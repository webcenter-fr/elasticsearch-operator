package logstash

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"

	"emperror.dev/errors"
	"github.com/sethvargo/go-password/password"
	discovercrd "github.com/webcenter-fr/elasticsearch-operator/api/discover/v1"
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/api/logstash/v1"
	"github.com/webcenter-fr/elasticsearch-operator/internal/controller/discover"
	logstashcontrollers "github.com/webcenter-fr/elasticsearch-operator/internal/controller/logstash"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"software.sslmate.com/src/go-pkcs12"
)

// buildLogstashSecrets permit to build secret from Logstash discovered
func buildLogstashSecrets(kk *discovercrd.Logstash, ls *logstashcrd.Logstash, secretCaLogstash *corev1.Secret, secretUserLogstash *corev1.Secret, secretCustomCaLogstash *corev1.Secret) (secrets []*corev1.Secret, err error) {
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

	// Compute CA certificates
	caCerts := make([]string, 0, 1)
	if secretCaLogstash != nil && secretCaLogstash.Data != nil && len(secretCaLogstash.Data["ca.crt"]) > 0 {
		caCerts = append(caCerts, string(secretCaLogstash.Data["ca.crt"]))
	}
	if secretCustomCaLogstash != nil && secretCustomCaLogstash.Data != nil {
		for certificateName, certificateData := range secretCustomCaLogstash.Data {
			if strings.HasSuffix(certificateName, ".crt") || strings.HasSuffix(certificateName, ".pem") {
				caCerts = append(caCerts, string(certificateData))
			}
		}
	}
	secretForFile.Data["ca.crt"] = []byte(strings.Join(caCerts, "\n"))

	// Compute Logstash URL
	logstashHosts := make([]string, 0, 1)
	if ls != nil {
		if kk.Spec.LogstashRef.ManagedLogstashRef.TargetService != "" {
			logstashHosts = append(logstashHosts, fmt.Sprintf("%s.%s.svc:%d", logstashcontrollers.GetServiceName(ls, kk.Spec.LogstashRef.ManagedLogstashRef.TargetService), ls.Namespace, kk.Spec.LogstashRef.ManagedLogstashRef.Port))
		} else {
			for i := 0; i < int(ls.Spec.Deployment.Replicas); i++ {
				logstashHosts = append(logstashHosts, fmt.Sprintf("%s-%d.%s.%s.svc:%d", logstashcontrollers.GetStatefulsetName(ls), i, logstashcontrollers.GetGlobalServiceName(ls), ls.Namespace, kk.Spec.LogstashRef.ManagedLogstashRef.Port))
			}
		}
	} else if kk.Spec.LogstashRef.IsExternal() && len(kk.Spec.LogstashRef.ExternalLogstashRef.Addresses) > 0 {
		logstashHosts = kk.Spec.LogstashRef.ExternalLogstashRef.Addresses
	}
	secretForEnv.Data[fmt.Sprintf("LOGSTASH_HOSTS_%s", envSuffix)] = []byte(strings.Join(logstashHosts, ","))

	// Compute Logstash credentials
	if secretUserLogstash != nil && secretUserLogstash.Data != nil {
		if cert, ok := secretUserLogstash.Data["user.crt"]; ok {
			secretForFile.Data["user.crt"] = cert
			userCrt = cert
		}
		if key, ok := secretUserLogstash.Data["user.key"]; ok {
			secretForFile.Data["user.key"] = key
			userKey = key
		}

		if ca, ok := secretUserLogstash.Data["ca.crt"]; ok {
			userCA = ca
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
		secretForEnv.Data[fmt.Sprintf("LOGSTASH_USER_PASSWORD_%s", envSuffix)] = []byte(keyStorePassword)
	}

	secrets = append(secrets, secretForEnv, secretForFile)

	return secrets, nil
}
