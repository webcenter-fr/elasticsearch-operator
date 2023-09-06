package metricbeat

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/disaster37/k8s-objectmatcher/patch"
	"github.com/disaster37/operator-sdk-extra/pkg/test"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/apis/beat/v1"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
	"github.com/webcenter-fr/elasticsearch-operator/apis/shared"
	elasticsearchcontrollers "github.com/webcenter-fr/elasticsearch-operator/controllers/elasticsearch"
	localhelper "github.com/webcenter-fr/elasticsearch-operator/pkg/helper"
	localtest "github.com/webcenter-fr/elasticsearch-operator/pkg/test"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	condition "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (t *MetricbeatControllerTestSuite) TestMetricbeatController() {
	key := types.NamespacedName{
		Name:      "t-mb-" + localhelper.RandomString(10),
		Namespace: "default",
	}
	fb := &beatcrd.Metricbeat{}
	data := map[string]any{}

	testCase := test.NewTestCase(t.T(), t.k8sClient, key, fb, 5*time.Second, data)
	testCase.Steps = []test.TestStep{
		doCreateMetricbeatStep(),
		doUpdateMetricbeatStep(),
		doDeleteMetricbeatStep(),
	}

	testCase.Run()
}

func doCreateMetricbeatStep() test.TestStep {
	return test.TestStep{
		Name: "create",
		Do: func(c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			logrus.Infof("=== Add new Metricbeat %s/%s ===", key.Namespace, key.Name)

			// First, create Elasticsearch
			es := &elasticsearchcrd.Elasticsearch{
				ObjectMeta: metav1.ObjectMeta{
					Name:      key.Name,
					Namespace: key.Namespace,
				},
				Spec: elasticsearchcrd.ElasticsearchSpec{
					Version: "8.6.0",
					NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
						{
							Name:     "all",
							Replicas: 1,
							Roles: []string{
								"master",
								"client",
								"data",
							},
						},
					},
				},
			}

			if err = c.Create(context.Background(), es); err != nil {
				return err
			}

			isTimeout, err := localtest.RunWithTimeout(func() error {
				if err := c.Get(context.Background(), key, es); err != nil {
					return err
				}

				// In envtest, no kubelet
				// So the Elasticsearch condition never set as true
				if condition.FindStatusCondition(es.Status.Conditions, elasticsearchcontrollers.ElasticsearchCondition.String()) != nil && condition.FindStatusCondition(es.Status.Conditions, elasticsearchcontrollers.ElasticsearchCondition.String()).Reason != "Initialize" {
					return nil
				}

				return errors.New("Not yet created")

			}, time.Second*30, time.Second*1)

			if err != nil || isTimeout {
				return err
			}

			mb := &beatcrd.Metricbeat{
				ObjectMeta: metav1.ObjectMeta{
					Name:      key.Name,
					Namespace: key.Namespace,
				},
				Spec: beatcrd.MetricbeatSpec{
					Version: "8.6.0",
					ElasticsearchRef: shared.ElasticsearchRef{
						ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
							Name: es.Name,
						},
					},
					Deployment: beatcrd.MetricbeatDeploymentSpec{
						Replicas: 2,
					},
					Config: map[string]string{
						"metricbeat.yml": `
pipeline.workers: 2
queue.type: persisted
`,
					},
					Module: map[string]string{
						"test.conf": "test",
					},
				},
			}

			if err = c.Create(context.Background(), mb); err != nil {
				return err
			}

			return nil
		},
		Check: func(t *testing.T, c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			mb := &beatcrd.Metricbeat{}
			var (
				s   *corev1.Secret
				svc *corev1.Service
				cm  *corev1.ConfigMap
				pdb *policyv1.PodDisruptionBudget
				sts *appv1.StatefulSet
			)

			isTimeout, err := localtest.RunWithTimeout(func() error {
				if err := c.Get(context.Background(), key, mb); err != nil {
					t.Fatal("Metricbeat not found")
				}

				// In envtest, no kubelet
				// So the Metricbeat condition never set as true
				if condition.FindStatusCondition(mb.Status.Conditions, MetricbeatCondition.String()) != nil && condition.FindStatusCondition(mb.Status.Conditions, MetricbeatCondition.String()).Reason != "Initialize" {
					return nil
				}

				return errors.New("Not yet created")

			}, time.Second*30, time.Second*1)
			if err != nil || isTimeout {
				t.Fatalf("All Metricbeat step provisionning not finished: %s", err.Error())
			}

			// Secrets for CA Elasticsearch
			s = &corev1.Secret{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetSecretNameForCAElasticsearch(mb)}, s); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, s.Data)
			assert.NotEmpty(t, s.OwnerReferences)
			assert.NotEmpty(t, s.Annotations[patch.LastAppliedConfig])

			// Secrets for credentials must exist
			s = &corev1.Secret{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetSecretNameForCredentials(mb)}, s); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, s.Data)
			assert.NotEmpty(t, s.OwnerReferences)
			assert.NotEmpty(t, s.Annotations[patch.LastAppliedConfig])

			// Global service must exist
			svc = &corev1.Service{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetGlobalServiceName(mb)}, svc); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, svc.OwnerReferences)
			assert.NotEmpty(t, svc.Annotations[patch.LastAppliedConfig])

			// ConfigMaps must exist
			cm = &corev1.ConfigMap{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetConfigMapConfigName(mb)}, cm); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, cm.OwnerReferences)
			assert.NotEmpty(t, cm.Annotations[patch.LastAppliedConfig])

			cm = &corev1.ConfigMap{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetConfigMapModuleName(mb)}, cm); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, cm.OwnerReferences)
			assert.NotEmpty(t, cm.Annotations[patch.LastAppliedConfig])

			// PDB must exist
			pdb = &policyv1.PodDisruptionBudget{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetPDBName(mb)}, pdb); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, pdb.OwnerReferences)
			assert.NotEmpty(t, pdb.Annotations[patch.LastAppliedConfig])

			// Statefulset musts exist
			sts = &appv1.StatefulSet{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetStatefulsetName(mb)}, sts); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, sts.OwnerReferences)
			assert.NotEmpty(t, sts.Annotations[patch.LastAppliedConfig])

			// Status must be update
			assert.NotEmpty(t, mb.Status.Phase)
			assert.False(t, *mb.Status.IsError)

			return nil
		},
	}
}

