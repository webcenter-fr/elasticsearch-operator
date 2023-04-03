package kibanaapi

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/disaster37/go-kibana-rest/v8"
	kbhandler "github.com/disaster37/kb-handler/v8"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/webcenter-fr/elasticsearch-operator/apis/shared"
	"github.com/webcenter-fr/elasticsearch-operator/controllers/common"
	elasticsearchcontrollers "github.com/webcenter-fr/elasticsearch-operator/controllers/elasticsearch"
	kibanacontrollers "github.com/webcenter-fr/elasticsearch-operator/controllers/kibana"
	core "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Reconciler struct {
	recorder   record.EventRecorder
	log        *logrus.Entry
	reconciler controller.Reconciler
	client.Client
}

func (r *Reconciler) SetLogger(log *logrus.Entry) {
	r.log = log
}

func (r *Reconciler) SetRecorder(recorder record.EventRecorder) {
	r.recorder = recorder
}

func (r *Reconciler) SetReconciler(reconciler controller.Reconciler) {
	r.reconciler = reconciler
}

func GetKibanaHandler(ctx context.Context, o client.Object, kbRef shared.KibanaRef, client client.Client, log *logrus.Entry) (kbHandler kbhandler.KibanaHandler, err error) {

	// Retrieve secret or elasticsearch resource that store the connexion credentials
	var (
		secretNS              types.NamespacedName
		url                   string
		isProvidedCredentials bool
	)
	selfSignedCertificate := false

	// If secret credentials is provided, use it in first priority
	if kbRef.KibanaCredentialSecretRef != nil {
		isProvidedCredentials = true
		secretNS = types.NamespacedName{
			Namespace: o.GetNamespace(),
			Name:      kbRef.KibanaCredentialSecretRef.Name,
		}
	}

	if kbRef.IsManaged() {

		// Get Kibana
		kb, err := common.GetKibanaFromRef(ctx, client, o, kbRef)
		if err != nil {
			return nil, errors.Wrap(err, "Error when get Kibana object from ref")
		}
		if kb == nil {
			return nil, errors.Errorf("Kibana %s/%s not found", kbRef.ManagedKibanaRef.Namespace, kbRef.ManagedKibanaRef.Name)
		}

		// If no Kibana secret credential provided and Elasticsearch is also managed, we can use Elasticsearc credentials secret
		if kbRef.KibanaCredentialSecretRef == nil {

			if kb.Spec.ElasticsearchRef.IsManaged() {
				es, err := common.GetElasticsearchFromRef(ctx, client, o, kb.Spec.ElasticsearchRef)
				if err != nil {
					return nil, errors.Wrap(err, "Error when get Elasticsearch object from ref")
				}
				if es == nil {
					return nil, errors.Errorf("Elasticsearch %s/%s not found", kb.Spec.ElasticsearchRef.ManagedElasticsearchRef.Namespace, kb.Spec.ElasticsearchRef.ManagedElasticsearchRef.Name)
				}

				secretNS = types.NamespacedName{
					Namespace: es.Namespace,
					Name:      elasticsearchcontrollers.GetSecretNameForCredentials(es),
				}

				isProvidedCredentials = false
			} else {
				return nil, errors.New("You must provide credential to access on Kibana API")
			}
		}

		// Compute URL
		if kb.IsTlsEnabled() {
			url = fmt.Sprintf("https://%s.%s.svc:5601", kibanacontrollers.GetServiceName(kb), kb.Namespace)
			selfSignedCertificate = true
		} else {
			url = fmt.Sprintf("http://%s.%s.svc:5601", kibanacontrollers.GetServiceName(kb), kb.Namespace)
		}

	} else if kbRef.IsExternal() {
		url = kbRef.ExternalKibanaRef.Address
	} else {
		log.Error("You must set the way to connect on Kibana")
		return nil, errors.New("You must set the way to connect on Kibana")
	}

	// Read secret to access on Kibana
	secret := &core.Secret{}

	if err = client.Get(ctx, secretNS, secret); err != nil {
		if k8serrors.IsNotFound(err) {
			log.Warnf("Secret %s/%s not yet exist, try later", secretNS.Namespace, secretNS.Name)
			return nil, nil
		}
		log.Errorf("Error when get secret %s/%s: %s", secretNS.Namespace, secretNS.Name, err.Error())
		return nil, err
	}

	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		TLSClientConfig:       &tls.Config{},
		ResponseHeaderTimeout: 10 * time.Second,
		DialContext:           (&net.Dialer{Timeout: 10 * time.Second}).DialContext,
	}
	cfg := kibana.Config{
		Address: url,
	}

	if !isProvidedCredentials {
		cfg.Username = "elastic"
		cfg.Password = string(secret.Data["elastic"])
	} else {
		if len(secret.Data["username"]) == 0 || len(secret.Data["password"]) == 0 {
			return nil, errors.Errorf("The secret %s/%s must contain key `username` and `password`", secret.Namespace, secret.Name)
		}
		cfg.Username = string(secret.Data["username"])
		cfg.Password = string(secret.Data["password"])
	}

	if selfSignedCertificate {
		transport.TLSClientConfig.InsecureSkipVerify = true
	}

	// Create Elasticsearch handler/client
	kbHandler, err = kbhandler.NewKibanaHandler(cfg, log)
	if err != nil {
		return nil, err
	}

	kbHandler.Client().Client.SetTransport(transport)

	if log.Logger.GetLevel() == logrus.DebugLevel {
		kbHandler.Client().Client.SetLogger(log)
		kbHandler.Client().Client.SetDebug(true)
	}

	return kbHandler, nil
}
