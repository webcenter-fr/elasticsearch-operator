package test

import (
	"os"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/webcenter-fr/elasticsearch-operator/pkg/helper"
	"sigs.k8s.io/yaml"
)

type TestOpt string

var CleanApi TestOpt = "cleanAPI"

func EqualFromYamlFile(t *testing.T, expectedYamlFile string, actual any, opts ...TestOpt) {

	if expectedYamlFile == "" {
		panic("expectedYamlFile must be provided")
	}

	// Read file
	f, err := os.ReadFile(expectedYamlFile)
	if err != nil {
		panic(err)
	}

	var n any

	// Create new object base from actual
	if reflect.ValueOf(actual).Kind() == reflect.Ptr {
		n = reflect.New(reflect.TypeOf(actual).Elem()).Interface()
	} else {
		n = reflect.New(reflect.TypeOf(actual)).Interface()
	}

	if err = yaml.Unmarshal(f, n); err != nil {
		panic(err)
	}

	for _, opt := range opts {
		switch opt {
		case CleanApi:
			mustCleanAPI(n)
			mustCleanAPI(actual)
		}
	}

	diff := helper.Diff(n, actual)

	if diff != "" {
		assert.Fail(t, diff)
	}

}

func mustCleanAPI(o any) {
	rt := reflect.TypeOf(o)
	if rt.Kind() != reflect.Ptr {
		panic("Object must be pointer")
	}
	rv := reflect.ValueOf(o).Elem()
	fKind := rv.FieldByName("Kind")
	if fKind.IsValid() {
		fKind.SetString("")
	}
	fApiVersion := rv.FieldByName("APIVersion")
	if fApiVersion.IsValid() {
		fApiVersion.SetString("")
	}
}