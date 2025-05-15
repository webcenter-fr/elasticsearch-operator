package elasticsearchapi

import (
	"context"
	"encoding/json"
	"time"

	"emperror.dev/errors"
	eshandler "github.com/disaster37/es-handler/v8"
	"github.com/disaster37/generic-objectmatcher/patch"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/controller/remote"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/helper"
	olivere "github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearchapi/v1"
	core "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type licenseReconciler struct {
	remote.RemoteReconcilerAction[*elasticsearchapicrd.License, *olivere.XPackInfoLicense, eshandler.ElasticsearchHandler]
	name string
}

func newLicenseReconciler(name string, client client.Client, recorder record.EventRecorder) remote.RemoteReconcilerAction[*elasticsearchapicrd.License, *olivere.XPackInfoLicense, eshandler.ElasticsearchHandler] {
	return &licenseReconciler{
		RemoteReconcilerAction: remote.NewRemoteReconcilerAction[*elasticsearchapicrd.License, *olivere.XPackInfoLicense, eshandler.ElasticsearchHandler](
			client,
			recorder,
		),
		name: name,
	}
}

func (h *licenseReconciler) GetRemoteHandler(ctx context.Context, req reconcile.Request, o *elasticsearchapicrd.License, logger *logrus.Entry) (handler remote.RemoteExternalReconciler[*elasticsearchapicrd.License, *olivere.XPackInfoLicense, eshandler.ElasticsearchHandler], res reconcile.Result, err error) {
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

	handler = newLicenseApiClient(esClient)

	return handler, res, nil
}

func (h *licenseReconciler) Read(ctx context.Context, o *elasticsearchapicrd.License, data map[string]any, handler remote.RemoteExternalReconciler[*elasticsearchapicrd.License, *olivere.XPackInfoLicense, eshandler.ElasticsearchHandler], logger *logrus.Entry) (read remote.RemoteRead[*olivere.XPackInfoLicense], res reconcile.Result, err error) {

	read, res, err = h.RemoteReconcilerAction.Read(ctx, o, data, handler, logger)
	if err != nil {
		return nil, res, err
	}

	// If not basic license, the license is stored on secret
	if !o.IsBasicLicense() {
		if o.Spec.SecretRef == nil {
			return nil, res, errors.New("You must set the secretRef to get license")
		}
		secret := &core.Secret{}
		secretNS := types.NamespacedName{
			Namespace: o.Namespace,
			Name:      o.Spec.SecretRef.Name,
		}
		if err = h.Client().Get(ctx, secretNS, secret); err != nil {
			if k8serrors.IsNotFound(err) {
				logger.Warnf("Secret %s not yet exist, try later", o.Spec.SecretRef.Name)
				h.Recorder().Eventf(o, core.EventTypeWarning, "Failed", "Secret %s not yet exist", o.Spec.SecretRef.Name)
				return read, reconcile.Result{RequeueAfter: 30 * time.Second}, nil
			}
			return nil, res, errors.Wrapf(err, "Error when get secret %s", o.Spec.SecretRef.Name)
		}
		licenseB, ok := secret.Data["license"]
		if !ok {
			return nil, res, errors.Wrapf(err, "Secret %s must have a license key", o.Spec.SecretRef.Name)
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

func (h *licenseReconciler) Create(ctx context.Context, o *elasticsearchapicrd.License, data map[string]any, handler remote.RemoteExternalReconciler[*elasticsearchapicrd.License, *olivere.XPackInfoLicense, eshandler.ElasticsearchHandler], object *olivere.XPackInfoLicense, logger *logrus.Entry) (res reconcile.Result, err error) {

	if o.IsBasicLicense() {

		if err = handler.Client().LicenseEnableBasic(); err != nil {
			return res, errors.Wrap(err, "Error when activate basic license")
		}

		logger.Info("Successfully enable basic license")
		h.Recorder().Event(o, core.EventTypeNormal, "Completed", "Enable basic license")
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

		logger.Infof("Successfully enable %s license", object.Type)
		h.Recorder().Eventf(o, core.EventTypeNormal, "Completed", "Enable %s license", object.Type)
	}

	return res, nil
}

func (h *licenseReconciler) Update(ctx context.Context, o *elasticsearchapicrd.License, data map[string]any, handler remote.RemoteExternalReconciler[*elasticsearchapicrd.License, *olivere.XPackInfoLicense, eshandler.ElasticsearchHandler], object *olivere.XPackInfoLicense, logger *logrus.Entry) (res reconcile.Result, err error) {
	return h.Create(ctx, o, data, handler, object, logger)
}

func (h *licenseReconciler) Diff(ctx context.Context, o *elasticsearchapicrd.License, read remote.RemoteRead[*olivere.XPackInfoLicense], data map[string]any, handler remote.RemoteExternalReconciler[*elasticsearchapicrd.License, *olivere.XPackInfoLicense, eshandler.ElasticsearchHandler], logger *logrus.Entry, ignoreDiff ...patch.CalculateOption) (diff remote.RemoteDiff[*olivere.XPackInfoLicense], res reconcile.Result, err error) {
	diff = remote.NewRemoteDiff[*olivere.XPackInfoLicense]()

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

func (h *licenseReconciler) OnSuccess(ctx context.Context, o *elasticsearchapicrd.License, data map[string]any, handler remote.RemoteExternalReconciler[*elasticsearchapicrd.License, *olivere.XPackInfoLicense, eshandler.ElasticsearchHandler], diff remote.RemoteDiff[*olivere.XPackInfoLicense], logger *logrus.Entry) (res reconcile.Result, err error) {

	if o.IsBasicLicense() {
		o.Status.LicenseType = "basic"
		o.Status.ExpireAt = ""
	} else {
		d, err := helper.Get(data, "license")
		if err != nil {
			return res, err
		}
		l := d.(*olivere.XPackInfoLicense)
		o.Status.ExpireAt = time.UnixMilli(int64(l.ExpiryMilis)).Format(time.RFC3339)
		o.Status.LicenseType = l.Type
	}

	return h.RemoteReconcilerAction.OnSuccess(ctx, o, data, handler, diff, logger)
}
