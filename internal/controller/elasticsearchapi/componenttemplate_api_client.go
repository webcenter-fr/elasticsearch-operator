package elasticsearchapi

import (
	"encoding/json"

	eshandler "github.com/disaster37/es-handler/v8"
	"github.com/disaster37/generic-objectmatcher/patch"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	olivere "github.com/olivere/elastic/v7"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearchapi/v1"
)

type componentTemplateApiClient struct {
	*controller.BasicRemoteExternalReconciler[*elasticsearchapicrd.ComponentTemplate, *olivere.IndicesGetComponentTemplate, eshandler.ElasticsearchHandler]
}

func newComponentTemplateApiClient(client eshandler.ElasticsearchHandler) controller.RemoteExternalReconciler[*elasticsearchapicrd.ComponentTemplate, *olivere.IndicesGetComponentTemplate, eshandler.ElasticsearchHandler] {
	return &componentTemplateApiClient{
		BasicRemoteExternalReconciler: controller.NewBasicRemoteExternalReconciler[*elasticsearchapicrd.ComponentTemplate, *olivere.IndicesGetComponentTemplate, eshandler.ElasticsearchHandler](client),
	}
}

func (h *componentTemplateApiClient) Build(o *elasticsearchapicrd.ComponentTemplate) (componentTemplate *olivere.IndicesGetComponentTemplate, err error) {
	if o.IsRawTemplate() {
		componentTemplate = &olivere.IndicesGetComponentTemplate{}
		if err := json.Unmarshal([]byte(o.Spec.Template), componentTemplate); err != nil {
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

		if o.Spec.Mappings != "" {
			if err := json.Unmarshal([]byte(o.Spec.Mappings), &componentTemplate.Template.Mappings); err != nil {
				return nil, err
			}
		}

		if o.Spec.Settings != "" {
			if err := json.Unmarshal([]byte(o.Spec.Settings), &componentTemplate.Template.Settings); err != nil {
				return nil, err
			}
		}

		if o.Spec.Aliases != "" {
			if err := json.Unmarshal([]byte(o.Spec.Aliases), &componentTemplate.Template.Aliases); err != nil {
				return nil, err
			}
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

func (h *componentTemplateApiClient) Diff(currentOject *olivere.IndicesGetComponentTemplate, expectedObject *olivere.IndicesGetComponentTemplate, originalObject *olivere.IndicesGetComponentTemplate, ignoresDiff ...patch.CalculateOption) (patchResult *patch.PatchResult, err error) {
	return h.Client().ComponentTemplateDiff(currentOject, expectedObject, originalObject)
}
