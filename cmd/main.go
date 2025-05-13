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

package main

import (
	"context"
	"crypto/tls"
	"flag"
	"os"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/pkg/helper"
	routev1 "github.com/openshift/api/route/v1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/sirupsen/logrus"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/api/beat/v1"
	cerebrocrd "github.com/webcenter-fr/elasticsearch-operator/api/cerebro/v1"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearch/v1"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearchapi/v1"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/api/kibana/v1"
	kibanaapicrd "github.com/webcenter-fr/elasticsearch-operator/api/kibanaapi/v1"
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/api/logstash/v1"
	cerebrocontrollers "github.com/webcenter-fr/elasticsearch-operator/internal/controller/cerebro"
	"github.com/webcenter-fr/elasticsearch-operator/internal/controller/common"
	elasticsearchcontrollers "github.com/webcenter-fr/elasticsearch-operator/internal/controller/elasticsearch"
	elasticsearchapicontrollers "github.com/webcenter-fr/elasticsearch-operator/internal/controller/elasticsearchapi"
	filebeatcontrollers "github.com/webcenter-fr/elasticsearch-operator/internal/controller/filebeat"
	kibanacontrollers "github.com/webcenter-fr/elasticsearch-operator/internal/controller/kibana"
	kibanaapicontrollers "github.com/webcenter-fr/elasticsearch-operator/internal/controller/kibanaapi"
	logstashcontrollers "github.com/webcenter-fr/elasticsearch-operator/internal/controller/logstash"
	metricbeatcontrollers "github.com/webcenter-fr/elasticsearch-operator/internal/controller/metricbeat"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/filters"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
	version  = "develop"
	commit   = ""
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(monitoringv1.AddToScheme(scheme))

	utilruntime.Must(elasticsearchcrd.AddToScheme(scheme))
	utilruntime.Must(kibanacrd.AddToScheme(scheme))
	utilruntime.Must(elasticsearchapicrd.AddToScheme(scheme))
	utilruntime.Must(logstashcrd.AddToScheme(scheme))
	utilruntime.Must(beatcrd.AddToScheme(scheme))
	utilruntime.Must(cerebrocrd.AddToScheme(scheme))
	utilruntime.Must(kibanaapicrd.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var secureMetrics bool
	var probeAddr string
	var tlsOpts []func(*tls.Config)
	flag.StringVar(&metricsAddr, "metrics-bind-address", "0", "The address the metrics endpoint binds to. "+
		"Use :8443 for HTTPS or :8080 for HTTP, or leave as 0 to disable the metrics service.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&secureMetrics, "metrics-secure", true,
		"If set, the metrics endpoint is served securely via HTTPS. Use --metrics-secure=false to use HTTP instead.")
	opts := zap.Options{
		Development: true,
		Level:       helper.GetZapLogLevelFromEnv(),
		Encoder:     helper.GetZapFormatterFromDev(),
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))
	log := logrus.New()
	log.SetLevel(helper.GetLogrusLogLevelFromEnv())
	log.SetFormatter(helper.GetLogrusFormatterFromEnv())

	// Log panics error and exit
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("Panic: %v", r)
			os.Exit(1)
		}
	}()

	var cacheNamespaces map[string]cache.Config
	watchNamespace, err := helper.GetWatchNamespaceFromEnv()
	if err != nil {
		setupLog.Info("WATCH_NAMESPACES env variable not setted, the manager will watch and manage resources in all namespaces")
	} else {
		setupLog.Info("Manager look only resources on namespaces %s", watchNamespace)
		watchNamespaces := helper.StringToSlice(watchNamespace, ",")
		cacheNamespaces = make(map[string]cache.Config)
		for _, namespace := range watchNamespaces {
			cacheNamespaces[namespace] = cache.Config{}
		}
	}

	helper.PrintVersion(ctrl.Log, metricsAddr, probeAddr)
	log.Infof("elasticsearch-operator version: %s - %s", version, commit)

	cfg := ctrl.GetConfigOrDie()
	timeout, err := helper.GetKubeClientTimeoutFromEnv()
	if err != nil {
		setupLog.Error(err, "KUBE_CLIENT_TIMEOUT must be a valid duration: %s", err.Error())
		os.Exit(1)
	}
	cfg.Timeout = timeout

	metricsServerOptions := server.Options{
		BindAddress:   metricsAddr,
		SecureServing: secureMetrics,
		// TODO(user): TLSOpts is used to allow configuring the TLS config used for the server. If certificates are
		// not provided, self-signed certificates will be generated by default. This option is not recommended for
		// production environments as self-signed certificates do not offer the same level of trust and security
		// as certificates issued by a trusted Certificate Authority (CA). The primary risk is potentially allowing
		// unauthorized access to sensitive metrics data. Consider replacing with CertDir, CertName, and KeyName
		// to provide certificates, ensuring the server communicates using trusted and secure certificates.
		TLSOpts: tlsOpts,
	}

	if secureMetrics {
		// FilterProvider is used to protect the metrics endpoint with authn/authz.
		// These configurations ensure that only authorized users and service accounts
		// can access the metrics endpoint. The RBAC are configured in 'config/rbac/kustomization.yaml'. More info:
		// https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.18.4/pkg/metrics/filters#WithAuthenticationAndAuthorization
		metricsServerOptions.FilterProvider = filters.WithAuthenticationAndAuthorization
	}
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:  scheme,
		Metrics: metricsServerOptions,
		WebhookServer: webhook.NewServer(webhook.Options{
			Port: 9443,
		}),
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "baa95990.k8s.webcenter.fr",
		Cache: cache.Options{
			DefaultNamespaces: cacheNamespaces,
		},
		// LeaderElectionReleaseOnCancel defines if the leader should step down voluntarily
		// when the Manager ends. This requires the binary to immediately end when the
		// Manager is stopped, otherwise, this setting is unsafe. Setting this significantly
		// speeds up voluntary leader transitions as the new leader don't have to wait
		// LeaseDuration time first.
		//
		// In the default scaffold provided, the program ends immediately after
		// the manager stops, so would be fine to enable this option. However,
		// if you are doing or is intended to do any operation such as perform cleanups
		// after the manager stops then its usage might be unsafe.
		// LeaderElectionReleaseOnCancel: true,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Migrate existing ressources
	clientDinamic, err := dynamic.NewForConfig(cfg)
	if err != nil {
		panic(err)
	}
	clientStd, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		panic(err)
	}
	if err = migrateElasticsearch(context.Background(), clientDinamic, clientStd, log.WithFields(logrus.Fields{"type": "MigrateElasticsearch"})); err != nil {
		setupLog.Error(err, "unable to migrate existing Elasticsearch cluster")
		os.Exit(1)
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
		mgr,
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
	elasticsearchController := elasticsearchcontrollers.NewElasticsearchReconciler(mgr.GetClient(), logrus.NewEntry(log), mgr.GetEventRecorderFor("elasticsearch-controller"), kubeCapability)
	if err = elasticsearchController.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Elasticsearch")
		os.Exit(1)
	}

	kibanaController := kibanacontrollers.NewKibanaReconciler(mgr.GetClient(), logrus.NewEntry(log), mgr.GetEventRecorderFor("kibana-controller"), kubeCapability)
	if err = kibanaController.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Kibana")
		os.Exit(1)
	}

	logstashController := logstashcontrollers.NewLogstashReconciler(mgr.GetClient(), logrus.NewEntry(log), mgr.GetEventRecorderFor("logstash-controller"), kubeCapability)
	if err = logstashController.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Logstash")
		os.Exit(1)
	}

	filebeatController := filebeatcontrollers.NewFilebeatReconciler(mgr.GetClient(), logrus.NewEntry(log), mgr.GetEventRecorderFor("filebeat-controller"), kubeCapability)
	if err = filebeatController.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Filebeat")
		os.Exit(1)
	}

	metricbeatController := metricbeatcontrollers.NewMetricbeatReconciler(mgr.GetClient(), logrus.NewEntry(log), mgr.GetEventRecorderFor("metricbeat-controller"), kubeCapability)
	if err = metricbeatController.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Metricbeat")
		os.Exit(1)
	}

	cerebroController := cerebrocontrollers.NewCerebroReconciler(mgr.GetClient(), logrus.NewEntry(log), mgr.GetEventRecorderFor("cerebro-controller"), kubeCapability)
	if err = cerebroController.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Cerebro")
		os.Exit(1)
	}

	elasticsearchUserController := elasticsearchapicontrollers.NewUserReconciler(mgr.GetClient(), logrus.NewEntry(log), mgr.GetEventRecorderFor("elasticsearch-user-controller"))
	if err = elasticsearchUserController.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ElasticsearchUser")
		os.Exit(1)
	}

	elasticsearchLicenseController := elasticsearchapicontrollers.NewLicenseReconciler(mgr.GetClient(), logrus.NewEntry(log), mgr.GetEventRecorderFor("elasticsearch-license-controller"))
	if err = elasticsearchLicenseController.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ElasticsearchLicense")
		os.Exit(1)
	}

	elasticsearchRoleController := elasticsearchapicontrollers.NewRoleReconciler(mgr.GetClient(), logrus.NewEntry(log), mgr.GetEventRecorderFor("elasticsearch-role-controller"))
	if err = elasticsearchRoleController.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ElasticsearchRole")
		os.Exit(1)
	}

	elasticsearchRoleMappingController := elasticsearchapicontrollers.NewRoleMappingReconciler(mgr.GetClient(), logrus.NewEntry(log), mgr.GetEventRecorderFor("elasticsearch-rolemapping-controller"))
	if err = elasticsearchRoleMappingController.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ElasticsearchRoleMapping")
		os.Exit(1)
	}

	elasticsearchIlmController := elasticsearchapicontrollers.NewIndexLifecyclePolicyReconciler(mgr.GetClient(), logrus.NewEntry(log), mgr.GetEventRecorderFor("elasticsearch-indexlifecyclepolicy-controller"))
	if err = elasticsearchIlmController.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ElasticsearchIndexLifecyclePolicy")
		os.Exit(1)
	}

	elasticsearchSlmController := elasticsearchapicontrollers.NewSnapshotLifecyclePolicyReconciler(mgr.GetClient(), logrus.NewEntry(log), mgr.GetEventRecorderFor("elasticsearch-snapshotlifecyclepolicy-controller"))
	if err = elasticsearchSlmController.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ElasticsearchSnapshotLifecyclePolicy")
		os.Exit(1)
	}

	elasticsearchSnapshotRepositoryController := elasticsearchapicontrollers.NewSnapshotRepositoryReconciler(mgr.GetClient(), logrus.NewEntry(log), mgr.GetEventRecorderFor("elasticsearch-snapshotrepository-controller"))
	if err = elasticsearchSnapshotRepositoryController.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ElasticsearchSnapshotRepository")
		os.Exit(1)
	}

	elasticsearchComponentTemplateController := elasticsearchapicontrollers.NewComponentTemplateReconciler(mgr.GetClient(), logrus.NewEntry(log), mgr.GetEventRecorderFor("elasticsearch-componenttemplate-controller"))
	if err = elasticsearchComponentTemplateController.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ElasticsearchComponentTemplate")
		os.Exit(1)
	}

	elasticsearchIndexTemplateController := elasticsearchapicontrollers.NewIndexTemplateReconciler(mgr.GetClient(), logrus.NewEntry(log), mgr.GetEventRecorderFor("elasticsearch-indextemplate-controller"))
	if err = elasticsearchIndexTemplateController.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ElasticsearchIndexTemplate")
		os.Exit(1)
	}

	elasticsearchWatchController := elasticsearchapicontrollers.NewWatchReconciler(mgr.GetClient(), logrus.NewEntry(log), mgr.GetEventRecorderFor("elasticsearch-indextemplate-controller"))
	if err = elasticsearchWatchController.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ElasticsearchWatch")
		os.Exit(1)
	}

	kibanaUserSpaceController := kibanaapicontrollers.NewUserSpaceReconciler(mgr.GetClient(), logrus.NewEntry(log), mgr.GetEventRecorderFor("kibana-userspace-controller"))
	if err = kibanaUserSpaceController.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "KibanaUserSpace")
		os.Exit(1)
	}

	kibanaRoleController := kibanaapicontrollers.NewRoleReconciler(mgr.GetClient(), logrus.NewEntry(log), mgr.GetEventRecorderFor("kibana-role-controller"))
	if err = kibanaRoleController.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "KibanaRole")
		os.Exit(1)
	}

	kibanaLogstashPipelineController := kibanaapicontrollers.NewLogstashPipelineReconciler(mgr.GetClient(), logrus.NewEntry(log), mgr.GetEventRecorderFor("kibana-logstashpipeline-controller"))
	if err = kibanaLogstashPipelineController.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "KibanaLogstashPipeline")
		os.Exit(1)
	}

	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
