package helper

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
)

func TestMergeSettings(t *testing.T) {
	var (
		m         map[string]string
		m2        map[string]string
		expectedM map[string]string
		err       error
	)

	// Normal case
	m = map[string]string{
		"elasticsearch.yml": `
dd: a
aa.bb: tutu
ff:
    gg:
        tt: plop
`,
	}
	m2 = map[string]string{
		"elasticsearch.yml": `
dd: a
aa.bb: tutu2
ff:
    gg:
        tt: plop
        kk: plop2
`,
	}
	expectedM = map[string]string{
		"elasticsearch.yml": `aa:
    bb: tutu2
dd: a
ff:
    gg:
        kk: plop2
        tt: plop
`,
	}

	m2, err = MergeSettings(m2, m)
	assert.NoError(t, err)
	assert.Empty(t, cmp.Diff(expectedM, m2))
}
