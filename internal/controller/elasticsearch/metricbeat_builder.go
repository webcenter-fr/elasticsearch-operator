package elasticsearch

import (
	"fmt"

	"github.com/disaster37/operator-sdk-extra/pkg/apis"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/api/beat/v1"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearch/v1"
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildMetricbeat permit to generate metricbeat
func buildMetricbeats(es *elasticsearchcrd.Elasticsearch) (mbs []beatcrd.Metricbeat, err error) {
	if !es.Spec.Monitoring.IsMetricbeatMonitoring(es.NumberOfReplicas()) {
		return nil, nil
	}

	mbs = make([]beatcrd.Metricbeat, 0, 1)

	metricbeatConfig := map[string]any{
		"setup.template.settings": map[string]any{
			"index.number_of_replicas": es.Spec.Monitoring.Metricbeat.NumberOfReplica,
		},
	}

	xpackModule := map[string]any{
		"module":        "elasticsearch",
		"xpack.enabled": true,
		"username":      "${SOURCE_METRICBEAT_USERNAME}",
		"password":      "${SOURCE_METRICBEAT_PASSWORD}",
		"scope":         "cluster",
	}

	if es.Spec.Monitoring.Metricbeat.RefreshPeriod == "" {
		xpackModule["period"] = "10s"
	} else {
		xpackModule["period"] = es.Spec.Monitoring.Metricbeat.RefreshPeriod
	}

	if es.Spec.Tls.IsTlsEnabled() {
		xpackModule["ssl"] = map[string]any{
			"enable":                  true,
			"certificate_authorities": "/usr/share/metricbeat/source-es-ca/ca.crt",
			"verification_mode":       "full",
		}
		xpackModule["hosts"] = fmt.Sprintf("https://%s.%s.svc:9200", GetGlobalServiceName(es), es.Namespace)

	} else {
		xpackModule["ssl"] = map[string]any{
			"enable": false,
		}
		xpackModule["hosts"] = fmt.Sprintf("http://%s.%s.svc:9200", GetGlobalServiceName(es), es.Namespace)
	}

	version := es.Spec.Version
	if es.Spec.Monitoring.Metricbeat.Version != "" {
		version = es.Spec.Monitoring.Metricbeat.Version
	}

	mb := &beatcrd.Metricbeat{
		ObjectMeta: metav1.ObjectMeta{
			Name:        GetMetricbeatName(es),
			Namespace:   es.Namespace,
			Labels:      es.Labels,      // not use getLabels() to avoid collision
			Annotations: es.Annotations, // not use getAnnotations() to avoid collision
		},
		Spec: beatcrd.MetricbeatSpec{
			Version:          version,
			ElasticsearchRef: es.Spec.Monitoring.Metricbeat.ElasticsearchRef,
			Modules: map[string][]apis.MapAny{
				"elasticsearch-xpack.yml": []apis.MapAny{
					{
						Data: xpackModule,
					},
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
										Name: GetSecretNameForCredentials(es),
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
	if es.Spec.Tls.IsTlsEnabled() {
		mb.Spec.Deployment.AdditionalVolumes = []shared.DeploymentVolumeSpec{
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

	mbs = append(mbs, *mb)

	return mbs, nil
}
