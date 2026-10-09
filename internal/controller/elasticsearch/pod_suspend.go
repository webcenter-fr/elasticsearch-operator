package elasticsearch

// Pod suspension (downtime mode) inspired by ECK (Elastic Cloud on Kubernetes)
// https://github.com/elastic/cloud-on-k8s
// Adapted for our operator's architecture and Apache 2.0 license.
// No ECK code has been copied directly.

import (
	"context"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// SuspendInitContainerName is the name of the init container that blocks
// suspended pods in Init state.
const SuspendInitContainerName = "suspend-check"

// GetSuspendedPodNames returns the set of pod names to suspend, parsed from
// the elasticsearch.k8s.webcenter.fr/suspend annotation (comma-separated list).
func GetSuspendedPodNames(es *elasticsearchcrd.Elasticsearch) map[string]bool {
	names := map[string]bool{}
	if es.Annotations == nil {
		return names
	}
	if val, ok := es.Annotations[elasticsearchcrd.ElasticsearchSuspendAnnotation]; ok {
		for _, name := range strings.Split(val, ",") {
			name = strings.TrimSpace(name)
			if name != "" {
				names[name] = true
			}
		}
	}
	return names
}

// IsPodSuspended checks if a specific pod should be suspended.
func IsPodSuspended(es *elasticsearchcrd.Elasticsearch, podName string) bool {
	return GetSuspendedPodNames(es)[podName]
}

// ReconcileSuspendedPods handles the pod suspension feature. When a pod is
// listed in the suspend annotation but is still running, it is deleted
// (gracefully) so it restarts and gets blocked by the suspend-check init
// container. When a pod is no longer listed, the init container exits by
// itself once the suspended-pods ConfigMap is updated.
func ReconcileSuspendedPods(ctx context.Context, c client.Client, es *elasticsearchcrd.Elasticsearch, logger *logrus.Entry) error {
	suspendedPods := GetSuspendedPodNames(es)
	if len(suspendedPods) > 0 {
		names := make([]string, 0, len(suspendedPods))
		for name := range suspendedPods {
			names = append(names, name)
		}
		logger.Infof("Pod suspension: %d pod(s) suspended via %s annotation: %s", len(suspendedPods), elasticsearchcrd.ElasticsearchSuspendAnnotation, strings.Join(names, ", "))
	}

	// List all ES pods
	podList := &corev1.PodList{}
	labelSelectors, err := labels.Parse(fmt.Sprintf("cluster=%s,%s=true", es.Name, elasticsearchcrd.ElasticsearchAnnotationKey))
	if err != nil {
		return err
	}
	if err := c.List(ctx, podList, &client.ListOptions{Namespace: es.Namespace, LabelSelector: labelSelectors}); err != nil {
		return err
	}

	for i := range podList.Items {
		pod := &podList.Items[i]
		shouldSuspend := suspendedPods[pod.Name]
		isSuspended := IsPodInSuspendedState(pod)

		switch {
		case shouldSuspend && !isSuspended && IsPodRunning(pod):
			// Pod should be suspended but is running: delete it (gracefully)
			// so it restarts and gets blocked by the suspend-check init container.
			logger.Infof("Suspending pod %s: deleting running pod so it restarts in suspended state", pod.Name)
			if err := c.Delete(ctx, pod); err != nil {
				return fmt.Errorf("failed to delete pod %s for suspension: %w", pod.Name, err)
			}

		case shouldSuspend && isSuspended:
			logger.Infof("Pod %s is suspended (downtime mode) - PVC remains mounted, use 'kubectl exec' to access data", pod.Name)

		case !shouldSuspend && isSuspended:
			// Pod is suspended but should not be: the init container will exit
			// once the suspended-pods ConfigMap is updated.
			logger.Infof("Pod %s is resuming from suspended state", pod.Name)
		}
	}

	return nil
}

// IsPodInSuspendedState checks if the pod is blocked in the suspend-check
// init container (suspended / downtime mode).
func IsPodInSuspendedState(pod *corev1.Pod) bool {
	for _, status := range pod.Status.InitContainerStatuses {
		if status.Name == SuspendInitContainerName && status.State.Running != nil {
			return true
		}
	}
	return false
}

// IsPodRunning checks if the main Elasticsearch container is running.
func IsPodRunning(pod *corev1.Pod) bool {
	for _, status := range pod.Status.ContainerStatuses {
		if status.Name == "elasticsearch" && status.State.Running != nil {
			return true
		}
	}
	return false
}

// IsPodReady checks if the pod is ready.
func IsPodReady(pod *corev1.Pod) bool {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}
