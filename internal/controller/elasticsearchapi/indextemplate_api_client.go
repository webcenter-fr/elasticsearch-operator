package elasticsearchapi

import (
	"encoding/json"

	eshandler "github.com/disaster37/es-handler/v8"
	"github.com/disaster37/generic-objectmatcher/patch"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	olivere "github.com/olivere/elastic/v7"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearchapi/v1"
)

type indexTemplateApiClient struct {
	*controller.BasicRemoteExternalReconciler[*elasticsearchapicrd.IndexTemplate, *olivere.IndicesGetIndexTemplate, eshandler.ElasticsearchHandler]
}

func newIndexTemplateApiClient(client eshandler.ElasticsearchHandler) controller.RemoteExternalReconciler[*elasticsearchapicrd.IndexTemplate, *olivere.IndicesGetIndexTemplate, eshandler.ElasticsearchHandler] {
	return &indexTemplateApiClient{
		BasicRemoteExternalReconciler: controller.NewBasicRemoteExternalReconciler[*elasticsearchapicrd.IndexTemplate, *olivere.IndicesGetIndexTemplate, eshandler.ElasticsearchHandler](client),
	}
}

func (h *indexTemplateApiClient) Build(o *elasticsearchapicrd.IndexTemplate) (indexTemplate *olivere.IndicesGetIndexTemplate, err error) {
	if o.IsRawTemplate() {
		indexTemplate = &olivere.IndicesGetIndexTemplate{}
		if err := json.Unmarshal([]byte(o.Spec.RawTemplate), indexTemplate); err != nil {
			return nil, err
		}
	} else {
		indexTemplate = &olivere.IndicesGetIndexTemplate{
			IndexPatterns:   o.Spec.IndexPatterns,
			ComposedOf:      o.Spec.ComposedOf,
			Priority:        o.Spec.Priority,
			Version:         o.Spec.Version,
			AllowAutoCreate: o.Spec.AllowAutoCreate,
		}

		if o.Spec.Template != nil {
			var settings, mappings, aliases map[string]any
			if o.Spec.Template.Settings != "" {
				settings = make(map[string]any)
				if err := json.Unmarshal([]byte(o.Spec.Template.Settings), &settings); err != nil {
					return nil, err
				}
			}
			if o.Spec.Template.Mappings != "" {
				mappings = make(map[string]any)
				if err := json.Unmarshal([]byte(o.Spec.Template.Mappings), &mappings); err != nil {
					return nil, err
				}
			}
			if o.Spec.Template.Aliases != "" {
				aliases = make(map[string]any)
				if err := json.Unmarshal([]byte(o.Spec.Template.Aliases), &aliases); err != nil {
					return nil, err
				}
			}
			indexTemplate.Template = &olivere.IndicesGetIndexTemplateData{
				Settings: settings,
				Mappings: mappings,
				Aliases:  aliases,
			}
		}

		if o.Spec.Meta != "" {
			meta := make(map[string]any)
			if err := json.Unmarshal([]byte(o.Spec.Meta), &meta); err != nil {
				return nil, err
			}
			indexTemplate.Meta = meta
		}
	}

	return indexTemplate, nil
}

func (h *indexTemplateApiClient) Get(o *elasticsearchapicrd.IndexTemplate) (object *olivere.IndicesGetIndexTemplate, err error) {
	return h.Client().IndexTemplateGet(o.GetExternalName())
}

func (h *indexTemplateApiClient) Create(object *olivere.IndicesGetIndexTemplate, o *elasticsearchapicrd.IndexTemplate) (err error) {
	return h.Client().IndexTemplateUpdate(o.GetExternalName(), object)
}

func (h *indexTemplateApiClient) Update(object *olivere.IndicesGetIndexTemplate, o *elasticsearchapicrd.IndexTemplate) (err error) {
	return h.Client().IndexTemplateUpdate(o.GetExternalName(), object)
}

func (h *indexTemplateApiClient) Delete(o *elasticsearchapicrd.IndexTemplate) (err error) {
	return h.Client().IndexTemplateDelete(o.GetExternalName())
}

func (h *indexTemplateApiClient) Diff(currentOject *olivere.IndicesGetIndexTemplate, expectedObject *olivere.IndicesGetIndexTemplate, originalObject *olivere.IndicesGetIndexTemplate, ignoresDiff ...patch.CalculateOption) (patchResult *patch.PatchResult, err error) {
	return h.Client().IndexTemplateDiff(currentOject, expectedObject, originalObject)
}
