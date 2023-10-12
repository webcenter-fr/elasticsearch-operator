package elasticsearchapi

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	eshandler "github.com/disaster37/es-handler/v8"
	"github.com/disaster37/es-handler/v8/mocks"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/pkg/mock"
	"github.com/disaster37/operator-sdk-extra/pkg/object"
	olivere "github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/apis/beat/v1"
	cerebrocrd "github.com/webcenter-fr/elasticsearch-operator/apis/cerebro/v1"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearchapi/v1"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/apis/kibana/v1"
	kibanaapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/kibanaapi/v1"
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/apis/logstash/v1"
	"go.uber.org/mock/gomock"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
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
	if err = controller.SetupIndexerWithManager(
		k8sManager,
		elasticsearchcrd.SetupElasticsearchIndexer,
		kibanacrd.SetupKibanaIndexer,
		logstashcrd.SetupLogstashIndexer,
		beatcrd.SetupFilebeatIndexer,
		beatcrd.SetupMetricbeatIndexer,
		cerebrocrd.SetupCerebroIndexer,
		cerebrocrd.SetupHostIndexer,
		elasticsearchapicrd.SetupLicenceIndexer,
		elasticsearchapicrd.SetupUserIndexexer,
	); err != nil {
		panic(err)
	}

	// Init controllers

	userReconciler := NewUserReconciler(
		k8sClient,
		logrus.NewEntry(logrus.StandardLogger()),
		k8sManager.GetEventRecorderFor("elasticsearch-user-controller"),
	)
	userReconciler.(*UserReconciler).reconcilerAction = mock.NewMockRemoteReconcilerAction[*elasticsearchapicrd.User, *olivere.XPackSecurityPutUserRequest, eshandler.ElasticsearchHandler](
		userReconciler.(*UserReconciler).reconcilerAction,
		func(ctx context.Context, req reconcile.Request, o object.RemoteObject) (handler controller.RemoteExternalReconciler[*elasticsearchapicrd.User, *olivere.XPackSecurityPutUserRequest, eshandler.ElasticsearchHandler], res reconcile.Result, err error) {
			return newUserApiClient(t.mockElasticsearchHandler), res, nil
		},
	)
	if err = userReconciler.SetupWithManager(k8sManager); err != nil {
		panic(err)
	}

	licenseReconciler := NewLicenseReconciler(
		k8sClient,
		logrus.NewEntry(logrus.StandardLogger()),
		k8sManager.GetEventRecorderFor("elasticsearch-license-controller"),
	)
	licenseReconciler.(*LicenseReconciler).reconcilerAction = mock.NewMockRemoteReconcilerAction[*elasticsearchapicrd.License, *olivere.XPackInfoLicense, eshandler.ElasticsearchHandler](
		licenseReconciler.(*LicenseReconciler).reconcilerAction,
		func(ctx context.Context, req reconcile.Request, o object.RemoteObject) (handler controller.RemoteExternalReconciler[*elasticsearchapicrd.License, *olivere.XPackInfoLicense, eshandler.ElasticsearchHandler], res reconcile.Result, err error) {
			return newLicenseApiClient(t.mockElasticsearchHandler), res, nil
		},
	)
	if err = licenseReconciler.SetupWithManager(k8sManager); err != nil {
		panic(err)
	}

	roleReconciler := NewRoleReconciler(
		k8sClient,
		logrus.NewEntry(logrus.StandardLogger()),
		k8sManager.GetEventRecorderFor("elasticsearch-role-controller"),
	)
	roleReconciler.(*RoleReconciler).reconcilerAction = mock.NewMockRemoteReconcilerAction[*elasticsearchapicrd.Role, *eshandler.XPackSecurityRole, eshandler.ElasticsearchHandler](
		roleReconciler.(*RoleReconciler).reconcilerAction,
		func(ctx context.Context, req reconcile.Request, o object.RemoteObject) (handler controller.RemoteExternalReconciler[*elasticsearchapicrd.Role, *eshandler.XPackSecurityRole, eshandler.ElasticsearchHandler], res reconcile.Result, err error) {
			return newRoleApiClient(t.mockElasticsearchHandler), res, nil
		},
	)
	if err = roleReconciler.SetupWithManager(k8sManager); err != nil {
		panic(err)
	}

	roleMappingReconciler := NewRoleMappingReconciler(
		k8sClient,
		logrus.NewEntry(logrus.StandardLogger()),
		k8sManager.GetEventRecorderFor("elasticsearch-rolemapping-controller"),
	)
	roleMappingReconciler.(*RoleMappingReconciler).reconcilerAction = mock.NewMockRemoteReconcilerAction[*elasticsearchapicrd.RoleMapping, *olivere.XPackSecurityRoleMapping, eshandler.ElasticsearchHandler](
		roleMappingReconciler.(*RoleMappingReconciler).reconcilerAction,
		func(ctx context.Context, req reconcile.Request, o object.RemoteObject) (handler controller.RemoteExternalReconciler[*elasticsearchapicrd.RoleMapping, *olivere.XPackSecurityRoleMapping, eshandler.ElasticsearchHandler], res reconcile.Result, err error) {
			return newRoleMappingApiClient(t.mockElasticsearchHandler), res, nil
		},
	)
	if err = roleMappingReconciler.SetupWithManager(k8sManager); err != nil {
		panic(err)
	}

	ilmReconciler := NewIndexLifecyclePolicyReconciler(
		k8sClient,
		logrus.NewEntry(logrus.StandardLogger()),
		k8sManager.GetEventRecorderFor("elasticsearch-indexlifecyclepolicy-controller"),
	)
	ilmReconciler.(*IndexLifecyclePolicyReconciler).reconcilerAction = mock.NewMockRemoteReconcilerAction[*elasticsearchapicrd.IndexLifecyclePolicy, *olivere.XPackIlmGetLifecycleResponse, eshandler.ElasticsearchHandler](
		ilmReconciler.(*IndexLifecyclePolicyReconciler).reconcilerAction,
		func(ctx context.Context, req reconcile.Request, o object.RemoteObject) (handler controller.RemoteExternalReconciler[*elasticsearchapicrd.IndexLifecyclePolicy, *olivere.XPackIlmGetLifecycleResponse, eshandler.ElasticsearchHandler], res reconcile.Result, err error) {
			return newIndexLifecyclePolicyApiClient(t.mockElasticsearchHandler), res, nil
		},
	)
	if err = ilmReconciler.SetupWithManager(k8sManager); err != nil {
		panic(err)
	}

	slmReconciler := NewSnapshotLifecyclePolicyReconciler(
		k8sClient,
		logrus.NewEntry(logrus.StandardLogger()),
		k8sManager.GetEventRecorderFor("elasticsearch-snapshotlifecyclepolicy-controller"),
	)
	slmReconciler.(*SnapshotLifecyclePolicyReconciler).reconcilerAction = mock.NewMockRemoteReconcilerAction[*elasticsearchapicrd.SnapshotLifecyclePolicy, *eshandler.SnapshotLifecyclePolicySpec, eshandler.ElasticsearchHandler](
		slmReconciler.(*SnapshotLifecyclePolicyReconciler).reconcilerAction,
		func(ctx context.Context, req reconcile.Request, o object.RemoteObject) (handler controller.RemoteExternalReconciler[*elasticsearchapicrd.SnapshotLifecyclePolicy, *eshandler.SnapshotLifecyclePolicySpec, eshandler.ElasticsearchHandler], res reconcile.Result, err error) {
			return newSnapshotLifecyclePolicyApiClient(t.mockElasticsearchHandler), res, nil
		},
	)
	if err = slmReconciler.SetupWithManager(k8sManager); err != nil {
		panic(err)
	}

	snapshotRepositoryReconciler := NewSnapshotRepositoryReconciler(
		k8sClient,
		logrus.NewEntry(logrus.StandardLogger()),
		k8sManager.GetEventRecorderFor("elasticsearch-snapshotrepository-controller"),
	)
	snapshotRepositoryReconciler.(*SnapshotRepositoryReconciler).reconcilerAction = mock.NewMockRemoteReconcilerAction[*elasticsearchapicrd.SnapshotRepository, *olivere.SnapshotRepositoryMetaData, eshandler.ElasticsearchHandler](
		snapshotRepositoryReconciler.(*SnapshotRepositoryReconciler).reconcilerAction,
		func(ctx context.Context, req reconcile.Request, o object.RemoteObject) (handler controller.RemoteExternalReconciler[*elasticsearchapicrd.SnapshotRepository, *olivere.SnapshotRepositoryMetaData, eshandler.ElasticsearchHandler], res reconcile.Result, err error) {
			return newSnapshotRepositoryApiClient(t.mockElasticsearchHandler), res, nil
		},
	)
	if err = snapshotRepositoryReconciler.SetupWithManager(k8sManager); err != nil {
		panic(err)
	}

	componentTemplateReconciler := NewComponentTemplateReconciler(
		k8sClient,
		logrus.NewEntry(logrus.StandardLogger()),
		k8sManager.GetEventRecorderFor("elasticsearch-componenttemplate-controller"),
	)
	componentTemplateReconciler.(*ComponentTemplateReconciler).reconcilerAction = mock.NewMockRemoteReconcilerAction[*elasticsearchapicrd.ComponentTemplate, *olivere.IndicesGetComponentTemplate, eshandler.ElasticsearchHandler](
		componentTemplateReconciler.(*ComponentTemplateReconciler).reconcilerAction,
		func(ctx context.Context, req reconcile.Request, o object.RemoteObject) (handler controller.RemoteExternalReconciler[*elasticsearchapicrd.ComponentTemplate, *olivere.IndicesGetComponentTemplate, eshandler.ElasticsearchHandler], res reconcile.Result, err error) {
			return newComponentTemplateApiClient(t.mockElasticsearchHandler), res, nil
		},
	)
	if err = componentTemplateReconciler.SetupWithManager(k8sManager); err != nil {
		panic(err)
	}

	indexTemplateReconciler := NewIndexTemplateReconciler(
		k8sClient,
		logrus.NewEntry(logrus.StandardLogger()),
		k8sManager.GetEventRecorderFor("elasticsearch-indextemplate-controller"),
	)
	indexTemplateReconciler.(*IndexTemplateReconciler).reconcilerAction = mock.NewMockRemoteReconcilerAction[*elasticsearchapicrd.IndexTemplate, *olivere.IndicesGetIndexTemplate, eshandler.ElasticsearchHandler](
		indexTemplateReconciler.(*IndexTemplateReconciler).reconcilerAction,
		func(ctx context.Context, req reconcile.Request, o object.RemoteObject) (handler controller.RemoteExternalReconciler[*elasticsearchapicrd.IndexTemplate, *olivere.IndicesGetIndexTemplate, eshandler.ElasticsearchHandler], res reconcile.Result, err error) {
			return newIndexTemplateApiClient(t.mockElasticsearchHandler), res, nil
		},
	)
	if err = indexTemplateReconciler.SetupWithManager(k8sManager); err != nil {
		panic(err)
	}

	watchReconciler := NewWatchReconciler(
		k8sClient,
		logrus.NewEntry(logrus.StandardLogger()),
		k8sManager.GetEventRecorderFor("elasticsearch-watch-controller"),
	)
	watchReconciler.(*WatchReconciler).reconcilerAction = mock.NewMockRemoteReconcilerAction[*elasticsearchapicrd.Watch, *olivere.XPackWatch, eshandler.ElasticsearchHandler](
		watchReconciler.(*WatchReconciler).reconcilerAction,
		func(ctx context.Context, req reconcile.Request, o object.RemoteObject) (handler controller.RemoteExternalReconciler[*elasticsearchapicrd.Watch, *olivere.XPackWatch, eshandler.ElasticsearchHandler], res reconcile.Result, err error) {
			return newWatchApiClient(t.mockElasticsearchHandler), res, nil
		},
	)
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
