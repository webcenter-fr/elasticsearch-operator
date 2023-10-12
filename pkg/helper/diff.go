package helper

import (
	"fmt"

	"github.com/disaster37/k8s-objectmatcher/patch"
	"github.com/disaster37/operator-sdk-extra/pkg/helper"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
)

// DiffLabels permit to diff labels
func DiffLabels(expected, current map[string]string) (diff string) {
	return helper.DiffMapString(expected, current, nil)
}

// DiffAnnotations permit to diff annotations
func DiffAnnotations(expected, current map[string]string) (diff string) {
	excludeKeys := []string{
		"kubectl.kubernetes.io/last-applied-configuration",
		patch.LastAppliedConfig,
		fmt.Sprintf("%s/sequence", elasticsearchcrd.ElasticsearchAnnotationKey),
	}
	return helper.DiffMapString(expected, current, excludeKeys)
}
