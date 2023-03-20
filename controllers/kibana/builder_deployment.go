package kibana

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/codingsince1985/checksum"
	"github.com/davecgh/go-spew/spew"
	"github.com/disaster37/k8sbuilder"
	"github.com/elastic/go-ucfg"
	"github.com/pkg/errors"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1alpha1"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/apis/kibana/v1alpha1"
	"github.com/webcenter-fr/elasticsearch-operator/controllers/elasticsearch"
	"github.com/webcenter-fr/elasticsearch-operator/pkg/helper"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/utils/pointer"
)

// BuildDeployment permit to generate deployment for Kibana
func BuildDeployment(kb *kibanacrd.Kibana, es *elasticsearchcrd.Elasticsearch, secretsChecksum []corev1.Secret, configMapsChecksum []corev1.ConfigMap) (dpl *appv1.Deployment, err error) {

	checksumAnnotations := map[string]string{}

	// Inject plugin for exporter Prometheus if needed
	if kb.IsPrometheusMonitoring() {
		kb.Spec.PluginsList = append(kb.Spec.PluginsList, GetExporterUrl(kb))
	}

	// Generate confimaps to know what file to mount
	// And to generate checksum
	configMap, err := BuildConfigMap(kb, es)
	if err != nil {
		return nil, errors.Wrap(err, "Error when generate configMap")
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
		checksumAnnotations[fmt.Sprintf("%s/configmap-%s", kibanacrd.KibanaAnnotationKey, cm.Name)] = sum
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
		checksumAnnotations[fmt.Sprintf("%s/secret-%s", kibanacrd.KibanaAnnotationKey, s.Name)] = sum
	}

	cb := k8sbuilder.NewContainerBuilder()
	ptb := k8sbuilder.NewPodTemplateBuilder()
	kibanaContainer := getKibanaContainer(kb.Spec.Deployment.PodTemplate)
	if kibanaContainer == nil {
		kibanaContainer = &corev1.Container{}
	}

	// Initialise Kibana container from user provided
	cb.WithContainer(kibanaContainer.DeepCopy()).
		Container().Name = "kibana"

	// Compute EnvFrom
	cb.WithEnvFrom(kb.Spec.Deployment.EnvFrom, k8sbuilder.Merge)

	// Compute Env
	probePath, err := computeProbePath(configMap)
	if err != nil {
		return nil, errors.Wrap(err, "Error when get probe path to use from kibana config")
	}
	cb.WithEnv(kb.Spec.Deployment.Env, k8sbuilder.Merge).
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
				Name:  "NODE_OPTIONS",
				Value: computeNodeOpts(kb),
			},
			{
				Name:  "SERVER_HOST",
				Value: "0.0.0.0",
			},
			{
				Name:  "SERVER_NAME",
				Value: kb.Name,
			},
			{
				Name:  "PROBE_PATH",
				Value: probePath,
			},
		}, k8sbuilder.Merge)
	if kb.IsTlsEnabled() {
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
	if kb.Spec.ElasticsearchRef.IsManaged() {
		cb.WithEnv([]corev1.EnvVar{
			{
				Name:  "ELASTICSEARCH_HOSTS",
				Value: computeElasticsearchHosts(kb, es),
			},
			{
				Name:  "ELASTICSEARCH_USERNAME",
				Value: "kibana_system",
			},
			{
				Name: "ELASTICSEARCH_PASSWORD",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: GetSecretNameForCredentials(kb),
						},
						Key: "kibana_system",
					},
				},
			},
		}, k8sbuilder.Merge)
	} else {
		cb.WithEnv([]corev1.EnvVar{
			{
				Name:  "ELASTICSEARCH_HOSTS",
				Value: computeElasticsearchHosts(kb, es),
			},
			{
				Name: "ELASTICSEARCH_USERNAME",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: kb.Spec.ElasticsearchRef.ExternalElasticsearchRef.SecretRef.Name,
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
							Name: kb.Spec.ElasticsearchRef.ExternalElasticsearchRef.SecretRef.Name,
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
			ContainerPort: 5601,
			Protocol:      corev1.ProtocolTCP,
		},
	}, k8sbuilder.Merge)

	// Compute resources
	cb.WithResource(kb.Spec.Deployment.Resources, k8sbuilder.Merge)

	// Compute image
	cb.WithImage(GetContainerImage(kb), k8sbuilder.OverwriteIfDefaultValue)

	// Compute image pull policy
	cb.WithImagePullPolicy(kb.Spec.ImagePullPolicy, k8sbuilder.OverwriteIfDefaultValue)

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
	cb.WithVolumeMount([]corev1.VolumeMount{
		{
			Name:      "config",
			MountPath: "/usr/share/kibana/config",
		},
	}, k8sbuilder.Merge)
	if len(kb.Spec.PluginsList) > 0 {
		cb.WithVolumeMount([]corev1.VolumeMount{
			{
				Name:      "plugin",
				MountPath: "/usr/share/kibana/plugins",
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
				Port: intstr.FromInt(5601),
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

# Implementation based on Kibana helm template

export NSS_SDB_USE_CACHE=no

HTTP_CODE=$(curl --output /dev/null -k -XGET -s --fail -L -w '%{http_code}' ${PROBE_SCHEME}://127.0.0.1:5601${PROBE_PATH})
RC=$?
if [[ ${RC} -ne 0 ]]; then
  echo "Failed to get Kibana"
  exit ${RC}
fi
if [[ ${HTTP_CODE} == "200" ]]; then
  exit 0
else
  echo "Kibana return code ${HTTP_CODE}"
  exit 1
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
				Port: intstr.FromInt(5601),
			},
		},
	}, k8sbuilder.OverwriteIfDefaultValue)

	// Initialise PodTemplate
	ptb.WithPodTemplateSpec(kb.Spec.Deployment.PodTemplate)

	// Compute labels
	// Do not set global labels here to avoid reconcile pod just because global label change
	ptb.WithLabels(map[string]string{
		"cluster":                     kb.Name,
		kibanacrd.KibanaAnnotationKey: "true",
	}).
		WithLabels(kb.Spec.Deployment.Labels, k8sbuilder.Merge)

	// Compute annotations
	// Do not set global annotation here to avoid reconcile pod just because global annotation change
	ptb.WithAnnotations(map[string]string{
		kibanacrd.KibanaAnnotationKey: "true",
	}).
		WithAnnotations(kb.Spec.Deployment.Annotations, k8sbuilder.Merge).
		WithAnnotations(checksumAnnotations, k8sbuilder.Merge)

	// Compute NodeSelector
	ptb.WithNodeSelector(kb.Spec.Deployment.NodeSelector, k8sbuilder.Merge)

	// Compute Termination grac period
	ptb.WithTerminationGracePeriodSeconds(30, k8sbuilder.OverwriteIfDefaultValue)

	// Compute toleration
	ptb.WithTolerations(kb.Spec.Deployment.Tolerations, k8sbuilder.Merge)

	// compute anti affinity
	antiAffinity, err := computeAntiAffinity(kb)
	if err != nil {
		return nil, errors.Wrap(err, "Error when compute anti affinity")
	}
	ptb.WithAffinity(corev1.Affinity{
		PodAntiAffinity: antiAffinity,
	}, k8sbuilder.Merge)

	// Compute containers
	ptb.WithContainers([]corev1.Container{*cb.Container()}, k8sbuilder.Merge)

	// Compute init containers
	if kb.Spec.KeystoreSecretRef != nil {
		kcb := k8sbuilder.NewContainerBuilder().WithContainer(&corev1.Container{
			Name:            "init-keystore",
			Image:           GetContainerImage(kb),
			ImagePullPolicy: kb.Spec.ImagePullPolicy,
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "keystore",
					MountPath: "/mnt/keystore",
				},
				{
					Name:      "kibana-keystore",
					MountPath: "/mnt/keystoreSecrets",
				},
			},
			Command: []string{
				"/bin/bash",
				"-c",
				`#!/usr/bin/env bash
set -euo pipefail

kibana-keystore create
for i in /mnt/keystoreSecrets/*; do
    key=$(basename $i)
    echo "Adding file $i to keystore key $key"
    kibana-keystore add -x "$key" < $i
done


cp -a /usr/share/kibana/config/kibana.keystore /mnt/keystore/
`,
			},
		})
		kcb.WithResource(kb.Spec.Deployment.InitContainerResources)

		ptb.WithInitContainers([]corev1.Container{*kcb.Container()}, k8sbuilder.Merge)
	}
	ccb := k8sbuilder.NewContainerBuilder().WithContainer(&corev1.Container{
		Name:            "init-filesystem",
		Image:           GetContainerImage(kb),
		ImagePullPolicy: kb.Spec.ImagePullPolicy,
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
				Name:      "config",
				MountPath: "/mnt/config",
			},
			{
				Name:      "tls",
				MountPath: "/mnt/certs",
			},
			{
				Name:      "keystore",
				MountPath: "/mnt/keystore",
			},
		},
	})

	// Inject env / envFrom to get proxy for exemple
	ccb.WithEnv(kb.Spec.Deployment.Env, k8sbuilder.Merge)
	ccb.WithEnvFrom(kb.Spec.Deployment.EnvFrom, k8sbuilder.Merge)

	var command strings.Builder
	command.WriteString(`#!/usr/bin/env bash
set -euo pipefail

# Move original config
echo "Move original kibana configs"
cp -a /usr/share/kibana/config/* /mnt/config/

# Move configmaps
if [ -d /mnt/configmap ]; then
  echo "Move custom configs"
  cp -f /mnt/configmap/* /mnt/config/
fi

# Move certificates
if [ -d /mnt/certs ]; then
  echo "Move cerficates"
  mkdir -p /mnt/config/api-cert
  cp /mnt/certs/* /mnt/config/api-cert/
fi

# Move CA Elasticsearch
if [ -d /mnt/ca-elasticsearch ]; then
  echo "Move CA certificate"
  mkdir -p /mnt/config/es-ca
  cp /mnt/ca-elasticsearch/* /mnt/config/es-ca/
fi

# Move keystore
if [ -f /mnt/keystore/kibana.keystore ]; then
  echo "Move keystore"
  cp /mnt/keystore/kibana.keystore /mnt/config
fi

# Set right
echo "Set right"
chown -R kibana:kibana /mnt/config

`)
	for _, plugin := range kb.Spec.PluginsList {
		command.WriteString(fmt.Sprintf("./bin/kibana-plugin install %s\n", plugin))
	}
	command.WriteString(`
if [ -d /mnt/plugins ]; then
  cp -a /usr/share/kibana/plugins/* /mnt/plugins/
  chown -R kibana:kibana /mnt/plugins
fi
`)
	ccb.Container().Command = []string{
		"/bin/bash",
		"-c",
		command.String(),
	}

	// Compute mount config map
	if configMap != nil {
		ccb.WithVolumeMount([]corev1.VolumeMount{
			{
				Name:      "kibana-config",
				MountPath: "/mnt/configmap",
			},
		}, k8sbuilder.Merge)
	}

	if kb.IsTlsEnabled() {
		ccb.WithVolumeMount([]corev1.VolumeMount{
			{
				Name:      "tls",
				MountPath: "/mnt/certs",
			},
		}, k8sbuilder.Merge)
	}

	if (kb.Spec.ElasticsearchRef.IsManaged() && es.IsTlsApiEnabled()) || (kb.Spec.ElasticsearchRef.IsExternal() && kb.Spec.ElasticsearchRef.ElasticsearchCaSecretRef != nil) {
		ccb.WithVolumeMount([]corev1.VolumeMount{
			{
				Name:      "ca-elasticsearch",
				MountPath: "/mnt/ca-elasticsearch",
			},
		}, k8sbuilder.Merge)
	}

	if len(kb.Spec.PluginsList) > 0 {
		ccb.WithVolumeMount([]corev1.VolumeMount{
			{
				Name:      "plugin",
				MountPath: "/mnt/plugins",
			},
		}, k8sbuilder.Merge)
	}
	ccb.WithResource(kb.Spec.Deployment.InitContainerResources)
	ptb.WithInitContainers([]corev1.Container{*ccb.Container()}, k8sbuilder.Merge)

	// Compute volumes
	ptb.WithVolumes([]corev1.Volume{
		{
			Name: "kibana-config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: GetConfigMapName(kb),
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

	if GetSecretNameForKeystore(kb) != "" {
		ptb.WithVolumes([]corev1.Volume{
			{
				Name: "kibana-keystore",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: GetSecretNameForKeystore(kb),
					},
				},
			},
		}, k8sbuilder.Merge)
	}
	if kb.IsTlsEnabled() {
		ptb.WithVolumes([]corev1.Volume{
			{
				Name: "tls",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: GetSecretNameForTls(kb),
					},
				},
			},
		}, k8sbuilder.Merge)
	}
	if kb.Spec.ElasticsearchRef.IsManaged() && es.IsTlsApiEnabled() && es.IsSelfManagedSecretForTlsApi() {
		ptb.WithVolumes([]corev1.Volume{
			{
				Name: "ca-elasticsearch",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: GetSecretNameForCAElasticsearch(kb),
					},
				},
			},
		}, k8sbuilder.Merge)
	} else if (kb.Spec.ElasticsearchRef.IsExternal() && kb.Spec.ElasticsearchRef.ElasticsearchCaSecretRef != nil) || (kb.Spec.ElasticsearchRef.IsManaged() && es.IsTlsApiEnabled() && kb.Spec.ElasticsearchRef.ElasticsearchCaSecretRef != nil) {
		ptb.WithVolumes([]corev1.Volume{
			{
				Name: "ca-elasticsearch",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: kb.Spec.ElasticsearchRef.ElasticsearchCaSecretRef.Name,
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
	ptb.PodTemplate().Name = GetDeploymentName(kb)

	// Compute pull secret
	ptb.PodTemplate().Spec.ImagePullSecrets = kb.Spec.ImagePullSecrets

	// Compute Deployment
	dpl = &appv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   kb.Namespace,
			Name:        GetDeploymentName(kb),
			Labels:      getLabels(kb, kb.Spec.Deployment.Labels),
			Annotations: getAnnotations(kb, kb.Spec.Deployment.Annotations),
		},
		Spec: appv1.DeploymentSpec{
			Replicas: pointer.Int32(kb.Spec.Deployment.Replicas),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"cluster":                     kb.Name,
					kibanacrd.KibanaAnnotationKey: "true",
				},
			},

			Template: *ptb.PodTemplate(),
		},
	}

	return dpl, nil
}

