package elasticsearchapi

import (
	"emperror.dev/errors"
	eshandler "github.com/disaster37/es-handler/v8"
	"github.com/disaster37/generic-objectmatcher/patch"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	olivere "github.com/olivere/elastic/v7"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearchapi/v1"
)

type licenseApiClient struct {
	*controller.BasicRemoteExternalReconciler[*elasticsearchapicrd.License, *olivere.XPackInfoLicense, eshandler.ElasticsearchHandler]
}

func newLicenseApiClient(client eshandler.ElasticsearchHandler) controller.RemoteExternalReconciler[*elasticsearchapicrd.License, *olivere.XPackInfoLicense, eshandler.ElasticsearchHandler] {
	return &licenseApiClient{
		BasicRemoteExternalReconciler: controller.NewBasicRemoteExternalReconciler[*elasticsearchapicrd.License, *olivere.XPackInfoLicense, eshandler.ElasticsearchHandler](client),
	}
}

func (h *licenseApiClient) Build(o *elasticsearchapicrd.License) (license *olivere.XPackInfoLicense, err error) {
	return license, err
}

func (h *licenseApiClient) Get(o *elasticsearchapicrd.License) (object *olivere.XPackInfoLicense, err error) {
	return h.Client().LicenseGet()
}

func (h *licenseApiClient) Create(object *olivere.XPackInfoLicense, o *elasticsearchapicrd.License) (err error) {
	return nil
}

func (h *licenseApiClient) Update(object *olivere.XPackInfoLicense, o *elasticsearchapicrd.License) (err error) {
	return nil
}

func (h *licenseApiClient) Delete(o *elasticsearchapicrd.License) (err error) {
	if !o.IsBasicLicense() {
		if err = h.Client().LicenseEnableBasic(); err != nil {
			return errors.Wrap(err, "Error when downgrade to basic license")
		}
	}

	return nil
}

func (h *licenseApiClient) Diff(currentOject *olivere.XPackInfoLicense, expectedObject *olivere.XPackInfoLicense, originalObject *olivere.XPackInfoLicense, ignoresDiff ...patch.CalculateOption) (patchResult *patch.PatchResult, err error) {
	return nil, nil
}
