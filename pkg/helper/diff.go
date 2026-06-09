package helper

import (
	"fmt"

	"github.com/disaster37/k8s-objectmatcher/patch"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/helper"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DiffLabels permit to diff labels
func DiffLabels(expected, current map[string]string) (diff string) {
	return helper.DiffMapString(expected, current, nil)
}

// DiffOwnerReferences permit to diff owner references
func DiffOwnerReferences(owner client.Object, object client.Object) (diff string, err error) {
	existing := metav1.GetControllerOf(object)
	if existing != nil {
		group, err := schema.ParseGroupVersion(existing.APIVersion)
		if err != nil {
			return "", err
		}
		if group.Group == owner.GetObjectKind().GroupVersionKind().Group && existing.Kind == owner.GetObjectKind().GroupVersionKind().Kind {
			if group.Version != owner.GetObjectKind().GroupVersionKind().Version {
				return fmt.Sprintf("Owner references differ: %s vs %s/%s", existing.APIVersion, owner.GetObjectKind().GroupVersionKind().Group, owner.GetObjectKind().GroupVersionKind().Version), nil
			}
		} else {
			return fmt.Sprintf("Owner references differ: %s.%s vs %s.%s/%s", existing.Kind, existing.APIVersion, owner.GetObjectKind().GroupVersionKind().Kind, owner.GetObjectKind().GroupVersionKind().Group, owner.GetObjectKind().GroupVersionKind().Version), nil
		}
	} else {
		return "Need to set owner references", nil
	}
	return "", nil
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
