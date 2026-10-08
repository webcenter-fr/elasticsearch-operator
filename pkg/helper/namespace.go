package helper

import (
	"os"

	"emperror.dev/errors"
)

const (
	operatorNamespaceEnvVar = "POD_NAMESPACE"
)

func GetOperatorNamespace() (ns string, err error) {
	ns, found := os.LookupEnv(operatorNamespaceEnvVar)
	if !found {
		return "", errors.Errorf("%s must be set", operatorNamespaceEnvVar)
	}

	return ns, nil
}
