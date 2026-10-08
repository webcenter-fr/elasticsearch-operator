package elasticsearchapi

import (
	"encoding/json"

	eshandler "github.com/disaster37/es-handler/v9"
	eshandlerpatch "github.com/disaster37/es-handler/v9/patch"
	"github.com/disaster37/generic-objectmatcher/patch"
	"github.com/disaster37/operator-sdk-extra/v3/pkg/controller/remote"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearchapi/v1"
)

type componentTemplateApiClient struct {
	remote.RemoteExternalReconciler[*elasticsearchapicrd.ComponentTemplate, *eshandlerpatch.ComponentTemplate, eshandler.ElasticsearchHandler]
}

func newComponentTemplateApiClient(client eshandler.ElasticsearchHandler) remote.RemoteExternalReconciler[*elasticsearchapicrd.ComponentTemplate, *eshandlerpatch.ComponentTemplate, eshandler.ElasticsearchHandler] {
	return &componentTemplateApiClient{
		RemoteExternalReconciler: remote.NewRemoteExternalReconciler[*elasticsearchapicrd.ComponentTemplate, *eshandlerpatch.ComponentTemplate, eshandler.ElasticsearchHandler](client),
	}
}

func (h *componentTemplateApiClient) Build(o *elasticsearchapicrd.ComponentTemplate) (componentTemplate *eshandlerpatch.ComponentTemplate, err error) {
	if o.IsRawTemplate() {
		componentTemplate = &eshandlerpatch.ComponentTemplate{}
		if err := json.Unmarshal([]byte(*o.Spec.RawTemplate), componentTemplate); err != nil {
			return nil, err
		}
	} else {
		componentTemplate = &eshandlerpatch.ComponentTemplate{
			Template: &eshandlerpatch.ComponentTemplateData{
				Settings: make(map[string]any),
				Mappings: make(map[string]any),
				Aliases:  make(map[string]any),
			},
		}

		if o.Spec.Mappings != nil && o.Spec.Mappings.Data != nil {
			componentTemplate.Template.Mappings = o.Spec.Mappings.Data
		}

		if o.Spec.Settings != nil && o.Spec.Settings.Data != nil {
			componentTemplate.Template.Settings = o.Spec.Settings.Data
		}

		if o.Spec.Aliases != nil && o.Spec.Aliases.Data != nil {
			componentTemplate.Template.Aliases = o.Spec.Aliases.Data
		}
	}

	return componentTemplate, nil
}

func (h *componentTemplateApiClient) Get(o *elasticsearchapicrd.ComponentTemplate) (object *eshandlerpatch.ComponentTemplate, err error) {
	return h.Client().ComponentTemplateGet(o.GetExternalName())
}

func (h *componentTemplateApiClient) Create(object *eshandlerpatch.ComponentTemplate, o *elasticsearchapicrd.ComponentTemplate) (err error) {
	return h.Client().ComponentTemplateUpdate(o.GetExternalName(), object)
}

func (h *componentTemplateApiClient) Update(object *eshandlerpatch.ComponentTemplate, o *elasticsearchapicrd.ComponentTemplate) (err error) {
	return h.Client().ComponentTemplateUpdate(o.GetExternalName(), object)
}

func (h *componentTemplateApiClient) Delete(o *elasticsearchapicrd.ComponentTemplate) (err error) {
	return h.Client().ComponentTemplateDelete(o.GetExternalName())
}

func (h *componentTemplateApiClient) Diff(currentOject *eshandlerpatch.ComponentTemplate, expectedObject *eshandlerpatch.ComponentTemplate, originalObject *eshandlerpatch.ComponentTemplate, o *elasticsearchapicrd.ComponentTemplate, ignoresDiff ...patch.CalculateOption) (patchResult *patch.PatchResult, err error) {
	return h.Client().ComponentTemplateDiff(currentOject, expectedObject, originalObject)
}
