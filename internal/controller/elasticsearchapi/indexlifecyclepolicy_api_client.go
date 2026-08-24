package elasticsearchapi

import (
	"encoding/json"

	"emperror.dev/errors"
	esapi "github.com/disaster37/elasticsearch/v9/api"
	eshandler "github.com/disaster37/es-handler/v9"
	"github.com/disaster37/generic-objectmatcher/patch"
	"github.com/disaster37/operator-sdk-extra/v3/pkg/controller/remote"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearchapi/v1"
	"sigs.k8s.io/yaml"
)

type indexLifecyclePolicyApiClient struct {
	remote.RemoteExternalReconciler[*elasticsearchapicrd.IndexLifecyclePolicy, *esapi.IlmPolicy, eshandler.ElasticsearchHandler]
}

func newIndexLifecyclePolicyApiClient(client eshandler.ElasticsearchHandler) remote.RemoteExternalReconciler[*elasticsearchapicrd.IndexLifecyclePolicy, *esapi.IlmPolicy, eshandler.ElasticsearchHandler] {
	return &indexLifecyclePolicyApiClient{
		RemoteExternalReconciler: remote.NewRemoteExternalReconciler[*elasticsearchapicrd.IndexLifecyclePolicy, *esapi.IlmPolicy, eshandler.ElasticsearchHandler](client),
	}
}

func (h *indexLifecyclePolicyApiClient) Build(o *elasticsearchapicrd.IndexLifecyclePolicy) (ilm *esapi.IlmPolicy, err error) {
	if (o.Spec.RawPolicy != nil) && (*o.Spec.RawPolicy != "") {
		ilm = &esapi.IlmPolicy{}
		if err = json.Unmarshal([]byte(*o.Spec.RawPolicy), ilm); err != nil {
			return nil, errors.Wrap(err, "Unable to convert expected policy to object")
		}
	} else {
		policy := map[string]any{}
		b, err := yaml.Marshal(o.Spec.Policy)
		if err != nil {
			return nil, errors.Wrap(err, "Error when marshal policy")
		}
		if err = yaml.Unmarshal(b, &policy); err != nil {
			return nil, errors.Wrap(err, "Error when unmarshall policy")
		}
		ilm = &esapi.IlmPolicy{
			Policy: policy,
		}
	}

	return ilm, nil
}

func (h *indexLifecyclePolicyApiClient) Get(o *elasticsearchapicrd.IndexLifecyclePolicy) (object *esapi.IlmPolicy, err error) {
	return h.Client().ILMGet(o.GetExternalName())
}

func (h *indexLifecyclePolicyApiClient) Create(object *esapi.IlmPolicy, o *elasticsearchapicrd.IndexLifecyclePolicy) (err error) {
	return h.Client().ILMUpdate(o.GetExternalName(), object)
}

func (h *indexLifecyclePolicyApiClient) Update(object *esapi.IlmPolicy, o *elasticsearchapicrd.IndexLifecyclePolicy) (err error) {
	return h.Client().ILMUpdate(o.GetExternalName(), object)
}

func (h *indexLifecyclePolicyApiClient) Delete(o *elasticsearchapicrd.IndexLifecyclePolicy) (err error) {
	return h.Client().ILMDelete(o.GetExternalName())
}

func (h *indexLifecyclePolicyApiClient) Diff(currentOject *esapi.IlmPolicy, expectedObject *esapi.IlmPolicy, originalObject *esapi.IlmPolicy, o *elasticsearchapicrd.IndexLifecyclePolicy, ignoresDiff ...patch.CalculateOption) (patchResult *patch.PatchResult, err error) {
	return h.Client().ILMDiff(currentOject, expectedObject, originalObject)
}
