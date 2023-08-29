package helper

import (
	"reflect"

	"dario.cat/mergo"
)

// Merge permit to merge unlimited same interface. The last src is the higher priority
// If some src are nil, it skip it
// It return error if dst is nil
func Merge(dst any, srcs ...any) (err error) {
	if dst != nil && reflect.ValueOf(dst).Kind() != reflect.Ptr {
		return mergo.ErrNonPointerArgument
	}

	for _, src := range srcs {
		if src == nil || reflect.ValueOf(src).IsNil() {
			continue
		}
		if err = mergo.Merge(dst, src); err != nil {
			return err
		}
	}

	return nil
}
