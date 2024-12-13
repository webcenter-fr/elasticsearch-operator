package elasticsearchapi

import (
	"encoding/json"

	"emperror.dev/errors"
	eshandler "github.com/disaster37/es-handler/v8"
	"github.com/disaster37/generic-objectmatcher/patch"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	olivere "github.com/olivere/elastic/v7"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearchapi/v1"
)

type indexLifecyclePolicyApiClient struct {
	*controller.BasicRemoteExternalReconciler[*elasticsearchapicrd.IndexLifecyclePolicy, *olivere.XPackIlmGetLifecycleResponse, eshandler.ElasticsearchHandler]
}

func newIndexLifecyclePolicyApiClient(client eshandler.ElasticsearchHandler) controller.RemoteExternalReconciler[*elasticsearchapicrd.IndexLifecyclePolicy, *olivere.XPackIlmGetLifecycleResponse, eshandler.ElasticsearchHandler] {
	return &indexLifecyclePolicyApiClient{
		BasicRemoteExternalReconciler: controller.NewBasicRemoteExternalReconciler[*elasticsearchapicrd.IndexLifecyclePolicy, *olivere.XPackIlmGetLifecycleResponse, eshandler.ElasticsearchHandler](client),
	}
}

func (h *indexLifecyclePolicyApiClient) Build(o *elasticsearchapicrd.IndexLifecyclePolicy) (ilm *olivere.XPackIlmGetLifecycleResponse, err error) {
	if o.Spec.Policy == "" {
		return nil, errors.New("The ILM policy must be provided")
	}

	ilm = &olivere.XPackIlmGetLifecycleResponse{}
	if err = json.Unmarshal([]byte(o.Spec.Policy), ilm); err != nil {
		return nil, errors.Wrap(err, "Unable to convert expected policy to object")
	}

	return ilm, nil
}

func (h *indexLifecyclePolicyApiClient) Get(o *elasticsearchapicrd.IndexLifecyclePolicy) (object *olivere.XPackIlmGetLifecycleResponse, err error) {
	return h.Client().ILMGet(o.GetExternalName())
}

func (h *indexLifecyclePolicyApiClient) Create(object *olivere.XPackIlmGetLifecycleResponse, o *elasticsearchapicrd.IndexLifecyclePolicy) (err error) {
	return h.Client().ILMUpdate(o.GetExternalName(), object)
}

func (h *indexLifecyclePolicyApiClient) Update(object *olivere.XPackIlmGetLifecycleResponse, o *elasticsearchapicrd.IndexLifecyclePolicy) (err error) {
	return h.Client().ILMUpdate(o.GetExternalName(), object)
}

func (h *indexLifecyclePolicyApiClient) Delete(o *elasticsearchapicrd.IndexLifecyclePolicy) (err error) {
	return h.Client().ILMDelete(o.GetExternalName())
}

func (h *indexLifecyclePolicyApiClient) Diff(currentOject *olivere.XPackIlmGetLifecycleResponse, expectedObject *olivere.XPackIlmGetLifecycleResponse, originalObject *olivere.XPackIlmGetLifecycleResponse, ignoresDiff ...patch.CalculateOption) (patchResult *patch.PatchResult, err error) {
	return h.Client().ILMDiff(currentOject, expectedObject, originalObject)
}
