package elasticsearch

import (
	"fmt"
	"strings"

	beatcrd "github.com/webcenter-fr/elasticsearch-operator/apis/beat/v1"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildMetricbeat permit to generate metricbeat
func BuildMetricbeat(es *elasticsearchcrd.Elasticsearch) (mb *beatcrd.Metricbeat, err error) {

	if !es.IsMetricbeatMonitoring() {
		return nil, nil
	}

	var sb strings.Builder

	if es.IsTlsApiEnabled() {
		sb.WriteString(`- module: elasticsearch
  xpack.enabled: true
  username: '${SOURCE_METRICBEAT_USERNAME}'
  password: '${SOURCE_METRICBEAT_PASSWORD}'
  ssl:
    enable: true
    certificate_authorities: '/usr/share/metricbeat/source-es-ca/ca.crt'
    verification_mode: full
  scope: cluster
`)

		if es.Spec.Monitoring.Metricbeat.RefreshPeriod == "" {
			sb.WriteString("  period: 10s\n")
		} else {
			sb.WriteString(fmt.Sprintf("  period: %s\n", es.Spec.Monitoring.Metricbeat.RefreshPeriod))
		}

		sb.WriteString(fmt.Sprintf("  hosts: https://%s.%s.svc:9200\n", GetGlobalServiceName(es), es.Namespace))
	} else {
		sb.WriteString(`- module: elasticsearch
  xpack.enabled: true
  username: '${SOURCE_METRICBEAT_USERNAME}'
  password: '${SOURCE_METRICBEAT_PASSWORD}'
  ssl:
    enable: false
  scope: cluster
`)

		if es.Spec.Monitoring.Metricbeat.RefreshPeriod == "" {
			sb.WriteString("  period: 10s\n")
		} else {
			sb.WriteString(fmt.Sprintf("  period: %s\n", es.Spec.Monitoring.Metricbeat.RefreshPeriod))
		}

		sb.WriteString(fmt.Sprintf("  hosts: http://%s.%s.svc:9200\n", GetGlobalServiceName(es), es.Namespace))
	}

	version := es.Spec.Version
	if es.Spec.Monitoring.Metricbeat.Version != "" {
		version = es.Spec.Monitoring.Metricbeat.Version
	}

	mb = &beatcrd.Metricbeat{
		ObjectMeta: metav1.ObjectMeta{
			Name:        GetMetricbeatName(es),
			Namespace:   es.Namespace,
			Labels:      es.Labels,      // not use getLabels() to avoid collision
			Annotations: es.Annotations, // not use getAnnotations() to avoid collision
		},
		Spec: beatcrd.MetricbeatSpec{
			Version:          version,
			ElasticsearchRef: es.Spec.Monitoring.Metricbeat.ElasticsearchRef,
			Module: map[string]string{
				"elasticsearch-xpack.yml": sb.String(),
			},
			Deployment: beatcrd.MetricbeatDeploymentSpec{
				Replicas: 1,
				Env: []corev1.EnvVar{
					{
						Name:  "SOURCE_METRICBEAT_USERNAME",
						Value: "remote_monitoring_user",
					},
					{
						Name: "SOURCE_METRICBEAT_PASSWORD",
						ValueFrom: &corev1.EnvVarSource{
							SecretKeyRef: &corev1.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: GetSecretNameForCredentials(es),
								},
								Key: "remote_monitoring_user",
							},
						},
					},
				},
			},
		},
	}

	// Compute resource
	if es.Spec.Monitoring.Metricbeat.Resources != nil {
		mb.Spec.Deployment.Resources = es.Spec.Monitoring.Metricbeat.Resources
	} else {
		mb.Spec.Deployment.Resources = &corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("100m"),
				corev1.ResourceMemory: resource.MustParse("100Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("300m"),
				corev1.ResourceMemory: resource.MustParse("200Mi"),
			},
		}
	}

	// Compute volumes
	if es.IsTlsApiEnabled() {
		mb.Spec.Deployment.AdditionalVolumes = []beatcrd.MetricbeatVolumeSpec{
			{
				Name: "ca-source-elasticsearch",
				VolumeMount: corev1.VolumeMount{
					MountPath: "/usr/share/metricbeat/source-es-ca",
				},
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: GetSecretNameForTlsApi(es),
						Items: []corev1.KeyToPath{
							{
								Key:  "ca.crt",
								Path: "ca.crt",
							},
						},
					},
				},
			},
		}
	}

	return mb, nil
}
