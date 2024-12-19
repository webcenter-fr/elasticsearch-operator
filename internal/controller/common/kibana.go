package common

import (
	"context"

	"emperror.dev/errors"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/api/kibana/v1"
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetKibanaRef permit to get Kibana
func GetKibanaFromRef(ctx context.Context, c client.Client, o client.Object, kbRef shared.KibanaRef) (kb *kibanacrd.Kibana, err error) {
	if !kbRef.IsManaged() {
		return nil, nil
	}

	kb = &kibanacrd.Kibana{}
	target := types.NamespacedName{Name: kbRef.ManagedKibanaRef.Name}
	if kbRef.ManagedKibanaRef.Namespace != "" {
		target.Namespace = kbRef.ManagedKibanaRef.Namespace
	} else {
		target.Namespace = o.GetNamespace()
	}
	if err = c.Get(ctx, target, kb); err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "Error when read kibana %s/%s", target.Namespace, target.Name)
	}

	return kb, nil
}