// getKibanaContainer permit to get Kibana container containning from pod template
func getKibanaContainer(podTemplate *corev1.PodTemplateSpec) (container *corev1.Container) {
	if podTemplate == nil {
		return nil
	}

	for _, p := range podTemplate.Spec.Containers {
		if p.Name == "kibana" {
			return &p
		}
	}

	return nil
}

// computeElasticsearchHosts permit to get the target Elasticsearch cluster to connect on
func computeElasticsearchHosts(kb *kibanacrd.Kibana, es *elasticsearchcrd.Elasticsearch) string {

	if kb.Spec.ElasticsearchRef.IsManaged() {
		scheme := "https"
		if !es.IsTlsApiEnabled() {
			scheme = "http"
		}
		if kb.Spec.ElasticsearchRef.ManagedElasticsearchRef.TargetNodeGroup == "" {
			return fmt.Sprintf("%s://%s.%s.svc:9200", scheme, elasticsearch.GetGlobalServiceName(es), es.Namespace)
		} else {
			return fmt.Sprintf("%s://%s.%s.svc:9200", scheme, elasticsearch.GetNodeGroupServiceName(es, kb.Spec.ElasticsearchRef.ManagedElasticsearchRef.TargetNodeGroup), es.Namespace)
		}
	}

	return fmt.Sprintf("[%s]", strings.Join(kb.Spec.ElasticsearchRef.ExternalElasticsearchRef.Addresses, ","))

}

