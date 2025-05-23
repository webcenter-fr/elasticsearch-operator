package filebeat

import (
	"bytes"
	"fmt"
	"strings"

	"emperror.dev/errors"
	"github.com/codingsince1985/checksum"
	"github.com/disaster37/k8sbuilder"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/api/beat/v1"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearch/v1"
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/api/logstash/v1"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/utils/ptr"
)

// GenerateStatefullset permit to generate statefullset
func buildStatefulsets(fb *beatcrd.Filebeat, es *elasticsearchcrd.Elasticsearch, ls *logstashcrd.Logstash, configMaps []*corev1.ConfigMap, secretsChecksum []*corev1.Secret, configMapsChecksum []*corev1.ConfigMap, isOpenshift bool) (statefullsets []*appv1.StatefulSet, err error) {
	// Check the secretRef is set when use Elasticsearch output
	if (fb.Spec.ElasticsearchRef.IsManaged() || fb.Spec.ElasticsearchRef.IsExternal()) && fb.Spec.ElasticsearchRef.SecretRef == nil {
		return nil, errors.New("You must set the secretRef when you use ElasticsearchRef")
	}

	statefullsets = make([]*appv1.StatefulSet, 0, 1)
	checksumAnnotations := map[string]string{}

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
		checksumAnnotations[fmt.Sprintf("%s/configmap-%s", beatcrd.FilebeatAnnotationKey, cm.Name)] = sum
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
		checksumAnnotations[fmt.Sprintf("%s/secret-%s", beatcrd.FilebeatAnnotationKey, s.Name)] = sum
	}

	cb := k8sbuilder.NewContainerBuilder()
	ptb := k8sbuilder.NewPodTemplateBuilder()

	filebeatContainer := getFilebeatContainer(fb.Spec.Deployment.PodTemplate)
	if filebeatContainer == nil {
		filebeatContainer = &corev1.Container{}
	}

	// Initialize Filebeat container from user provided
	cb.WithContainer(filebeatContainer.DeepCopy()).
		Container().Name = "filebeat"

	// Compute EnvFrom
	cb.WithEnvFrom(fb.Spec.Deployment.EnvFrom, k8sbuilder.Merge)

	// Compute Env
	cb.WithEnv(fb.Spec.Deployment.Env, k8sbuilder.Merge).
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

	// Inject Elasticsearch credentials if provided
	if fb.Spec.ElasticsearchRef.IsManaged() || fb.Spec.ElasticsearchRef.IsExternal() {
		cb.WithEnv([]corev1.EnvVar{
			{
				Name: "ELASTICSEARCH_USERNAME",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: fb.Spec.ElasticsearchRef.SecretRef.Name,
						},
						Key: "username",
					},
				},
			},
			{
				Name: "ELASTICSEARCH_PASSWORD",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: fb.Spec.ElasticsearchRef.SecretRef.Name,
						},
						Key: "password",
					},
				},
			},
		}, k8sbuilder.Merge)
	}

	// Compute ports
	cb.WithPort(fb.Spec.Deployment.Ports, k8sbuilder.Merge).
		WithPort([]corev1.ContainerPort{
			{
				Name:          "http",
				ContainerPort: 5066,
				Protocol:      corev1.ProtocolTCP,
			},
		}, k8sbuilder.Merge)

	// Compute resources
	cb.WithResource(fb.Spec.Deployment.Resources, k8sbuilder.Merge)

	// Compute image
	cb.WithImage(GetContainerImage(fb), k8sbuilder.OverwriteIfDefaultValue)

	// Compute image pull policy
	cb.WithImagePullPolicy(fb.Spec.ImagePullPolicy, k8sbuilder.OverwriteIfDefaultValue)

	// Compute security context
	cb.WithSecurityContext(&corev1.SecurityContext{
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{
				"ALL",
			},
		},
		AllowPrivilegeEscalation: ptr.To(false),
		Privileged:               ptr.To(false),
		RunAsUser:                ptr.To[int64](0),
		RunAsNonRoot:             ptr.To[bool](false),
		RunAsGroup:               ptr.To[int64](1000),
	}, k8sbuilder.OverwriteIfDefaultValue)

	// Compute volume mount
	additionalVolumeMounts := make([]corev1.VolumeMount, 0, len(fb.Spec.Deployment.AdditionalVolumes))
	for _, vol := range fb.Spec.Deployment.AdditionalVolumes {
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
			Name:      "filebeat-data",
			MountPath: "/usr/share/filebeat/data",
		},
	}, k8sbuilder.Merge)

	// Mount configmap of type module
	for _, cm := range configMaps {
		// Mount config in the right path
		switch cm.Name {
		case GetConfigMapModuleName(fb):
			cb.WithVolumeMount([]corev1.VolumeMount{
				{
					Name:      "filebeat-module",
					MountPath: "/usr/share/filebeat/modules.d",
				},
			}, k8sbuilder.Merge)
		case GetConfigMapConfigName(fb):
			for file := range cm.Data {
				cb.WithVolumeMount([]corev1.VolumeMount{
					{
						Name:      "filebeat-config",
						MountPath: fmt.Sprintf("/usr/share/filebeat/%s", file),
						SubPath:   file,
					},
				}, k8sbuilder.Merge)
			}
		}
	}

	// Compute mount CA elasticsearch
	if fb.Spec.ElasticsearchRef.ElasticsearchCaSecretRef != nil {
		cb.WithVolumeMount([]corev1.VolumeMount{
			{
				Name:      "ca-custom-elasticsearch",
				MountPath: "/usr/share/filebeat/es-custom-ca",
			},
		}, k8sbuilder.Merge)
	}
	if fb.Spec.ElasticsearchRef.IsManaged() && es.Spec.Tls.IsTlsEnabled() && es.Spec.Tls.IsSelfManagedSecretForTls() {
		cb.WithVolumeMount([]corev1.VolumeMount{
			{
				Name:      "ca-elasticsearch",
				MountPath: "/usr/share/filebeat/es-ca",
			},
		}, k8sbuilder.Merge)
	}

	// Compute mount CA logstash
	if fb.Spec.LogstashRef.LogstashCaSecretRef != nil {
		cb.WithVolumeMount([]corev1.VolumeMount{
			{
				Name:      "ca-custom-logstash",
				MountPath: "/usr/share/filebeat/ls-custom-ca",
			},
		}, k8sbuilder.Merge)
	}
	if ls != nil && ls.Spec.Pki.IsEnabled() && ls.Spec.Pki.HasBeatCertificate() {
		cb.WithVolumeMount([]corev1.VolumeMount{
			{
				Name:      "ca-logstash",
				MountPath: "/usr/share/filebeat/ls-ca",
			},
		}, k8sbuilder.Merge)
	}

	// Mount pki
	if fb.Spec.Pki.IsEnabled() {
		cb.WithVolumeMount([]corev1.VolumeMount{
			{
				Name:      "filebeat-certs",
				MountPath: "/usr/share/filebeat/certs",
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
	ptb.WithPodTemplateSpec(fb.Spec.Deployment.PodTemplate)

	// Compute labels
	// Do not set global labels here to avoid reconcile pod just because global label change
	ptb.WithLabels(map[string]string{
		"cluster":                     fb.Name,
		beatcrd.FilebeatAnnotationKey: "true",
	}).
		WithLabels(fb.Spec.Deployment.Labels, k8sbuilder.Merge)

	// Compute annotations
	// Do not set global annotation here to avoid reconcile pod just because global annotation change
	ptb.WithAnnotations(map[string]string{
		beatcrd.FilebeatAnnotationKey: "true",
	}).
		WithAnnotations(fb.Spec.Deployment.Annotations, k8sbuilder.Merge).
		WithAnnotations(checksumAnnotations, k8sbuilder.Merge)

	// Compute NodeSelector
	ptb.WithNodeSelector(fb.Spec.Deployment.NodeSelector, k8sbuilder.OverwriteIfDefaultValue)

	// Compute Termination grac period
	ptb.WithTerminationGracePeriodSeconds(60, k8sbuilder.OverwriteIfDefaultValue)

	// Compute toleration
	ptb.WithTolerations(fb.Spec.Deployment.Tolerations, k8sbuilder.OverwriteIfDefaultValue)

	// compute anti affinity
	antiAffinity, err := computeAntiAffinity(fb)
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
		Image:           GetContainerImage(fb),
		ImagePullPolicy: fb.Spec.ImagePullPolicy,
		SecurityContext: &corev1.SecurityContext{
			Privileged: ptr.To(false),
			RunAsUser:  ptr.To[int64](0),
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
				Name:      "filebeat-data",
				MountPath: "/mnt/data",
			},
		},
	})

	// Inject env / envFrom to get proxy for exemple
	ccb.WithEnv(fb.Spec.Deployment.Env, k8sbuilder.Merge)
	ccb.WithEnvFrom(fb.Spec.Deployment.EnvFrom, k8sbuilder.Merge)

	var command strings.Builder
	command.WriteString(`#!/usr/bin/env bash
set -euo pipefail

# Set right
echo "Set right"
chown -v root:root /mnt/data
`)

	ccb.Container().Command = []string{
		"/bin/bash",
		"-c",
		command.String(),
	}

	ccb.WithResource(fb.Spec.Deployment.InitContainerResources)
	ptb.WithInitContainers([]corev1.Container{*ccb.Container()}, k8sbuilder.Merge)

	// Compute volumes
	additionalVolume := make([]corev1.Volume, 0, len(fb.Spec.Deployment.AdditionalVolumes))
	for _, vol := range fb.Spec.Deployment.AdditionalVolumes {
		additionalVolume = append(additionalVolume, corev1.Volume{
			Name:         vol.Name,
			VolumeSource: vol.VolumeSource,
		})
	}
	ptb.WithVolumes(additionalVolume, k8sbuilder.Merge)
	if fb.IsPersistence() && fb.Spec.Deployment.Persistence.Volume != nil {
		ptb.WithVolumes([]corev1.Volume{
			{
				Name:         "filebeat-data",
				VolumeSource: *fb.Spec.Deployment.Persistence.Volume,
			},
		}, k8sbuilder.Merge)
	} else if !fb.IsPersistence() {
		ptb.WithVolumes([]corev1.Volume{
			{
				Name: "filebeat-data",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
		}, k8sbuilder.Merge)
	}

	// Add configmap volumes
	for _, cm := range configMaps {
		switch cm.Name {
		case GetConfigMapConfigName(fb):
			ptb.WithVolumes([]corev1.Volume{
				{
					Name: "filebeat-config",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: cm.Name,
							},
						},
					},
				},
			}, k8sbuilder.Merge)
		case GetConfigMapModuleName(fb):
			ptb.WithVolumes([]corev1.Volume{
				{
					Name: "filebeat-module",
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
	if fb.Spec.ElasticsearchRef.ElasticsearchCaSecretRef != nil {
		ptb.WithVolumes([]corev1.Volume{
			{
				Name: "ca-custom-elasticsearch",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: fb.Spec.ElasticsearchRef.ElasticsearchCaSecretRef.Name,
					},
				},
			},
		}, k8sbuilder.Merge)
	}
	if fb.Spec.ElasticsearchRef.IsManaged() && es.Spec.Tls.IsTlsEnabled() && es.Spec.Tls.IsSelfManagedSecretForTls() {
		ptb.WithVolumes([]corev1.Volume{
			{
				Name: "ca-elasticsearch",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: GetSecretNameForCAElasticsearch(fb),
					},
				},
			},
		}, k8sbuilder.Merge)
	}

	// Logstash CA secret
	if fb.Spec.LogstashRef.LogstashCaSecretRef != nil {
		ptb.WithVolumes([]corev1.Volume{
			{
				Name: "ca-custom-logstash",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: fb.Spec.LogstashRef.LogstashCaSecretRef.Name,
					},
				},
			},
		}, k8sbuilder.Merge)
	}
	if ls != nil && ls.Spec.Pki.IsEnabled() && ls.Spec.Pki.HasBeatCertificate() {
		ptb.WithVolumes([]corev1.Volume{
			{
				Name: "ca-logstash",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: GetSecretNameForCALogstash(fb),
					},
				},
			},
		}, k8sbuilder.Merge)
	}

	// Add PKI volume
	if fb.Spec.Pki.IsEnabled() {
		ptb.WithVolumes([]corev1.Volume{
			{
				Name: "filebeat-certs",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: GetSecretNameForTls(fb),
					},
				},
			},
		}, k8sbuilder.Merge)
	}

	// Compute Security context
	// Compute Security context
	ptb.WithSecurityContext(&corev1.PodSecurityContext{
		FSGroup: ptr.To[int64](1000),
	}, k8sbuilder.Merge)

	// On Openshift, we need to run Elasticsearch with specific serviceAccount that is binding to anyuid scc
	if isOpenshift {
		ptb.PodTemplate().Spec.ServiceAccountName = GetServiceAccountName(fb)
	}

	// Compute pod template name
	ptb.PodTemplate().Name = GetStatefulsetName(fb)

	// Compute image pull secret
	ptb.PodTemplate().Spec.ImagePullSecrets = fb.Spec.ImagePullSecrets

	// Compute Statefullset
	statefullset := &appv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   fb.Namespace,
			Name:        GetStatefulsetName(fb),
			Labels:      getLabels(fb, fb.Spec.Deployment.Labels),
			Annotations: getAnnotations(fb, fb.Spec.Deployment.Annotations),
		},
		Spec: appv1.StatefulSetSpec{
			Replicas:            ptr.To(fb.Spec.Deployment.Replicas),
			PodManagementPolicy: appv1.ParallelPodManagement,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"cluster":                     fb.Name,
					beatcrd.FilebeatAnnotationKey: "true",
				},
			},
			ServiceName: GetGlobalServiceName(fb),

			Template: *ptb.PodTemplate(),
		},
	}

	// Compute persistence
	if fb.IsPersistence() && fb.Spec.Deployment.Persistence.VolumeClaim != nil {
		statefullset.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "filebeat-data",
					Labels:      fb.Spec.Deployment.Persistence.VolumeClaim.Labels,
					Annotations: fb.Spec.Deployment.Persistence.VolumeClaim.Annotations,
				},
				Spec: fb.Spec.Deployment.Persistence.VolumeClaim.PersistentVolumeClaimSpec,
			},
		}
	}

	statefullsets = append(statefullsets, statefullset)

	return statefullsets, nil
}

