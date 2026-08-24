package elasticsearchapi

import (
	"emperror.dev/errors"
	esapi "github.com/disaster37/elasticsearch/v9/api"
	eshandler "github.com/disaster37/es-handler/v9"
	"github.com/disaster37/generic-objectmatcher/patch"
	"github.com/disaster37/operator-sdk-extra/v3/pkg/controller/remote"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearchapi/v1"
)

type licenseApiClient struct {
	remote.RemoteExternalReconciler[*elasticsearchapicrd.License, *esapi.LicenseInfo, eshandler.ElasticsearchHandler]
}

func newLicenseApiClient(client eshandler.ElasticsearchHandler) remote.RemoteExternalReconciler[*elasticsearchapicrd.License, *esapi.LicenseInfo, eshandler.ElasticsearchHandler] {
	return &licenseApiClient{
		RemoteExternalReconciler: remote.NewRemoteExternalReconciler[*elasticsearchapicrd.License, *esapi.LicenseInfo, eshandler.ElasticsearchHandler](client),
	}
}

func (h *licenseApiClient) Build(o *elasticsearchapicrd.License) (license *esapi.LicenseInfo, err error) {
	return license, err
}

func (h *licenseApiClient) Get(o *elasticsearchapicrd.License) (object *esapi.LicenseInfo, err error) {
	return h.Client().LicenseGet()
}

func (h *licenseApiClient) Create(object *esapi.LicenseInfo, o *elasticsearchapicrd.License) (err error) {
	return nil
}

func (h *licenseApiClient) Update(object *esapi.LicenseInfo, o *elasticsearchapicrd.License) (err error) {
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

func (h *licenseApiClient) Diff(currentOject *esapi.LicenseInfo, expectedObject *esapi.LicenseInfo, originalObject *esapi.LicenseInfo, o *elasticsearchapicrd.License, ignoresDiff ...patch.CalculateOption) (patchResult *patch.PatchResult, err error) {
	return nil, nil
}
