package elasticsearchapi

import (
	"encoding/json"

	eshandler "github.com/disaster37/es-handler/v9"
	eshandlerpatch "github.com/disaster37/es-handler/v9/patch"
	"github.com/disaster37/generic-objectmatcher/patch"
	"github.com/disaster37/operator-sdk-extra/v3/pkg/controller/remote"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearchapi/v1"
)

type indexTemplateApiClient struct {
	remote.RemoteExternalReconciler[*elasticsearchapicrd.IndexTemplate, *eshandlerpatch.IndexTemplate, eshandler.ElasticsearchHandler]
}

func newIndexTemplateApiClient(client eshandler.ElasticsearchHandler) remote.RemoteExternalReconciler[*elasticsearchapicrd.IndexTemplate, *eshandlerpatch.IndexTemplate, eshandler.ElasticsearchHandler] {
	return &indexTemplateApiClient{
		RemoteExternalReconciler: remote.NewRemoteExternalReconciler[*elasticsearchapicrd.IndexTemplate, *eshandlerpatch.IndexTemplate, eshandler.ElasticsearchHandler](client),
	}
}

func (h *indexTemplateApiClient) Build(o *elasticsearchapicrd.IndexTemplate) (indexTemplate *eshandlerpatch.IndexTemplate, err error) {
	if o.IsRawTemplate() {
		indexTemplate = &eshandlerpatch.IndexTemplate{}
		if err := json.Unmarshal([]byte(*o.Spec.RawTemplate), indexTemplate); err != nil {
			return nil, err
		}
	} else {
		indexTemplate = &eshandlerpatch.IndexTemplate{
			IndexPatterns: o.Spec.IndexPatterns,
			ComposedOf:    o.Spec.ComposedOf,
			Priority:      o.Spec.Priority,
			Version:       o.Spec.Version,
		}

		if o.Spec.Template != nil {
			var settings, mappings, aliases map[string]any
			if o.Spec.Template.Settings != nil {
				settings = o.Spec.Template.Settings.Data
			}
			if o.Spec.Template.Mappings != nil {
				mappings = o.Spec.Template.Mappings.Data
			}
			if o.Spec.Template.Aliases != nil {
				aliases = o.Spec.Template.Aliases.Data
			}
			indexTemplate.Template = &eshandlerpatch.IndexTemplateData{
				Settings: settings,
				Mappings: mappings,
				Aliases:  aliases,
			}
		}

		if o.Spec.Meta != nil {
			indexTemplate.Meta = o.Spec.Meta.Data
		}
	}

	return indexTemplate, nil
}

func (h *indexTemplateApiClient) Get(o *elasticsearchapicrd.IndexTemplate) (object *eshandlerpatch.IndexTemplate, err error) {
	return h.Client().IndexTemplateGet(o.GetExternalName())
}

func (h *indexTemplateApiClient) Create(object *eshandlerpatch.IndexTemplate, o *elasticsearchapicrd.IndexTemplate) (err error) {
	return h.Client().IndexTemplateUpdate(o.GetExternalName(), object)
}

func (h *indexTemplateApiClient) Update(object *eshandlerpatch.IndexTemplate, o *elasticsearchapicrd.IndexTemplate) (err error) {
	return h.Client().IndexTemplateUpdate(o.GetExternalName(), object)
}

func (h *indexTemplateApiClient) Delete(o *elasticsearchapicrd.IndexTemplate) (err error) {
	return h.Client().IndexTemplateDelete(o.GetExternalName())
}

func (h *indexTemplateApiClient) Diff(currentOject *eshandlerpatch.IndexTemplate, expectedObject *eshandlerpatch.IndexTemplate, originalObject *eshandlerpatch.IndexTemplate, o *elasticsearchapicrd.IndexTemplate, ignoresDiff ...patch.CalculateOption) (patchResult *patch.PatchResult, err error) {
	return h.Client().IndexTemplateDiff(currentOject, expectedObject, originalObject)
}
