package helper

import (
	"fmt"

	"github.com/disaster37/k8s-objectmatcher/patch"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/kr/pretty"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
)

// Diff is cmp.Diff with custom function to compare slices with pretty.Sprint
func Diff(expected, current any) string {
	return cmp.Diff(expected, current, cmpopts.SortSlices(func(x, y any) bool {
		return pretty.Sprint(x) < pretty.Sprint(y)
	}))
}

// DiffMapString permit to diff map[string]string
func DiffMapString(expected, current map[string]string, excludeKeys []string) string {
	tmpExpected := map[string]string{}
	tmpCurrent := map[string]string{}

Loop:
	for key, val := range expected {
		for _, excludeKey := range excludeKeys {
			if key == excludeKey {
				continue Loop
			}
		}
		tmpExpected[key] = val
	}

Loop2:
	for key, val := range current {
		for _, excludeKey := range excludeKeys {
			if key == excludeKey {
				continue Loop2
			}
		}
		tmpCurrent[key] = val
	}

	return cmp.Diff(tmpCurrent, tmpExpected)
}

// DiffLabels permit to diff labels
func DiffLabels(expected, current map[string]string) (diff string) {
	return DiffMapString(expected, current, nil)
}

// DiffAnnotations permit to diff annotations
func DiffAnnotations(expected, current map[string]string) (diff string) {
	excludeKeys := []string{
		"kubectl.kubernetes.io/last-applied-configuration",
		patch.LastAppliedConfig,
		fmt.Sprintf("%s/sequence", elasticsearchcrd.ElasticsearchAnnotationKey),
	}
	return DiffMapString(expected, current, excludeKeys)
}
