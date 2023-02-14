package helper

import (
	"errors"
	"reflect"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

func SetLastOriginal(o client.Object, original any) (err error) {
	if o == nil {
		return errors.New("Object can't be nil")
	}

	rt := reflect.TypeOf(o)
	if rt.Kind() != reflect.Ptr {
		panic("Object must be pointer")
	}
	rv := reflect.ValueOf(o).Elem()

	omStatus := rv.FieldByName("Status")
	if !omStatus.IsValid() {
		return errors.New("Object must have `Status field`")
	}

	omOriginal := omStatus.FieldByName("OriginalObject")
	if !omOriginal.IsValid() {
		return errors.New("Object must have `Status.OriginalObject field`")
	}

	originalZip, err := ZipAndBase64Encode(original)
	if err != nil {
		return err
	}

	omOriginal.SetString(originalZip)

	return nil

}
