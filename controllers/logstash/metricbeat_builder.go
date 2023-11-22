package logstash

import (
	"fmt"
	"strings"

	beatcrd "github.com/webcenter-fr/elasticsearch-operator/apis/beat/v1"
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/apis/logstash/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildMetricbeat permit to generate metricbeat
func buildMetricbeats(ls *logstashcrd.Logstash) (metricbeats []beatcrd.Metricbeat, err error) {

	if !ls.Spec.Monitoring.IsMetricbeatMonitoring(ls.Spec.Deployment.Replicas) {
		return nil, nil
	}

	var sb strings.Builder
	metricbeats = make([]beatcrd.Metricbeat, 0, 1)

	sb.WriteString(`- module: logstash
  xpack.enabled: true
  username: '${SOURCE_METRICBEAT_USERNAME}'
  password: '${SOURCE_METRICBEAT_PASSWORD}'
  metricsets:
    - node
    - node_stats
`)

	if ls.Spec.Monitoring.Metricbeat.RefreshPeriod == "" {
		sb.WriteString("  period: 10s\n")
	} else {
		sb.WriteString(fmt.Sprintf("  period: %s\n", ls.Spec.Monitoring.Metricbeat.RefreshPeriod))
	}

	sb.WriteString(fmt.Sprintf("  hosts: [%s]\n", strings.Join(getLogstashTargets(ls), ", ")))

	version := ls.Spec.Version
	if ls.Spec.Monitoring.Metricbeat.Version != "" {
		version = ls.Spec.Monitoring.Metricbeat.Version
	}

	mb := &beatcrd.Metricbeat{
		ObjectMeta: metav1.ObjectMeta{
			Name:        GetMetricbeatName(ls),
			Namespace:   ls.Namespace,
			Labels:      ls.Labels,      // not use getLabels() to avoid collision
			Annotations: ls.Annotations, // not use getAnnotations() to avoid collision
		},
		Spec: beatcrd.MetricbeatSpec{
			Version:          version,
			ElasticsearchRef: ls.Spec.Monitoring.Metricbeat.ElasticsearchRef,
			Module: map[string]string{
				"logstash-xpack.yml": sb.String(),
			},
			Config: map[string]string{
				"metricbeat.yml": fmt.Sprintf("setup.template.settings:\n  index.number_of_replicas: %d", ls.Spec.Monitoring.Metricbeat.NumberOfReplica),
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
									Name: GetSecretNameForCredentials(ls),
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
	if ls.Spec.Monitoring.Metricbeat.Resources != nil {
		mb.Spec.Deployment.Resources = ls.Spec.Monitoring.Metricbeat.Resources
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

	metricbeats = append(metricbeats, *mb)

	return metricbeats, nil
}

func getLogstashTargets(ls *logstashcrd.Logstash) (podNames []string) {
	podNames = make([]string, 0, ls.Spec.Deployment.Replicas)

	for i := 0; i < int(ls.Spec.Deployment.Replicas); i++ {
		podNames = append(podNames, fmt.Sprintf("http://%s-%d.%s.%s.svc:9600", GetStatefulsetName(ls), i, GetGlobalServiceName(ls), ls.Namespace))
	}

	return podNames
}
