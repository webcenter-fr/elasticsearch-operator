package kafka

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"

	"emperror.dev/errors"
	strimzicrd "github.com/RedHatInsights/strimzi-client-go/apis/kafka.strimzi.io/v1beta2"
	"github.com/sethvargo/go-password/password"
	discovercrd "github.com/webcenter-fr/elasticsearch-operator/api/discover/v1"
	"github.com/webcenter-fr/elasticsearch-operator/internal/controller/discover"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"software.sslmate.com/src/go-pkcs12"
)

// buildKafkaSecrets permit to build secret from Kafka discovered
func buildKafkaSecrets(kk *discovercrd.Kafka, kafkaCluster *strimzicrd.Kafka, secretCaKafka *corev1.Secret, secretUserKafka *corev1.Secret, secretCustomCaKafka *corev1.Secret) (secrets []*corev1.Secret, err error) {
	certificates := make([]*x509.Certificate, 0)
	caCrt := make([][]byte, 0, 1)
	secrets = make([]*corev1.Secret, 0, 2)
	envSuffix := getEnvSuffix(kk)
	var (
		userKey     []byte
		userCrt     []byte
		userCA      []byte
		certificate *x509.Certificate
	)

	secretForEnv := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GetSecretNameForEnv(kk),
			Namespace: kk.Namespace,
			Labels:    getLabels(kk),
			Annotations: getAnnotations(kk, map[string]string{
				fmt.Sprintf("%s/type", discovercrd.DiscoverAnnotationKey): "env",
			}),
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{},
	}

	secretForFile := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GetSecretNameForFile(kk),
			Namespace: kk.Namespace,
			Labels:    getLabels(kk),
			Annotations: getAnnotations(kk, map[string]string{
				fmt.Sprintf("%s/type", discovercrd.DiscoverAnnotationKey): "file",
			}),
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{},
	}

	// Add custom CA certificates
	if kk.Spec.KafkaRef.KafkaCaSecretRef != nil && secretCustomCaKafka != nil {
		for certificateName, certificateData := range secretCustomCaKafka.Data {
			if strings.HasSuffix(certificateName, ".crt") || strings.HasSuffix(certificateName, ".pem") {
				certificate, err = discover.LoadCertFromPem(certificateData)
				if err != nil {
					return nil, errors.Wrapf(err, "Error when load %s certificate", certificateName)
				}
				certificates = append(certificates, certificate)
				caCrt = append(caCrt, certificateData)
			}
		}
	}

	// When external kafka
	if kk.Spec.KafkaRef.IsExternal() {
		secretForEnv.Data[fmt.Sprintf("KAFKA_BOOTSTRAP_SERVERS_%s", envSuffix)] = []byte(strings.Join(kk.Spec.KafkaRef.ExternalKafkaRef.Addresses, ","))

		if secretUserKafka != nil && secretUserKafka.Data != nil {
			if len(secretUserKafka.Data["user.key"]) > 0 {
				secretForFile.Data["user.key"] = secretUserKafka.Data["user.key"]
				userKey = secretUserKafka.Data["user.key"]
			}
			if len(secretUserKafka.Data["user.crt"]) > 0 {
				secretForFile.Data["user.crt"] = secretUserKafka.Data["user.crt"]
				userCrt = secretUserKafka.Data["user.crt"]
			}
			if len(secretUserKafka.Data["ca.crt"]) > 0 {
				userCA = secretUserKafka.Data["ca.crt"]
			}
		}
	}

	// When managed kafka
	if kk.Spec.KafkaRef.IsManaged() && kafkaCluster != nil {
		var (
			bootstrapUrl    string
			listenerIndex   = -1
			targetListener  = kk.Spec.KafkaRef.ManagedKafkaRef.TargetListener
			statusListeners = kafkaCluster.Status.Listeners
		)

		// Search bootstrap URL from the status listeners
		for i := range statusListeners {
			listener := &statusListeners[i]
			listenerName := ""
			if listener.Name != nil {
				listenerName = *listener.Name
			}

			if targetListener != "" {
				// User asked for a specific listener, it must exist
				if listenerName != targetListener {
					continue
				}
			} else {
				// By default, select the first internal or clusterip listener
				specListener := getSpecListener(kafkaCluster, listenerName, i)
				if specListener == nil || (specListener.Type != strimzicrd.KafkaSpecKafkaListenersElemTypeInternal && specListener.Type != strimzicrd.KafkaSpecKafkaListenersElemTypeClusterIp) {
					continue
				}
			}

			if listener.BootstrapServers == nil {
				return nil, errors.Errorf("Kafka listener %s has no bootstrap servers yet", listenerName)
			}
			bootstrapUrl = *listener.BootstrapServers
			listenerIndex = i
			break
		}

		if targetListener != "" && listenerIndex == -1 {
			return nil, errors.Errorf("Target listener %s not found on Kafka %s", targetListener, kafkaCluster.Name)
		}
		if listenerIndex == -1 && len(statusListeners) > 0 {
			if statusListeners[0].BootstrapServers == nil {
				listenerName := ""
				if statusListeners[0].Name != nil {
					listenerName = *statusListeners[0].Name
				}
				return nil, errors.Errorf("Kafka listener %s has no bootstrap servers yet", listenerName)
			}
			bootstrapUrl = *statusListeners[0].BootstrapServers
			listenerIndex = 0
		}
		if listenerIndex == -1 {
			return nil, errors.Errorf("No usable listener found on Kafka %s", kafkaCluster.Name)
		}
		secretForEnv.Data[fmt.Sprintf("KAFKA_BOOTSTRAP_SERVERS_%s", envSuffix)] = []byte(bootstrapUrl)

		// Discover auth type
		var authorization *strimzicrd.KafkaSpecKafkaAuthorization
		if kafkaCluster.Spec != nil {
			authorization = kafkaCluster.Spec.Kafka.Authorization
		}
		authentication := getListenerAuthentication(kafkaCluster, statusListeners[listenerIndex], listenerIndex)
		if authentication != nil {
			secretForEnv.Data[fmt.Sprintf("KAFKA_AUTH_TYPE_%s", envSuffix)] = []byte(authentication.Type)
			if authentication.Type == strimzicrd.KafkaSpecKafkaListenersElemAuthenticationTypeOauth {

				if authentication.ClientId != nil {
					secretForEnv.Data[fmt.Sprintf("KAFKA_OAUTH_CLIENT_ID_%s", envSuffix)] = []byte(*authentication.ClientId)
				} else if authorization != nil && authorization.ClientId != nil {
					secretForEnv.Data[fmt.Sprintf("KAFKA_OAUTH_CLIENT_ID_%s", envSuffix)] = []byte(*authorization.ClientId)
				}

				if authentication.TokenEndpointUri != nil {
					secretForEnv.Data[fmt.Sprintf("KAFKA_OAUTH_TOKEN_ENDPOINT_URI_%s", envSuffix)] = []byte(*authentication.TokenEndpointUri)
				} else if authorization != nil && authorization.TokenEndpointUri != nil {
					secretForEnv.Data[fmt.Sprintf("KAFKA_OAUTH_TOKEN_ENDPOINT_URI_%s", envSuffix)] = []byte(*authorization.TokenEndpointUri)
				}

				if authentication.ValidIssuerUri != nil {
					secretForEnv.Data[fmt.Sprintf("KAFKA_OAUTH_VALID_ISSUER_URI_%s", envSuffix)] = []byte(*authentication.ValidIssuerUri)
				}

				if authentication.JwksEndpointUri != nil {
					secretForEnv.Data[fmt.Sprintf("KAFKA_OAUTH_JWKS_ENDPOINT_URI_%s", envSuffix)] = []byte(*authentication.JwksEndpointUri)
				}

				if authentication.UserNameClaim != nil {
					secretForEnv.Data[fmt.Sprintf("KAFKA_OAUTH_USERNAME_CLAIM_%s", envSuffix)] = []byte(*authentication.UserNameClaim)
				}
			}
		}

		// Add CA certificate if is managed by Strimzi
		if kafkaHaveClusterCa(kafkaCluster) && secretCaKafka != nil {
			certificate, err = discover.LoadCertFromPem(secretCaKafka.Data["ca.crt"])
			if err != nil {
				return nil, errors.Wrap(err, "Error when load ca.crt certificate")
			}
			certificates = append(certificates, certificate)
			caCrt = append(caCrt, secretCaKafka.Data["ca.crt"])
		}

		// Add user certificate if is managed by Strimzi
		if kk.Spec.KafkaRef.ManagedKafkaRef.UserRef != nil && secretUserKafka != nil && secretUserKafka.Data != nil {
			if len(secretUserKafka.Data["user.key"]) > 0 {
				secretForFile.Data["user.key"] = secretUserKafka.Data["user.key"]
				userKey = secretUserKafka.Data["user.key"]
			}
			if len(secretUserKafka.Data["user.crt"]) > 0 {
				secretForFile.Data["user.crt"] = secretUserKafka.Data["user.crt"]
				userCrt = secretUserKafka.Data["user.crt"]
			}
			if len(secretUserKafka.Data["ca.crt"]) > 0 {
				userCA = secretUserKafka.Data["ca.crt"]
			}
		}
	}

	// Generate user keystore
	if len(userKey) > 0 && len(userCrt) > 0 {
		var (
			privateKey any
			usersCa    []*x509.Certificate
		)

		// Generate p12 keystore
		certificate, err = discover.LoadCertFromPem(userCrt)
		if err != nil {
			return nil, errors.Wrap(err, "Error when load user.crt certificate")
		}
		block, _ := pem.Decode(userKey)
		if block == nil {
			return nil, errors.New("private key error!")
		}
		switch block.Type {
		case "PRIVATE KEY":
			privateKey, err = x509.ParsePKCS8PrivateKey(block.Bytes)
		case "RSA PRIVATE KEY":
			privateKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
		case "EC PRIVATE KEY":
			privateKey, err = x509.ParseECPrivateKey(block.Bytes)
		default:
			return nil, errors.Errorf("Unsupported private key type: %s", block.Type)
		}
		if err != nil {
			return nil, errors.Wrap(err, "Error when load user.key certificate")
		}
		if len(userCA) > 0 {
			caUser, err := discover.LoadCertFromPem(userCA)
			if err != nil {
				return nil, errors.Wrap(err, "Error when load ca.crt certificate from user secret")
			}
			usersCa = append(usersCa, caUser)
		}
		keyStorePassword, err := password.Generate(64, 10, 0, false, true)
		if err != nil {
			return nil, errors.Wrap(err, "Error when generate keystore password")
		}

		p12, err := pkcs12.Modern.Encode(privateKey, certificate, usersCa, keyStorePassword)
		if err != nil {
			return nil, errors.Wrap(err, "Error when encode p12 keystore")
		}
		secretForFile.Data["user.p12"] = p12
		secretForEnv.Data[fmt.Sprintf("KAFKA_USER_PASSWORD_%s", envSuffix)] = []byte(keyStorePassword)
	}

	// Generate CA trustore
	caPassword, err := password.Generate(64, 10, 0, false, true)
	if err != nil {
		return nil, errors.Wrap(err, "Error when generate CA trustore password")
	}
	caTrustore, err := pkcs12.Modern.EncodeTrustStore(certificates, caPassword)
	if err != nil {
		return nil, errors.Wrap(err, "Error when encode CA trustore")
	}
	secretForFile.Data["ca.crt"] = bytes.Join(caCrt, []byte("\n"))
	secretForFile.Data["ca.p12"] = caTrustore
	secretForEnv.Data[fmt.Sprintf("KAFKA_CA_TRUSTSTORE_PASSWORD_%s", envSuffix)] = []byte(caPassword)

	secrets = append(secrets, secretForEnv, secretForFile)

	return secrets, nil
}
