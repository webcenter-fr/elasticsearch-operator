package v1

import (
	"github.com/disaster37/operator-sdk-extra/v2/pkg/object"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func validateImmutableName(current, old object.RemoteObject) *field.Error {
	if current.GetExternalName() != old.GetExternalName() {
		return field.Forbidden(field.NewPath("spec").Child("name"), "The field 'spec.name' is immutable")
	}

	return nil
}
