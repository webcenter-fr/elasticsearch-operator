package helper

import "sigs.k8s.io/yaml"

// ToYamlOrDie convert object to YAML. If error it panic
func ToYamlOrDie(data any) string {
	b, err := yaml.Marshal(data)
	if err != nil {
		panic(err)
	}

	return string(b)
}
