package common

import (
	"reflect"

	"k8s.io/apimachinery/pkg/util/json"
)

// ToJsonOrDie return object as json format
func ToJsonOrDie(o any) string {

	if reflect.ValueOf(o).IsNil() {
		return ""
	}

	b, err := json.Marshal(o)
	if err != nil {
		panic(err)
	}

	return string(b)
}
