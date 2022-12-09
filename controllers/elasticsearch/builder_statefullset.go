package elasticsearch

import (
	"fmt"
	"strings"

	"github.com/codingsince1985/checksum"
	"github.com/disaster37/k8sbuilder"
	"github.com/pkg/errors"
	"github.com/thoas/go-funk"
	elasticsearchapi "github.com/webcenter-fr/elasticsearch-operator/api/v1alpha1"
	"github.com/webcenter-fr/elasticsearch-operator/pkg/helper"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
)

var (
	roleList = []string{
		"master",
		"data",
		"ingest",
		"ml",
		"remote_cluster_client",
		"transform",
	}
)

// GenerateStatefullsets permit to generate statefullsets for each node groups
func BuildStatefullsets(es *elasticsearchapi.Elasticsearch) (statefullsets []*appv1.StatefulSet, err error) {
	var (
		sts *appv1.StatefulSet
	)

	for _, nodeGroup := range es.Spec.NodeGroups {

		cb := k8sbuilder.NewContainerBuilder()
		ptb := k8sbuilder.NewPodTemplateBuilder()
		globalElasticsearchContainer := getElasticsearchContainer(es.Spec.GlobalNodeGroup.PodTemplate)
		if globalElasticsearchContainer == nil {
			globalElasticsearchContainer = &corev1.Container{}
		}
		localElasticsearchContainer := getElasticsearchContainer(nodeGroup.PodTemplate)
		if localElasticsearchContainer == nil {
			localElasticsearchContainer = &corev1.Container{}
		}

		// Initialise Elasticsearch container from user provided
		cb.WithContainer(globalElasticsearchContainer).
			WithContainer(localElasticsearchContainer, k8sbuilder.Merge).
			Container().Name = "elasticsearch"

		// Compute EnvFrom
		cb.WithEnvFrom(es.Spec.GlobalNodeGroup.EnvFrom).
			WithEnvFrom(nodeGroup.EnvFrom, k8sbuilder.Merge)

		// Compute Env
		cb.WithEnv(es.Spec.GlobalNodeGroup.Env).
			WithEnv(nodeGroup.Env, k8sbuilder.Merge).
			WithEnv(computeRoles(nodeGroup.Roles), k8sbuilder.Merge).
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
				{
					Name:  "ELASTICSEARCH_JAVA_OPTS",
					Value: computeJavaOpts(es, &nodeGroup),
				},
				{
					Name:  "cluster.initial_master_nodes",
					Value: computeInitialMasterNodes(es),
				},
				{
					Name:  "discovery.seed_hosts",
					Value: computeDiscoverySeedHosts(es),
				},
				{
					Name:  "cluster.name",
					Value: es.Name,
				},
				{
					Name:  "network.host",
					Value: "0.0.0.0",
				},
				{
					Name:  "bootstrap.memory_lock",
					Value: "true",
				},
				{
					Name: "ELASTIC_PASSWORD",
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: GetSecretNameForCredentials(es),
							},
							Key: "elastic",
						},
					},
				},
			}, k8sbuilder.Merge)
		if len(es.Spec.NodeGroups) == 1 && es.Spec.NodeGroups[0].Replicas == 1 {
			// Cluster with only one node
			cb.WithEnv([]corev1.EnvVar{
				{
					Name:  "discovery.type",
					Value: "single-node",
				},
			}, k8sbuilder.Merge)
		}

		// Compute ports
		cb.WithPort([]corev1.ContainerPort{
			{
				Name:          "http",
				ContainerPort: 9200,
				Protocol:      corev1.ProtocolTCP,
			},
			{
				Name:          "transport",
				ContainerPort: 9300,
				Protocol:      corev1.ProtocolTCP,
			},
		}, k8sbuilder.Merge)

		// Compute resources
		cb.WithResource(nodeGroup.Resources, k8sbuilder.Merge)

		// Compute image
		cb.WithImage(GetContainerImage(es), k8sbuilder.OverwriteIfDefaultValue)

		// Compute image pull policy
		cb.WithImagePullPolicy(es.Spec.ImagePullPolicy).
			WithImagePullPolicy(globalElasticsearchContainer.ImagePullPolicy, k8sbuilder.Merge).
			WithImagePullPolicy(localElasticsearchContainer.ImagePullPolicy, k8sbuilder.Merge)

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
		additionalVolumeMounts := make([]corev1.VolumeMount, 0, len(es.Spec.GlobalNodeGroup.AdditionalVolumes))
		for _, vol := range es.Spec.GlobalNodeGroup.AdditionalVolumes {
			additionalVolumeMounts = append(additionalVolumeMounts, corev1.VolumeMount{
				Name:      vol.Name,
				MountPath: vol.MountPath,
				ReadOnly:  vol.ReadOnly,
				SubPath:   vol.SubPath,
			})
		}
		cb.WithVolumeMount(additionalVolumeMounts).
			WithVolumeMount([]corev1.VolumeMount{
				{
					Name:      "node-tls",
					MountPath: "/usr/share/elasticsearch/config/certs/node",
				},
				{
					Name:      "api-tls",
					MountPath: "/usr/share/elasticsearch/config/certs/api",
				},
				{
					Name:      "keystore",
					MountPath: "/mnt/keystore",
				},
			}, k8sbuilder.Merge)
		if nodeGroup.Persistence != nil && (nodeGroup.Persistence.Volume != nil || nodeGroup.Persistence.VolumeClaimSpec != nil) {
			cb.WithVolumeMount([]corev1.VolumeMount{
				{
					Name:      "elasticsearch-data",
					MountPath: "/usr/share/elasticsearch/data",
				},
			}, k8sbuilder.Merge)
		}
		// Compute mount config maps
		configMaps, err := BuildConfigMaps(es)
		if err != nil {
			return nil, errors.Wrap(err, "Error when generate configMaps")
		}
		for _, configMap := range configMaps {
			if configMap.Name == GetNodeGroupConfigMapName(es, nodeGroup.Name) {
				additionalVolumeMounts = make([]corev1.VolumeMount, 0, len(configMap.Data))
				for key := range configMap.Data {
					additionalVolumeMounts = append(additionalVolumeMounts, corev1.VolumeMount{
						Name:      "elasticsearch-config",
						MountPath: fmt.Sprintf("/usr/share/elasticsearch/config/%s", key),
						SubPath:   key,
					})
				}
				cb.WithVolumeMount(additionalVolumeMounts, k8sbuilder.Merge)
			}
		}

		// Compute liveness
		cb.WithLivenessProbe(&corev1.Probe{
			TimeoutSeconds:   5,
			PeriodSeconds:    30,
			FailureThreshold: 10,
			SuccessThreshold: 1,
			ProbeHandler: corev1.ProbeHandler{
				TCPSocket: &corev1.TCPSocketAction{
					Port: intstr.FromInt(9300),
				},
			},
		}, k8sbuilder.OverwriteIfDefaultValue)

		// Compute readiness
		cb.WithReadinessProbe(&corev1.Probe{
			TimeoutSeconds:   5,
			PeriodSeconds:    30,
			FailureThreshold: 3,
			SuccessThreshold: 1,
			ProbeHandler: corev1.ProbeHandler{
				TCPSocket: &corev1.TCPSocketAction{
					Port: intstr.FromInt(9200),
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
					Port: intstr.FromInt(9200),
				},
			},
		}, k8sbuilder.OverwriteIfDefaultValue)

		// Add specific command to handle plugin installation and keystore
		var pluginInstallation strings.Builder
		pluginInstallation.WriteString(`#!/usr/bin/env bash
set -euo pipefail

if [[ -f /mnt/keystore/elasticsearch.keystore ]]; then
  cp /mnt/keystore/elasticsearch.keystore /usr/share/elasticsearch/config/elasticsearch.keystore
fi
`)
		for _, plugin := range es.Spec.PluginsList {
			pluginInstallation.WriteString(fmt.Sprintf("./bin/elasticsearch-plugin install -b %s\n", plugin))
		}
		pluginInstallation.WriteString("bash elasticsearch-docker-entrypoint.sh\n")
		cb.Container().Command = []string{
			"sh",
			"-c",
			pluginInstallation.String(),
		}

		// Initialise PodTemplate
		ptb.WithPodTemplateSpec(es.Spec.GlobalNodeGroup.PodTemplate).
			WithPodTemplateSpec(nodeGroup.PodTemplate, k8sbuilder.Merge)

		// Compute labels
		ptb.WithLabels(es.Labels, k8sbuilder.Merge).
			WithLabels(es.Spec.GlobalNodeGroup.Labels, k8sbuilder.Merge).
			WithLabels(nodeGroup.Labels, k8sbuilder.Merge).WithLabels(map[string]string{
			"cluster":   es.Name,
			"nodeGroup": nodeGroup.Name,
		}, k8sbuilder.Merge)

		// Computes pods annotations to force reconcil
		configMapChecksumAnnotations := map[string]string{}
		for _, configMap := range configMaps {
			for file, contend := range configMap.Data {
				sum, err := checksum.SHA256sumReader(strings.NewReader(contend))
				if err != nil {
					return nil, errors.Wrapf(err, "Error when generate checksum for %s/%s", configMap.Name, file)
				}
				configMapChecksumAnnotations[fmt.Sprintf("%s/checksum-%s", elasticsearchAnnotationKey, file)] = sum
			}
		}

		// Compute annotations
		ptb.WithAnnotations(es.Annotations, k8sbuilder.Merge).
			WithAnnotations(es.Spec.GlobalNodeGroup.Annotations, k8sbuilder.Merge).
			WithAnnotations(nodeGroup.Annotations, k8sbuilder.Merge).
			WithAnnotations(configMapChecksumAnnotations, k8sbuilder.Merge)

		// Compute NodeSelector
		ptb.WithNodeSelector(nodeGroup.NodeSelector, k8sbuilder.Merge)

		// Compute Termination grac period
		ptb.WithTerminationGracePeriodSeconds(120, k8sbuilder.OverwriteIfDefaultValue)

		// Compute toleration
		ptb.WithTolerations(nodeGroup.Tolerations, k8sbuilder.Merge)

		// compute anti affinity
		antiAffinity, err := computeAntiAffinity(es, &nodeGroup)
		if err != nil {
			return nil, errors.Wrapf(err, "Error when compute anti affinity for %s", nodeGroup.Name)
		}
		ptb.WithAffinity(corev1.Affinity{
			PodAntiAffinity: antiAffinity,
		}, k8sbuilder.Merge)

		// Compute containers
		ptb.WithContainers([]corev1.Container{*cb.Container()}, k8sbuilder.Merge)

		// Compute init containers
		if es.Spec.SetVMMaxMapCount == nil || *es.Spec.SetVMMaxMapCount {
			icb := k8sbuilder.NewContainerBuilder().WithContainer(&corev1.Container{
				Name:            "configure-sysctl",
				Image:           GetContainerImage(es),
				ImagePullPolicy: es.Spec.ImagePullPolicy,
				SecurityContext: &corev1.SecurityContext{
					Privileged: pointer.Bool(true),
					RunAsUser:  pointer.Int64(0),
				},
				Command: []string{
					"sysctl",
					"-w",
					"vm.max_map_count=262144",
				},
			})
			icb.WithResource(es.Spec.GlobalNodeGroup.InitContainerResources)

			ptb.WithInitContainers([]corev1.Container{*icb.Container()}, k8sbuilder.Merge)
		}
		if es.Spec.GlobalNodeGroup.KeystoreSecretRef != nil {
			kcb := k8sbuilder.NewContainerBuilder().WithContainer(&corev1.Container{
				Name:            "init-keystore",
				Image:           GetContainerImage(es),
				ImagePullPolicy: es.Spec.ImagePullPolicy,
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      "keystore",
						MountPath: "/mnt/keystore",
					},
					{
						Name:      "elasticsearch-keystore",
						MountPath: "/mnt/keystoreSecrets",
					},
				},
				Command: []string{
					"/bin/bash",
					"-c",
					`
#!/bin/bash
set -e

elasticsearch-keystore create
for i in /mnt/keystoreSecrets/*/*; do
    key=$(basename $i)
    echo "Adding file $i to keystore key $key"
    elasticsearch-keystore add-file "$key" "$i"
done

# Add the bootstrap password since otherwise the Elasticsearch entrypoint tries to do this on startup
if [ ! -z ${ELASTIC_PASSWORD+x} ]; then
  echo 'Adding env $ELASTIC_PASSWORD to keystore as key bootstrap.password'
  echo "$ELASTIC_PASSWORD" | elasticsearch-keystore add -x bootstrap.password
fi

cp -a /usr/share/elasticsearch/config/elasticsearch.keystore /mnt/keystore/
					`,
				},
			})
			kcb.WithResource(es.Spec.GlobalNodeGroup.InitContainerResources)

			ptb.WithInitContainers([]corev1.Container{*kcb.Container()}, k8sbuilder.Merge)
		}

		// Compute volumes
		ptb.WithVolumes([]corev1.Volume{
			{
				Name: "node-tls",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: GetSecretNameForTlsTransport(es),
					},
				},
			},
			{
				Name: "api-tls",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: GetSecretNameForTlsApi(es),
					},
				},
			},
			{
				Name: "elasticsearch-config",
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: GetNodeGroupConfigMapName(es, nodeGroup.Name),
						},
					},
				},
			},
			{
				Name: "keystore",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
		}, k8sbuilder.Merge)
		additionalVolume := make([]corev1.Volume, 0, len(es.Spec.GlobalNodeGroup.AdditionalVolumes))
		for _, vol := range es.Spec.GlobalNodeGroup.AdditionalVolumes {
			additionalVolume = append(additionalVolume, corev1.Volume{
				Name:         vol.Name,
				VolumeSource: vol.VolumeSource,
			})
		}
		ptb.WithVolumes(additionalVolume, k8sbuilder.Merge)
		if GetSecretNameForKeystore(es) != "" {
			ptb.WithVolumes([]corev1.Volume{
				{
					Name: "elasticsearch-keystore",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: GetSecretNameForKeystore(es),
						},
					},
				},
			}, k8sbuilder.Merge)
		}
		if nodeGroup.Persistence != nil && nodeGroup.Persistence.VolumeClaimSpec == nil && nodeGroup.Persistence.Volume != nil {
			ptb.WithVolumes([]corev1.Volume{
				{
					Name:         "elasticsearch-data",
					VolumeSource: *nodeGroup.Persistence.Volume,
				},
			}, k8sbuilder.Merge)
		}

		// Compute Security context
		ptb.WithSecurityContext(&corev1.PodSecurityContext{
			FSGroup: pointer.Int64(1000),
		}, k8sbuilder.Merge)

		ptb.PodTemplate().Name = GetNodeGroupName(es, nodeGroup.Name)

		// Compute Statefullset
		sts = &appv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   es.Namespace,
				Name:        GetNodeGroupName(es, nodeGroup.Name),
				Labels:      getLabels(es, map[string]string{"nodeGroup": nodeGroup.Name}),
				Annotations: getAnnotations(es),
			},
			Spec: appv1.StatefulSetSpec{
				Replicas: pointer.Int32(nodeGroup.Replicas),
				// Start all node to create cluster
				PodManagementPolicy: appv1.ParallelPodManagement,
				ServiceName:         GetNodeGroupServiceNameHeadless(es, nodeGroup.Name),
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"cluster":   es.Name,
						"nodeGroup": es.Name,
					},
				},

				Template: *ptb.PodTemplate(),
			},
		}

		// Compute persistence
		if nodeGroup.Persistence != nil && nodeGroup.Persistence.VolumeClaimSpec != nil {
			sts.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "elasticsearch-data",
					},
					Spec: *nodeGroup.Persistence.VolumeClaimSpec,
				},
			}
		}

		statefullsets = append(statefullsets, sts)
	}

	return statefullsets, nil
}

