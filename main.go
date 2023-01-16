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
	"flag"
	"os"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/sirupsen/logrus"

	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1alpha1"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearchapi/v1alpha1"
	elasticsearchapiv1alpha1 "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearchapi/v1alpha1"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/apis/kibana/v1alpha1"
	elasticsearchcontrollers "github.com/webcenter-fr/elasticsearch-operator/controllers/elasticsearch"
	elasticsearchapicontrollers "github.com/webcenter-fr/elasticsearch-operator/controllers/elasticsearchapi"
	kibanacontrollers "github.com/webcenter-fr/elasticsearch-operator/controllers/kibana"
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

	utilruntime.Must(elasticsearchcrd.AddToScheme(scheme))
	utilruntime.Must(kibanacrd.AddToScheme(scheme))
	utilruntime.Must(elasticsearchapicrd.AddToScheme(scheme))
	utilruntime.Must(elasticsearchapiv1alpha1.AddToScheme(scheme))
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

	// Add indexers on License to track secret change
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &elasticsearchapicrd.License{}, "spec.secretRef.name", func(o client.Object) []string {
		p := o.(*elasticsearchapicrd.License)
		if p.Spec.SecretRef != nil {
			return []string{p.Spec.SecretRef.Name}
		}
		return []string{}
	}); err != nil {
		panic(err)
	}

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &elasticsearchapicrd.User{}, "spec.secretRef.name", func(o client.Object) []string {
		p := o.(*elasticsearchapicrd.User)
		if p.Spec.SecretRef != nil {
			return []string{p.Spec.SecretRef.Name}
		}
		return []string{}
	}); err != nil {
		panic(err)
	}

	// Add indexers on Elasticsearch to track secret change
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &elasticsearchcrd.Elasticsearch{}, "spec.tls.certificateSecretRef.name", func(o client.Object) []string {
		p := o.(*elasticsearchcrd.Elasticsearch)
		if p.IsTlsApiEnabled() && !p.IsSelfManagedSecretForTlsApi() {
			return []string{p.Spec.Tls.CertificateSecretRef.Name}
		}
		return []string{}
	}); err != nil {
		panic(err)
	}

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &elasticsearchcrd.Elasticsearch{}, "spec.globalNodeGroup.keystoreSecretRef.name", func(o client.Object) []string {
		p := o.(*elasticsearchcrd.Elasticsearch)
		if p.Spec.GlobalNodeGroup.KeystoreSecretRef != nil {
			return []string{p.Spec.GlobalNodeGroup.KeystoreSecretRef.Name}
		}
		return []string{}
	}); err != nil {
		panic(err)
	}

	// Add indexers on Kibana to track secret change
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &kibanacrd.Kibana{}, "spec.tls.certificateSecretRef.name", func(o client.Object) []string {
		p := o.(*kibanacrd.Kibana)
		if p.IsTlsEnabled() && !p.IsSelfManagedSecretForTls() {
			return []string{p.Spec.Tls.CertificateSecretRef.Name}
		}
		return []string{}
	}); err != nil {
		panic(err)
	}

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &kibanacrd.Kibana{}, "spec.keystoreSecretRef.name", func(o client.Object) []string {
		p := o.(*kibanacrd.Kibana)
		if p.Spec.KeystoreSecretRef != nil {
			return []string{p.Spec.KeystoreSecretRef.Name}
		}
		return []string{}
	}); err != nil {
		panic(err)
	}

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &kibanacrd.Kibana{}, "spec.elasticsearchRef.name", func(o client.Object) []string {
		p := o.(*kibanacrd.Kibana)
		if p.Spec.KeystoreSecretRef != nil {
			return []string{p.Spec.KeystoreSecretRef.Name}
		}
		return []string{}
	}); err != nil {
		panic(err)
	}

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

	elasticsearchLicenseController := elasticsearchapicontrollers.NewUserReconciler(mgr.GetClient(), mgr.GetScheme())
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
