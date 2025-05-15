package filebeat

import (
	"fmt"

	"github.com/disaster37/operator-sdk-extra/v2/pkg/apis"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/api/beat/v1"
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildMetricbeat permit to generate metricbeat
func buildMetricbeats(fb *beatcrd.Filebeat) (metricbeats []*beatcrd.Metricbeat, err error) {
	if !fb.Spec.Monitoring.IsMetricbeatMonitoring(fb.Spec.Deployment.Replicas) {
		return nil, nil
	}

	metricbeats = make([]*beatcrd.Metricbeat, 0, 1)

	metricbeatConfig := map[string]any{
		"setup.template.settings": map[string]any{
			"index.number_of_replicas": fb.Spec.Monitoring.Metricbeat.NumberOfReplica,
		},
	}

	xpackModule := map[string]any{
		"module":        "beat",
		"xpack.enabled": true,
		"metricsets": []string{
			"stats",
			"state",
		},
		"hosts": getFilebeatTargets(fb),
	}

	if fb.Spec.Monitoring.Metricbeat.RefreshPeriod == "" {
		xpackModule["period"] = "10s"
	} else {
		xpackModule["period"] = fb.Spec.Monitoring.Metricbeat.RefreshPeriod
	}

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
			Modules: &apis.MapAny{
				Data: map[string]any{
					"beat-xpack.yml": []map[string]any{xpackModule},
				},
			},
			Config: &apis.MapAny{
				Data: metricbeatConfig,
			},
			Deployment: beatcrd.MetricbeatDeploymentSpec{
				Deployment: shared.Deployment{
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

	metricbeats = append(metricbeats, mb)

	return metricbeats, nil
}

func getFilebeatTargets(fb *beatcrd.Filebeat) (podNames []string) {
	podNames = make([]string, 0, fb.Spec.Deployment.Replicas)

	for i := 0; i < int(fb.Spec.Deployment.Replicas); i++ {
		podNames = append(podNames, fmt.Sprintf("http://%s-%d.%s.%s.svc:5066", GetStatefulsetName(fb), i, GetGlobalServiceName(fb), fb.Namespace))
	}

	return podNames
}
