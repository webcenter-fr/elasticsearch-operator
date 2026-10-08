package elasticsearchapi

import (
	"context"
	"time"

	"emperror.dev/errors"
	eshandler "github.com/disaster37/es-handler/v9"
	"github.com/disaster37/generic-objectmatcher/patch"
	"github.com/disaster37/operator-sdk-extra/v3/pkg/controller/remote"
	"github.com/sethvargo/go-password/password"
	"github.com/sirupsen/logrus"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearchapi/v1"
	localhelper "github.com/webcenter-fr/elasticsearch-operator/pkg/helper"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type userReconciler struct {
	remote.RemoteReconcilerAction[*elasticsearchapicrd.User, *eshandler.SecurityPutUserRequest, eshandler.ElasticsearchHandler]
	name string
}

func newUserReconciler(name string, client client.Client, recorder record.EventRecorder) remote.RemoteReconcilerAction[*elasticsearchapicrd.User, *eshandler.SecurityPutUserRequest, eshandler.ElasticsearchHandler] {
	return &userReconciler{
		RemoteReconcilerAction: remote.NewRemoteReconcilerAction[*elasticsearchapicrd.User, *eshandler.SecurityPutUserRequest, eshandler.ElasticsearchHandler](
			client,
			recorder,
		),
		name: name,
	}
}

func (h *userReconciler) GetRemoteHandler(ctx context.Context, req reconcile.Request, o *elasticsearchapicrd.User, logger *logrus.Entry) (handler remote.RemoteExternalReconciler[*elasticsearchapicrd.User, *eshandler.SecurityPutUserRequest, eshandler.ElasticsearchHandler], res reconcile.Result, err error) {
	esClient, err := GetElasticsearchHandler(ctx, o, o.Spec.ElasticsearchRef, h.Client(), logger)
	if err != nil && o.DeletionTimestamp.IsZero() {
		return nil, res, err
	}

	// Elastic not ready
	if esClient == nil {
		if o.DeletionTimestamp.IsZero() {
			return nil, reconcile.Result{RequeueAfter: 60 * time.Second}, nil
		}

		return nil, res, nil
	}

	handler = newUserApiClient(esClient)

	return handler, res, nil
}

func (h *userReconciler) Read(ctx context.Context, o *elasticsearchapicrd.User, data map[string]any, handler remote.RemoteExternalReconciler[*elasticsearchapicrd.User, *eshandler.SecurityPutUserRequest, eshandler.ElasticsearchHandler], logger *logrus.Entry) (read remote.RemoteRead[*eshandler.SecurityPutUserRequest], res reconcile.Result, err error) {
	read, res, err = h.RemoteReconcilerAction.Read(ctx, o, data, handler, logger)
	if err != nil {
		return nil, res, err
	}

	// Read password from secret if needed and inject it on expected user
	if !o.IsAutoGeneratePassword() && o.Spec.SecretRef != nil {
		secret := &corev1.Secret{}
		secretNS := types.NamespacedName{
			Namespace: o.Namespace,
			Name:      o.Spec.SecretRef.Name,
		}
		if err = h.Client().Get(ctx, secretNS, secret); err != nil {
			if k8serrors.IsNotFound(err) {
				logger.Warnf("Secret %s not yet exist, try later", o.Spec.SecretRef.Name)
				h.Recorder().Eventf(o, corev1.EventTypeWarning, "Failed", "Secret %s not yet exist", o.Spec.SecretRef.Name)
				return nil, reconcile.Result{RequeueAfter: 30 * time.Second}, nil
			}
			return nil, res, errors.Wrapf(err, "Error when get secret %s", o.Spec.SecretRef.Name)
		}
		passwordB, ok := secret.Data[o.Spec.SecretRef.Key]
		if !ok {
			return nil, res, errors.Wrapf(err, "Secret %s must have a %s key", o.Spec.SecretRef.Name, o.Spec.SecretRef.Key)
		}

		read.GetExpectedObject().Password = string(passwordB)
		read.GetExpectedObject().PasswordHash = ""
	} else if o.IsAutoGeneratePassword() {
		secret := &corev1.Secret{}
		var expectedPassword string
		if err = h.Client().Get(ctx, types.NamespacedName{Namespace: o.Namespace, Name: GetUserSecretWhenAutoGeneratePassword(o)}, secret); err != nil {
			if k8serrors.IsNotFound(err) {
				// Create secret and generate password
				expectedPassword, err = password.Generate(64, 10, 0, false, true)
				if err != nil {
					return nil, res, errors.Wrap(err, "Error when generate password")
				}
				secret = &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      GetUserSecretWhenAutoGeneratePassword(o),
						Namespace: o.Namespace,
					},
					Data: map[string][]byte{
						"password": []byte(expectedPassword),
						"username": []byte(o.GetExternalName()),
					},
				}
				// Set owner
				err = ctrl.SetControllerReference(o, secret, h.Client().Scheme())
				if err != nil {
					return nil, res, errors.Wrapf(err, "Error when set owner reference on object '%s'", secret.GetName())
				}
				if err = h.Client().Create(ctx, secret); err != nil {
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
			secret.Data["username"] = []byte(o.GetExternalName())
			if err = h.Client().Update(ctx, secret); err != nil {
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

func (h *userReconciler) Delete(ctx context.Context, o *elasticsearchapicrd.User, data map[string]any, handler remote.RemoteExternalReconciler[*elasticsearchapicrd.User, *eshandler.SecurityPutUserRequest, eshandler.ElasticsearchHandler], logger *logrus.Entry) (err error) {
	if o.IsProtected() {
		return nil
	}

	return h.RemoteReconcilerAction.Delete(ctx, o, data, handler, logger)
}

func (h *userReconciler) OnSuccess(ctx context.Context, o *elasticsearchapicrd.User, data map[string]any, handler remote.RemoteExternalReconciler[*elasticsearchapicrd.User, *eshandler.SecurityPutUserRequest, eshandler.ElasticsearchHandler], diff remote.RemoteDiff[*eshandler.SecurityPutUserRequest], logger *logrus.Entry) (res reconcile.Result, err error) {
	// Update passwordHash if needed on status
	if diff.NeedCreate() || diff.NeedUpdate() {
		var passwordHash string

		if o.IsAutoGeneratePassword() || o.Spec.SecretRef != nil {
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

		} else if o.Spec.PasswordHash != o.Status.PasswordHash {
			passwordHash = o.Spec.PasswordHash
		}

		if passwordHash != "" {
			o.Status.PasswordHash = passwordHash
		}
	}

	return h.RemoteReconcilerAction.OnSuccess(ctx, o, data, handler, diff, logger)
}

func (h *userReconciler) Diff(ctx context.Context, o *elasticsearchapicrd.User, read remote.RemoteRead[*eshandler.SecurityPutUserRequest], data map[string]any, handler remote.RemoteExternalReconciler[*elasticsearchapicrd.User, *eshandler.SecurityPutUserRequest, eshandler.ElasticsearchHandler], logger *logrus.Entry, ignoreDiff ...patch.CalculateOption) (diff remote.RemoteDiff[*eshandler.SecurityPutUserRequest], res reconcile.Result, err error) {
	var currentUser *eshandler.SecurityPutUserRequest

	// If it is protected user, only manage the password
	if o.IsProtected() {
		currentUser = &eshandler.SecurityPutUserRequest{
			Enabled:      read.GetCurrentObject().Enabled,
			Password:     read.GetCurrentObject().Password,
			PasswordHash: read.GetCurrentObject().PasswordHash,
		}

		read.SetCurrentObject(currentUser)
	}

	if read.GetExpectedObject().Password != "" {
		// Password is provided by secret
		if localhelper.CheckPasswordHash(read.GetExpectedObject().Password, o.Status.PasswordHash) {
			read.GetExpectedObject().Password = ""
			read.GetExpectedObject().PasswordHash = ""
		}
	} else if o.Spec.PasswordHash == o.Status.PasswordHash {
		// Password hash is provided
		read.GetExpectedObject().Password = ""
		read.GetExpectedObject().PasswordHash = ""
	}

	return h.RemoteReconcilerAction.Diff(ctx, o, read, data, handler, logger, ignoreDiff...)
}
