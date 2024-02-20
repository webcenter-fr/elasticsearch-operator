package elasticsearchapi

import (
	"context"
	"time"

	"emperror.dev/errors"
	eshandler "github.com/disaster37/es-handler/v8"
	"github.com/disaster37/generic-objectmatcher/patch"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/pkg/object"
	olivere "github.com/olivere/elastic/v7"
	"github.com/sethvargo/go-password/password"
	"github.com/sirupsen/logrus"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearchapi/v1"
	localhelper "github.com/webcenter-fr/elasticsearch-operator/pkg/helper"
	core "k8s.io/api/core/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type userReconciler struct {
	controller.RemoteReconcilerAction[*elasticsearchapicrd.User, *olivere.XPackSecurityPutUserRequest, eshandler.ElasticsearchHandler]
	controller.BaseReconciler
}

func newUserReconciler(client client.Client, logger *logrus.Entry, recorder record.EventRecorder) controller.RemoteReconcilerAction[*elasticsearchapicrd.User, *olivere.XPackSecurityPutUserRequest, eshandler.ElasticsearchHandler] {
	return &userReconciler{
		RemoteReconcilerAction: controller.NewRemoteReconcilerAction[*elasticsearchapicrd.User, *olivere.XPackSecurityPutUserRequest, eshandler.ElasticsearchHandler](
			client,
			logger,
			recorder,
		),
		BaseReconciler: controller.BaseReconciler{
			Client:   client,
			Log:      logger,
			Recorder: recorder,
		},
	}
}

func (h *userReconciler) GetRemoteHandler(ctx context.Context, req ctrl.Request, o object.RemoteObject) (handler controller.RemoteExternalReconciler[*elasticsearchapicrd.User, *olivere.XPackSecurityPutUserRequest, eshandler.ElasticsearchHandler], res ctrl.Result, err error) {
	user := o.(*elasticsearchapicrd.User)
	esClient, err := GetElasticsearchHandler(ctx, user, user.Spec.ElasticsearchRef, h.BaseReconciler.Client, h.BaseReconciler.Log)
	if err != nil && user.DeletionTimestamp.IsZero() {
		return nil, res, err
	}

	// Elastic not ready
	if esClient == nil {
		return nil, ctrl.Result{RequeueAfter: 60 * time.Second}, nil
	}

	handler = newUserApiClient(esClient)

	return handler, res, nil
}

func (h *userReconciler) Read(ctx context.Context, o object.RemoteObject, data map[string]any, handler controller.RemoteExternalReconciler[*elasticsearchapicrd.User, *olivere.XPackSecurityPutUserRequest, eshandler.ElasticsearchHandler]) (read controller.RemoteRead[*olivere.XPackSecurityPutUserRequest], res ctrl.Result, err error) {
	read, res, err = h.RemoteReconcilerAction.Read(ctx, o, data, handler)
	if err != nil {
		return nil, res, err
	}

	user := o.(*elasticsearchapicrd.User)

	// Read password from secret if needed and inject it on expected user
	if !user.IsAutoGeneratePassword() && user.Spec.SecretRef != nil {
		secret := &core.Secret{}
		secretNS := types.NamespacedName{
			Namespace: user.Namespace,
			Name:      user.Spec.SecretRef.Name,
		}
		if err = h.Get(ctx, secretNS, secret); err != nil {
			if k8serrors.IsNotFound(err) {
				h.Log.Warnf("Secret %s not yet exist, try later", user.Spec.SecretRef.Name)
				h.Recorder.Eventf(o, core.EventTypeWarning, "Failed", "Secret %s not yet exist", user.Spec.SecretRef.Name)
				return nil, ctrl.Result{RequeueAfter: 30 * time.Second}, nil
			}
			return nil, res, errors.Wrapf(err, "Error when get secret %s", user.Spec.SecretRef.Name)
		}
		passwordB, ok := secret.Data[user.Spec.SecretRef.Key]
		if !ok {
			return nil, res, errors.Wrapf(err, "Secret %s must have a %s key", user.Spec.SecretRef.Name, user.Spec.SecretRef.Key)
		}

		read.GetExpectedObject().Password = string(passwordB)
		read.GetExpectedObject().PasswordHash = ""
	} else if user.IsAutoGeneratePassword() {
		secret := &corev1.Secret{}
		var expectedPassword string
		if err = h.Get(ctx, types.NamespacedName{Namespace: user.Namespace, Name: GetUserSecretWhenAutoGeneratePassword(user)}, secret); err != nil {
			if k8serrors.IsNotFound(err) {
				// Create secret and generate password
				expectedPassword, err = password.Generate(64, 10, 0, false, true)
				if err != nil {
					return nil, res, errors.Wrap(err, "Error when generate password")
				}
				secret = &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      GetUserSecretWhenAutoGeneratePassword(user),
						Namespace: user.Namespace,
					},
					Data: map[string][]byte{
						"password": []byte(expectedPassword),
						"username": []byte(user.GetExternalName()),
					},
				}
				// Set owner
				err = ctrl.SetControllerReference(o, secret, h.Client.Scheme())
				if err != nil {
					return nil, res, errors.Wrapf(err, "Error when set owner reference on object '%s'", secret.GetName())
				}
				if err = h.Client.Create(ctx, secret); err != nil {
					return nil, res, errors.Wrap(err, "Error when create secret that store auto generated password")
				}

			}
		}
		if len(secret.Data["password"]) == 0 || len(secret.Data["username"]) == 0 {
			// The password entry not exist, create it
			expectedPassword, err = password.Generate(64, 10, 0, false, true)
			if err != nil {
				return nil, res, errors.Wrap(err, "Error when generate password")
			}
			secret.Data["password"] = []byte(expectedPassword)
			secret.Data["username"] = []byte(user.GetExternalName())
			if err = h.Client.Update(ctx, secret); err != nil {
				return nil, res, errors.Wrap(err, "Error when update secret that store auto generated password")
			}

		} else {
			expectedPassword = string(secret.Data["password"])
		}

		read.GetExpectedObject().Password = expectedPassword
		read.GetExpectedObject().PasswordHash = ""
	}

	return read, res, nil

}

