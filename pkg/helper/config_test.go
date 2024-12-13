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
		res       map[string]string
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

	res, err = MergeSettings(m2, m)
	assert.NoError(t, err)
	assert.Empty(t, cmp.Diff(expectedM, res))

	// When one of them is nil
	res, err = MergeSettings(nil, m)
	assert.NoError(t, err)
	assert.Empty(t, cmp.Diff(m, res))

	res, err = MergeSettings(m, nil)
	assert.NoError(t, err)
	assert.Empty(t, cmp.Diff(m, res))

	// When keys is not the same
	m = map[string]string{
		"elasticsearch.yml": `
aa: aa
`,
	}
	m2 = map[string]string{
		"elasticsearch.yml": `
bb: bb
`,
	}
	expectedM = map[string]string{
		"elasticsearch.yml": `aa: aa
bb: bb
`,
	}

	res, err = MergeSettings(m2, m)
	assert.NoError(t, err)
	assert.Empty(t, cmp.Diff(expectedM, res))

	// When error
	m = map[string]string{
		"elasticsearch.yml": `
dd: dd
 fff: ff
`,
	}
	m2 = map[string]string{
		"elasticsearch.yml": `
bb: bb
`,
	}

	_, err = MergeSettings(m2, m)
	assert.Error(t, err)

	_, err = MergeSettings(m, m2)
	assert.Error(t, err)
}

func TestGetSetting(t *testing.T) {
	var (
		err  error
		val  string
		tVal string
	)

	// Normal case

	tVal = `
key1: value1
key2: value2
key3: value3
`

	val, err = GetSetting("key2", []byte(tVal))
	assert.NoError(t, err)
	assert.Equal(t, "value2", val)

	// When config is empty
	val, err = GetSetting("key", nil)
	assert.NoError(t, err)
	assert.Equal(t, "", val)

	// When key empty
	_, err = GetSetting("", []byte(tVal))
	assert.Error(t, err)

	// When config is not yaml, error
	tVal = `
key1: value1
 key2: value2
key3: value3
`

	_, err = GetSetting("key2", []byte(tVal))
	assert.Error(t, err)
}
