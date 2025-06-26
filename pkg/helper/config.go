package helper

import (
	"encoding/json"
	"fmt"
	"strings"

	"dario.cat/mergo"
	"emperror.dev/errors"
	"github.com/davecgh/go-spew/spew"
	"github.com/elastic/go-ucfg"
	ucfgyaml "github.com/elastic/go-ucfg/yaml"
	"github.com/thoas/go-funk"
	"gopkg.in/yaml.v3"
	"github.com/magiconair/properties"
	ucfgjson "github.com/elastic/go-ucfg/json"
)

// MergeSettings have different behavior in fact of content
// When YAML content, it will merge the content of the two maps
// When JSON content, it will merge the content of the two maps
// When properties content, it will merge the content of the two maps
// For other it just append the content of the two maps
// dst have the priority
func MergeSettings(dst, src map[string]string) (res map[string]string, err error) {
	if dst == nil {
		return src, nil
	}

	if src == nil {
		return dst, nil
	}

	res = map[string]string{}

	dstKeys := funk.Keys(dst).([]string)
	srcKeys := funk.Keys(src).([]string)

	// Compute diffrente keys
	dstDiff, srcDiff := funk.DifferenceString(dstKeys, srcKeys)
	for _, key := range dstDiff {
		res[key] = dst[key]
	}
	for _, key := range srcDiff {
		res[key] = src[key]
	}

	// Compute interset keys
	intersectKeys := funk.IntersectString(dstKeys, srcKeys)
	for _, key := range intersectKeys {
		if strings.HasSuffix(key, ".yaml") || strings.HasSuffix(key, ".yml") {
			dstConfig, err := ucfgyaml.NewConfig([]byte(dst[key]), ucfg.PathSep("."))
			if err != nil {
				return nil, errors.Wrapf(err, "Error when init config for key %s: %s\n", key, spew.Sprint(dst))
			}
			srcConfig, err := ucfgyaml.NewConfig([]byte(src[key]), ucfg.PathSep("."))
			if err != nil {
				return nil, errors.Wrapf(err, "Error when init config for key %s: %s\n", key, spew.Sprint(src))
			}

			dstUnpack := map[string]any{}
			srcUnpack := map[string]any{}
			if err = dstConfig.Unpack(&dstUnpack, ucfg.PathSep(".")); err != nil {
				return nil, errors.Wrapf(err, "Error when unpack config for key %s", key)
			}
			if err = srcConfig.Unpack(&srcUnpack, ucfg.PathSep(".")); err != nil {
				return nil, errors.Wrapf(err, "Error when unpack config for key %s", key)
			}

			// Merge contend
			if err = mergo.Merge(&dstUnpack, srcUnpack); err != nil {
				return nil, errors.Wrapf(err, "Error when merge config for key %s", key)
			}

			// Convert to string
			configByte, err := yaml.Marshal(dstUnpack)
			if err != nil {
				return nil, errors.Wrapf(err, "Error when unmarshall contend for key %s", key)
			}

			res[key] = string(configByte)

		} else if strings.HasSuffix(key, ".json") {
			dstConfig, err := ucfgjson.NewConfig([]byte(dst[key]), ucfg.PathSep("."))
			if err != nil {
				return nil, errors.Wrapf(err, "Error when init config for key %s: %s\n", key, spew.Sprint(dst))
			}
			srcConfig, err := ucfgjson.NewConfig([]byte(src[key]), ucfg.PathSep("."))
			if err != nil {
				return nil, errors.Wrapf(err, "Error when init config for key %s: %s\n", key, spew.Sprint(src))
			}

			dstUnpack := map[string]any{}
			srcUnpack := map[string]any{}
			if err = dstConfig.Unpack(&dstUnpack, ucfg.PathSep(".")); err != nil {
				return nil, errors.Wrapf(err, "Error when unpack config for key %s", key)
			}
			if err = srcConfig.Unpack(&srcUnpack, ucfg.PathSep(".")); err != nil {
				return nil, errors.Wrapf(err, "Error when unpack config for key %s", key)
			}

			// Merge contend
			if err = mergo.Merge(&dstUnpack, srcUnpack); err != nil {
				return nil, errors.Wrapf(err, "Error when merge config for key %s", key)
			}

			// Convert to string
			configByte, err := json.MarshalIndent(dstUnpack, "", "  ")
			if err != nil {
				return nil, errors.Wrapf(err, "Error when unmarshall contend for key %s", key)
			}

			res[key] = string(configByte)

		} else if strings.HasSuffix(key, ".properties") {

			p1, err := properties.LoadString(dst[key])
			if err != nil {
				return nil, errors.Wrapf(err, "Error when load properties for key %s", key)
			}
			p2, err := properties.LoadString(src[key])
			if err != nil {
				return nil, errors.Wrapf(err, "Error when load properties for key %s", key)
			}
			p2.Merge(p1)
			res[key] = p2.String()

		} else {
			res[key] = fmt.Sprintf("%s\n%s", src[key], dst[key])
		}
	}

	return res, nil
}

func GetSetting(key string, config []byte) (value string, err error) {
	if key == "" {
		return "", errors.New("You must provide key")
	}

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
