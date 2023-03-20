package metricbeat

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/codingsince1985/checksum"
	"github.com/disaster37/k8sbuilder"
	"github.com/pkg/errors"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/apis/beat/v1alpha1"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1alpha1"
	elasticsearchcontrollers "github.com/webcenter-fr/elasticsearch-operator/controllers/elasticsearch"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/utils/pointer"
)

// GenerateStatefullset permit to generate statefullset
func BuildStatefulset(mb *beatcrd.Metricbeat, es *elasticsearchcrd.Elasticsearch, secretsChecksum []corev1.Secret, configMapsChecksum []corev1.ConfigMap) (statefullset *appv1.StatefulSet, err error) {

	checksumAnnotations := map[string]string{}

	// Generate confimaps to know what file to mount
	configMaps, err := BuildConfigMaps(mb, es)
	if err != nil {
		return nil, errors.Wrap(err, "Error when generate configMaps")
	}

	// checksum for configmap
	for _, cm := range configMapsChecksum {
		j, err := json.Marshal(cm.Data)
		if err != nil {
			return nil, errors.Wrapf(err, "Error when convert data of configMap %s on json string", cm.Name)
		}
		sum, err := checksum.SHA256sumReader(bytes.NewReader(j))
		if err != nil {
			return nil, errors.Wrapf(err, "Error when generate checksum for extra configMap %s", cm.Name)
		}
		checksumAnnotations[fmt.Sprintf("%s/configmap-%s", beatcrd.MetricbeatAnnotationKey, cm.Name)] = sum
	}
	// checksum for secret
	for _, s := range secretsChecksum {
		j, err := json.Marshal(s.Data)
		if err != nil {
			return nil, errors.Wrapf(err, "Error when convert data of secret %s on json string", s.Name)
		}
		sum, err := checksum.SHA256sumReader(bytes.NewReader(j))
		if err != nil {
			return nil, errors.Wrapf(err, "Error when generate checksum for extra secret %s", s.Name)
		}
		checksumAnnotations[fmt.Sprintf("%s/secret-%s", beatcrd.MetricbeatAnnotationKey, s.Name)] = sum
	}

	cb := k8sbuilder.NewContainerBuilder()
	ptb := k8sbuilder.NewPodTemplateBuilder()

	metricbeatContainer := getMetricbeatContainer(mb.Spec.Deployment.PodTemplate)
	if metricbeatContainer == nil {
		metricbeatContainer = &corev1.Container{}
	}

	// Initialize FMetricbeat container from user provided
	cb.WithContainer(metricbeatContainer.DeepCopy()).
		Container().Name = "metricbeat"

	// Compute EnvFrom
	cb.WithEnvFrom(mb.Spec.Deployment.EnvFrom, k8sbuilder.Merge)

	// Compute Env
	cb.WithEnv(mb.Spec.Deployment.Env, k8sbuilder.Merge).
		WithEnv([]corev1.EnvVar{
			{
				Name: "NODE_NAME",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						APIVersion: "v1",
						FieldPath:  "spec.nodeName",
					},
				},
			},
			{
				Name: "NAMESPACE",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						APIVersion: "v1",
						FieldPath:  "metadata.namespace",
					},
				},
			},
			{
				Name: "POD_NAME",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						APIVersion: "v1",
						FieldPath:  "metadata.name",
					},
				},
			},
			{
				Name: "POD_IP",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						APIVersion: "v1",
						FieldPath:  "status.podIP",
					},
				},
			},
		}, k8sbuilder.Merge)

	if mb.Spec.ElasticsearchRef.IsManaged() {
		cb.WithEnv([]corev1.EnvVar{
			{
				Name:  "ELASTICSEARCH_HOST",
				Value: elasticsearchcontrollers.GetPublicUrl(es, mb.Spec.ElasticsearchRef.ManagedElasticsearchRef.TargetNodeGroup, false),
			},
			{
				Name:  "METRICBEAT_USERNAME",
				Value: "remote_monitoring_user",
			},
			{
				Name: "METRICBEAT_PASSWORD",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: GetSecretNameForCredentials(mb),
						},
						Key: "remote_monitoring_user",
					},
				},
			},
		}, k8sbuilder.Merge)
	} else {
		cb.WithEnv([]corev1.EnvVar{
			{
				Name:  "ELASTICSEARCH_HOST",
				Value: mb.Spec.ElasticsearchRef.ExternalElasticsearchRef.Addresses[0],
			},
			{
				Name: "METRICBEAT_USERNAME",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: mb.Spec.ElasticsearchRef.ExternalElasticsearchRef.SecretRef.Name,
						},
						Key: "username",
					},
				},
			},
			{
				Name: "METRICBEAT_PASSWORD",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: mb.Spec.ElasticsearchRef.ExternalElasticsearchRef.SecretRef.Name,
						},
						Key: "password",
					},
				},
			},
		}, k8sbuilder.Merge)
	}

	// Compute ports
	cb.WithPort([]corev1.ContainerPort{
		{
			Name:          "http",
			ContainerPort: 5066,
			Protocol:      corev1.ProtocolTCP,
		},
	}, k8sbuilder.Merge)

	// Compute resources
	cb.WithResource(mb.Spec.Deployment.Resources, k8sbuilder.Merge)

	// Compute image
	cb.WithImage(GetContainerImage(mb), k8sbuilder.OverwriteIfDefaultValue)

	// Compute image pull policy
	cb.WithImagePullPolicy(mb.Spec.ImagePullPolicy, k8sbuilder.OverwriteIfDefaultValue)

	// Compute security context
	cb.WithSecurityContext(&corev1.SecurityContext{
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{
				"ALL",
			},
		},
		RunAsUser:    pointer.Int64(1000),
		RunAsNonRoot: pointer.Bool(true),
	}, k8sbuilder.OverwriteIfDefaultValue)

	// Compute volume mount
	additionalVolumeMounts := make([]corev1.VolumeMount, 0, len(mb.Spec.Deployment.AdditionalVolumes))
	for _, vol := range mb.Spec.Deployment.AdditionalVolumes {
		additionalVolumeMounts = append(additionalVolumeMounts, corev1.VolumeMount{
			Name:      vol.Name,
			MountPath: vol.MountPath,
			ReadOnly:  vol.ReadOnly,
			SubPath:   vol.SubPath,
		})
	}
	cb.WithVolumeMount(additionalVolumeMounts, k8sbuilder.Merge)

	cb.WithVolumeMount([]corev1.VolumeMount{
		{
			Name:      "metricbeat-data",
			MountPath: "/usr/share/metricbeat/data",
		},
	}, k8sbuilder.Merge)

	// Mount configmap of type module
	for _, cm := range configMaps {
		// Mount config in the right path
		switch cm.Name {
		case GetConfigMapModuleName(mb):
			cb.WithVolumeMount([]corev1.VolumeMount{
				{
					Name:      "metricbeat-module",
					MountPath: "/usr/share/metricbeat/modules.d",
				},
			}, k8sbuilder.Merge)
		case GetConfigMapConfigName(mb):
			for file := range cm.Data {
				cb.WithVolumeMount([]corev1.VolumeMount{
					{
						Name:      "metricbeat-config",
						MountPath: fmt.Sprintf("/usr/share/metricbeat/%s", file),
						SubPath:   file,
					},
				}, k8sbuilder.Merge)
			}
		}
	}

	// Compute mount CA elasticsearch
	if (mb.Spec.ElasticsearchRef.IsManaged() && es.IsTlsApiEnabled()) || (mb.Spec.ElasticsearchRef.IsExternal() && mb.Spec.ElasticsearchRef.ElasticsearchCaSecretRef != nil) {
		cb.WithVolumeMount([]corev1.VolumeMount{
			{
				Name:      "ca-elasticsearch",
				MountPath: "/usr/share/metricbeat/es-ca",
			},
		}, k8sbuilder.Merge)
	}

	// Compute liveness
	cb.WithLivenessProbe(&corev1.Probe{
		TimeoutSeconds:   5,
		PeriodSeconds:    30,
		FailureThreshold: 3,
		SuccessThreshold: 1,
		ProbeHandler: corev1.ProbeHandler{
			TCPSocket: &corev1.TCPSocketAction{
				Port: intstr.FromInt(5066),
			},
		},
	}, k8sbuilder.OverwriteIfDefaultValue)

	// Compute readiness
	cb.WithReadinessProbe(&corev1.Probe{
		TimeoutSeconds:   5,
		PeriodSeconds:    10,
		FailureThreshold: 3,
		SuccessThreshold: 1,
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: "/",
				Port: intstr.FromInt(5066),
			},
		},
	}, k8sbuilder.OverwriteIfDefaultValue)

	// Compute startup
	cb.WithStartupProbe(&corev1.Probe{
		InitialDelaySeconds: 10,
		TimeoutSeconds:      5,
		PeriodSeconds:       10,
		FailureThreshold:    30,
		SuccessThreshold:    1,
		ProbeHandler: corev1.ProbeHandler{
			TCPSocket: &corev1.TCPSocketAction{
				Port: intstr.FromInt(5066),
			},
		},
	}, k8sbuilder.OverwriteIfDefaultValue)

	// Initialise PodTemplate
	ptb.WithPodTemplateSpec(mb.Spec.Deployment.PodTemplate)

	// Compute labels
	// Do not set global labels here to avoid reconcile pod just because global label change
	ptb.WithLabels(map[string]string{
		"cluster":                       mb.Name,
		beatcrd.MetricbeatAnnotationKey: "true",
	}).
		WithLabels(mb.Spec.Deployment.Labels, k8sbuilder.Merge)

	// Compute annotations
	// Do not set global annotation here to avoid reconcile pod just because global annotation change
	ptb.WithAnnotations(map[string]string{
		beatcrd.MetricbeatAnnotationKey: "true",
	}).
		WithAnnotations(mb.Spec.Deployment.Annotations, k8sbuilder.Merge).
		WithAnnotations(checksumAnnotations, k8sbuilder.Merge)

	// Compute NodeSelector
	ptb.WithNodeSelector(mb.Spec.Deployment.NodeSelector, k8sbuilder.OverwriteIfDefaultValue)

	// Compute Termination grac period
	ptb.WithTerminationGracePeriodSeconds(60, k8sbuilder.OverwriteIfDefaultValue)

	// Compute toleration
	ptb.WithTolerations(mb.Spec.Deployment.Tolerations, k8sbuilder.OverwriteIfDefaultValue)

	// compute anti affinity
	antiAffinity, err := computeAntiAffinity(mb)
	if err != nil {
		return nil, errors.Wrap(err, "Error when compute anti affinity")
	}
	ptb.WithAffinity(corev1.Affinity{
		PodAntiAffinity: antiAffinity,
	}, k8sbuilder.OverwriteIfDefaultValue)

	// Compute containers
	ptb.WithContainers([]corev1.Container{*cb.Container()}, k8sbuilder.Merge)

	// Compute init containers
	ccb := k8sbuilder.NewContainerBuilder().WithContainer(&corev1.Container{
		Name:            "init-filesystem",
		Image:           GetContainerImage(mb),
		ImagePullPolicy: mb.Spec.ImagePullPolicy,
		SecurityContext: &corev1.SecurityContext{
			RunAsUser: pointer.Int64(0),
		},
		Env: []corev1.EnvVar{
			{
				Name: "NODE_NAME",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						APIVersion: "v1",
						FieldPath:  "spec.nodeName",
					},
				},
			},
			{
				Name: "NAMESPACE",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						APIVersion: "v1",
						FieldPath:  "metadata.namespace",
					},
				},
			},
			{
				Name: "POD_NAME",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						APIVersion: "v1",
						FieldPath:  "metadata.name",
					},
				},
			},
			{
				Name: "POD_IP",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						APIVersion: "v1",
						FieldPath:  "status.podIP",
					},
				},
			},
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "metricbeat-data",
				MountPath: "/mnt/data",
			},
		},
	})

	// Inject env / envFrom to get proxy for exemple
	ccb.WithEnv(mb.Spec.Deployment.Env, k8sbuilder.Merge)
	ccb.WithEnvFrom(mb.Spec.Deployment.EnvFrom, k8sbuilder.Merge)

	var command strings.Builder
	command.WriteString(`#!/usr/bin/env bash
set -euo pipefail

# Set right
echo "Set right"
chown -v metricbeat:metricbeat /mnt/data
`)

	ccb.Container().Command = []string{
		"/bin/bash",
		"-c",
		command.String(),
	}

	ccb.WithResource(mb.Spec.Deployment.InitContainerResources)
	ptb.WithInitContainers([]corev1.Container{*ccb.Container()}, k8sbuilder.Merge)

	// Compute volumes
	additionalVolume := make([]corev1.Volume, 0, len(mb.Spec.Deployment.AdditionalVolumes))
	for _, vol := range mb.Spec.Deployment.AdditionalVolumes {
		additionalVolume = append(additionalVolume, corev1.Volume{
			Name:         vol.Name,
			VolumeSource: vol.VolumeSource,
		})
	}
	ptb.WithVolumes(additionalVolume, k8sbuilder.Merge)
	if mb.IsPersistence() && mb.Spec.Deployment.Persistence.Volume != nil {
		ptb.WithVolumes([]corev1.Volume{
			{
				Name:         "metricbeat-data",
				VolumeSource: *mb.Spec.Deployment.Persistence.Volume,
			},
		}, k8sbuilder.Merge)
	} else if !mb.IsPersistence() {
		ptb.WithVolumes([]corev1.Volume{
			{
				Name: "metricbeat-data",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
		}, k8sbuilder.Merge)
	}

	// Add configmap volumes
	for _, cm := range configMaps {
		switch cm.Name {
		case GetConfigMapConfigName(mb):
			ptb.WithVolumes([]corev1.Volume{
				{
					Name: "metricbeat-config",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: cm.Name,
							},
						},
					},
				},
			}, k8sbuilder.Merge)
		case GetConfigMapModuleName(mb):
			ptb.WithVolumes([]corev1.Volume{
				{
					Name: "metricbeat-module",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: cm.Name,
							},
						},
					},
				},
			}, k8sbuilder.Merge)
		}
	}

	// Elasticsearch CA secret
	if mb.Spec.ElasticsearchRef.IsManaged() && es.IsTlsApiEnabled() && es.IsSelfManagedSecretForTlsApi() {
		ptb.WithVolumes([]corev1.Volume{
			{
				Name: "ca-elasticsearch",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: GetSecretNameForCAElasticsearch(mb),
					},
				},
			},
		}, k8sbuilder.Merge)
	} else if (mb.Spec.ElasticsearchRef.IsExternal() && mb.Spec.ElasticsearchRef.ElasticsearchCaSecretRef != nil) || (mb.Spec.ElasticsearchRef.IsManaged() && es.IsTlsApiEnabled() && mb.Spec.ElasticsearchRef.ElasticsearchCaSecretRef != nil) {
		ptb.WithVolumes([]corev1.Volume{
			{
				Name: "ca-elasticsearch",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: mb.Spec.ElasticsearchRef.ElasticsearchCaSecretRef.Name,
					},
				},
			},
		}, k8sbuilder.Merge)
	}

	// Compute Security context
	ptb.WithSecurityContext(&corev1.PodSecurityContext{
		FSGroup: pointer.Int64(1000),
	}, k8sbuilder.Merge)

	// Compute pod template name
	ptb.PodTemplate().Name = GetStatefulsetName(mb)

	// Compute image pull secret
	ptb.PodTemplate().Spec.ImagePullSecrets = mb.Spec.ImagePullSecrets

	// Compute Statefullset
	statefullset = &appv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   mb.Namespace,
			Name:        GetStatefulsetName(mb),
			Labels:      getLabels(mb, mb.Spec.Deployment.Labels),
			Annotations: getAnnotations(mb, mb.Spec.Deployment.Annotations),
		},
		Spec: appv1.StatefulSetSpec{
			Replicas: pointer.Int32(mb.Spec.Deployment.Replicas),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"cluster":                       mb.Name,
					beatcrd.MetricbeatAnnotationKey: "true",
				},
			},
			ServiceName: GetGlobalServiceName(mb),

			Template: *ptb.PodTemplate(),
		},
	}

	// Compute persistence
	if mb.IsPersistence() && mb.Spec.Deployment.Persistence.VolumeClaimSpec != nil {
		statefullset.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "metricbeat-data",
				},
				Spec: *mb.Spec.Deployment.Persistence.VolumeClaimSpec,
			},
		}
	}

	return statefullset, nil
}

