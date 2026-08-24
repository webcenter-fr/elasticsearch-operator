package elasticsearchapi

import (
	"context"
	"fmt"

	"emperror.dev/errors"
	elasticsearch "github.com/disaster37/elasticsearch/v9"
	eshandler "github.com/disaster37/es-handler/v9"
	"github.com/sirupsen/logrus"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearchapi/v1"
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
	"github.com/webcenter-fr/elasticsearch-operator/internal/controller/common"
	elasticsearchcontrollers "github.com/webcenter-fr/elasticsearch-operator/internal/controller/elasticsearch"
	core "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetElasticsearchHandler(ctx context.Context, o client.Object, esRef shared.ElasticsearchRef, client client.Client, log *logrus.Entry) (esHandler eshandler.ElasticsearchHandler, err error) {
	// Retrieve secret or elasticsearch resource that store the connexion credentials
	var secretNS types.NamespacedName
	secretName := ""
	isManaged := false
	addresses := []string{}
	selfSignedCertificate := false
	allowInsecureHTTP := false
	if esRef.IsManaged() {
		isManaged = true

		// Get Elasticsearch
		es, err := common.GetElasticsearchFromRef(ctx, client, o, esRef)
		if err != nil {
			return nil, errors.Wrap(err, "Error when get Elasticsearch object from ref")
		}
		if es == nil {
			return nil, errors.Errorf("Elasticsearch %s/%s not found", esRef.ManagedElasticsearchRef.Namespace, esRef.ManagedElasticsearchRef.Name)
		}

		// Get secret that store credential
		secretName = elasticsearchcontrollers.GetSecretNameForCredentials(es)

		serviceName := elasticsearchcontrollers.GetGlobalServiceName(es)
		if esRef.ManagedElasticsearchRef.TargetNodeGroup != "" {
			serviceName = elasticsearchcontrollers.GetNodeGroupServiceName(es, esRef.ManagedElasticsearchRef.TargetNodeGroup)
		}

		if !es.Spec.Tls.IsTlsEnabled() {
			addresses = append(addresses, fmt.Sprintf("http://%s.%s.svc:9200", serviceName, es.Namespace))
			allowInsecureHTTP = true
		} else {
			addresses = append(addresses, fmt.Sprintf("https://%s.%s.svc:9200", serviceName, es.Namespace))
			selfSignedCertificate = true
		}

		secretNS = types.NamespacedName{
			Namespace: es.Namespace,
			Name:      secretName,
		}

	} else if esRef.IsExternal() {
		if esRef.SecretRef == nil {
			return nil, errors.New("You must set the secretRef when you use external Elasticsearch")
		}
		secretName = esRef.SecretRef.Name
		addresses = esRef.ExternalElasticsearchRef.Addresses
		// The v9 client rejects credentials over plaintext http:// unless
		// explicitly allowed. External clusters may legitimately use http, so
		// preserve the previous behavior by opting in.
		allowInsecureHTTP = true

		secretNS = types.NamespacedName{
			Namespace: o.GetNamespace(),
			Name:      secretName,
		}
	} else {
		log.Error("You must set the way to connect on Elasticsearch")
		return nil, errors.New("You must set the way to connect on Elasticsearch")
	}

	// Read settings to access on Elasticsearch api
	secret := &core.Secret{}

	if err = client.Get(ctx, secretNS, secret); err != nil {
		if k8serrors.IsNotFound(err) {
			log.Warnf("Secret %s not yet exist, try later", secretName)
			return nil, nil
		}
		log.Errorf("Error when get resource: %s", err.Error())
		return nil, err
	}

	cfg := &elasticsearch.Config{
		Addresses:         addresses,
		TLSSkipVerify:     selfSignedCertificate,
		AllowInsecureHTTP: allowInsecureHTTP,
		Timeout:           common.ESClientTimeout,
	}

	if isManaged {
		cfg.Username = "elastic"
		cfg.Password = string(secret.Data["elastic"])
	} else {
		if len(secret.Data["username"]) == 0 || len(secret.Data["password"]) == 0 {
			return nil, errors.Errorf("The secret %s must contain key `username` and `password`", secret.Name)
		}
		cfg.Username = string(secret.Data["username"])
		cfg.Password = string(secret.Data["password"])
	}

	// Create Elasticsearch handler/client
	esHandler, err = eshandler.NewElasticsearchHandler(cfg, log)
	if err != nil {
		return nil, err
	}

	return esHandler, nil
}

func GetUserSecretWhenAutoGeneratePassword(user *elasticsearchapicrd.User) string {
	return fmt.Sprintf("%s-credential-es", user.Name)
}
