package common

import (
	helperdiff "github.com/webcenter-fr/elasticsearch-operator/pkg/helper"
)

func DiffLabels(expected, current map[string]string) (diff string) {
	return helperdiff.DiffMapString(expected, current, nil)
}

func DiffAnnotations(expected, current map[string]string) (diff string) {
	excludeKeys := []string{
		"kubectl.kubernetes.io/last-applied-configuration",
	}
	return helperdiff.DiffMapString(expected, current, excludeKeys)
}
