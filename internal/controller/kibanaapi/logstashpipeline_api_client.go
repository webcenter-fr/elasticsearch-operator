package kibanaapi

import (
	"github.com/disaster37/generic-objectmatcher/patch"
	"github.com/disaster37/go-kibana-rest/v8/kbapi"
	kbhandler "github.com/disaster37/kb-handler/v8"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/controller/remote"
	kibanaapicrd "github.com/webcenter-fr/elasticsearch-operator/api/kibanaapi/v1"
)

type logstashPipelineApiClient struct {
	remote.RemoteExternalReconciler[*kibanaapicrd.LogstashPipeline, *kbapi.LogstashPipeline, kbhandler.KibanaHandler]
}

func newLogstashPipelineApiClient(client kbhandler.KibanaHandler) remote.RemoteExternalReconciler[*kibanaapicrd.LogstashPipeline, *kbapi.LogstashPipeline, kbhandler.KibanaHandler] {
	return &logstashPipelineApiClient{
		RemoteExternalReconciler: remote.NewRemoteExternalReconciler[*kibanaapicrd.LogstashPipeline, *kbapi.LogstashPipeline, kbhandler.KibanaHandler](client),
	}
}

func (h *logstashPipelineApiClient) Build(o *kibanaapicrd.LogstashPipeline) (pipeline *kbapi.LogstashPipeline, err error) {
	pipeline = &kbapi.LogstashPipeline{
		ID:          o.GetExternalName(),
		Description: o.Spec.Description,
		Pipeline:    o.Spec.Pipeline,
	}

	if o.Spec.Settings != nil {
		pipeline.Settings = o.Spec.Settings.Data
	}

	return pipeline, nil
}

func (h *logstashPipelineApiClient) Get(o *kibanaapicrd.LogstashPipeline) (object *kbapi.LogstashPipeline, err error) {
	return h.Client().LogstashPipelineGet(o.GetExternalName())
}

func (h *logstashPipelineApiClient) Create(object *kbapi.LogstashPipeline, o *kibanaapicrd.LogstashPipeline) (err error) {
	return h.Client().LogstashPipelineUpdate(object)
}

func (h *logstashPipelineApiClient) Update(object *kbapi.LogstashPipeline, o *kibanaapicrd.LogstashPipeline) (err error) {
	return h.Client().LogstashPipelineUpdate(object)
}

func (h *logstashPipelineApiClient) Delete(o *kibanaapicrd.LogstashPipeline) (err error) {
	return h.Client().LogstashPipelineDelete(o.GetGenerateName())
}

func (h *logstashPipelineApiClient) Diff(currentOject *kbapi.LogstashPipeline, expectedObject *kbapi.LogstashPipeline, originalObject *kbapi.LogstashPipeline, o *kibanaapicrd.LogstashPipeline, ignoresDiff ...patch.CalculateOption) (patchResult *patch.PatchResult, err error) {
	return h.Client().LogstashPipelineDiff(currentOject, expectedObject, originalObject)
}
