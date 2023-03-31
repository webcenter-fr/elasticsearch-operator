package kibanaapi

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/disaster37/kb-handler/v8/mocks"
	"github.com/disaster37/operator-sdk-extra/pkg/mock"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/apis/beat/v1alpha1"
	cerebrocrd "github.com/webcenter-fr/elasticsearch-operator/apis/cerebro/v1alpha1"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1alpha1"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearchapi/v1alpha1"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/apis/kibana/v1alpha1"
	kibanaapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/kibanaapi/v1alpha1"
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/apis/logstash/v1alpha1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	//+kubebuilder:scaffold:imports
	//+kubebuilder:scaffold:imports
)

var testEnv *envtest.Environment

type KibanaapiControllerTestSuite struct {
	suite.Suite
	k8sClient         client.Client
	cfg               *rest.Config
	mockCtrl          *gomock.Controller
	mockKibanaHandler *mocks.MockKibanaHandler
}

func TestKibanachapiControllerSuite(t *testing.T) {
	suite.Run(t, new(KibanaapiControllerTestSuite))
}

func (t *KibanaapiControllerTestSuite) SetupSuite() {

	t.mockCtrl = gomock.NewController(t.T())
	t.mockKibanaHandler = mocks.NewMockKibanaHandler(t.mockCtrl)

	logf.SetLogger(zap.New(zap.UseDevMode(true)))
	logrus.SetLevel(logrus.TraceLevel)
	logrus.SetFormatter(&logrus.TextFormatter{
		DisableQuote: true,
	})

	// Setup testenv
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("../..", "config", "crd", "bases"),
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
	kibanaapicrd.MustSetUpIndex(k8sManager)
	elasticsearchapicrd.MustSetUpIndex(k8sManager)
	elasticsearchcrd.MustSetUpIndex(k8sManager)
	kibanacrd.MustSetUpIndex(k8sManager)
	logstashcrd.MustSetUpIndex(k8sManager)
	beatcrd.MustSetUpIndexForFilebeat(k8sManager)
	beatcrd.MustSetUpIndexForMetricbeat(k8sManager)
	cerebrocrd.MustSetUpIndexCerebro(k8sManager)
	cerebrocrd.MustSetUpIndexHost(k8sManager)

	// Init controllers

	roleReconciler := NewRoleReconciler(k8sClient, scheme.Scheme)
	roleReconciler.SetLogger(logrus.WithFields(logrus.Fields{
		"type": "kibanaRoleController",
	}))
	roleReconciler.SetRecorder(k8sManager.GetEventRecorderFor("kibana-role-controller"))
	roleReconciler.SetReconciler(mock.NewMockReconciler(roleReconciler, t.mockKibanaHandler))
	if err = roleReconciler.SetupWithManager(k8sManager); err != nil {
		panic(err)
	}

	spaceReconciler := NewUserSpaceReconciler(k8sClient, scheme.Scheme)
	spaceReconciler.SetLogger(logrus.WithFields(logrus.Fields{
		"type": "kibanaUserSpaceController",
	}))
	spaceReconciler.SetRecorder(k8sManager.GetEventRecorderFor("kibana-user-space-controller"))
	spaceReconciler.SetReconciler(mock.NewMockReconciler(spaceReconciler, t.mockKibanaHandler))
	if err = spaceReconciler.SetupWithManager(k8sManager); err != nil {
		panic(err)
	}

	pipelineReconciler := NewLogstashPipelineReconciler(k8sClient, scheme.Scheme)
	pipelineReconciler.SetLogger(logrus.WithFields(logrus.Fields{
		"type": "kibanaLogstashPipelineController",
	}))
	pipelineReconciler.SetRecorder(k8sManager.GetEventRecorderFor("kibana-logstash-pipeline-controller"))
	pipelineReconciler.SetReconciler(mock.NewMockReconciler(pipelineReconciler, t.mockKibanaHandler))
	if err = pipelineReconciler.SetupWithManager(k8sManager); err != nil {
		panic(err)
	}

	go func() {
		err = k8sManager.Start(ctrl.SetupSignalHandler())
		if err != nil {
			panic(err)
		}
	}()
}

func (t *KibanaapiControllerTestSuite) TearDownSuite() {

	// Teardown the test environment once controller is fnished.
	// Otherwise from Kubernetes 1.21+, teardon timeouts waiting on
	// kube-apiserver to return
	err := testEnv.Stop()
	if err != nil {
		panic(err)
	}
}

func (t *KibanaapiControllerTestSuite) BeforeTest(suiteName, testName string) {

}

func (t *KibanaapiControllerTestSuite) AfterTest(suiteName, testName string) {
	defer t.mockCtrl.Finish()
}
