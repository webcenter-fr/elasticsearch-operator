package cerebro

import (
	"bytes"
	"fmt"

	"emperror.dev/errors"
	"github.com/codingsince1985/checksum"
	"github.com/disaster37/k8sbuilder"
	cerebrocrd "github.com/webcenter-fr/elasticsearch-operator/api/cerebro/v1"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/utils/ptr"
)

// computeChecksum marshals data to JSON and returns its SHA-256 hex digest.
func computeChecksum(name string, data any) (string, error) {
	j, err := json.Marshal(data)
	if err != nil {
		return "", errors.Wrapf(err, "Error when convert data of %s on json string", name)
	}
	sum, err := checksum.SHA256sumReader(bytes.NewReader(j))
	if err != nil {
		return "", errors.Wrapf(err, "Error when generate checksum for %s", name)
	}
	return sum, nil
}

// BuildDeployment permit to generate deployment
func buildDeployments(cerebro *cerebrocrd.Cerebro, secretsChecksum []*corev1.Secret, configMapsChecksum []*corev1.ConfigMap, isOpenshift bool) (dpls []*appv1.Deployment, err error) {
	dpls = make([]*appv1.Deployment, 0, 1)
	checksumAnnotations := map[string]string{}

	// checksum for configmap
	for _, cm := range configMapsChecksum {
		sum, err := computeChecksum(cm.Name, cm.Data)
		if err != nil {
			return nil, err
		}
		checksumAnnotations[fmt.Sprintf("%s/configmap-%s", cerebrocrd.CerebroAnnotationKey, cm.Name)] = sum
	}
	// checksum for secret
	for _, s := range secretsChecksum {
		sum, err := computeChecksum(s.Name, s.Data)
		if err != nil {
			return nil, err
		}
		checksumAnnotations[fmt.Sprintf("%s/secret-%s", cerebrocrd.CerebroAnnotationKey, s.Name)] = sum
	}

	cb := k8sbuilder.NewContainerBuilder()
	ptb := k8sbuilder.NewPodTemplateBuilder()
	cerebroContainer := getContainer(cerebro.Spec.Deployment.PodTemplate)
	if cerebroContainer == nil {
		cerebroContainer = &corev1.Container{}
	}

	// Initialise container from user provided
	cb.WithContainer(cerebroContainer.DeepCopy()).
		Container().Name = "cerebro"
	cb.Container().Args = []string{
		"--config",
		"/opt/cerebro/conf/application.yaml",
	}

	// Compute EnvFrom
	cb.WithEnvFrom(cerebro.Spec.Deployment.EnvFrom, k8sbuilder.Merge)

	// Compute Env
	cb.WithEnv(cerebro.Spec.Deployment.Env, k8sbuilder.Merge).
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
				Name: "CEREBRO_SECRET",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: GetSecretNameForApplication(cerebro),
						},
						Key: "session-key",
					},
				},
			},
		}, k8sbuilder.Merge)

	// Compute ports
	cb.WithPort([]corev1.ContainerPort{
		{
			Name:          "http",
			ContainerPort: 9000,
			Protocol:      corev1.ProtocolTCP,
		},
	}, k8sbuilder.Merge)

	// Compute resources
	cb.WithResource(cerebro.Spec.Deployment.Resources, k8sbuilder.Merge)

	// Compute image
	cb.WithImage(GetContainerImage(cerebro), k8sbuilder.OverwriteIfDefaultValue)

	// Compute image pull policy
	cb.WithImagePullPolicy(cerebro.Spec.ImagePullPolicy, k8sbuilder.OverwriteIfDefaultValue)

	// Compute security context
	sc := &corev1.SecurityContext{
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{"ALL"},
		},
		AllowPrivilegeEscalation: ptr.To(false),
		ReadOnlyRootFilesystem:   ptr.To(true),
		Privileged:               ptr.To(false),
		RunAsNonRoot:             ptr.To(true),
	}
	if !isOpenshift {
		sc.RunAsUser = ptr.To[int64](1000)
		sc.RunAsGroup = ptr.To[int64](1000)
	}
	cb.WithSecurityContext(sc, k8sbuilder.OverwriteIfDefaultValue)

	// Compute volume mount
	cb.WithVolumeMount([]corev1.VolumeMount{
		{
			Name:      "config",
			MountPath: "/opt/cerebro/conf",
		},
		{
			Name:      "data",
			MountPath: "/data",
		},
		{
			Name:      "tmp",
			MountPath: "/tmp",
		},
	}, k8sbuilder.Merge)

	// Compute liveness
	cb.WithLivenessProbe(&corev1.Probe{
		TimeoutSeconds:   5,
		PeriodSeconds:    30,
		FailureThreshold: 3,
		SuccessThreshold: 1,
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path:   "/favicon.ico",
				Port:   intstr.FromInt(9000),
				Scheme: corev1.URISchemeHTTP,
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
				Path:   "/favicon.ico",
				Port:   intstr.FromInt(9000),
				Scheme: corev1.URISchemeHTTP,
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
				Port: intstr.FromInt(9000),
			},
		},
	}, k8sbuilder.OverwriteIfDefaultValue)

	// Initialise PodTemplate
	ptb.WithPodTemplateSpec(cerebro.Spec.Deployment.PodTemplate)

	// Compute labels
	// Do not set global labels here to avoid reconcile pod just because global label change
	ptb.WithLabels(map[string]string{
		"cluster":                       cerebro.Name,
		cerebrocrd.CerebroAnnotationKey: "true",
	}).
		WithLabels(cerebro.Spec.Deployment.Labels, k8sbuilder.Merge)

	// Compute annotations
	// Do not set global annotation here to avoid reconcile pod just because global annotation change
	ptb.WithAnnotations(map[string]string{
		cerebrocrd.CerebroAnnotationKey: "true",
	}).
		WithAnnotations(cerebro.Spec.Deployment.Annotations, k8sbuilder.Merge).
		WithAnnotations(checksumAnnotations, k8sbuilder.Merge)

	// Compute NodeSelector
	ptb.WithNodeSelector(cerebro.Spec.Deployment.NodeSelector, k8sbuilder.Merge)

	// Compute Termination grac period
	ptb.WithTerminationGracePeriodSeconds(30, k8sbuilder.OverwriteIfDefaultValue)

	// Compute toleration
	ptb.WithTolerations(cerebro.Spec.Deployment.Tolerations, k8sbuilder.Merge)

	// Compute containers
	ptb.WithContainers([]corev1.Container{*cb.Container()}, k8sbuilder.Merge)

	// Compute volumes
	ptb.WithVolumes([]corev1.Volume{
		{
			Name: "config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: GetConfigMapName(cerebro),
					},
				},
			},
		},
		{
			Name: "data",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: "tmp",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	}, k8sbuilder.Merge)

	// Compute Security context
	if !isOpenshift {
		ptb.WithSecurityContext(&corev1.PodSecurityContext{
			FSGroup: ptr.To[int64](1000),
		}, k8sbuilder.Merge)
	}

	// Compute pod template name
	ptb.PodTemplate().Name = GetDeploymentName(cerebro)

	// Compute pull secret
	ptb.PodTemplate().Spec.ImagePullSecrets = cerebro.Spec.ImagePullSecrets

	// Compute Deployment
	dpl := &appv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   cerebro.Namespace,
			Name:        GetDeploymentName(cerebro),
			Labels:      getLabels(cerebro, cerebro.Spec.Deployment.Labels),
			Annotations: getAnnotations(cerebro, cerebro.Spec.Deployment.Annotations),
		},
		Spec: appv1.DeploymentSpec{
			Replicas: ptr.To[int32](cerebro.Spec.Deployment.Replicas),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"cluster":                       cerebro.Name,
					cerebrocrd.CerebroAnnotationKey: "true",
				},
			},

			Template: *ptb.PodTemplate(),
		},
	}

	dpls = append(dpls, dpl)

	return dpls, nil
}

// getContainer permit to get container containning from pod template
func getContainer(podTemplate *corev1.PodTemplateSpec) (container *corev1.Container) {
	if podTemplate == nil {
		return nil
	}

	for _, p := range podTemplate.Spec.Containers {
		if p.Name == "cerebro" {
			return &p
		}
	}

	return nil
}
