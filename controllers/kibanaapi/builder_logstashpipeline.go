package kibanaapi

import (
	"encoding/json"

	"github.com/disaster37/go-kibana-rest/v8/kbapi"
	kibanaapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/kibanaapi/v1"
)

// BuildLogstashPipeline permit to build Logstash pipeline
func BuildLogstashPipeline(o *kibanaapicrd.LogstashPipeline) (pipeline *kbapi.LogstashPipeline, err error) {

	pipeline = &kbapi.LogstashPipeline{
		ID:          o.GetPipelineName(),
		Description: o.Spec.Description,
		Pipeline:    o.Spec.Pipeline,
	}

	if o.Spec.Settings != "" {
		s := make(map[string]any)
		if err := json.Unmarshal([]byte(o.Spec.Settings), &s); err != nil {
			return nil, err
		}

		pipeline.Settings = s
	}

	return pipeline, nil
}
