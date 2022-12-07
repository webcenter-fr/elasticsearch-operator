package test

import "sigs.k8s.io/yaml"

func ToYaml(s any) string {
	b, err := yaml.Marshal(s)
	if err != nil {
		panic(err)
	}

	return string(b)
}
