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
	"flag"
	"os"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	beatcrd "github.com/webcenter-fr/elasticsearch-operator/apis/beat/v1alpha1"
	beatv1alpha1 "github.com/webcenter-fr/elasticsearch-operator/apis/beat/v1alpha1"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1alpha1"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearchapi/v1alpha1"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/apis/kibana/v1alpha1"
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/apis/logstash/v1alpha1"
	elasticsearchcontrollers "github.com/webcenter-fr/elasticsearch-operator/controllers/elasticsearch"
	elasticsearchapicontrollers "github.com/webcenter-fr/elasticsearch-operator/controllers/elasticsearchapi"
	filebeatcontrollers "github.com/webcenter-fr/elasticsearch-operator/controllers/filebeat"
	kibanacontrollers "github.com/webcenter-fr/elasticsearch-operator/controllers/kibana"
	logstashcontrollers "github.com/webcenter-fr/elasticsearch-operator/controllers/logstash"
	metricbeatcontrollers "github.com/webcenter-fr/elasticsearch-operator/controllers/metricbeat"
	"github.com/webcenter-fr/elasticsearch-operator/pkg/helper"
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
	utilruntime.Must(beatv1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	os.Setenv("CAPATH", "/dev/null")
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	opts := zap.Options{
		Development: true,
		Level:       getZapLogLevel(),
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))
	log := logrus.New()
	log.SetLevel(getLogrusLogLevel())

	watchNamespace, err := getWatchNamespace()
	var namespace string
	var multiNamespacesCached cache.NewCacheFunc
	if err != nil {
		setupLog.Info("WATCH_NAMESPACES env variable not setted, the manager will watch and manage resources in all namespaces")
	} else {
		setupLog.Info("Manager look only resources on namespaces %s", watchNamespace)
		watchNamespaces := helper.StringToSlice(watchNamespace, ",")
		if len(watchNamespaces) == 1 {
			namespace = watchNamespace
		} else {
			multiNamespacesCached = cache.MultiNamespacedCacheBuilder(watchNamespaces)
		}
	}

	printVersion(ctrl.Log, metricsAddr, probeAddr)
	log.Infof("elasticsearch-operator version: %s - %s", version, commit)

	cfg := ctrl.GetConfigOrDie()
	timeout, err := getKubeClientTimeout()
	if err != nil {
		setupLog.Error(err, "KUBE_CLIENT_TIMEOUT must be a valid duration: %s", err.Error())
		os.Exit(1)
	}
	cfg.Timeout = timeout

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "baa95990.k8s.webcenter.fr",
		Namespace:              namespace,
		NewCache:               multiNamespacesCached,
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

	// Add indexers
	elasticsearchcrd.MustSetUpIndex(mgr)
	kibanacrd.MustSetUpIndex(mgr)
	logstashcrd.MustSetUpIndex(mgr)
	beatcrd.MustSetUpIndexForFilebeat(mgr)
	beatcrd.MustSetUpIndexForMetricbeat(mgr)
	elasticsearchapicontrollers.MustSetUpIndex(mgr)

	// Init controllers
	elasticsearchController := elasticsearchcontrollers.NewElasticsearchReconciler(mgr.GetClient(), mgr.GetScheme())
	elasticsearchController.SetLogger(log.WithFields(logrus.Fields{
		"type": "ElasticsearchController",
	}))
	elasticsearchController.SetRecorder(mgr.GetEventRecorderFor("elasticsearch-controller"))
	elasticsearchController.SetReconciler(elasticsearchController)
	if err = elasticsearchController.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Elasticsearch")
		os.Exit(1)
	}

	kibanaController := kibanacontrollers.NewKibanaReconciler(mgr.GetClient(), mgr.GetScheme())
	kibanaController.SetLogger(log.WithFields(logrus.Fields{
		"type": "KibanaController",
	}))
	kibanaController.SetRecorder(mgr.GetEventRecorderFor("kibana-controller"))
	kibanaController.SetReconciler(kibanaController)
	if err = kibanaController.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Kibana")
		os.Exit(1)
	}

	logstashController := logstashcontrollers.NewLogstashReconciler(mgr.GetClient(), mgr.GetScheme())
	logstashController.SetLogger(log.WithFields(logrus.Fields{
		"type": "LogstashController",
	}))
	logstashController.SetRecorder(mgr.GetEventRecorderFor("logstash-controller"))
	logstashController.SetReconciler(logstashController)
	if err = logstashController.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Logstash")
		os.Exit(1)
	}

	filebeatController := filebeatcontrollers.NewFilebeatReconciler(mgr.GetClient(), mgr.GetScheme())
	filebeatController.SetLogger(log.WithFields(logrus.Fields{
		"type": "FilebeatController",
	}))
	filebeatController.SetRecorder(mgr.GetEventRecorderFor("filebeat-controller"))
	filebeatController.SetReconciler(filebeatController)
	if err = filebeatController.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Filebeat")
		os.Exit(1)
	}

	metricbeatController := metricbeatcontrollers.NewMetricbeatReconciler(mgr.GetClient(), mgr.GetScheme())
	metricbeatController.SetLogger(log.WithFields(logrus.Fields{
		"type": "MetricbeatController",
	}))
	metricbeatController.SetRecorder(mgr.GetEventRecorderFor("metricbeat-controller"))
	metricbeatController.SetReconciler(metricbeatController)
	if err = metricbeatController.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Metricbeat")
		os.Exit(1)
	}

	elasticsearchUserController := elasticsearchapicontrollers.NewUserReconciler(mgr.GetClient(), mgr.GetScheme())
	elasticsearchUserController.SetLogger(log.WithFields(logrus.Fields{
		"type": "ElasticsearchUserController",
	}))
	elasticsearchUserController.SetRecorder(mgr.GetEventRecorderFor("elasticsearch-user-controller"))
	elasticsearchUserController.SetReconciler(elasticsearchUserController)
	if err = elasticsearchUserController.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ElasticsearchUser")
		os.Exit(1)
	}

	elasticsearchLicenseController := elasticsearchapicontrollers.NewLicenseReconciler(mgr.GetClient(), mgr.GetScheme())
	elasticsearchLicenseController.SetLogger(log.WithFields(logrus.Fields{
		"type": "ElasticsearchLicenseController",
	}))
	elasticsearchLicenseController.SetRecorder(mgr.GetEventRecorderFor("elasticsearch-license-controller"))
	elasticsearchLicenseController.SetReconciler(elasticsearchLicenseController)
	if err = elasticsearchLicenseController.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ElasticsearchLicense")
		os.Exit(1)
	}

	elasticsearchRoleController := elasticsearchapicontrollers.NewRoleReconciler(mgr.GetClient(), mgr.GetScheme())
	elasticsearchRoleController.SetLogger(log.WithFields(logrus.Fields{
		"type": "ElasticsearchRoleController",
	}))
	elasticsearchRoleController.SetRecorder(mgr.GetEventRecorderFor("elasticsearch-role-controller"))
	elasticsearchRoleController.SetReconciler(elasticsearchRoleController)
	if err = elasticsearchRoleController.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ElasticsearchRole")
		os.Exit(1)
	}

	elasticsearchRoleMappingController := elasticsearchapicontrollers.NewRoleMappingReconciler(mgr.GetClient(), mgr.GetScheme())
	elasticsearchRoleMappingController.SetLogger(log.WithFields(logrus.Fields{
		"type": "ElasticsearchRoleMappingController",
	}))
	elasticsearchRoleMappingController.SetRecorder(mgr.GetEventRecorderFor("elasticsearch-rolemapping-controller"))
	elasticsearchRoleMappingController.SetReconciler(elasticsearchRoleMappingController)
	if err = elasticsearchRoleMappingController.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ElasticsearchRoleMapping")
		os.Exit(1)
	}

	elasticsearchIlmController := elasticsearchapicontrollers.NewIndexLifecyclePolicyReconciler(mgr.GetClient(), mgr.GetScheme())
	elasticsearchIlmController.SetLogger(log.WithFields(logrus.Fields{
		"type": "ElasticsearchIndexLifecyclePolicyController",
	}))
	elasticsearchIlmController.SetRecorder(mgr.GetEventRecorderFor("elasticsearch-indexlifecyclepolicy-controller"))
	elasticsearchIlmController.SetReconciler(elasticsearchIlmController)
	if err = elasticsearchIlmController.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ElasticsearchIndexLifecyclePolicy")
		os.Exit(1)
	}

	elasticsearchSlmController := elasticsearchapicontrollers.NewSnapshotLifecyclePolicyReconciler(mgr.GetClient(), mgr.GetScheme())
	elasticsearchSlmController.SetLogger(log.WithFields(logrus.Fields{
		"type": "ElasticsearchSnapshotLifecyclePolicyController",
	}))
	elasticsearchSlmController.SetRecorder(mgr.GetEventRecorderFor("elasticsearch-snapshotlifecyclepolicy-controller"))
	elasticsearchSlmController.SetReconciler(elasticsearchSlmController)
	if err = elasticsearchSlmController.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ElasticsearchSnapshotLifecyclePolicy")
		os.Exit(1)
	}

	elasticsearchSnapshotRepositoryController := elasticsearchapicontrollers.NewSnapshotRepositoryReconciler(mgr.GetClient(), mgr.GetScheme())
	elasticsearchSnapshotRepositoryController.SetLogger(log.WithFields(logrus.Fields{
		"type": "ElasticsearchSnapshotRepositoryController",
	}))
	elasticsearchSnapshotRepositoryController.SetRecorder(mgr.GetEventRecorderFor("elasticsearch-snapshotrepository-controller"))
	elasticsearchSnapshotRepositoryController.SetReconciler(elasticsearchSnapshotRepositoryController)
	if err = elasticsearchSnapshotRepositoryController.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ElasticsearchSnapshotRepository")
		os.Exit(1)
	}

	elasticsearchComponentTemplateController := elasticsearchapicontrollers.NewComponentTemplateReconciler(mgr.GetClient(), mgr.GetScheme())
	elasticsearchComponentTemplateController.SetLogger(log.WithFields(logrus.Fields{
		"type": "ElasticsearchComponentTemplateController",
	}))
	elasticsearchComponentTemplateController.SetRecorder(mgr.GetEventRecorderFor("elasticsearch-componenttemplate-controller"))
	elasticsearchComponentTemplateController.SetReconciler(elasticsearchComponentTemplateController)
	if err = elasticsearchComponentTemplateController.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ElasticsearchComponentTemplate")
		os.Exit(1)
	}

	elasticsearchIndexTemplateController := elasticsearchapicontrollers.NewIndexTemplateReconciler(mgr.GetClient(), mgr.GetScheme())
	elasticsearchIndexTemplateController.SetLogger(log.WithFields(logrus.Fields{
		"type": "ElasticsearchIndexTemplateController",
	}))
	elasticsearchIndexTemplateController.SetRecorder(mgr.GetEventRecorderFor("elasticsearch-indextemplate-controller"))
	elasticsearchIndexTemplateController.SetReconciler(elasticsearchIndexTemplateController)
	if err = elasticsearchIndexTemplateController.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ElasticsearchIndexTemplate")
		os.Exit(1)
	}

	elasticsearchWatchController := elasticsearchapicontrollers.NewWatchReconciler(mgr.GetClient(), mgr.GetScheme())
	elasticsearchWatchController.SetLogger(log.WithFields(logrus.Fields{
		"type": "ElasticsearchWatchController",
	}))
	elasticsearchWatchController.SetRecorder(mgr.GetEventRecorderFor("elasticsearch-indextemplate-controller"))
	elasticsearchWatchController.SetReconciler(elasticsearchWatchController)
	if err = elasticsearchWatchController.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ElasticsearchWatch")
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
