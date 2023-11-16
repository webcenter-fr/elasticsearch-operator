package filebeat

import (
	"fmt"
	"strings"

	beatcrd "github.com/webcenter-fr/elasticsearch-operator/apis/beat/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildMetricbeat permit to generate metricbeat
func buildMetricbeats(fb *beatcrd.Filebeat) (metricbeats []beatcrd.Metricbeat, err error) {

	if !fb.IsMetricbeatMonitoring() {
		return nil, nil
	}

	metricbeats = make([]beatcrd.Metricbeat, 0, 1)

	var sb strings.Builder

	sb.WriteString(`- module: beat
  xpack.enabled: true
  metricsets:
    - stats
    - state
`)

	if fb.Spec.Monitoring.Metricbeat.RefreshPeriod == "" {
		sb.WriteString("  period: 10s\n")
	} else {
		sb.WriteString(fmt.Sprintf("  period: %s\n", fb.Spec.Monitoring.Metricbeat.RefreshPeriod))
	}

	sb.WriteString(fmt.Sprintf("  hosts: [%s]\n", strings.Join(getFilebeatTargets(fb), ", ")))

	version := fb.Spec.Version
	if fb.Spec.Monitoring.Metricbeat.Version != "" {
		version = fb.Spec.Monitoring.Metricbeat.Version
	}

	mb := &beatcrd.Metricbeat{
		ObjectMeta: metav1.ObjectMeta{
			Name:        GetMetricbeatName(fb),
			Namespace:   fb.Namespace,
			Labels:      fb.Labels,      // not use getLabels() to avoid collision
			Annotations: fb.Annotations, // not use getAnnotations() to avoid collision
		},
		Spec: beatcrd.MetricbeatSpec{
			Version:          version,
			ElasticsearchRef: fb.Spec.Monitoring.Metricbeat.ElasticsearchRef,
			Module: map[string]string{
				"beat-xpack.yml": sb.String(),
			},
			Config: map[string]string{
				"metricbeat.yml": fmt.Sprintf("setup.template.settings:\n  index.number_of_replicas: %d", fb.Spec.Monitoring.Metricbeat.NumberOfReplica),
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
									Name: GetSecretNameForCredentials(fb),
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
	if fb.Spec.Monitoring.Metricbeat.Resources != nil {
		mb.Spec.Deployment.Resources = fb.Spec.Monitoring.Metricbeat.Resources
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

func getFilebeatTargets(fb *beatcrd.Filebeat) (podNames []string) {
	podNames = make([]string, 0, fb.Spec.Deployment.Replicas)

	for i := 0; i < int(fb.Spec.Deployment.Replicas); i++ {
		podNames = append(podNames, fmt.Sprintf("http://%s-%d.%s.%s.svc:5066", GetStatefulsetName(fb), i, GetGlobalServiceName(fb), fb.Namespace))
	}

	return podNames
}