// computeAntiAffinity permit to get  anti affinity spec
// Default to soft anti affinity
func computeAntiAffinity(kb *kibanacrd.Kibana) (antiAffinity *corev1.PodAntiAffinity, err error) {
	var expectedAntiAffinity *kibanacrd.KibanaAntiAffinitySpec

	antiAffinity = &corev1.PodAntiAffinity{}
	topologyKey := "kubernetes.io/hostname"

	// Check if need to merge anti affinity spec
	if kb.Spec.Deployment.AntiAffinity != nil {
		expectedAntiAffinity = &kibanacrd.KibanaAntiAffinitySpec{}
		if err = helper.Merge(expectedAntiAffinity, kb.Spec.Deployment.AntiAffinity); err != nil {
			return nil, errors.Wrap(err, "Error when merge global anti affinity")
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
						"cluster":                     kb.Name,
						kibanacrd.KibanaAnnotationKey: "true",
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
						"cluster":                     kb.Name,
						kibanacrd.KibanaAnnotationKey: "true",
					},
				},
			},
		},
	}

	return antiAffinity, nil
}

// computeNodeOpts permit to get computed NODE_OPTIONS
func computeNodeOpts(kb *kibanacrd.Kibana) string {
	nodeOpts := []string{}

	if kb.Spec.Deployment.Node != "" {
		nodeOpts = append(nodeOpts, kb.Spec.Deployment.Node)
	}

	return strings.Join(nodeOpts, " ")
}

// computeProbePath permit to compute the probe path to use on readynessProbe
func computeProbePath(cm *corev1.ConfigMap) (path string, err error) {
	if cm == nil || cm.Data["kibana.yml"] == "" {
		return "/app/kibana", nil
	}

	path, err = helper.GetSetting("server.basePath", []byte(cm.Data["kibana.yml"]))
	if ucfg.ErrMissing == err {
		return "/app/kibana", nil
	}

	spew.Dump(cm.Data["kibana.yml"])

	return path, nil

}