func (h *userReconciler) Delete(ctx context.Context, o object.RemoteObject, data map[string]any, handler controller.RemoteExternalReconciler[*elasticsearchapicrd.User, *olivere.XPackSecurityPutUserRequest, eshandler.ElasticsearchHandler]) (err error) {
	user := o.(*elasticsearchapicrd.User)

	if user.IsProtected() {
		return nil
	}

	return h.RemoteReconcilerAction.Delete(ctx, o, data, handler)
}

func (h *userReconciler) OnSuccess(ctx context.Context, o object.RemoteObject, data map[string]any, handler controller.RemoteExternalReconciler[*elasticsearchapicrd.User, *olivere.XPackSecurityPutUserRequest, eshandler.ElasticsearchHandler], diff controller.RemoteDiff[*olivere.XPackSecurityPutUserRequest]) (res ctrl.Result, err error) {
	// Update passwordHash if needed on status
	if diff.NeedCreate() || diff.NeedUpdate() {
		user := o.(*elasticsearchapicrd.User)
		var passwordHash string

		if user.IsAutoGeneratePassword() || user.Spec.SecretRef != nil {
			if diff.NeedCreate() && diff.GetObjectToCreate().Password != "" {
				passwordHash, err = localhelper.HashPassword(diff.GetObjectToCreate().Password)
				if err != nil {
					return res, errors.Wrap(err, "Error when hash password")
				}
			}

			if diff.NeedUpdate() && diff.GetObjectToUpdate().Password != "" {
				passwordHash, err = localhelper.HashPassword(diff.GetObjectToUpdate().Password)
				if err != nil {
					return res, errors.Wrap(err, "Error when hash password")
				}
			}

		} else if user.Spec.PasswordHash != user.Status.PasswordHash {
			passwordHash = user.Spec.PasswordHash
		}

		if passwordHash != "" {
			user.Status.PasswordHash = passwordHash
		}
	}

	return h.RemoteReconcilerAction.OnSuccess(ctx, o, data, handler, diff)
}

func (h *userReconciler) Diff(ctx context.Context, o object.RemoteObject, read controller.RemoteRead[*olivere.XPackSecurityPutUserRequest], data map[string]any, handler controller.RemoteExternalReconciler[*elasticsearchapicrd.User, *olivere.XPackSecurityPutUserRequest, eshandler.ElasticsearchHandler], ignoreDiff ...patch.CalculateOption) (diff controller.RemoteDiff[*olivere.XPackSecurityPutUserRequest], res ctrl.Result, err error) {
	user := o.(*elasticsearchapicrd.User)

	var currentUser *olivere.XPackSecurityPutUserRequest

	// If it is protected user, only manage the password
	if user.IsProtected() {
		currentUser = &olivere.XPackSecurityPutUserRequest{
			Enabled:      read.GetCurrentObject().Enabled,
			Password:     read.GetCurrentObject().Password,
			PasswordHash: read.GetCurrentObject().PasswordHash,
		}

		read.SetCurrentObject(currentUser)
	}

	if read.GetExpectedObject().Password != "" {
		// Password is provided by secret
		if localhelper.CheckPasswordHash(read.GetExpectedObject().Password, user.Status.PasswordHash) {
			read.GetExpectedObject().Password = ""
			read.GetExpectedObject().PasswordHash = ""
		}
	} else if user.Spec.PasswordHash == user.Status.PasswordHash {
		// Password hash is provided
		read.GetExpectedObject().Password = ""
		read.GetExpectedObject().PasswordHash = ""
	}

	return h.RemoteReconcilerAction.Diff(ctx, o, read, data, handler, ignoreDiff...)

}
