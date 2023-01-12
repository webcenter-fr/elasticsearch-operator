package helper

import (
	"github.com/davecgh/go-spew/spew"
	"github.com/elastic/go-ucfg"
	ucfgyaml "github.com/elastic/go-ucfg/yaml"
	"github.com/imdario/mergo"
	"github.com/pkg/errors"
	"github.com/thoas/go-funk"
	"gopkg.in/yaml.v3"
)

// MergeSettings permit to merge map[string] that contain yaml as string
// m1 have the priority
func MergeSettings(m1, m2 map[string]string) (res map[string]string, err error) {

	if m1 == nil {
		return m2, nil
	}

	if m2 == nil {
		return m1, nil
	}

	res = map[string]string{}

	m1Keys := funk.Keys(m1).([]string)
	m2Keys := funk.Keys(m2).([]string)

	// Compute diffrente keys
	m1Diff, m2Diff := funk.DifferenceString(m1Keys, m2Keys)
	for _, key := range m1Diff {
		res[key] = m1[key]
	}
	for _, key := range m2Diff {
		res[key] = m2[key]
	}

	// Compute interset keys
	intersectKeys := funk.IntersectString(m1Keys, m2Keys)
	for _, key := range intersectKeys {
		m1Config, err := ucfgyaml.NewConfig([]byte(m1[key]), ucfg.PathSep("."))
		if err != nil {
			return nil, errors.Wrapf(err, "Error when init config for key %s: %s\n", key, spew.Sprint(m1))
		}
		m2Config, err := ucfgyaml.NewConfig([]byte(m2[key]), ucfg.PathSep("."))
		if err != nil {
			return nil, errors.Wrapf(err, "Error when init config for key %s: %s\n", key, spew.Sprint(m2))
		}

		m1Unpack := map[string]any{}
		m2Unpack := map[string]any{}
		if err = m1Config.Unpack(&m1Unpack, ucfg.PathSep(".")); err != nil {
			return nil, errors.Wrapf(err, "Error when unpack config for key %s", key)
		}
		if err = m2Config.Unpack(&m2Unpack, ucfg.PathSep(".")); err != nil {
			return nil, errors.Wrapf(err, "Error when unpack config for key %s", key)
		}

		// Merge contend
		if err = mergo.Merge(&m1Unpack, m2Unpack); err != nil {
			return nil, errors.Wrapf(err, "Error when merge config for key %s", key)
		}

		// Convert to string
		configByte, err := yaml.Marshal(m1Unpack)
		if err != nil {
			return nil, errors.Wrapf(err, "Error when unmarshall contend for key %s", key)
		}

		res[key] = string(configByte)
	}

	return res, nil
}

func GetSetting(key string, config []byte) (value string, err error) {
	if len(config) == 0 {
		return "", nil
	}

	yConfig, err := ucfgyaml.NewConfig(config, ucfg.PathSep("."))
	if err != nil {
		return "", errors.Wrapf(err, "Error when load config: %s", spew.Sprint(config))
	}

	hasField, err := yConfig.Has(key, -1, ucfg.PathSep("."))
	if err != nil {
		return "", errors.Wrapf(err, "Error when check if field %s exist", key)
	}

	if hasField {
		return yConfig.String(key, -1, ucfg.PathSep("."))
	}

	return "", ucfg.ErrMissing

}
