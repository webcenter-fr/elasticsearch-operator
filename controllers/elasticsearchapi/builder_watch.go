package elasticsearchapi

import (
	"encoding/json"

	"github.com/pkg/errors"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearchapi/v1"

	olivere "github.com/olivere/elastic/v7"
)

// BuildWatch permit to build watch
func BuildWatch(o *elasticsearchapicrd.Watch) (watch *olivere.XPackWatch, err error) {

	watch = &olivere.XPackWatch{
		ThrottlePeriod:         o.Spec.ThrottlePeriod,
		ThrottlePeriodInMillis: o.Spec.ThrottlePeriodInMillis,
	}

	if o.Spec.Trigger != "" {
		trigger := make(map[string]map[string]any)
		if err := json.Unmarshal([]byte(o.Spec.Trigger), &trigger); err != nil {
			return nil, errors.Wrap(err, "Error when decode trigger")
		}
		watch.Trigger = trigger
	}

	if o.Spec.Input != "" {
		input := make(map[string]map[string]any)
		if err := json.Unmarshal([]byte(o.Spec.Input), &input); err != nil {
			return nil, errors.Wrap(err, "Error when decode input")
		}
		watch.Input = input
	}

	if o.Spec.Condition != "" {
		condition := make(map[string]map[string]any)
		if err := json.Unmarshal([]byte(o.Spec.Condition), &condition); err != nil {
			return nil, errors.Wrap(err, "Error when decode condition")
		}
		watch.Condition = condition
	}

	if o.Spec.Transform != "" {
		transform := make(map[string]any)
		if err := json.Unmarshal([]byte(o.Spec.Transform), &transform); err != nil {
			return nil, errors.Wrap(err, "Error when decode transform")

		}
		watch.Transform = transform
	}

	if o.Spec.Actions != "" {
		actions := make(map[string]map[string]any)
		if err := json.Unmarshal([]byte(o.Spec.Actions), &actions); err != nil {
			return nil, errors.Wrap(err, "Error when decode actions")
		}
		watch.Actions = actions
	}

	if o.Spec.Metadata != "" {
		meta := make(map[string]any)
		if err := json.Unmarshal([]byte(o.Spec.Metadata), &meta); err != nil {
			return nil, errors.Wrap(err, "Error when decode metadata")
		}
		watch.Metadata = meta
	}

	return watch, nil
}
