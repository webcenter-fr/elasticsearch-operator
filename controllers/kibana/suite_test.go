package kibana

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1alpha1"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearchapi/v1alpha1"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/apis/kibana/v1alpha1"
	elasticsearchcontrollers "github.com/webcenter-fr/elasticsearch-operator/controllers/elasticsearch"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	//+kubebuilder:scaffold:imports
)

var testEnv *envtest.Environment

type KibanaControllerTestSuite struct {
	suite.Suite
	k8sClient client.Client
	cfg       *rest.Config
}

func TestControllerSuite(t *testing.T) {
	suite.Run(t, new(KibanaControllerTestSuite))
}

func (t *KibanaControllerTestSuite) SetupSuite() {

	logf.SetLogger(zap.New(zap.UseDevMode(true)))
	logrus.SetLevel(logrus.TraceLevel)
	logrus.SetFormatter(&logrus.TextFormatter{
		DisableQuote: true,
	})

	// Setup testenv
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("../..", "config", "crd", "bases"),
			filepath.Join("../..", "config", "crd", "externals"),
		},
		ErrorIfCRDPathMissing:    true,
		ControlPlaneStopTimeout:  120 * time.Second,
		ControlPlaneStartTimeout: 120 * time.Second,
	}
	cfg, err := testEnv.Start()
	if err != nil {
		panic(err)
	}
	t.cfg = cfg

	// Add CRD sheme
	err = scheme.AddToScheme(scheme.Scheme)
	if err != nil {
		panic(err)
	}
	err = elasticsearchapicrd.AddToScheme(scheme.Scheme)
	if err != nil {
		panic(err)
	}
	err = elasticsearchcrd.AddToScheme(scheme.Scheme)
	if err != nil {
		panic(err)
	}
	err = kibanacrd.AddToScheme(scheme.Scheme)
	if err != nil {
		panic(err)
	}
	err = monitoringv1.AddToScheme(scheme.Scheme)
	if err != nil {
		panic(err)
	}

	// Init k8smanager and k8sclient
	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	if err != nil {
		panic(err)
	}
	k8sClient := k8sManager.GetClient()
	t.k8sClient = k8sClient

	// Add indexers on Elasticsearch to track secret change
	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &elasticsearchcrd.Elasticsearch{}, "spec.tls.certificateSecretRef.name", func(o client.Object) []string {
		p := o.(*elasticsearchcrd.Elasticsearch)
		if p.IsTlsApiEnabled() && !p.IsSelfManagedSecretForTlsApi() {
			return []string{p.Spec.Tls.CertificateSecretRef.Name}
		}
		return []string{}
	}); err != nil {
		panic(err)
	}

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &elasticsearchcrd.Elasticsearch{}, "spec.globalNodeGroup.keystoreSecretRef.name", func(o client.Object) []string {
		p := o.(*elasticsearchcrd.Elasticsearch)
		if p.Spec.GlobalNodeGroup.KeystoreSecretRef != nil {
			return []string{p.Spec.GlobalNodeGroup.KeystoreSecretRef.Name}
		}
		return []string{}
	}); err != nil {
		panic(err)
	}

	// Add indexers on Kibana to track secret change
	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &kibanacrd.Kibana{}, "spec.tls.certificateSecretRef.name", func(o client.Object) []string {
		p := o.(*kibanacrd.Kibana)
		if p.IsTlsEnabled() && !p.IsSelfManagedSecretForTls() {
			return []string{p.Spec.Tls.CertificateSecretRef.Name}
		}
		return []string{}
	}); err != nil {
		panic(err)
	}

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &kibanacrd.Kibana{}, "spec.keystoreSecretRef.name", func(o client.Object) []string {
		p := o.(*kibanacrd.Kibana)
		if p.Spec.KeystoreSecretRef != nil {
			return []string{p.Spec.KeystoreSecretRef.Name}
		}
		return []string{}
	}); err != nil {
		panic(err)
	}

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &kibanacrd.Kibana{}, "spec.elasticsearchRef.managed.name", func(o client.Object) []string {
		p := o.(*kibanacrd.Kibana)
		if p.Spec.ElasticsearchRef.IsManaged() {
			return []string{p.Spec.ElasticsearchRef.ManagedElasticsearchRef.Name}
		}
		return []string{}
	}); err != nil {
		panic(err)
	}

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &kibanacrd.Kibana{}, "spec.elasticsearchRef.elasticsearchCASecretRef.name", func(o client.Object) []string {
		p := o.(*kibanacrd.Kibana)
		if p.Spec.ElasticsearchRef.ElasticsearchCaSecretRef != nil {
			return []string{p.Spec.ElasticsearchRef.ElasticsearchCaSecretRef.Name}
		}
		return []string{}
	}); err != nil {
		panic(err)
	}

	if err := k8sManager.GetFieldIndexer().IndexField(context.Background(), &kibanacrd.Kibana{}, "spec.elasticsearchRef.external.secretRef.name", func(o client.Object) []string {
		p := o.(*kibanacrd.Kibana)
		if p.Spec.ElasticsearchRef.IsExternal() && p.Spec.ElasticsearchRef.ExternalElasticsearchRef.SecretRef != nil {
			return []string{p.Spec.ElasticsearchRef.ExternalElasticsearchRef.SecretRef.Name}
		}
		return []string{}
	}); err != nil {
		panic(err)
	}

	// Init controllers

	elasticsearchReconciler := elasticsearchcontrollers.NewElasticsearchReconciler(k8sClient, scheme.Scheme)
	elasticsearchReconciler.SetLogger(logrus.WithFields(logrus.Fields{
		"type": "elasticsearchController",
	}))
	elasticsearchReconciler.SetRecorder(k8sManager.GetEventRecorderFor("elasticsearch-controller"))
	elasticsearchReconciler.SetReconciler(elasticsearchReconciler)
	if err = elasticsearchReconciler.SetupWithManager(k8sManager); err != nil {
		panic(err)
	}

	kibanaReconciler := NewKibanaReconciler(k8sClient, scheme.Scheme)
	kibanaReconciler.SetLogger(logrus.WithFields(logrus.Fields{
		"type": "kibanaController",
	}))
	kibanaReconciler.SetRecorder(k8sManager.GetEventRecorderFor("kibana-controller"))
	kibanaReconciler.SetReconciler(kibanaReconciler)
	if err = kibanaReconciler.SetupWithManager(k8sManager); err != nil {
		panic(err)
	}

	go func() {
		err = k8sManager.Start(ctrl.SetupSignalHandler())
		if err != nil {
			panic(err)
		}
	}()

}

func (t *KibanaControllerTestSuite) TearDownSuite() {

	// Teardown the test environment once controller is fnished.
	// Otherwise from Kubernetes 1.21+, teardon timeouts waiting on
	// kube-apiserver to return
	err := testEnv.Stop()
	if err != nil {
		panic(err)
	}

}

func (t *KibanaControllerTestSuite) BeforeTest(suiteName, testName string) {

}

func (t *KibanaControllerTestSuite) AfterTest(suiteName, testName string) {
}
