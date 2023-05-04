package kibana

import (
	"fmt"
	"strings"

	beatcrd "github.com/webcenter-fr/elasticsearch-operator/apis/beat/v1"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/apis/kibana/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildMetricbeat permit to generate metricbeat
func BuildMetricbeat(kb *kibanacrd.Kibana) (mb *beatcrd.Metricbeat, err error) {

	if !kb.IsMetricbeatMonitoring() {
		return nil, nil
	}

	var sb strings.Builder

	if kb.IsTlsEnabled() {
		sb.WriteString(`- module: kibana
  xpack.enabled: true
  username: '${SOURCE_METRICBEAT_USERNAME}'
  password: '${SOURCE_METRICBEAT_PASSWORD}'
  ssl:
    enable: true
    certificate_authorities: '/usr/share/metricbeat/source-kb-ca/ca.crt'
    verification_mode: full
  metricsets:
    - stats
`)

		if kb.Spec.Monitoring.Metricbeat.RefreshPeriod == "" {
			sb.WriteString("  period: 10s\n")
		} else {
			sb.WriteString(fmt.Sprintf("  period: %s\n", kb.Spec.Monitoring.Metricbeat.RefreshPeriod))
		}

		sb.WriteString(fmt.Sprintf("  hosts: https://%s.%s.svc:5601\n", GetServiceName(kb), kb.Namespace))
	} else {
		sb.WriteString(`- module: kibana
  xpack.enabled: true
  username: '${SOURCE_METRICBEAT_USERNAME}'
  password: '${SOURCE_METRICBEAT_PASSWORD}'
  ssl:
    enable: false
  metricsets:
    - stats
`)

		if kb.Spec.Monitoring.Metricbeat.RefreshPeriod == "" {
			sb.WriteString("  period: 10s\n")
		} else {
			sb.WriteString(fmt.Sprintf("  period: %s\n", kb.Spec.Monitoring.Metricbeat.RefreshPeriod))
		}

		sb.WriteString(fmt.Sprintf("  hosts: http://%s.%s.svc:5601\n", GetServiceName(kb), kb.Namespace))
	}

	mb = &beatcrd.Metricbeat{
		ObjectMeta: metav1.ObjectMeta{
			Name:        GetMetricbeatName(kb),
			Namespace:   kb.Namespace,
			Labels:      kb.Labels,      // not use getLabels() to avoid collision
			Annotations: kb.Annotations, // not use getAnnotations() to avoid collision
		},
		Spec: beatcrd.MetricbeatSpec{
			Version:          kb.Spec.Version,
			ElasticsearchRef: kb.Spec.Monitoring.Metricbeat.ElasticsearchRef,
			Module: map[string]string{
				"kibana-xpack.yml": sb.String(),
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
									Name: GetSecretNameForCredentials(kb),
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
	if kb.Spec.Monitoring.Metricbeat.Resources != nil {
		mb.Spec.Deployment.Resources = kb.Spec.Monitoring.Metricbeat.Resources
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
	if kb.IsTlsEnabled() {
		mb.Spec.Deployment.AdditionalVolumes = []beatcrd.MetricbeatVolumeSpec{
			{
				Name: "ca-source-kibana",
				VolumeMount: corev1.VolumeMount{
					MountPath: "/usr/share/metricbeat/source-kb-ca",
				},
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: GetSecretNameForTls(kb),
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
