package elasticsearch

import (
	"fmt"

	"github.com/disaster37/k8sbuilder"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
)

// BuildDeploymentExporter permit to generate deployment for exporter
func BuildDeploymentExporter(es *elasticsearchcrd.Elasticsearch) (dpl *appv1.Deployment, err error) {

	if !es.IsPrometheusMonitoring() {
		return nil, nil
	}

	cb := k8sbuilder.NewContainerBuilder()
	ptb := k8sbuilder.NewPodTemplateBuilder()
	exporterContainer := &corev1.Container{}

	// Initialise Kibana container from user provided
	cb.WithContainer(exporterContainer).
		Container().Name = "exporter"

	// Command and arguments
	scheme := "https"
	if !es.IsTlsApiEnabled() {
		scheme = "http"
	}

	cb.Container().Args = []string{
		fmt.Sprintf("--es.uri=%s://%s.%s.svc:9200", scheme, GetGlobalServiceName(es), es.Namespace),
		"--es.ssl-skip-verify",
		"--es.all",
		"--collector.clustersettings",
		"--collector.cluster-info",
		"--es.indices",
		"--es.indices_settings",
		"--es.indices_mappings",
		"--es.aliases",
		"--es.ilm",
		"--es.shards",
		"--es.snapshots",
		"--es.slm",
		"--es.data_stream",
	}

	// Compute Env
	cb.WithEnv([]corev1.EnvVar{
		{
			Name:  "ES_USERNAME",
			Value: "elastic",
		},
		{
			Name: "ES_PASSWORD",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: GetSecretNameForCredentials(es),
					},
					Key: "elastic",
				},
			},
		},
	})

	// Compute ports
	cb.WithPort([]corev1.ContainerPort{
		{
			Name:          "exporter",
			ContainerPort: 9114,
			Protocol:      corev1.ProtocolTCP,
		},
	})

	// Compute resources
	if es.Spec.Monitoring.Prometheus.Resources != nil {
		cb.WithResource(es.Spec.Monitoring.Prometheus.Resources)
	} else {
		cb.WithResource(&corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("25m"),
				corev1.ResourceMemory: resource.MustParse("64Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("100m"),
				corev1.ResourceMemory: resource.MustParse("512Mi"),
			},
		})
	}

	// Compute image
	cb.WithImage(GetExporterImage(es))

	// Compute image pull policy
	cb.WithImagePullPolicy(es.Spec.Monitoring.Prometheus.ImagePullPolicy)

	// Compute security context
	cb.WithSecurityContext(&corev1.SecurityContext{
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{
				"ALL",
			},
		},
		RunAsNonRoot: pointer.Bool(true),
		RunAsUser:    pointer.Int64(1000),
	})

	// Compute liveness
	cb.WithLivenessProbe(&corev1.Probe{
		TimeoutSeconds:   10,
		PeriodSeconds:    30,
		FailureThreshold: 3,
		SuccessThreshold: 1,
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: "/healthz",
				Port: intstr.FromInt(9114),
			},
		},
	})

	// Compute readiness
	cb.WithReadinessProbe(&corev1.Probe{
		TimeoutSeconds:   10,
		PeriodSeconds:    30,
		FailureThreshold: 3,
		SuccessThreshold: 1,
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: "/healthz",
				Port: intstr.FromInt(9114),
			},
		},
	})

	// Compute startup
	cb.WithStartupProbe(&corev1.Probe{
		InitialDelaySeconds: 10,
		TimeoutSeconds:      5,
		PeriodSeconds:       10,
		FailureThreshold:    30,
		SuccessThreshold:    1,
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: "/healthz",
				Port: intstr.FromInt(9114),
			},
		},
	})

	// Compute labels
	ptb.WithLabels(map[string]string{
		"exporter":      "true",
		"elasticsearch": "true",
	}, k8sbuilder.Merge)

	// Compute containers
	ptb.WithContainers([]corev1.Container{*cb.Container()}, k8sbuilder.Merge)

	// Pod template name
	ptb.PodTemplate().Name = GetExporterDeployementName(es)

	// Image pull secret
	ptb.PodTemplate().Spec.ImagePullSecrets = es.Spec.Monitoring.Prometheus.ImagePullSecrets

	// Compute Deployment
	dpl = &appv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   es.Namespace,
			Name:        GetExporterDeployementName(es),
			Labels:      getLabels(es),
			Annotations: getAnnotations(es),
		},
		Spec: appv1.DeploymentSpec{
			Replicas: pointer.Int32(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"exporter":      "true",
					"elasticsearch": "true",
				},
			},
			Template: *ptb.PodTemplate(),
		},
	}

	return dpl, nil
}
