package elasticsearchapi

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/disaster37/es-handler/v8/mocks"
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

type ElasticsearchapiControllerTestSuite struct {
	suite.Suite
	k8sClient                client.Client
	cfg                      *rest.Config
	mockCtrl                 *gomock.Controller
	mockElasticsearchHandler *mocks.MockElasticsearchHandler
}

func TestElasticsearchapiControllerSuite(t *testing.T) {
	suite.Run(t, new(ElasticsearchapiControllerTestSuite))
}

func (t *ElasticsearchapiControllerTestSuite) SetupSuite() {

	t.mockCtrl = gomock.NewController(t.T())
	t.mockElasticsearchHandler = mocks.NewMockElasticsearchHandler(t.mockCtrl)

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
	elasticsearchapicrd.MustSetUpIndex(k8sManager)
	elasticsearchcrd.MustSetUpIndex(k8sManager)
	kibanacrd.MustSetUpIndex(k8sManager)
	logstashcrd.MustSetUpIndex(k8sManager)
	beatcrd.MustSetUpIndexForFilebeat(k8sManager)
	beatcrd.MustSetUpIndexForMetricbeat(k8sManager)
	cerebrocrd.MustSetUpIndexCerebro(k8sManager)
	cerebrocrd.MustSetUpIndexHost(k8sManager)
	kibanaapicrd.MustSetUpIndex(k8sManager)

	// Init controllers
	userReconciler := NewUserReconciler(k8sClient, scheme.Scheme)
	userReconciler.SetLogger(logrus.WithFields(logrus.Fields{
		"type": "elasticsearchUserController",
	}))
	userReconciler.SetRecorder(k8sManager.GetEventRecorderFor("elasticsearch-user-controller"))
	userReconciler.SetReconciler(mock.NewMockReconciler(userReconciler, t.mockElasticsearchHandler))
	if err = userReconciler.SetupWithManager(k8sManager); err != nil {
		panic(err)
	}

	licenseReconciler := NewLicenseReconciler(k8sClient, scheme.Scheme)
	licenseReconciler.SetLogger(logrus.WithFields(logrus.Fields{
		"type": "elasticsearchLicenseController",
	}))
	licenseReconciler.SetRecorder(k8sManager.GetEventRecorderFor("elasticsearch-license-controller"))
	licenseReconciler.SetReconciler(mock.NewMockReconciler(licenseReconciler, t.mockElasticsearchHandler))
	if err = licenseReconciler.SetupWithManager(k8sManager); err != nil {
		panic(err)
	}

	roleReconciler := NewRoleReconciler(k8sClient, scheme.Scheme)
	roleReconciler.SetLogger(logrus.WithFields(logrus.Fields{
		"type": "elasticsearchRoleController",
	}))
	roleReconciler.SetRecorder(k8sManager.GetEventRecorderFor("elasticsearch-role-controller"))
	roleReconciler.SetReconciler(mock.NewMockReconciler(roleReconciler, t.mockElasticsearchHandler))
	if err = roleReconciler.SetupWithManager(k8sManager); err != nil {
		panic(err)
	}

	roleMappingReconciler := NewRoleMappingReconciler(k8sClient, scheme.Scheme)
	roleMappingReconciler.SetLogger(logrus.WithFields(logrus.Fields{
		"type": "elasticsearchRoleMappingController",
	}))
	roleMappingReconciler.SetRecorder(k8sManager.GetEventRecorderFor("elasticsearch-rolemapping-controller"))
	roleMappingReconciler.SetReconciler(mock.NewMockReconciler(roleMappingReconciler, t.mockElasticsearchHandler))
	if err = roleMappingReconciler.SetupWithManager(k8sManager); err != nil {
		panic(err)
	}

	ilmReconciler := NewIndexLifecyclePolicyReconciler(k8sClient, scheme.Scheme)
	ilmReconciler.SetLogger(logrus.WithFields(logrus.Fields{
		"type": "elasticsearchIndexLifecyclePolicyController",
	}))
	ilmReconciler.SetRecorder(k8sManager.GetEventRecorderFor("elasticsearch-indexlifecyclepolicy-controller"))
	ilmReconciler.SetReconciler(mock.NewMockReconciler(ilmReconciler, t.mockElasticsearchHandler))
	if err = ilmReconciler.SetupWithManager(k8sManager); err != nil {
		panic(err)
	}

	slmReconciler := NewSnapshotLifecyclePolicyReconciler(k8sClient, scheme.Scheme)
	slmReconciler.SetLogger(logrus.WithFields(logrus.Fields{
		"type": "elasticsearchSnapshotLifecyclePolicyController",
	}))
	slmReconciler.SetRecorder(k8sManager.GetEventRecorderFor("elasticsearch-snapshotlifecyclepolicy-controller"))
	slmReconciler.SetReconciler(mock.NewMockReconciler(slmReconciler, t.mockElasticsearchHandler))
	if err = slmReconciler.SetupWithManager(k8sManager); err != nil {
		panic(err)
	}

	snapshotRepositoryReconciler := NewSnapshotRepositoryReconciler(k8sClient, scheme.Scheme)
	snapshotRepositoryReconciler.SetLogger(logrus.WithFields(logrus.Fields{
		"type": "elasticsearchSnapshotRepositoryController",
	}))
	snapshotRepositoryReconciler.SetRecorder(k8sManager.GetEventRecorderFor("elasticsearch-snapshotrepository-controller"))
	snapshotRepositoryReconciler.SetReconciler(mock.NewMockReconciler(snapshotRepositoryReconciler, t.mockElasticsearchHandler))
	if err = snapshotRepositoryReconciler.SetupWithManager(k8sManager); err != nil {
		panic(err)
	}

	componentTemplateReconciler := NewComponentTemplateReconciler(k8sClient, scheme.Scheme)
	componentTemplateReconciler.SetLogger(logrus.WithFields(logrus.Fields{
		"type": "elasticsearchComponentTemplateController",
	}))
	componentTemplateReconciler.SetRecorder(k8sManager.GetEventRecorderFor("elasticsearch-componenttemplate-controller"))
	componentTemplateReconciler.SetReconciler(mock.NewMockReconciler(componentTemplateReconciler, t.mockElasticsearchHandler))
	if err = componentTemplateReconciler.SetupWithManager(k8sManager); err != nil {
		panic(err)
	}

	indexTemplateReconciler := NewIndexTemplateReconciler(k8sClient, scheme.Scheme)
	indexTemplateReconciler.SetLogger(logrus.WithFields(logrus.Fields{
		"type": "elasticsearchIndexTemplateController",
	}))
	indexTemplateReconciler.SetRecorder(k8sManager.GetEventRecorderFor("elasticsearch-indextemplate-controller"))
	indexTemplateReconciler.SetReconciler(mock.NewMockReconciler(indexTemplateReconciler, t.mockElasticsearchHandler))
	if err = indexTemplateReconciler.SetupWithManager(k8sManager); err != nil {
		panic(err)
	}

	watchReconciler := NewWatchReconciler(k8sClient, scheme.Scheme)
	watchReconciler.SetLogger(logrus.WithFields(logrus.Fields{
		"type": "elasticsearchWatchController",
	}))
	watchReconciler.SetRecorder(k8sManager.GetEventRecorderFor("elasticsearch-watch-controller"))
	watchReconciler.SetReconciler(mock.NewMockReconciler(watchReconciler, t.mockElasticsearchHandler))
	if err = watchReconciler.SetupWithManager(k8sManager); err != nil {
		panic(err)
	}

	go func() {
		err = k8sManager.Start(ctrl.SetupSignalHandler())
		if err != nil {
			panic(err)
		}
	}()
}

func (t *ElasticsearchapiControllerTestSuite) TearDownSuite() {

	// Teardown the test environment once controller is fnished.
	// Otherwise from Kubernetes 1.21+, teardon timeouts waiting on
	// kube-apiserver to return
	err := testEnv.Stop()
	if err != nil {
		panic(err)
	}
}

func (t *ElasticsearchapiControllerTestSuite) BeforeTest(suiteName, testName string) {

}

func (t *ElasticsearchapiControllerTestSuite) AfterTest(suiteName, testName string) {
	defer t.mockCtrl.Finish()
}
