package elasticsearch

import (
	"bytes"
	"fmt"
	"strings"

	"emperror.dev/errors"
	"github.com/codingsince1985/checksum"
	"github.com/disaster37/k8sbuilder"
	"github.com/disaster37/operator-sdk-extra/pkg/helper"
	"github.com/thoas/go-funk"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/utils/ptr"
)

var (
	roleList = []string{
		"master",
		"data",
		"data_content",
		"data_hot",
		"data_warm",
		"data_cold",
		"data_frozen",
		"ingest",
		"ml",
		"remote_cluster_client",
		"transform",
	}
)

// GenerateStatefullsets permit to generate statefullsets for each node groups
func buildStatefulsets(es *elasticsearchcrd.Elasticsearch, secretsChecksum []corev1.Secret, configMapsChecksum []corev1.ConfigMap) (statefullsets []appv1.StatefulSet, err error) {
	var (
		sts *appv1.StatefulSet
	)

	checksumAnnotations := map[string]string{}

	// Generate confimaps to know what file to mount
	// And to generate checksum
	configMaps, err := buildConfigMaps(es)
	if err != nil {
		return nil, errors.Wrap(err, "Error when generate configMaps")
	}

	// checksum for secret
	// Use annotations sequence instead compute checksum if provided
	for _, s := range secretsChecksum {

		sum := s.Annotations[fmt.Sprintf("%s/sequence", elasticsearchcrd.ElasticsearchAnnotationKey)]
		if sum == "" {
			j, err := json.Marshal(s.Data)
			if err != nil {
				return nil, errors.Wrapf(err, "Error when convert data of secret %s on json string", s.Name)
			}
			sum, err = checksum.SHA256sumReader(bytes.NewReader(j))
			if err != nil {
				return nil, errors.Wrapf(err, "Error when generate checksum for extra secret %s", s.Name)
			}
		}

		checksumAnnotations[fmt.Sprintf("%s/secret-%s", elasticsearchcrd.ElasticsearchAnnotationKey, s.Name)] = sum
	}

	// Compute cluster name
	clusterName := es.Name
	if es.Spec.ClusterName != "" {
		clusterName = es.Spec.ClusterName
	}

	for _, nodeGroup := range es.Spec.NodeGroups {
		nodeGroupCheckSumAnnotations := map[string]string{}
		for key, checksum := range checksumAnnotations {
			nodeGroupCheckSumAnnotations[key] = checksum
		}

		// checksum for configmap
		for _, cm := range configMapsChecksum {
			// Keep only configMap for this nodeGroup or global configMap
			if cm.Labels == nil || cm.Labels["nodeGroup"] == "" || cm.Labels["nodeGroup"] == nodeGroup.Name {
				j, err := json.Marshal(cm.Data)
				if err != nil {
					return nil, errors.Wrapf(err, "Error when convert data of configMap %s on json string", cm.Name)
				}
				sum, err := checksum.SHA256sumReader(bytes.NewReader(j))
				if err != nil {
					return nil, errors.Wrapf(err, "Error when generate checksum for extra configMap %s", cm.Name)
				}
				nodeGroupCheckSumAnnotations[fmt.Sprintf("%s/configmap-%s", elasticsearchcrd.ElasticsearchAnnotationKey, cm.Name)] = sum
			}
		}

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
		cb.WithContainer(globalElasticsearchContainer.DeepCopy()).
			WithContainer(localElasticsearchContainer.DeepCopy(), k8sbuilder.Merge).
			Container().Name = "elasticsearch"

		// Compute EnvFrom
		cb.WithEnvFrom(es.Spec.GlobalNodeGroup.EnvFrom, k8sbuilder.Merge).
			WithEnvFrom(nodeGroup.EnvFrom, k8sbuilder.Merge).
			WithEnvFrom([]corev1.EnvFromSource{
				{
					ConfigMapRef: &corev1.ConfigMapEnvSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: GetBootstrappingConfigMapName(es),
						},
					},
				},
			}, k8sbuilder.Merge)

		// Compute Env
		cb.WithEnv(es.Spec.GlobalNodeGroup.Env, k8sbuilder.Merge).
			WithEnv(nodeGroup.Env, k8sbuilder.Merge).
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
					Name: "node.name",
					ValueFrom: &corev1.EnvVarSource{
						FieldRef: &corev1.ObjectFieldSelector{
							APIVersion: "v1",
							FieldPath:  "metadata.name",
						},
					},
				},
				{
					Name: "node.attr.node_name",
					ValueFrom: &corev1.EnvVarSource{
						FieldRef: &corev1.ObjectFieldSelector{
							APIVersion: "v1",
							FieldPath:  "spec.nodeName",
						},
					},
				},
				{
					Name:  "ELASTICSEARCH_JAVA_OPTS",
					Value: computeJavaOpts(es, &nodeGroup),
				},
				{
					Name:  "cluster.name",
					Value: clusterName,
				},
				{
					Name:  "network.host",
					Value: "0.0.0.0",
				},
				{
					Name: "network.publish_host",
					ValueFrom: &corev1.EnvVarSource{
						FieldRef: &corev1.ObjectFieldSelector{
							APIVersion: "v1",
							FieldPath:  "metadata.name",
						},
					},
				},
				{
					Name:  "node.roles",
					Value: computeRoles(nodeGroup.Roles),
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

		if es.IsTlsApiEnabled() {
			cb.WithEnv([]corev1.EnvVar{
				{
					Name:  "PROBE_SCHEME",
					Value: "https",
				},
			}, k8sbuilder.Merge)
		} else {
			cb.WithEnv([]corev1.EnvVar{
				{
					Name:  "PROBE_SCHEME",
					Value: "http",
				},
			}, k8sbuilder.Merge)
		}
		if nodeGroup.WaitClusterStatus == "" {
			cb.WithEnv([]corev1.EnvVar{
				{
					Name:  "PROBE_WAIT_STATUS",
					Value: "green",
				},
			}, k8sbuilder.Merge)
		} else {
			cb.WithEnv([]corev1.EnvVar{
				{
					Name:  "PROBE_WAIT_STATUS",
					Value: nodeGroup.WaitClusterStatus,
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
		cb.WithImagePullPolicy(es.Spec.ImagePullPolicy, k8sbuilder.OverwriteIfDefaultValue)
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
		additionalVolumeMounts := make([]corev1.VolumeMount, 0, len(es.Spec.GlobalNodeGroup.AdditionalVolumes))
		for _, vol := range es.Spec.GlobalNodeGroup.AdditionalVolumes {
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
					MountPath: "/usr/share/elasticsearch/config",
				},
			}, k8sbuilder.Merge)

		cb.WithVolumeMount([]corev1.VolumeMount{
			{
				Name:      "elasticsearch-data",
				MountPath: "/usr/share/elasticsearch/data",
			},
		}, k8sbuilder.Merge)

		if len(es.Spec.PluginsList) > 0 {
			cb.WithVolumeMount([]corev1.VolumeMount{
				{
					Name:      "plugin",
					MountPath: "/usr/share/elasticsearch/plugins",
				},
			}, k8sbuilder.Merge)
		}

		if es.Spec.GlobalNodeGroup.CacertsSecretRef != nil {
			cb.WithVolumeMount([]corev1.VolumeMount{
				{
					Name:      "cacerts",
					MountPath: "/usr/share/elasticsearch/jdk/lib/security",
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
					Port: intstr.FromInt(9300),
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
				Exec: &corev1.ExecAction{
					Command: []string{
						"/bin/bash",
						"-c",
						`#!/usr/bin/env bash
set -euo pipefail

# Implementation based on Elasticsearch helm template

export NSS_SDB_USE_CACHE=no

STARTER_FILE=/tmp/.es_starter_file
if [ -f ${STARTER_FILE} ]; then
  HTTP_CODE=$(curl --output /dev/null -k -XGET -s -w '%{http_code}' -u elastic:${ELASTIC_PASSWORD} ${PROBE_SCHEME}://127.0.0.1:9200/)
  RC=$?
  if [[ ${RC} -ne 0 ]]; then
    echo "Failed to get Elasticsearch API"
    exit ${RC}
  fi
  if [[ ${HTTP_CODE} == "200" ]]; then
    exit 0
  else
    echo "Elasticsearch API return code ${HTTP_CODE}"
    exit 1
  fi
else
  HTTP_CODE=$(curl --output /dev/null -k -XGET -s -w '%{http_code}' -u elastic:${ELASTIC_PASSWORD} --fail ${PROBE_SCHEME}://127.0.0.1:9200/_cluster/health?wait_for_status=${PROBE_WAIT_STATUS}&timeout=1s)
  RC=$?
  if [[ ${RC} -ne 0 ]]; then
    echo "Failed to get Elasticsearch API"
    exit ${RC}
  fi
  if [[ ${HTTP_CODE} == "200" ]]; then
    touch ${STARTER_FILE}
    exit 0
  else
    echo "Elasticsearch API return code ${HTTP_CODE}"
    exit 1
  fi
fi
`,
					},
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

		// Initialise PodTemplate
		ptb.WithPodTemplateSpec(es.Spec.GlobalNodeGroup.PodTemplate).
			WithPodTemplateSpec(nodeGroup.PodTemplate, k8sbuilder.Merge)

		// Compute labels
		// Do not set global labels here to avoid reconcile pod just because global label change
		ptb.WithLabels(map[string]string{
			"cluster":   es.Name,
			"nodeGroup": nodeGroup.Name,
			elasticsearchcrd.ElasticsearchAnnotationKey: "true",
		}).
			WithLabels(es.Spec.GlobalNodeGroup.Labels, k8sbuilder.Merge).
			WithLabels(nodeGroup.Labels, k8sbuilder.Merge)

		// Compute annotations
		// Do not set global annotation here to avoid reconcile pod just because global annotation change
		ptb.WithAnnotations(map[string]string{
			elasticsearchcrd.ElasticsearchAnnotationKey: "true",
		}).
			WithAnnotations(es.Spec.GlobalNodeGroup.Annotations, k8sbuilder.Merge).
			WithAnnotations(nodeGroup.Annotations, k8sbuilder.Merge).
			WithAnnotations(nodeGroupCheckSumAnnotations, k8sbuilder.Merge)

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
					Privileged: ptr.To[bool](true),
					RunAsUser:  ptr.To[int64](0),
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
					`#!/usr/bin/env bash
set -euo pipefail

elasticsearch-keystore create
for i in /mnt/keystoreSecrets/*; do
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
		if es.Spec.GlobalNodeGroup.CacertsSecretRef != nil {
			tcb := k8sbuilder.NewContainerBuilder().WithContainer(&corev1.Container{
				Name:            "init-cacerts",
				Image:           GetContainerImage(es),
				ImagePullPolicy: es.Spec.ImagePullPolicy,
				SecurityContext: &corev1.SecurityContext{
					RunAsUser: ptr.To[int64](0),
				},
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      "cacerts",
						MountPath: "/mnt/cacerts",
					},
					{
						Name:      "elasticsearch-cacerts",
						MountPath: "/mnt/cacertsSecrets",
					},
				},
				Command: []string{
					"/bin/bash",
					"-c",
					`#!/usr/bin/env bash
set -euo pipefail

for i in /mnt/cacertsSecrets/*; do
    key=$(basename $i)
    echo "Import certificat $i with name $key"
    /usr/share/elasticsearch/jdk/bin/keytool -import -trustcacerts -cacerts -noprompt -alias $key -storepass changeit -file $i
done

cp -a /usr/share/elasticsearch/jdk/lib/security/* /mnt/cacerts/
`,
				},
			})
			tcb.WithResource(es.Spec.GlobalNodeGroup.InitContainerResources)

			ptb.WithInitContainers([]corev1.Container{*tcb.Container()}, k8sbuilder.Merge)
		}
		ccb := k8sbuilder.NewContainerBuilder().WithContainer(&corev1.Container{
			Name:            "init-filesystem",
			Image:           GetContainerImage(es),
			ImagePullPolicy: es.Spec.ImagePullPolicy,
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
					Name:      "node-tls",
					MountPath: "/mnt/certs/node",
				},
				{
					Name:      "api-tls",
					MountPath: "/mnt/certs/api",
				},
				{
					Name:      "keystore",
					MountPath: "/mnt/keystore",
				},
				{
					Name:      "elasticsearch-data",
					MountPath: "/mnt/data",
				},
			},
		})
		var command strings.Builder

		// Inject env and envFrom get proxy conf for exemple
		ccb.WithEnv(globalElasticsearchContainer.Env, k8sbuilder.Merge).
			WithEnv(localElasticsearchContainer.Env, k8sbuilder.Merge)
		ccb.WithEnvFrom(globalElasticsearchContainer.EnvFrom, k8sbuilder.Merge).
			WithEnvFrom(localElasticsearchContainer.EnvFrom, k8sbuilder.Merge)

		command.WriteString(`#!/usr/bin/env bash
set -euo pipefail

# Move original config
echo "Move original elasticsearch configs"
cp -a /usr/share/elasticsearch/config/* /mnt/config/

# Move configmaps
if [ -d /mnt/configmap ]; then
  echo "Move custom configs"
  cp -f /mnt/configmap/* /mnt/config/
fi

# Move certificates
echo "Move cerficates"
mkdir -p /mnt/config/api-cert /mnt/config/transport-cert
cp /mnt/certs/api/* /mnt/config/api-cert/
cp /mnt/certs/node/ca.crt /mnt/config/transport-cert/
cp /mnt/certs/node/${POD_NAME}.crt /mnt/config/transport-cert/
cp /mnt/certs/node/${POD_NAME}.key /mnt/config/transport-cert/

# Move keystore
if [ -f /mnt/keystore/elasticsearch.keystore ]; then
  echo "Move keystore"
  cp /mnt/keystore/elasticsearch.keystore /mnt/config
fi

# Set right
echo "Set right"
chown -R elasticsearch:elasticsearch /mnt/config
chown elasticsearch:elasticsearch /mnt/data
chmod 775 /mnt/data

`)
		for _, plugin := range es.Spec.PluginsList {
			command.WriteString(fmt.Sprintf("./bin/elasticsearch-plugin install -b %s\n", plugin))
		}
		command.WriteString(`
if [ -d /mnt/plugins ]; then
  cp -a /usr/share/elasticsearch/plugins/* /mnt/plugins/
  chown -R elasticsearch:elasticsearch /mnt/plugins
fi
`)
		ccb.Container().Command = []string{
			"/bin/bash",
			"-c",
			command.String(),
		}

		// Compute mount config maps
		for _, configMap := range configMaps {
			if configMap.Name == GetNodeGroupConfigMapName(es, nodeGroup.Name) {
				additionalVolumeMounts = make([]corev1.VolumeMount, 0, len(configMap.Data))
				for key := range configMap.Data {
					additionalVolumeMounts = append(additionalVolumeMounts, corev1.VolumeMount{
						Name:      "elasticsearch-config",
						MountPath: fmt.Sprintf("/mnt/configmap/%s", key),
						SubPath:   key,
					})
				}
				ccb.WithVolumeMount(additionalVolumeMounts, k8sbuilder.Merge)
			}
		}
		if len(es.Spec.PluginsList) > 0 {
			ccb.WithVolumeMount([]corev1.VolumeMount{
				{
					Name:      "plugin",
					MountPath: "/mnt/plugins",
				},
			}, k8sbuilder.Merge)
		}
		ccb.WithResource(es.Spec.GlobalNodeGroup.InitContainerResources)
		ptb.WithInitContainers([]corev1.Container{*ccb.Container()}, k8sbuilder.Merge)

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
			{
				Name: "cacerts",
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
		if GetSecretNameForCacerts(es) != "" {
			ptb.WithVolumes([]corev1.Volume{
				{
					Name: "elasticsearch-cacerts",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: GetSecretNameForCacerts(es),
						},
					},
				},
			}, k8sbuilder.Merge)
		}
		if nodeGroup.IsPersistence() && nodeGroup.Persistence.Volume != nil {
			ptb.WithVolumes([]corev1.Volume{
				{
					Name:         "elasticsearch-data",
					VolumeSource: *nodeGroup.Persistence.Volume,
				},
			}, k8sbuilder.Merge)
		} else if !nodeGroup.IsPersistence() {
			ptb.WithVolumes([]corev1.Volume{
				{
					Name: "elasticsearch-data",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			}, k8sbuilder.Merge)
		}

		// Compute Security context
		ptb.WithSecurityContext(&corev1.PodSecurityContext{
			FSGroup: ptr.To[int64](1000),
		}, k8sbuilder.Merge)

		// Compute pod template name
		ptb.PodTemplate().Name = GetNodeGroupName(es, nodeGroup.Name)

		// Compute image pull secret
		ptb.PodTemplate().Spec.ImagePullSecrets = es.Spec.ImagePullSecrets

		// Compute Statefullset
		sts = &appv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   es.Namespace,
				Name:        GetNodeGroupName(es, nodeGroup.Name),
				Labels:      getLabels(es, map[string]string{"nodeGroup": nodeGroup.Name}, es.Spec.GlobalNodeGroup.Labels, nodeGroup.Labels),
				Annotations: getAnnotations(es, es.Spec.GlobalNodeGroup.Annotations, nodeGroup.Annotations),
			},
			Spec: appv1.StatefulSetSpec{
				Replicas: ptr.To[int32](nodeGroup.Replicas),
				// Start all node to create cluster
				PodManagementPolicy: appv1.ParallelPodManagement,
				ServiceName:         GetNodeGroupServiceNameHeadless(es, nodeGroup.Name),
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"cluster":   es.Name,
						"nodeGroup": nodeGroup.Name,
						elasticsearchcrd.ElasticsearchAnnotationKey: "true",
					},
				},

				Template: *ptb.PodTemplate(),
			},
		}

		// Compute persistence
		if nodeGroup.IsPersistence() && nodeGroup.Persistence.VolumeClaimSpec != nil {
			sts.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "elasticsearch-data",
					},
					Spec: *nodeGroup.Persistence.VolumeClaimSpec,
				},
			}
		}

		statefullsets = append(statefullsets, *sts)
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
func computeEnvFroms(es *elasticsearchcrd.Elasticsearch, nodeGroup *elasticsearchcrd.ElasticsearchNodeGroupSpec) (envFroms []corev1.EnvFromSource) {

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
func computeAntiAffinity(es *elasticsearchcrd.Elasticsearch, nodeGroup *elasticsearchcrd.ElasticsearchNodeGroupSpec) (antiAffinity *corev1.PodAntiAffinity, err error) {
	var expectedAntiAffinity *elasticsearchcrd.ElasticsearchAntiAffinitySpec

	antiAffinity = &corev1.PodAntiAffinity{}
	topologyKey := "kubernetes.io/hostname"

	// Check if need to merge anti affinity spec
	if nodeGroup.AntiAffinity != nil || es.Spec.GlobalNodeGroup.AntiAffinity != nil {
		expectedAntiAffinity = &elasticsearchcrd.ElasticsearchAntiAffinitySpec{}
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
						elasticsearchcrd.ElasticsearchAnnotationKey: "true",
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
						elasticsearchcrd.ElasticsearchAnnotationKey: "true",
					},
				},
			},
		},
	}

	return antiAffinity, nil
}

// computeRoles permit to compute les roles of node groups
func computeRoles(roles []string) string {
	computedRoles := make([]string, 0, len(roles))

	for _, role := range roleList {
		if funk.ContainsString(roles, role) {
			computedRoles = append(computedRoles, role)
		}
	}

	return strings.Join(computedRoles, ", ")
}

// getJavaOpts permit to get computed JAVA_OPTS
func computeJavaOpts(es *elasticsearchcrd.Elasticsearch, nodeGroup *elasticsearchcrd.ElasticsearchNodeGroupSpec) string {
	javaOpts := []string{}

	if es.Spec.GlobalNodeGroup.Jvm != "" {
		javaOpts = append(javaOpts, es.Spec.GlobalNodeGroup.Jvm)
	}

	if nodeGroup.Jvm != "" {
		javaOpts = append(javaOpts, nodeGroup.Jvm)
	}

	return strings.Join(javaOpts, " ")
}
