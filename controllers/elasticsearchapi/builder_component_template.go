package elasticsearchapi

import (
	"encoding/json"

	olivere "github.com/olivere/elastic/v7"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearchapi/v1"
)

// BuildComponentTemplate permit to build component template
func BuildComponentTemplate(o *elasticsearchapicrd.ComponentTemplate) (componentTemplate *olivere.IndicesGetComponentTemplate, err error) {

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