// getElasticsearchContainer permit to get Elasticsearch container containning from pod template
func getElasticsearchContainer(podTemplate *corev1.PodTemplateSpec) (container *corev1.Container) {
	if podTemplate == nil {
		return nil
	}

	for _, p := range podTemplate.Spec.Containers {
		if p.Name == "elasticsearch" {
			return &p
		}
	}

	return nil
}

// computeEnvFroms permit to compute the envFrom list
// It just append all, without to keep unique object
func computeEnvFroms(es *elasticsearchapi.Elasticsearch, nodeGroup *elasticsearchapi.NodeGroupSpec) (envFroms []corev1.EnvFromSource) {

	secrets := make([]any, 0)
	configMaps := make([]any, 0)
	finalSecrets := make([]any, 0)
	finalConfigMaps := make([]any, 0)

	for _, ef := range nodeGroup.EnvFrom {
		if ef.ConfigMapRef != nil {
			configMaps = append(configMaps, ef)
		} else if ef.SecretRef != nil {
			secrets = append(secrets, ef)
		}
	}

	for _, ef := range es.Spec.GlobalNodeGroup.EnvFrom {
		if ef.ConfigMapRef != nil {
			configMaps = append(configMaps, ef)
		} else if ef.SecretRef != nil {
			secrets = append(secrets, ef)
		}
	}

	k8sbuilder.MergeSliceOrDie(&finalSecrets, "SecretRef.LocalObjectReference.Name", secrets)
	k8sbuilder.MergeSliceOrDie(&finalConfigMaps, "ConfigMapRef.LocalObjectReference.Name", configMaps)
	envFroms = make([]corev1.EnvFromSource, 0, len(finalSecrets)+len(finalConfigMaps))

	for _, item := range finalSecrets {
		envFroms = append(envFroms, item.(corev1.EnvFromSource))
	}
	for _, item := range finalConfigMaps {
		envFroms = append(envFroms, item.(corev1.EnvFromSource))
	}

	return envFroms
}

