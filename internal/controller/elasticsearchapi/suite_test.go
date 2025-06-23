package elasticsearchapi

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"path/filepath"
	"testing"
	"time"

	eshandler "github.com/disaster37/es-handler/v8"
	"github.com/disaster37/es-handler/v8/mocks"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/controller/remote"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/mock"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/test"
	olivere "github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/api/beat/v1"
	cerebrocrd "github.com/webcenter-fr/elasticsearch-operator/api/cerebro/v1"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearch/v1"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearchapi/v1"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/api/kibana/v1"
	kibanaapicrd "github.com/webcenter-fr/elasticsearch-operator/api/kibanaapi/v1"
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/api/logstash/v1"
	"go.uber.org/mock/gomock"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
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
			filepath.Join("../../..", "config", "crd", "bases"),
			filepath.Join("../../..", "config", "crd", "externals"),
		},
		ErrorIfCRDPathMissing:    true,
		ControlPlaneStopTimeout:  120 * time.Second,
		ControlPlaneStartTimeout: 120 * time.Second,
		WebhookInstallOptions: envtest.WebhookInstallOptions{
			Paths: []string{filepath.Join("..", "..", "..", "config", "webhook")},
		},
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
	webhookInstallOptions := &testEnv.WebhookInstallOptions
	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
		WebhookServer: webhook.NewServer(webhook.Options{
			Host:    webhookInstallOptions.LocalServingHost,
			Port:    webhookInstallOptions.LocalServingPort,
			CertDir: webhookInstallOptions.LocalServingCertDir,
		}),
		LeaderElection: false,
		Metrics:        metricsserver.Options{BindAddress: "0"},
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
		elasticsearchapicrd.SetupComponentTemplateIndexer,
		elasticsearchapicrd.SetupIndexLifecyclePolicyIndexer,
		elasticsearchapicrd.SetupIndexTemplateIndexer,
		elasticsearchapicrd.SetupLicenceIndexer,
		elasticsearchapicrd.SetupRoleIndexer,
		elasticsearchapicrd.SetupRoleMappingIndexer,
		elasticsearchapicrd.SetupSnapshotLifecyclePolicyIndexer,
		elasticsearchapicrd.SetupSnapshotRepositoryIndexer,
		elasticsearchapicrd.SetupUserIndexexer,
		elasticsearchapicrd.SetupWatchIndexer,
		kibanaapicrd.SetupLogstashPipelineIndexer,
		kibanaapicrd.SetupRoleIndexer,
		kibanaapicrd.SetupUserSpaceIndexer,
	); err != nil {
		panic(err)
	}

	// Add webhooks
	if err := controller.SetupWebhookWithManager(
		k8sManager,
		k8sClient,
		beatcrd.SetupFilebeatWebhookWithManager(logrus.NewEntry(logrus.StandardLogger())),
		beatcrd.SetupMetricbeatWebhookWithManager(logrus.NewEntry(logrus.StandardLogger())),
		cerebrocrd.SetupHostWebhookWithManager(logrus.NewEntry(logrus.StandardLogger())),
		kibanacrd.SetupKibanaWebhookWithManager(logrus.NewEntry(logrus.StandardLogger())),
		logstashcrd.SetupLogstashWebhookWithManager(logrus.NewEntry(logrus.StandardLogger())),
		elasticsearchapicrd.SetupComponentTemplateWebhookWithManager(logrus.NewEntry(logrus.StandardLogger())),
		elasticsearchapicrd.SetupIndexLifecyclePolicyWebhookWithManager(logrus.NewEntry(logrus.StandardLogger())),
		elasticsearchapicrd.SetupIndexTemplateWebhookWithManager(logrus.NewEntry(logrus.StandardLogger())),
		elasticsearchapicrd.SetupLicenseWebhookWithManager(logrus.NewEntry(logrus.StandardLogger())),
		elasticsearchapicrd.SetupRoleWebhookWithManager(logrus.NewEntry(logrus.StandardLogger())),
		elasticsearchapicrd.SetupRoleMappingWebhookWithManager(logrus.NewEntry(logrus.StandardLogger())),
		elasticsearchapicrd.SetupSnapshotLifecyclePolicyWebhookWithManager(logrus.NewEntry(logrus.StandardLogger())),
		elasticsearchapicrd.SetupSnapshotRepositoryWebhookWithManager(logrus.NewEntry(logrus.StandardLogger())),
		elasticsearchapicrd.SetupUserWebhookWithManager(logrus.NewEntry(logrus.StandardLogger())),
		elasticsearchapicrd.SetupWatchWebhookWithManager(logrus.NewEntry(logrus.StandardLogger())),
		kibanaapicrd.SetupLogstashPipelineWebhookWithManager(logrus.NewEntry(logrus.StandardLogger())),
		kibanaapicrd.SetupRoleWebhookWithManager(logrus.NewEntry(logrus.StandardLogger())),
		kibanaapicrd.SetupUserSpaceWebhookWithManager(logrus.NewEntry(logrus.StandardLogger())),
	); err != nil {
		panic(err)
	}

	// Init controllers

	userReconciler := NewUserReconciler(
		k8sClient,
		logrus.NewEntry(logrus.StandardLogger()),
		k8sManager.GetEventRecorderFor("elasticsearch-user-controller"),
	)
	userReconciler.(*UserReconciler).RemoteReconcilerAction = mock.NewMockRemoteReconcilerAction[*elasticsearchapicrd.User, *olivere.XPackSecurityPutUserRequest, eshandler.ElasticsearchHandler](
		userReconciler.(*UserReconciler).RemoteReconcilerAction,
		func(ctx context.Context, req reconcile.Request, o *elasticsearchapicrd.User, logger *logrus.Entry) (handler remote.RemoteExternalReconciler[*elasticsearchapicrd.User, *olivere.XPackSecurityPutUserRequest, eshandler.ElasticsearchHandler], res reconcile.Result, err error) {
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
	licenseReconciler.(*LicenseReconciler).RemoteReconcilerAction = mock.NewMockRemoteReconcilerAction[*elasticsearchapicrd.License, *olivere.XPackInfoLicense, eshandler.ElasticsearchHandler](
		licenseReconciler.(*LicenseReconciler).RemoteReconcilerAction,
		func(ctx context.Context, req reconcile.Request, o *elasticsearchapicrd.License, logger *logrus.Entry) (handler remote.RemoteExternalReconciler[*elasticsearchapicrd.License, *olivere.XPackInfoLicense, eshandler.ElasticsearchHandler], res reconcile.Result, err error) {
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
	roleReconciler.(*RoleReconciler).RemoteReconcilerAction = mock.NewMockRemoteReconcilerAction[*elasticsearchapicrd.Role, *eshandler.XPackSecurityRole, eshandler.ElasticsearchHandler](
		roleReconciler.(*RoleReconciler).RemoteReconcilerAction,
		func(ctx context.Context, req reconcile.Request, o *elasticsearchapicrd.Role, logger *logrus.Entry) (handler remote.RemoteExternalReconciler[*elasticsearchapicrd.Role, *eshandler.XPackSecurityRole, eshandler.ElasticsearchHandler], res reconcile.Result, err error) {
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
	roleMappingReconciler.(*RoleMappingReconciler).RemoteReconcilerAction = mock.NewMockRemoteReconcilerAction[*elasticsearchapicrd.RoleMapping, *olivere.XPackSecurityRoleMapping, eshandler.ElasticsearchHandler](
		roleMappingReconciler.(*RoleMappingReconciler).RemoteReconcilerAction,
		func(ctx context.Context, req reconcile.Request, o *elasticsearchapicrd.RoleMapping, logger *logrus.Entry) (handler remote.RemoteExternalReconciler[*elasticsearchapicrd.RoleMapping, *olivere.XPackSecurityRoleMapping, eshandler.ElasticsearchHandler], res reconcile.Result, err error) {
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
	ilmReconciler.(*IndexLifecyclePolicyReconciler).RemoteReconcilerAction = mock.NewMockRemoteReconcilerAction[*elasticsearchapicrd.IndexLifecyclePolicy, *olivere.XPackIlmGetLifecycleResponse, eshandler.ElasticsearchHandler](
		ilmReconciler.(*IndexLifecyclePolicyReconciler).RemoteReconcilerAction,
		func(ctx context.Context, req reconcile.Request, o *elasticsearchapicrd.IndexLifecyclePolicy, logger *logrus.Entry) (handler remote.RemoteExternalReconciler[*elasticsearchapicrd.IndexLifecyclePolicy, *olivere.XPackIlmGetLifecycleResponse, eshandler.ElasticsearchHandler], res reconcile.Result, err error) {
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
	slmReconciler.(*SnapshotLifecyclePolicyReconciler).RemoteReconcilerAction = mock.NewMockRemoteReconcilerAction[*elasticsearchapicrd.SnapshotLifecyclePolicy, *eshandler.SnapshotLifecyclePolicySpec, eshandler.ElasticsearchHandler](
		slmReconciler.(*SnapshotLifecyclePolicyReconciler).RemoteReconcilerAction,
		func(ctx context.Context, req reconcile.Request, o *elasticsearchapicrd.SnapshotLifecyclePolicy, logger *logrus.Entry) (handler remote.RemoteExternalReconciler[*elasticsearchapicrd.SnapshotLifecyclePolicy, *eshandler.SnapshotLifecyclePolicySpec, eshandler.ElasticsearchHandler], res reconcile.Result, err error) {
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
	snapshotRepositoryReconciler.(*SnapshotRepositoryReconciler).RemoteReconcilerAction = mock.NewMockRemoteReconcilerAction[*elasticsearchapicrd.SnapshotRepository, *olivere.SnapshotRepositoryMetaData, eshandler.ElasticsearchHandler](
		snapshotRepositoryReconciler.(*SnapshotRepositoryReconciler).RemoteReconcilerAction,
		func(ctx context.Context, req reconcile.Request, o *elasticsearchapicrd.SnapshotRepository, logger *logrus.Entry) (handler remote.RemoteExternalReconciler[*elasticsearchapicrd.SnapshotRepository, *olivere.SnapshotRepositoryMetaData, eshandler.ElasticsearchHandler], res reconcile.Result, err error) {
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
	componentTemplateReconciler.(*ComponentTemplateReconciler).RemoteReconcilerAction = mock.NewMockRemoteReconcilerAction[*elasticsearchapicrd.ComponentTemplate, *olivere.IndicesGetComponentTemplate, eshandler.ElasticsearchHandler](
		componentTemplateReconciler.(*ComponentTemplateReconciler).RemoteReconcilerAction,
		func(ctx context.Context, req reconcile.Request, o *elasticsearchapicrd.ComponentTemplate, logger *logrus.Entry) (handler remote.RemoteExternalReconciler[*elasticsearchapicrd.ComponentTemplate, *olivere.IndicesGetComponentTemplate, eshandler.ElasticsearchHandler], res reconcile.Result, err error) {
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
	indexTemplateReconciler.(*IndexTemplateReconciler).RemoteReconcilerAction = mock.NewMockRemoteReconcilerAction[*elasticsearchapicrd.IndexTemplate, *olivere.IndicesGetIndexTemplate, eshandler.ElasticsearchHandler](
		indexTemplateReconciler.(*IndexTemplateReconciler).RemoteReconcilerAction,
		func(ctx context.Context, req reconcile.Request, o *elasticsearchapicrd.IndexTemplate, logger *logrus.Entry) (handler remote.RemoteExternalReconciler[*elasticsearchapicrd.IndexTemplate, *olivere.IndicesGetIndexTemplate, eshandler.ElasticsearchHandler], res reconcile.Result, err error) {
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
	watchReconciler.(*WatchReconciler).RemoteReconcilerAction = mock.NewMockRemoteReconcilerAction[*elasticsearchapicrd.Watch, *olivere.XPackWatch, eshandler.ElasticsearchHandler](
		watchReconciler.(*WatchReconciler).RemoteReconcilerAction,
		func(ctx context.Context, req reconcile.Request, o *elasticsearchapicrd.Watch, logger *logrus.Entry) (handler remote.RemoteExternalReconciler[*elasticsearchapicrd.Watch, *olivere.XPackWatch, eshandler.ElasticsearchHandler], res reconcile.Result, err error) {
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

	// wait for the webhook server to get ready
	dialer := &net.Dialer{Timeout: time.Second}
	addrPort := fmt.Sprintf("%s:%d", webhookInstallOptions.LocalServingHost, webhookInstallOptions.LocalServingPort)
	isTimeout, err := test.RunWithTimeout(func() error {
		conn, err := tls.DialWithDialer(dialer, "tcp", addrPort, &tls.Config{InsecureSkipVerify: true})
		if err != nil {
			return err
		}
		return conn.Close()
	}, time.Second*30, time.Second*1)
	if err != nil || isTimeout {
		panic("Webhook not ready")
	}
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
