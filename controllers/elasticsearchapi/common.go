package elasticsearchapi

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"time"

	eshandler "github.com/disaster37/es-handler/v8"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/elastic/elastic-transport-go/v8/elastictransport"
	elastic "github.com/elastic/go-elasticsearch/v8"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearchapi/v1alpha1"
	"github.com/webcenter-fr/elasticsearch-operator/apis/shared"
	"github.com/webcenter-fr/elasticsearch-operator/controllers/common"
	elasticsearchcontrollers "github.com/webcenter-fr/elasticsearch-operator/controllers/elasticsearch"
	core "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
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

func MustSetUpIndex(k8sManager manager.Manager) {
	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &elasticsearchapicrd.License{}, "spec.secretRef.name", func(o client.Object) []string {
		p := o.(*elasticsearchapicrd.License)
		if p.Spec.SecretRef != nil {
			return []string{p.Spec.SecretRef.Name}
		}
		return []string{}
	}); err != nil {
		panic(err)
	}

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &elasticsearchapicrd.User{}, "spec.secretRef.name", func(o client.Object) []string {
		p := o.(*elasticsearchapicrd.User)
		if p.Spec.SecretRef != nil {
			return []string{p.Spec.SecretRef.Name}
		}
		return []string{}
	}); err != nil {
		panic(err)
	}
}

func GetElasticsearchHandler(ctx context.Context, o client.Object, esRef shared.ElasticsearchRef, client client.Client, log *logrus.Entry) (esHandler eshandler.ElasticsearchHandler, err error) {

	// Retrieve secret or elasticsearch resource that store the connexion credentials
	var secretNS types.NamespacedName
	secretName := ""
	isManaged := false
	hosts := []string{}
	selfSignedCertificate := false
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

		if !es.IsTlsApiEnabled() {
			hosts = append(hosts, fmt.Sprintf("http://%s.%s.svc:9200", serviceName, es.Namespace))
		} else {
			hosts = append(hosts, fmt.Sprintf("https://%s.%s.svc:9200", serviceName, es.Namespace))
			selfSignedCertificate = true
		}

		secretNS = types.NamespacedName{
			Namespace: es.Namespace,
			Name:      secretName,
		}

	} else if esRef.IsExternal() {
		secretName = esRef.ExternalElasticsearchRef.SecretRef.Name
		hosts = esRef.ExternalElasticsearchRef.Addresses

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

	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		TLSClientConfig:       &tls.Config{},
		ResponseHeaderTimeout: 10 * time.Second,
		DialContext:           (&net.Dialer{Timeout: 10 * time.Second}).DialContext,
	}
	cfg := elastic.Config{
		Transport: transport,
		Addresses: hosts,
	}

	if log.Logger.GetLevel() == logrus.DebugLevel {
		cfg.Logger = &elastictransport.JSONLogger{EnableRequestBody: true, EnableResponseBody: true, Output: log.Logger.Out}
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
