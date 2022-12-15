package helper

import (
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/kr/pretty"
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
