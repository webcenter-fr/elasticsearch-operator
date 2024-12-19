package elasticsearchapi

import (
	"encoding/json"

	"emperror.dev/errors"
	eshandler "github.com/disaster37/es-handler/v8"
	"github.com/disaster37/generic-objectmatcher/patch"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	olivere "github.com/olivere/elastic/v7"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearchapi/v1"
)

type snapshotRepositoryApiClient struct {
	*controller.BasicRemoteExternalReconciler[*elasticsearchapicrd.SnapshotRepository, *olivere.SnapshotRepositoryMetaData, eshandler.ElasticsearchHandler]
}

func newSnapshotRepositoryApiClient(client eshandler.ElasticsearchHandler) controller.RemoteExternalReconciler[*elasticsearchapicrd.SnapshotRepository, *olivere.SnapshotRepositoryMetaData, eshandler.ElasticsearchHandler] {
	return &snapshotRepositoryApiClient{
		BasicRemoteExternalReconciler: controller.NewBasicRemoteExternalReconciler[*elasticsearchapicrd.SnapshotRepository, *olivere.SnapshotRepositoryMetaData, eshandler.ElasticsearchHandler](client),
	}
}

func (h *snapshotRepositoryApiClient) Build(o *elasticsearchapicrd.SnapshotRepository) (sr *olivere.SnapshotRepositoryMetaData, err error) {
	settings := map[string]any{}

	if o.Spec.Settings != "" {
		if err = json.Unmarshal([]byte(o.Spec.Settings), &settings); err != nil {
			return nil, errors.Wrap(err, "Unable to generate snapshot repository")
		}
	}

	sr = &olivere.SnapshotRepositoryMetaData{
		Type:     o.Spec.Type,
		Settings: settings,
	}

	return sr, nil
}

func (h *snapshotRepositoryApiClient) Get(o *elasticsearchapicrd.SnapshotRepository) (object *olivere.SnapshotRepositoryMetaData, err error) {
	return h.Client().SnapshotRepositoryGet(o.GetExternalName())
}

func (h *snapshotRepositoryApiClient) Create(object *olivere.SnapshotRepositoryMetaData, o *elasticsearchapicrd.SnapshotRepository) (err error) {
	return h.Client().SnapshotRepositoryUpdate(o.GetExternalName(), object)
}

func (h *snapshotRepositoryApiClient) Update(object *olivere.SnapshotRepositoryMetaData, o *elasticsearchapicrd.SnapshotRepository) (err error) {
	return h.Client().SnapshotRepositoryUpdate(o.GetExternalName(), object)
}

func (h *snapshotRepositoryApiClient) Delete(o *elasticsearchapicrd.SnapshotRepository) (err error) {
	return h.Client().SnapshotRepositoryDelete(o.GetExternalName())
}

func (h *snapshotRepositoryApiClient) Diff(currentOject *olivere.SnapshotRepositoryMetaData, expectedObject *olivere.SnapshotRepositoryMetaData, originalObject *olivere.SnapshotRepositoryMetaData, ignoresDiff ...patch.CalculateOption) (patchResult *patch.PatchResult, err error) {
	return h.Client().SnapshotRepositoryDiff(currentOject, expectedObject, originalObject)
}
