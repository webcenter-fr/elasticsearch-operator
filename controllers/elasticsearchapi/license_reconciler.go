package elasticsearchapi

import (
	"context"
	"encoding/json"
	"time"

	"emperror.dev/errors"
	eshandler "github.com/disaster37/es-handler/v8"
	"github.com/disaster37/generic-objectmatcher/patch"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/pkg/helper"
	"github.com/disaster37/operator-sdk-extra/pkg/object"
	olivere "github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearchapi/v1"
	core "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type licenseReconciler struct {
	controller.RemoteReconcilerAction[*elasticsearchapicrd.License, *olivere.XPackInfoLicense, eshandler.ElasticsearchHandler]
	controller.BaseReconciler
}

func newLicenseReconciler(client client.Client, logger *logrus.Entry, recorder record.EventRecorder) controller.RemoteReconcilerAction[*elasticsearchapicrd.License, *olivere.XPackInfoLicense, eshandler.ElasticsearchHandler] {
	return &licenseReconciler{
		RemoteReconcilerAction: controller.NewRemoteReconcilerAction[*elasticsearchapicrd.License, *olivere.XPackInfoLicense, eshandler.ElasticsearchHandler](
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

func (h *licenseReconciler) GetRemoteHandler(ctx context.Context, req ctrl.Request, o object.RemoteObject) (handler controller.RemoteExternalReconciler[*elasticsearchapicrd.License, *olivere.XPackInfoLicense, eshandler.ElasticsearchHandler], res ctrl.Result, err error) {
	license := o.(*elasticsearchapicrd.License)
	esClient, err := GetElasticsearchHandler(ctx, license, license.Spec.ElasticsearchRef, h.BaseReconciler.Client, h.BaseReconciler.Log)
	if err != nil && license.DeletionTimestamp.IsZero() {
		return nil, res, err
	}

	// Elastic not ready
	if esClient == nil {
		return nil, ctrl.Result{RequeueAfter: 60 * time.Second}, nil
	}

	handler = newLicenseApiClient(esClient)

	return handler, res, nil
}

func (h *licenseReconciler) Read(ctx context.Context, o object.RemoteObject, data map[string]any, handler controller.RemoteExternalReconciler[*elasticsearchapicrd.License, *olivere.XPackInfoLicense, eshandler.ElasticsearchHandler]) (read controller.RemoteRead[*olivere.XPackInfoLicense], res ctrl.Result, err error) {
	license := o.(*elasticsearchapicrd.License)

	read, res, err = h.RemoteReconcilerAction.Read(ctx, o, data, handler)
	if err != nil {
		return nil, res, err
	}

	// If not basic license, the license is stored on secret
	if !license.IsBasicLicense() {
		if license.Spec.SecretRef == nil {
			return nil, res, errors.New("You must set the secretRef to get license")
		}
		secret := &core.Secret{}
		secretNS := types.NamespacedName{
			Namespace: license.Namespace,
			Name:      license.Spec.SecretRef.Name,
		}
		if err = h.Get(ctx, secretNS, secret); err != nil {
			if k8serrors.IsNotFound(err) {
				h.Log.Warnf("Secret %s not yet exist, try later", license.Spec.SecretRef.Name)
				h.Recorder.Eventf(o, core.EventTypeWarning, "Failed", "Secret %s not yet exist", license.Spec.SecretRef.Name)
				return read, ctrl.Result{RequeueAfter: 30 * time.Second}, nil
			}
			return nil, res, errors.Wrapf(err, "Error when get secret %s", license.Spec.SecretRef.Name)
		}
		licenseB, ok := secret.Data["license"]
		if !ok {
			return nil, res, errors.Wrapf(err, "Secret %s must have a license key", license.Spec.SecretRef.Name)
		}
		expectedLicense := &olivere.XPackInfoServiceResponse{}
		if err = json.Unmarshal(licenseB, expectedLicense); err != nil {
			return nil, res, errors.Wrap(err, "License contend is invalid")
		}
		read.SetExpectedObject(&expectedLicense.License)
		data["rawLicense"] = string(licenseB)
		data["license"] = &expectedLicense.License
	} else {
		read.SetExpectedObject(&olivere.XPackInfoLicense{
			Type: "basic",
		})
	}

	return read, res, nil
}

func (h *licenseReconciler) Create(ctx context.Context, o object.RemoteObject, data map[string]any, handler controller.RemoteExternalReconciler[*elasticsearchapicrd.License, *olivere.XPackInfoLicense, eshandler.ElasticsearchHandler], object *olivere.XPackInfoLicense) (res ctrl.Result, err error) {
	license := o.(*elasticsearchapicrd.License)

	if license.IsBasicLicense() {

		if err = handler.Client().LicenseEnableBasic(); err != nil {
			return res, errors.Wrap(err, "Error when activate basic license")
		}

		h.Log.Info("Successfully enable basic license")
		h.Recorder.Event(o, core.EventTypeNormal, "Completed", "Enable basic license")
	} else {
		// Enterprise or platinium license
		d, err := helper.Get(data, "rawLicense")
		if err != nil {
			return res, err
		}
		rawLicense := d.(string)

		if err = handler.Client().LicenseUpdate(rawLicense); err != nil {
			return res, errors.Wrap(err, "Error when add enterprise license on Elasticsearch")
		}

		h.Log.Infof("Successfully enable %s license", object.Type)
		h.Recorder.Eventf(o, core.EventTypeNormal, "Completed", "Enable %s license", object.Type)
	}

	return res, nil
}

func (h *licenseReconciler) Update(ctx context.Context, o object.RemoteObject, data map[string]any, handler controller.RemoteExternalReconciler[*elasticsearchapicrd.License, *olivere.XPackInfoLicense, eshandler.ElasticsearchHandler], object *olivere.XPackInfoLicense) (res ctrl.Result, err error) {
	return h.Create(ctx, o, data, handler, object)
}

func (h *licenseReconciler) Diff(ctx context.Context, o object.RemoteObject, read controller.RemoteRead[*olivere.XPackInfoLicense], data map[string]any, handler controller.RemoteExternalReconciler[*elasticsearchapicrd.License, *olivere.XPackInfoLicense, eshandler.ElasticsearchHandler], ignoreDiff ...patch.CalculateOption) (diff controller.RemoteDiff[*olivere.XPackInfoLicense], res ctrl.Result, err error) {
	diff = controller.NewBasicRemoteDiff[*olivere.XPackInfoLicense]()

	// Not yet license
	if read.GetCurrentObject() == nil {
		diff.AddDiff("Add new license")
		diff.SetObjectToCreate(read.GetExpectedObject())
		return diff, res, nil
	}

	if handler.Client().LicenseDiff(read.GetCurrentObject(), read.GetExpectedObject()) {
		diff.AddDiff("Update the current license")
		diff.SetObjectToUpdate(read.GetExpectedObject())
	}

	return diff, res, nil
}

func (h *licenseReconciler) OnSuccess(ctx context.Context, o object.RemoteObject, data map[string]any, handler controller.RemoteExternalReconciler[*elasticsearchapicrd.License, *olivere.XPackInfoLicense, eshandler.ElasticsearchHandler], diff controller.RemoteDiff[*olivere.XPackInfoLicense]) (res ctrl.Result, err error) {
	license := o.(*elasticsearchapicrd.License)

	if license.IsBasicLicense() {
		license.Status.LicenseType = "basic"
		license.Status.ExpireAt = ""
	} else {
		d, err := helper.Get(data, "license")
		if err != nil {
			return res, err
		}
		l := d.(*olivere.XPackInfoLicense)
		license.Status.ExpireAt = time.UnixMilli(int64(l.ExpiryMilis)).Format(time.RFC3339)
		license.Status.LicenseType = l.Type
	}

	return h.RemoteReconcilerAction.OnSuccess(ctx, license, data, handler, diff)

}
