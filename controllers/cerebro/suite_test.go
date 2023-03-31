package cerebro

import (
	"path/filepath"
	"testing"
	"time"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/apis/beat/v1alpha1"
	cerebrocrd "github.com/webcenter-fr/elasticsearch-operator/apis/cerebro/v1alpha1"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1alpha1"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearchapi/v1alpha1"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/apis/kibana/v1alpha1"
	kibanaapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/kibanaapi/v1alpha1"
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/apis/logstash/v1alpha1"
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

type CerebroControllerTestSuite struct {
	suite.Suite
	k8sClient client.Client
	cfg       *rest.Config
}

func TestControllerSuite(t *testing.T) {
	suite.Run(t, new(CerebroControllerTestSuite))
}

func (t *CerebroControllerTestSuite) SetupSuite() {

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
	err = logstashcrd.AddToScheme(scheme.Scheme)
	if err != nil {
		panic(err)
	}
	err = beatcrd.AddToScheme(scheme.Scheme)
	if err != nil {
		panic(err)
	}
	err = cerebrocrd.AddToScheme(scheme.Scheme)
	if err != nil {
		panic(err)
	}
	err = monitoringv1.AddToScheme(scheme.Scheme)
	if err != nil {
		panic(err)
	}
	err = kibanaapicrd.AddToScheme(scheme.Scheme)
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

	// Add indexers
	elasticsearchcrd.MustSetUpIndex(k8sManager)
	kibanacrd.MustSetUpIndex(k8sManager)
	logstashcrd.MustSetUpIndex(k8sManager)
	beatcrd.MustSetUpIndexForFilebeat(k8sManager)
	beatcrd.MustSetUpIndexForMetricbeat(k8sManager)
	cerebrocrd.MustSetUpIndexCerebro(k8sManager)
	cerebrocrd.MustSetUpIndexHost(k8sManager)
	kibanaapicrd.MustSetUpIndex(k8sManager)

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

	cerebroReconciler := NewCerebroReconciler(k8sClient, scheme.Scheme)
	cerebroReconciler.SetLogger(logrus.WithFields(logrus.Fields{
		"type": "cerebroController",
	}))
	cerebroReconciler.SetRecorder(k8sManager.GetEventRecorderFor("cerebro-controller"))
	cerebroReconciler.SetReconciler(cerebroReconciler)
	if err = cerebroReconciler.SetupWithManager(k8sManager); err != nil {
		panic(err)
	}

	go func() {
		err = k8sManager.Start(ctrl.SetupSignalHandler())
		if err != nil {
			panic(err)
		}
	}()

}

func (t *CerebroControllerTestSuite) TearDownSuite() {

	// Teardown the test environment once controller is fnished.
	// Otherwise from Kubernetes 1.21+, teardon timeouts waiting on
	// kube-apiserver to return
	err := testEnv.Stop()
	if err != nil {
		panic(err)
	}

}

func (t *CerebroControllerTestSuite) BeforeTest(suiteName, testName string) {

}

func (t *CerebroControllerTestSuite) AfterTest(suiteName, testName string) {
}