func doUpdateMetricbeatStep() test.TestStep {
	return test.TestStep{
		Name: "update",
		Do: func(c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			logrus.Infof("=== Update Metricbeat cluster %s/%s ===", key.Namespace, key.Name)

			if o == nil {
				return errors.New("Metricbeat is null")
			}
			mb := o.(*beatcrd.Metricbeat)

			// Add labels must force to update all resources
			mb.Labels = map[string]string{
				"test": "fu",
			}

			data["lastVersion"] = mb.ResourceVersion

			if err = c.Update(context.Background(), mb); err != nil {
				return err
			}

			time.Sleep(5 * time.Second)

			return nil
		},
		Check: func(t *testing.T, c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			mb := &beatcrd.Metricbeat{}

			var (
				s   *corev1.Secret
				svc *corev1.Service
				cm  *corev1.ConfigMap
				pdb *policyv1.PodDisruptionBudget
				sts *appv1.StatefulSet
			)

			lastVersion := data["lastVersion"].(string)

			isTimeout, err := localtest.RunWithTimeout(func() error {
				if err := c.Get(context.Background(), key, mb); err != nil {
					t.Fatal("Metricbeat not found")
				}

				// In envtest, no kubelet
				// So the Metricbeat condition never set as true
				if lastVersion != mb.ResourceVersion && (mb.Status.Phase == MetricbeatPhaseStarting.String()) {
					return nil
				}

				return errors.New("Not yet updated")

			}, time.Second*30, time.Second*1)
			if err != nil || isTimeout {
				t.Fatalf("All Metricbeat step upgrading not finished: %s", err.Error())
			}

			// Secrets for CA Elasticsearch
			s = &corev1.Secret{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetSecretNameForCAElasticsearch(mb)}, s); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, s.Data)
			assert.NotEmpty(t, s.OwnerReferences)
			assert.NotEmpty(t, s.Annotations[patch.LastAppliedConfig])
			assert.Equal(t, "fu", s.Labels["test"])

			// Secrets for credentials must exist
			s = &corev1.Secret{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetSecretNameForCredentials(mb)}, s); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, s.Data)
			assert.NotEmpty(t, s.OwnerReferences)
			assert.NotEmpty(t, s.Annotations[patch.LastAppliedConfig])
			assert.Equal(t, "fu", s.Labels["test"])

			// Global service must exist
			svc = &corev1.Service{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetGlobalServiceName(mb)}, svc); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, svc.OwnerReferences)
			assert.NotEmpty(t, svc.Annotations[patch.LastAppliedConfig])
			assert.Equal(t, "fu", svc.Labels["test"])

			// ConfigMaps must exist
			cm = &corev1.ConfigMap{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetConfigMapConfigName(mb)}, cm); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, cm.OwnerReferences)
			assert.NotEmpty(t, cm.Annotations[patch.LastAppliedConfig])
			assert.Equal(t, "fu", cm.Labels["test"])

			cm = &corev1.ConfigMap{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetConfigMapModuleName(mb)}, cm); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, cm.OwnerReferences)
			assert.NotEmpty(t, cm.Annotations[patch.LastAppliedConfig])
			assert.Equal(t, "fu", cm.Labels["test"])

			// PDB must exist
			pdb = &policyv1.PodDisruptionBudget{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetPDBName(mb)}, pdb); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, pdb.OwnerReferences)
			assert.NotEmpty(t, pdb.Annotations[patch.LastAppliedConfig])
			assert.Equal(t, "fu", pdb.Labels["test"])

			// Statefulset musts exist
			sts = &appv1.StatefulSet{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetStatefulsetName(mb)}, sts); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, sts.OwnerReferences)
			assert.NotEmpty(t, sts.Annotations[patch.LastAppliedConfig])
			assert.Equal(t, "fu", sts.Labels["test"])

			// Status must be update
			assert.NotEmpty(t, mb.Status.Phase)
			assert.False(t, *mb.Status.IsError)

			return nil
		},
	}
}

func doDeleteMetricbeatStep() test.TestStep {
	return test.TestStep{
		Name: "delete",
		Do: func(c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			logrus.Infof("=== Delete Metricbeat cluster %s/%s ===", key.Namespace, key.Name)

			if o == nil {
				return errors.New("Metricbeat is null")
			}
			mb := o.(*beatcrd.Metricbeat)

			wait := int64(0)
			if err = c.Delete(context.Background(), mb, &client.DeleteOptions{GracePeriodSeconds: &wait}); err != nil {
				return err
			}

			return nil
		},
		Check: func(t *testing.T, c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {

			// In envtest, no kubelet
			// So the cascading children delation not works

			return nil
		},
	}
}