// getFilebeatContainer permit to get Filebeat container containning from pod template
func getFilebeatContainer(podTemplate *corev1.PodTemplateSpec) (container *corev1.Container) {
	if podTemplate == nil {
		return nil
	}

	for _, p := range podTemplate.Spec.Containers {
		if p.Name == "filebeat" {
			return &p
		}
	}

	return nil
}

// computeAntiAffinity permit to get  anti affinity spec
// Default to soft anti affinity
func computeAntiAffinity(fb *beatcrd.Filebeat) (antiAffinity *corev1.PodAntiAffinity, err error) {
	antiAffinity = &corev1.PodAntiAffinity{}
	topologyKey := "kubernetes.io/hostname"

	// Compute the antiAffinity
	if fb.Spec.Deployment.AntiAffinity != nil && fb.Spec.Deployment.AntiAffinity.TopologyKey != "" {
		topologyKey = fb.Spec.Deployment.AntiAffinity.TopologyKey
	}
	if fb.Spec.Deployment.AntiAffinity != nil && fb.Spec.Deployment.AntiAffinity.Type == "hard" {

		antiAffinity.RequiredDuringSchedulingIgnoredDuringExecution = []corev1.PodAffinityTerm{
			{
				TopologyKey: topologyKey,
				LabelSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"cluster":                     fb.Name,
						beatcrd.FilebeatAnnotationKey: "true",
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
						"cluster":                     fb.Name,
						beatcrd.FilebeatAnnotationKey: "true",
					},
				},
			},
		},
	}

	return antiAffinity, nil
}
