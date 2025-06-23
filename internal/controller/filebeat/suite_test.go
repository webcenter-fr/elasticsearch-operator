/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package filebeat

import (
	"crypto/tls"
	"fmt"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/disaster37/operator-sdk-extra/v2/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/helper"
	"github.com/disaster37/operator-sdk-extra/v2/pkg/test"
	routev1 "github.com/openshift/api/route/v1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/api/beat/v1"
	cerebrocrd "github.com/webcenter-fr/elasticsearch-operator/api/cerebro/v1"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearch/v1"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearchapi/v1"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/api/kibana/v1"
	kibanaapicrd "github.com/webcenter-fr/elasticsearch-operator/api/kibanaapi/v1"
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/api/logstash/v1"
	"github.com/webcenter-fr/elasticsearch-operator/internal/controller/common"
	elasticsearchcontrollers "github.com/webcenter-fr/elasticsearch-operator/internal/controller/elasticsearch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	//+kubebuilder:scaffold:imports
)

var testEnv *envtest.Environment

type FilebeatControllerTestSuite struct {
	suite.Suite
	k8sClient client.Client
	cfg       *rest.Config
}

func TestFilebeatControllerSuite(t *testing.T) {
	suite.Run(t, new(FilebeatControllerTestSuite))
}

func (t *FilebeatControllerTestSuite) SetupSuite() {
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
	err = monitoringv1.AddToScheme(scheme.Scheme)
	if err != nil {
		panic(err)
	}
	err = kibanaapicrd.AddToScheme(scheme.Scheme)
	if err != nil {
		panic(err)
	}
	err = routev1.AddToScheme(scheme.Scheme)
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

	clientStd, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		panic(err)
	}
	kubeCapability := common.KubernetesCapability{}
	if helper.HasCRD(clientStd, monitoringv1.SchemeGroupVersion) {
		kubeCapability.HasPrometheus = true
	}
	if helper.HasCRD(clientStd, routev1.SchemeGroupVersion) {
		kubeCapability.HasRoute = true
	}

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
	elasticsearchReconciler := elasticsearchcontrollers.NewElasticsearchReconciler(k8sClient, logrus.NewEntry(logrus.StandardLogger()), k8sManager.GetEventRecorderFor("elasticsearch-controller"), kubeCapability)
	if err = elasticsearchReconciler.SetupWithManager(k8sManager); err != nil {
		panic(err)
	}

	filebeatReconciler := NewFilebeatReconciler(k8sClient, logrus.NewEntry(logrus.StandardLogger()), k8sManager.GetEventRecorderFor("filebeat-controller"), kubeCapability)
	if err = filebeatReconciler.SetupWithManager(k8sManager); err != nil {
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

func (t *FilebeatControllerTestSuite) TearDownSuite() {
	// Teardown the test environment once controller is fnished.
	// Otherwise from Kubernetes 1.21+, teardon timeouts waiting on
	// kube-apiserver to return
	err := testEnv.Stop()
	if err != nil {
		panic(err)
	}
}

func (t *FilebeatControllerTestSuite) BeforeTest(suiteName, testName string) {
}

func (t *FilebeatControllerTestSuite) AfterTest(suiteName, testName string) {
}
