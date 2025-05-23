package elasticsearchapi

import (
	"encoding/json"

	eshandler "github.com/disaster37/es-handler/v8"
	"github.com/disaster37/generic-objectmatcher/patch"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/controller/remote"
	olivere "github.com/olivere/elastic/v7"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearchapi/v1"
)

type componentTemplateApiClient struct {
	remote.RemoteExternalReconciler[*elasticsearchapicrd.ComponentTemplate, *olivere.IndicesGetComponentTemplate, eshandler.ElasticsearchHandler]
}

func newComponentTemplateApiClient(client eshandler.ElasticsearchHandler) remote.RemoteExternalReconciler[*elasticsearchapicrd.ComponentTemplate, *olivere.IndicesGetComponentTemplate, eshandler.ElasticsearchHandler] {
	return &componentTemplateApiClient{
		RemoteExternalReconciler: remote.NewRemoteExternalReconciler[*elasticsearchapicrd.ComponentTemplate, *olivere.IndicesGetComponentTemplate, eshandler.ElasticsearchHandler](client),
	}
}

func (h *componentTemplateApiClient) Build(o *elasticsearchapicrd.ComponentTemplate) (componentTemplate *olivere.IndicesGetComponentTemplate, err error) {
	if o.IsRawTemplate() {
		componentTemplate = &olivere.IndicesGetComponentTemplate{}
		if err := json.Unmarshal([]byte(*o.Spec.RawTemplate), componentTemplate); err != nil {
			return nil, err
		}
	} else {
		componentTemplate = &olivere.IndicesGetComponentTemplate{
			Template: &olivere.IndicesGetComponentTemplateData{
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

func (h *componentTemplateApiClient) Get(o *elasticsearchapicrd.ComponentTemplate) (object *olivere.IndicesGetComponentTemplate, err error) {
	return h.Client().ComponentTemplateGet(o.GetExternalName())
}

func (h *componentTemplateApiClient) Create(object *olivere.IndicesGetComponentTemplate, o *elasticsearchapicrd.ComponentTemplate) (err error) {
	return h.Client().ComponentTemplateUpdate(o.GetExternalName(), object)
}

func (h *componentTemplateApiClient) Update(object *olivere.IndicesGetComponentTemplate, o *elasticsearchapicrd.ComponentTemplate) (err error) {
	return h.Client().ComponentTemplateUpdate(o.GetExternalName(), object)
}

func (h *componentTemplateApiClient) Delete(o *elasticsearchapicrd.ComponentTemplate) (err error) {
	return h.Client().ComponentTemplateDelete(o.GetExternalName())
}

func (h *componentTemplateApiClient) Diff(currentOject *olivere.IndicesGetComponentTemplate, expectedObject *olivere.IndicesGetComponentTemplate, originalObject *olivere.IndicesGetComponentTemplate, o *elasticsearchapicrd.ComponentTemplate, ignoresDiff ...patch.CalculateOption) (patchResult *patch.PatchResult, err error) {
	return h.Client().ComponentTemplateDiff(currentOject, expectedObject, originalObject)
}
