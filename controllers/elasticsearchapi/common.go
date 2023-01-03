package elasticsearchapi

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"

	eshandler "github.com/disaster37/es-handler/v8"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	elastic "github.com/elastic/go-elasticsearch/v8"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1alpha1"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearchapi/v1alpha1"
	elasticsearchcontrollers "github.com/webcenter-fr/elasticsearch-operator/controllers/elasticsearch"
	core "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
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

func GetElasticsearchHandler(ctx context.Context, o *elasticsearchapicrd.ElasticsearchRefSpec, client client.Client, req ctrl.Request, log *logrus.Entry) (esHandler eshandler.ElasticsearchHandler, err error) {

	// Retrieve secret or elasticsearch resource that store the connexion credentials
	secretName := ""
	isManagedByOperator := false
	hosts := []string{}
	selfSignedCertificate := false
	if o != nil && o.IsManagedByOperator() {
		isManagedByOperator = true
		// From Elasticsearch resource
		es := &elasticsearchcrd.Elasticsearch{}
		if err = client.Get(context.Background(), types.NamespacedName{Namespace: req.Namespace, Name: o.Name}, es); err != nil {
			if k8serrors.IsNotFound(err) {
				log.Warnf("Elasticsearch %s not yet exist, try later", o.Name)
				return nil, nil
			}
			log.Errorf("Error when get resource: %s", err.Error())
			return nil, err
		}

		// Get secret that store credential
		secretName = elasticsearchcontrollers.GetSecretNameForCredentials(es)

		if !es.IsTlsApiEnabled() {
			hosts = append(hosts, fmt.Sprintf("http://%s.%s.svc:9200", elasticsearchcontrollers.GetGlobalServiceName(es), es.Namespace))
		} else {
			hosts = append(hosts, fmt.Sprintf("https://%s.%s.svc:9200", elasticsearchcontrollers.GetGlobalServiceName(es), es.Namespace))
			selfSignedCertificate = true
		}

	} else if len(o.Addresses) > 0 && o.SecretName != "" {
		secretName = o.SecretName
		hosts = o.Addresses
	} else {
		log.Error("You must set the way to connect on Elasticsearch")
		return nil, errors.New("You must set the way to connect on Elasticsearch")
	}

	// Read settings to access on Elasticsearch api
	secret := &core.Secret{}
	secretNS := types.NamespacedName{
		Namespace: req.NamespacedName.Namespace,
		Name:      secretName,
	}
	if err = client.Get(ctx, secretNS, secret); err != nil {
		if k8serrors.IsNotFound(err) {
			log.Warnf("Secret %s not yet exist, try later", secretName)
			return nil, nil
		}
		log.Errorf("Error when get resource: %s", err.Error())
		return nil, err
	}

	transport := &http.Transport{
		Proxy:           http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{},
	}
	cfg := elastic.Config{
		Transport: transport,
		Addresses: hosts,
	}

	if isManagedByOperator {
		cfg.Username = "elastic"
		cfg.Password = string(secret.Data["elastic"])
	} else {
		for user, password := range secret.Data {
			cfg.Username = user
			cfg.Password = string(password)
			break
		}
	}

	if selfSignedCertificate {
		transport.TLSClientConfig.InsecureSkipVerify = true
	}

	// Create Elasticsearch handler/client
	esHandler, err = eshandler.NewElasticsearchHandler(cfg, log)
	if err != nil {
		return nil, err
	}

	return esHandler, nil
}
