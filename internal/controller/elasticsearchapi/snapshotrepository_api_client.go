package elasticsearchapi

import (
	"fmt"

	esapi "github.com/disaster37/elasticsearch/v9/api"
	eshandler "github.com/disaster37/es-handler/v9"
	"github.com/disaster37/generic-objectmatcher/patch"
	"github.com/disaster37/operator-sdk-extra/v3/pkg/controller/remote"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearchapi/v1"
)

type snapshotRepositoryApiClient struct {
	remote.RemoteExternalReconciler[*elasticsearchapicrd.SnapshotRepository, *esapi.SnapshotRepository, eshandler.ElasticsearchHandler]
}

func newSnapshotRepositoryApiClient(client eshandler.ElasticsearchHandler) remote.RemoteExternalReconciler[*elasticsearchapicrd.SnapshotRepository, *esapi.SnapshotRepository, eshandler.ElasticsearchHandler] {
	return &snapshotRepositoryApiClient{
		RemoteExternalReconciler: remote.NewRemoteExternalReconciler[*elasticsearchapicrd.SnapshotRepository, *esapi.SnapshotRepository, eshandler.ElasticsearchHandler](client),
	}
}

func (h *snapshotRepositoryApiClient) Build(o *elasticsearchapicrd.SnapshotRepository) (sr *esapi.SnapshotRepository, err error) {
	sr = &esapi.SnapshotRepository{
		Type: o.Spec.Type,
	}

	if o.Spec.Settings != nil {
		sr.Settings = make(map[string]string, len(o.Spec.Settings.Data))
		for k, v := range o.Spec.Settings.Data {
			sr.Settings[k] = fmt.Sprint(v)
		}
	}

	return sr, nil
}

func (h *snapshotRepositoryApiClient) Get(o *elasticsearchapicrd.SnapshotRepository) (object *esapi.SnapshotRepository, err error) {
	return h.Client().SnapshotRepositoryGet(o.GetExternalName())
}

func (h *snapshotRepositoryApiClient) Create(object *esapi.SnapshotRepository, o *elasticsearchapicrd.SnapshotRepository) (err error) {
	return h.Client().SnapshotRepositoryUpdate(o.GetExternalName(), object)
}

func (h *snapshotRepositoryApiClient) Update(object *esapi.SnapshotRepository, o *elasticsearchapicrd.SnapshotRepository) (err error) {
	return h.Client().SnapshotRepositoryUpdate(o.GetExternalName(), object)
}

func (h *snapshotRepositoryApiClient) Delete(o *elasticsearchapicrd.SnapshotRepository) (err error) {
	return h.Client().SnapshotRepositoryDelete(o.GetExternalName())
}

func (h *snapshotRepositoryApiClient) Diff(currentOject *esapi.SnapshotRepository, expectedObject *esapi.SnapshotRepository, originalObject *esapi.SnapshotRepository, o *elasticsearchapicrd.SnapshotRepository, ignoresDiff ...patch.CalculateOption) (patchResult *patch.PatchResult, err error) {
	return h.Client().SnapshotRepositoryDiff(currentOject, expectedObject, originalObject)
}
