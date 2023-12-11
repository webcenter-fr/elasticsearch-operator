package logstash

import (
	"bytes"
	"fmt"
	"strings"

	"emperror.dev/errors"
	"github.com/codingsince1985/checksum"
	"github.com/disaster37/k8sbuilder"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/apis/logstash/v1"
	elasticsearchcontrollers "github.com/webcenter-fr/elasticsearch-operator/controllers/elasticsearch"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/utils/ptr"
)

// GenerateStatefullset permit to generate statefullset
func buildStatefulsets(ls *logstashcrd.Logstash, es *elasticsearchcrd.Elasticsearch, secretsChecksum []corev1.Secret, configMapsChecksum []corev1.ConfigMap) (statefullsets []appv1.StatefulSet, err error) {

	statefullsets = make([]appv1.StatefulSet, 0, 1)
	checksumAnnotations := map[string]string{}

	// Generate confimaps to know what file to mount
	configMaps, err := buildConfigMaps(ls)
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
		checksumAnnotations[fmt.Sprintf("%s/configmap-%s", logstashcrd.LogstashAnnotationKey, cm.Name)] = sum
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
		checksumAnnotations[fmt.Sprintf("%s/secret-%s", logstashcrd.LogstashAnnotationKey, s.Name)] = sum
	}

	cb := k8sbuilder.NewContainerBuilder()
	ptb := k8sbuilder.NewPodTemplateBuilder()

	logstashContainer := getLogstashContainer(ls.Spec.Deployment.PodTemplate)
	if logstashContainer == nil {
		logstashContainer = &corev1.Container{}
	}

	// Initialize Logstash container from user provided
	cb.WithContainer(logstashContainer.DeepCopy()).
		Container().Name = "logstash"

	// Compute EnvFrom
	cb.WithEnvFrom(ls.Spec.Deployment.EnvFrom, k8sbuilder.Merge)

	// Compute Env
	cb.WithEnv(ls.Spec.Deployment.Env, k8sbuilder.Merge).
		WithEnv([]corev1.EnvVar{
			{
				Name: "NODE_NAME",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						APIVersion: "v1",
						FieldPath:  "metadata.name",
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
			{
				Name:  "LS_JAVA_OPTS",
				Value: ls.Spec.Deployment.Jvm,
			},
			{
				Name:  "HTTP_HOST",
				Value: "0.0.0.0",
			},
		}, k8sbuilder.Merge)

	// Inject Elasticsearch CA Path if provided
	if es != nil && es.Spec.Tls.IsTlsEnabled() || ls.Spec.ElasticsearchRef.IsExternal() && ls.Spec.ElasticsearchRef.ElasticsearchCaSecretRef != nil {
		cb.WithEnv([]corev1.EnvVar{
			{
				Name:  "ELASTICSEARCH_CA_PATH",
				Value: "/usr/share/logstash/config/es-ca/ca.crt",
			},
		}, k8sbuilder.Merge)
	}

	// Inject Elasticsearch targets if provided
	if es != nil {
		cb.WithEnv([]corev1.EnvVar{
			{
				Name:  "ELASTICSEARCH_HOST",
				Value: elasticsearchcontrollers.GetPublicUrl(es, ls.Spec.ElasticsearchRef.ManagedElasticsearchRef.TargetNodeGroup, false),
			},
		}, k8sbuilder.Merge)
	} else if ls.Spec.ElasticsearchRef.IsExternal() && len(ls.Spec.ElasticsearchRef.ExternalElasticsearchRef.Addresses) > 0 {
		cb.WithEnv([]corev1.EnvVar{
			{
				Name:  "ELASTICSEARCH_HOST",
				Value: ls.Spec.ElasticsearchRef.ExternalElasticsearchRef.Addresses[0],
			},
		}, k8sbuilder.Merge)
	}

	// Inject Elasticsearch credentials if provided
	if ls.Spec.ElasticsearchRef.SecretRef != nil {
		cb.WithEnv([]corev1.EnvVar{
			{
				Name: "ELASTICSEARCH_USERNAME",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: ls.Spec.ElasticsearchRef.SecretRef.Name,
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
							Name: ls.Spec.ElasticsearchRef.SecretRef.Name,
						},
						Key: "password",
					},
				},
			},
		}, k8sbuilder.Merge)
	}

	// Compute ports
	cb.WithPort(ls.Spec.Deployment.Ports, k8sbuilder.Merge).
		WithPort([]corev1.ContainerPort{
			{
				Name:          "http",
				ContainerPort: 9600,
				Protocol:      corev1.ProtocolTCP,
			},
		}, k8sbuilder.Merge)

	// Compute resources
	cb.WithResource(ls.Spec.Deployment.Resources, k8sbuilder.Merge)

	// Compute image
	cb.WithImage(GetContainerImage(ls), k8sbuilder.OverwriteIfDefaultValue)

	// Compute image pull policy
	cb.WithImagePullPolicy(ls.Spec.ImagePullPolicy, k8sbuilder.OverwriteIfDefaultValue)

	// Compute security context
	cb.WithSecurityContext(&corev1.SecurityContext{
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{
				"ALL",
			},
		},
		RunAsUser:    ptr.To[int64](1000),
		RunAsNonRoot: ptr.To[bool](true),
	}, k8sbuilder.OverwriteIfDefaultValue)

	// Compute volume mount
	additionalVolumeMounts := make([]corev1.VolumeMount, 0, len(ls.Spec.Deployment.AdditionalVolumes))
	for _, vol := range ls.Spec.Deployment.AdditionalVolumes {
		additionalVolumeMounts = append(additionalVolumeMounts, corev1.VolumeMount{
			Name:      vol.Name,
			MountPath: vol.MountPath,
			ReadOnly:  vol.ReadOnly,
			SubPath:   vol.SubPath,
		})
	}
	cb.WithVolumeMount(additionalVolumeMounts, k8sbuilder.Merge).
		WithVolumeMount([]corev1.VolumeMount{
			{
				Name:      "config",
				MountPath: "/usr/share/logstash/config",
			},
		}, k8sbuilder.Merge)
	cb.WithVolumeMount([]corev1.VolumeMount{
		{
			Name:      "logstash-data",
			MountPath: "/usr/share/logstash/data",
		},
	}, k8sbuilder.Merge)
	if len(ls.Spec.PluginsList) > 0 {
		cb.WithVolumeMount([]corev1.VolumeMount{
			{
				Name:      "plugin",
				MountPath: "/usr/share/elogstash/plugins",
			},
		}, k8sbuilder.Merge)
	}

	// Mount configmap of type pipeline or pattern
	for _, cm := range configMaps {
		// Mount config in the right path
		switch cm.Name {
		case GetConfigMapPipelineName(ls):
			cb.WithVolumeMount([]corev1.VolumeMount{
				{
					Name:      "logstash-pipeline",
					MountPath: "/usr/share/logstash/pipeline",
				},
			}, k8sbuilder.Merge)
		case GetConfigMapPatternName(ls):
			cb.WithVolumeMount([]corev1.VolumeMount{
				{
					Name:      "logstash-pattern",
					MountPath: "/usr/share/logstash/patterns",
				},
			}, k8sbuilder.Merge)
		}
	}

	// Compute liveness
	cb.WithLivenessProbe(&corev1.Probe{
		TimeoutSeconds:   5,
		PeriodSeconds:    30,
		FailureThreshold: 3,
		SuccessThreshold: 1,
		ProbeHandler: corev1.ProbeHandler{
			TCPSocket: &corev1.TCPSocketAction{
				Port: intstr.FromInt(9600),
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
				Port: intstr.FromInt(9600),
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
				Port: intstr.FromInt(9600),
			},
		},
	}, k8sbuilder.OverwriteIfDefaultValue)

	// Initialise PodTemplate
	ptb.WithPodTemplateSpec(ls.Spec.Deployment.PodTemplate)

	// Compute labels
	// Do not set global labels here to avoid reconcile pod just because global label change
	ptb.WithLabels(map[string]string{
		"cluster":                         ls.Name,
		logstashcrd.LogstashAnnotationKey: "true",
	}).
		WithLabels(ls.Spec.Deployment.Labels, k8sbuilder.Merge)

	// Compute annotations
	// Do not set global annotation here to avoid reconcile pod just because global annotation change
	ptb.WithAnnotations(map[string]string{
		logstashcrd.LogstashAnnotationKey: "true",
	}).
		WithAnnotations(ls.Spec.Deployment.Annotations, k8sbuilder.Merge).
		WithAnnotations(checksumAnnotations, k8sbuilder.Merge)

	// Compute NodeSelector
	ptb.WithNodeSelector(ls.Spec.Deployment.NodeSelector, k8sbuilder.OverwriteIfDefaultValue)

	// Compute Termination grac period
	ptb.WithTerminationGracePeriodSeconds(120, k8sbuilder.OverwriteIfDefaultValue)

	// Compute toleration
	ptb.WithTolerations(ls.Spec.Deployment.Tolerations, k8sbuilder.OverwriteIfDefaultValue)

	// compute anti affinity
	antiAffinity, err := computeAntiAffinity(ls)
	if err != nil {
		return nil, errors.Wrap(err, "Error when compute anti affinity")
	}
	ptb.WithAffinity(corev1.Affinity{
		PodAntiAffinity: antiAffinity,
	}, k8sbuilder.OverwriteIfDefaultValue)

	// Compute containers
	ptb.WithContainers([]corev1.Container{*cb.Container()}, k8sbuilder.Merge)

	// Compute init containers
	if ls.Spec.KeystoreSecretRef != nil {
		kcb := k8sbuilder.NewContainerBuilder().WithContainer(&corev1.Container{
			Name:            "init-keystore",
			Image:           GetContainerImage(ls),
			ImagePullPolicy: ls.Spec.ImagePullPolicy,
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "keystore",
					MountPath: "/mnt/keystore",
				},
				{
					Name:      "logstash-keystore",
					MountPath: "/mnt/keystoreSecrets",
				},
			},
			Command: []string{
				"/bin/bash",
				"-c",
				`#!/usr/bin/env bash
set -euo pipefail

logstash-keystore create
for i in /mnt/keystoreSecrets/*; do
    key=$(basename $i)
    echo "Adding file $i to keystore key $key"
    logstash-keystore add -x "$key" < $i
done

cp -a /usr/share/logstash/config/logstash.keystore /mnt/keystore/
`,
			},
		})
		kcb.WithResource(ls.Spec.Deployment.InitContainerResources)

		ptb.WithInitContainers([]corev1.Container{*kcb.Container()}, k8sbuilder.Merge)
	}
	ccb := k8sbuilder.NewContainerBuilder().WithContainer(&corev1.Container{
		Name:            "init-filesystem",
		Image:           GetContainerImage(ls),
		ImagePullPolicy: ls.Spec.ImagePullPolicy,
		SecurityContext: &corev1.SecurityContext{
			RunAsUser: ptr.To[int64](0),
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
				Name:      "config",
				MountPath: "/mnt/config",
			},
			{
				Name:      "keystore",
				MountPath: "/mnt/keystore",
			},
			{
				Name:      "logstash-data",
				MountPath: "/mnt/data",
			},
		},
	})

	// Inject env / envFrom to get proxy for exemple
	ccb.WithEnv(ls.Spec.Deployment.Env, k8sbuilder.Merge)
	ccb.WithEnvFrom(ls.Spec.Deployment.EnvFrom, k8sbuilder.Merge)

	var command strings.Builder
	command.WriteString(`#!/usr/bin/env bash
set -euo pipefail

# Move original config
echo "Move original logstash configs"
cp -a /usr/share/logstash/config/* /mnt/config/

# Move configmaps
if [ -d /mnt/configmap ]; then
  echo "Move custom configs"
  cp -f /mnt/configmap/* /mnt/config/
fi

# Move CA Elasticsearch
if [ -d /mnt/ca-elasticsearch ]; then
  echo "Move CA certificate"
  mkdir -p /mnt/config/es-ca
  cp /mnt/ca-elasticsearch/* /mnt/config/es-ca/
fi

# Move keystore
if [ -f /mnt/keystore/logstash.keystore ]; then
  echo "Move keystore"
  cp /mnt/keystore/logstash.keystore /mnt/config
fi

# Set right
echo "Set right"
chown -R logstash:logstash /mnt/config
chown -v logstash:logstash /mnt/data
`)
	for _, plugin := range ls.Spec.PluginsList {
		command.WriteString(fmt.Sprintf("./bin/logstash-plugin install %s\n", plugin))
	}
	command.WriteString(`
if [ -d /mnt/plugins ]; then
  cp -a /usr/share/logstash/plugins/* /mnt/plugins/
  chown -R logstash:logstash /mnt/plugins
fi
`)
	ccb.Container().Command = []string{
		"/bin/bash",
		"-c",
		command.String(),
	}

	// Compute mount config maps
	for _, configMap := range configMaps {

		if configMap.Name == GetConfigMapConfigName(ls) {
			ccb.WithVolumeMount([]corev1.VolumeMount{
				{
					Name:      "logstash-config",
					MountPath: "/mnt/configmap",
				},
			}, k8sbuilder.Merge)

			break
		}
	}
	if len(ls.Spec.PluginsList) > 0 {
		ccb.WithVolumeMount([]corev1.VolumeMount{
			{
				Name:      "plugin",
				MountPath: "/mnt/plugins",
			},
		}, k8sbuilder.Merge)
	}

	// Compute mount CA elasticsearch
	if (ls.Spec.ElasticsearchRef.IsManaged() && es.Spec.Tls.IsTlsEnabled()) || (ls.Spec.ElasticsearchRef.IsExternal() && ls.Spec.ElasticsearchRef.ElasticsearchCaSecretRef != nil) {
		ccb.WithVolumeMount([]corev1.VolumeMount{
			{
				Name:      "ca-elasticsearch",
				MountPath: "/mnt/ca-elasticsearch",
			},
		}, k8sbuilder.Merge)
	}

	ccb.WithResource(ls.Spec.Deployment.InitContainerResources)
	ptb.WithInitContainers([]corev1.Container{*ccb.Container()}, k8sbuilder.Merge)

	// Compute volumes
	ptb.WithVolumes([]corev1.Volume{
		{
			Name: "keystore",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: "config",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: "plugin",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	}, k8sbuilder.Merge)
	additionalVolume := make([]corev1.Volume, 0, len(ls.Spec.Deployment.AdditionalVolumes))
	for _, vol := range ls.Spec.Deployment.AdditionalVolumes {
		additionalVolume = append(additionalVolume, corev1.Volume{
			Name:         vol.Name,
			VolumeSource: vol.VolumeSource,
		})
	}
	ptb.WithVolumes(additionalVolume, k8sbuilder.Merge)
	if GetSecretNameForKeystore(ls) != "" {
		ptb.WithVolumes([]corev1.Volume{
			{
				Name: "logstash-keystore",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: GetSecretNameForKeystore(ls),
					},
				},
			},
		}, k8sbuilder.Merge)
	}
	if ls.IsPersistence() && ls.Spec.Deployment.Persistence.Volume != nil {
		ptb.WithVolumes([]corev1.Volume{
			{
				Name:         "logstash-data",
				VolumeSource: *ls.Spec.Deployment.Persistence.Volume,
			},
		}, k8sbuilder.Merge)
	} else if !ls.IsPersistence() {
		ptb.WithVolumes([]corev1.Volume{
			{
				Name: "logstash-data",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
		}, k8sbuilder.Merge)
	}

	// Add configmap volumes
	for _, cm := range configMaps {
		switch cm.Name {
		case GetConfigMapConfigName(ls):
			ptb.WithVolumes([]corev1.Volume{
				{
					Name: "logstash-config",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: cm.Name,
							},
						},
					},
				},
			}, k8sbuilder.Merge)
		case GetConfigMapPipelineName(ls):
			ptb.WithVolumes([]corev1.Volume{
				{
					Name: "logstash-pipeline",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: cm.Name,
							},
						},
					},
				},
			}, k8sbuilder.Merge)
		case GetConfigMapPatternName(ls):
			ptb.WithVolumes([]corev1.Volume{
				{
					Name: "logstash-pattern",
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

	if ls.Spec.ElasticsearchRef.IsManaged() && es.Spec.Tls.IsTlsEnabled() && es.Spec.Tls.IsSelfManagedSecretForTls() {
		ptb.WithVolumes([]corev1.Volume{
			{
				Name: "ca-elasticsearch",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: GetSecretNameForCAElasticsearch(ls),
					},
				},
			},
		}, k8sbuilder.Merge)
	} else if (ls.Spec.ElasticsearchRef.IsExternal() && ls.Spec.ElasticsearchRef.ElasticsearchCaSecretRef != nil) || (ls.Spec.ElasticsearchRef.IsManaged() && es.Spec.Tls.IsTlsEnabled() && ls.Spec.ElasticsearchRef.ElasticsearchCaSecretRef != nil) {
		ptb.WithVolumes([]corev1.Volume{
			{
				Name: "ca-elasticsearch",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: ls.Spec.ElasticsearchRef.ElasticsearchCaSecretRef.Name,
					},
				},
			},
		}, k8sbuilder.Merge)
	}

	// Compute Security context
	ptb.WithSecurityContext(&corev1.PodSecurityContext{
		FSGroup: ptr.To[int64](1000),
	}, k8sbuilder.Merge)

	// Compute pod template name
	ptb.PodTemplate().Name = GetStatefulsetName(ls)

	// Compute image pull secret
	ptb.PodTemplate().Spec.ImagePullSecrets = ls.Spec.ImagePullSecrets

	// Compute Statefullset
	statefullset := &appv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   ls.Namespace,
			Name:        GetStatefulsetName(ls),
			Labels:      getLabels(ls, ls.Spec.Deployment.Labels),
			Annotations: getAnnotations(ls, ls.Spec.Deployment.Annotations),
		},
		Spec: appv1.StatefulSetSpec{
			Replicas: ptr.To[int32](ls.Spec.Deployment.Replicas),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"cluster":                         ls.Name,
					logstashcrd.LogstashAnnotationKey: "true",
				},
			},
			ServiceName: GetGlobalServiceName(ls),

			Template: *ptb.PodTemplate(),
		},
	}

	// Compute persistence
	if ls.IsPersistence() && ls.Spec.Deployment.Persistence.VolumeClaimSpec != nil {
		statefullset.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "logstash-data",
				},
				Spec: *ls.Spec.Deployment.Persistence.VolumeClaimSpec,
			},
		}
	}

	statefullsets = append(statefullsets, *statefullset)

	return statefullsets, nil
}

