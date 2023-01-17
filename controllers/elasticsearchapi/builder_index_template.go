package elasticsearchapi

import (
	"encoding/json"

	olivere "github.com/olivere/elastic/v7"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearchapi/v1alpha1"
)

// BuildIndexTemplate permit to build index template
func BuildIndexTemplate(o *elasticsearchapicrd.IndexTemplate) (indexTemplate *olivere.IndicesGetIndexTemplate, err error) {

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

	return indexTemplate, nil
}
