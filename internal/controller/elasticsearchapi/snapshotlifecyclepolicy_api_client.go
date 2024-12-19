package elasticsearchapi

import (
	eshandler "github.com/disaster37/es-handler/v8"
	"github.com/disaster37/generic-objectmatcher/patch"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearchapi/v1"
)

type snapshotLifecyclePolicyApiClient struct {
	*controller.BasicRemoteExternalReconciler[*elasticsearchapicrd.SnapshotLifecyclePolicy, *eshandler.SnapshotLifecyclePolicySpec, eshandler.ElasticsearchHandler]
}

func newSnapshotLifecyclePolicyApiClient(client eshandler.ElasticsearchHandler) controller.RemoteExternalReconciler[*elasticsearchapicrd.SnapshotLifecyclePolicy, *eshandler.SnapshotLifecyclePolicySpec, eshandler.ElasticsearchHandler] {
	return &snapshotLifecyclePolicyApiClient{
		BasicRemoteExternalReconciler: controller.NewBasicRemoteExternalReconciler[*elasticsearchapicrd.SnapshotLifecyclePolicy, *eshandler.SnapshotLifecyclePolicySpec, eshandler.ElasticsearchHandler](client),
	}
}

func (h *snapshotLifecyclePolicyApiClient) Build(o *elasticsearchapicrd.SnapshotLifecyclePolicy) (slm *eshandler.SnapshotLifecyclePolicySpec, err error) {
	slm = &eshandler.SnapshotLifecyclePolicySpec{
		Schedule:   o.Spec.Schedule,
		Name:       o.Spec.Name,
		Repository: o.Spec.Repository,
		Config: eshandler.ElasticsearchSLMConfig{
			ExpendWildcards:    o.Spec.Config.ExpendWildcards,
			IgnoreUnavailable:  o.Spec.Config.IgnoreUnavailable,
			IncludeGlobalState: o.Spec.Config.IncludeGlobalState,
			Indices:            o.Spec.Config.Indices,
			FeatureStates:      o.Spec.Config.FeatureStates,
			Metadata:           o.Spec.Config.Metadata,
			Partial:            o.Spec.Config.Partial,
		},
	}

	if o.Spec.Retention != nil {
		slm.Retention = &eshandler.ElasticsearchSLMRetention{
			ExpireAfter: o.Spec.Retention.ExpireAfter,
			MaxCount:    o.Spec.Retention.MaxCount,
			MinCount:    o.Spec.Retention.MinCount,
		}
	}

	return slm, nil
}

func (h *snapshotLifecyclePolicyApiClient) Get(o *elasticsearchapicrd.SnapshotLifecyclePolicy) (object *eshandler.SnapshotLifecyclePolicySpec, err error) {
	return h.Client().SLMGet(o.GetExternalName())
}

func (h *snapshotLifecyclePolicyApiClient) Create(object *eshandler.SnapshotLifecyclePolicySpec, o *elasticsearchapicrd.SnapshotLifecyclePolicy) (err error) {
	return h.Client().SLMUpdate(o.GetExternalName(), object)
}

func (h *snapshotLifecyclePolicyApiClient) Update(object *eshandler.SnapshotLifecyclePolicySpec, o *elasticsearchapicrd.SnapshotLifecyclePolicy) (err error) {
	return h.Client().SLMUpdate(o.GetExternalName(), object)
}

func (h *snapshotLifecyclePolicyApiClient) Delete(o *elasticsearchapicrd.SnapshotLifecyclePolicy) (err error) {
	return h.Client().SLMDelete(o.GetExternalName())
}

func (h *snapshotLifecyclePolicyApiClient) Diff(currentOject *eshandler.SnapshotLifecyclePolicySpec, expectedObject *eshandler.SnapshotLifecyclePolicySpec, originalObject *eshandler.SnapshotLifecyclePolicySpec, ignoresDiff ...patch.CalculateOption) (patchResult *patch.PatchResult, err error) {
	return h.Client().SLMDiff(currentOject, expectedObject, originalObject)
}