// computeAntiAffinity permit to get  anti affinity spec
// Default to soft anti affinity
func computeAntiAffinity(es *elasticsearchapi.Elasticsearch, nodeGroup *elasticsearchapi.NodeGroupSpec) (antiAffinity *corev1.PodAntiAffinity, err error) {
	var expectedAntiAffinity *elasticsearchapi.AntiAffinitySpec

	antiAffinity = &corev1.PodAntiAffinity{}
	topologyKey := "kubernetes.io/hostname"

	// Check if need to merge anti affinity spec
	if nodeGroup.AntiAffinity != nil || es.Spec.GlobalNodeGroup.AntiAffinity != nil {
		expectedAntiAffinity = &elasticsearchapi.AntiAffinitySpec{}
		if err = helper.Merge(expectedAntiAffinity, nodeGroup.AntiAffinity, funk.Get(es.Spec.GlobalNodeGroup, "AntiAffinity")); err != nil {
			return nil, errors.Wrapf(err, "Error when merge global anti affinity  with node group %s", nodeGroup.Name)
		}
	}

	// Compute the antiAffinity
	if expectedAntiAffinity != nil && expectedAntiAffinity.TopologyKey != "" {
		topologyKey = expectedAntiAffinity.TopologyKey
	}
	if expectedAntiAffinity != nil && expectedAntiAffinity.Type == "hard" {

		antiAffinity.RequiredDuringSchedulingIgnoredDuringExecution = []corev1.PodAffinityTerm{
			{
				TopologyKey: topologyKey,
				LabelSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"cluster":   es.Name,
						"nodeGroup": nodeGroup.Name,
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
						"cluster":   es.Name,
						"nodeGroup": nodeGroup.Name,
					},
				},
			},
		},
	}

	return antiAffinity, nil
}