// getLogstashContainer permit to get Logstash container containning from pod template
func getLogstashContainer(podTemplate *corev1.PodTemplateSpec) (container *corev1.Container) {
	if podTemplate == nil {
		return nil
	}

	for _, p := range podTemplate.Spec.Containers {
		if p.Name == "logstash" {
			return &p
		}
	}

	return nil
}

// computeAntiAffinity permit to get  anti affinity spec
// Default to soft anti affinity
func computeAntiAffinity(ls *logstashcrd.Logstash) (antiAffinity *corev1.PodAntiAffinity, err error) {

	antiAffinity = &corev1.PodAntiAffinity{}
	topologyKey := "kubernetes.io/hostname"

	// Compute the antiAffinity
	if ls.Spec.Deployment.AntiAffinity != nil && ls.Spec.Deployment.AntiAffinity.TopologyKey != "" {
		topologyKey = ls.Spec.Deployment.AntiAffinity.TopologyKey
	}
	if ls.Spec.Deployment.AntiAffinity != nil && ls.Spec.Deployment.AntiAffinity.Type == "hard" {

		antiAffinity.RequiredDuringSchedulingIgnoredDuringExecution = []corev1.PodAffinityTerm{
			{
				TopologyKey: topologyKey,
				LabelSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"cluster":                         ls.Name,
						logstashcrd.LogstashAnnotationKey: "true",
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
						"cluster":                         ls.Name,
						logstashcrd.LogstashAnnotationKey: "true",
					},
				},
			},
		},
	}

	return antiAffinity, nil
}
