package elasticsearchapi

import (
	eshandler "github.com/disaster37/es-handler/v8"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearchapi/v1alpha1"
)

// BuildSnapshotLifecyclePolicy permit to build SLM policy
func BuildSnapshotLifecyclePolicy(o *elasticsearchapicrd.SnapshotLifecyclePolicy) (slm *eshandler.SnapshotLifecyclePolicySpec, err error) {

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