// computeRoles permit to compute les roles of node groups
func computeRoles(roles []string) (envs []corev1.EnvVar) {
	envs = make([]corev1.EnvVar, 0, len(roles))

	for _, role := range roleList {
		if funk.ContainsString(roles, role) {
			envs = append(envs, corev1.EnvVar{
				Name:  fmt.Sprintf("node.%s", role),
				Value: "true",
			})
		} else {
			envs = append(envs, corev1.EnvVar{
				Name:  fmt.Sprintf("node.%s", role),
				Value: "false",
			})
		}
	}

	return envs
}

// getJavaOpts permit to get computed JAVA_OPTS
func computeJavaOpts(es *elasticsearchapi.Elasticsearch, nodeGroup *elasticsearchapi.NodeGroupSpec) string {
	javaOpts := []string{}

	if es.Spec.GlobalNodeGroup.Jvm != "" {
		javaOpts = append(javaOpts, es.Spec.GlobalNodeGroup.Jvm)
	}

	if nodeGroup.Jvm != "" {
		javaOpts = append(javaOpts, nodeGroup.Jvm)
	}

	return strings.Join(javaOpts, " ")
}

// computeInitialMasterNodes create the list of all master nodes
func computeInitialMasterNodes(es *elasticsearchapi.Elasticsearch) string {
	masterNodes := make([]string, 0, 3)
	for _, nodeGroup := range es.Spec.NodeGroups {
		if IsMasterRole(es, nodeGroup.Name) {
			masterNodes = append(masterNodes, GetNodeGroupNodeNames(es, nodeGroup.Name)...)
		}
	}

	return strings.Join(masterNodes, " ")
}

// computeDiscoverySeedHosts create the list of all headless service of all masters node groups
func computeDiscoverySeedHosts(es *elasticsearchapi.Elasticsearch) string {
	serviceNames := make([]string, 0, 1)

	for _, nodeGroup := range es.Spec.NodeGroups {
		if IsMasterRole(es, nodeGroup.Name) {
			serviceNames = append(serviceNames, GetNodeGroupServiceNameHeadless(es, nodeGroup.Name))
		}
	}

	return strings.Join(serviceNames, " ")
}