// getMetricbeatContainer permit to get Metricbeat container containning from pod template
func getMetricbeatContainer(podTemplate *corev1.PodTemplateSpec) (container *corev1.Container) {
	if podTemplate == nil {
		return nil
	}

	for _, p := range podTemplate.Spec.Containers {
		if p.Name == "metricbeat" {
			return &p
		}
	}

	return nil
}

// computeAntiAffinity permit to get  anti affinity spec
// Default to soft anti affinity
func computeAntiAffinity(mb *beatcrd.Metricbeat) (antiAffinity *corev1.PodAntiAffinity, err error) {

	antiAffinity = &corev1.PodAntiAffinity{}
	topologyKey := "kubernetes.io/hostname"

	// Compute the antiAffinity
	if mb.Spec.Deployment.AntiAffinity != nil && mb.Spec.Deployment.AntiAffinity.TopologyKey != "" {
		topologyKey = mb.Spec.Deployment.AntiAffinity.TopologyKey
	}
	if mb.Spec.Deployment.AntiAffinity != nil && mb.Spec.Deployment.AntiAffinity.Type == "hard" {

		antiAffinity.RequiredDuringSchedulingIgnoredDuringExecution = []corev1.PodAffinityTerm{
			{
				TopologyKey: topologyKey,
				LabelSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"cluster":                       mb.Name,
						beatcrd.MetricbeatAnnotationKey: "true",
					},
				},
			},
		}

		return antiAffinity, nil
	}

	antiAffinity.PreferredDuringSchedulingIgnoredDuringExecution = []corev1.WeightedPodAffinityTerm{
		{
			Weight: 10,
			PodAffinityTerm: corev1.PodAffinityTerm{
				TopologyKey: topologyKey,
				LabelSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"cluster":                       mb.Name,
						beatcrd.MetricbeatAnnotationKey: "true",
					},
				},
			},
		},
	}

	return antiAffinity, nil
}
