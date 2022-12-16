package elasticsearch

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/disaster37/k8s-objectmatcher/patch"
	"github.com/disaster37/operator-sdk-extra/pkg/test"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	elasticsearchapi "github.com/webcenter-fr/elasticsearch-operator/api/v1alpha1"
	localhelper "github.com/webcenter-fr/elasticsearch-operator/pkg/helper"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	condition "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (t *ControllerTestSuite) TestElasticsearchController() {
	key := types.NamespacedName{
		Name:      "t-csg-" + localhelper.RandomString(10),
		Namespace: "default",
	}
	es := &elasticsearchapi.Elasticsearch{}
	data := map[string]any{}

	testCase := test.NewTestCase(t.T(), t.k8sClient, key, es, 5*time.Second, data)
	testCase.Steps = []test.TestStep{
		doCreateElasticsearchStep(),
		doUpdateElasticsearchStep(),
		doDeleteElasticsearchStep(),
	}

	testCase.Run()
}

func doCreateElasticsearchStep() test.TestStep {
	return test.TestStep{
		Name: "create",
		Do: func(c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			logrus.Infof("=== Add new Elasticsearch cluster %s/%s ===", key.Namespace, key.Name)

			es := &elasticsearchapi.Elasticsearch{
				ObjectMeta: metav1.ObjectMeta{
					Name:      key.Name,
					Namespace: key.Namespace,
				},
				Spec: elasticsearchapi.ElasticsearchSpec{
					Endpoint: elasticsearchapi.EndpointSpec{
						Ingress: &elasticsearchapi.IngressSpec{
							Enabled: true,
							Host:    "test.cluster.local",
							SecretRef: &corev1.LocalObjectReference{
								Name: "test-tls",
							},
						},
						LoadBalancer: &elasticsearchapi.LoadBalancerSpec{
							Enabled: true,
						},
					},
					NodeGroups: []elasticsearchapi.NodeGroupSpec{
						{
							Name:     "single",
							Replicas: 1,
							Roles: []string{
								"master",
								"data",
								"ingest",
							},
							Resources: &corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("300m"),
									corev1.ResourceMemory: resource.MustParse("1Gi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("1000m"),
									corev1.ResourceMemory: resource.MustParse("1Gi"),
								},
							},
						},
					},
				},
			}

			if err = c.Create(context.Background(), es); err != nil {
				return err
			}

			return nil
		},
		Check: func(t *testing.T, c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			es := &elasticsearchapi.Elasticsearch{}
			var (
				s   *corev1.Secret
				svc *corev1.Service
				i   *networkingv1.Ingress
				cm  *corev1.ConfigMap
				pdb *policyv1.PodDisruptionBudget
				sts *appv1.StatefulSet
			)

			isTimeout, err := RunWithTimeout(func() error {
				if err := c.Get(context.Background(), key, es); err != nil {
					t.Fatal("Elasticsearch not found")
				}

				// In envtest, no kubelet
				// So the Elasticsearch condition never set as true
				if condition.FindStatusCondition(es.Status.Conditions, ElasticsearchCondition) != nil && condition.FindStatusCondition(es.Status.Conditions, ElasticsearchCondition).Reason != "Initialize" {
					return nil
				}

				return errors.New("Not yet created")

			}, time.Second*30, time.Second*1)
			if err != nil || isTimeout {
				t.Fatalf("All Elasticsearch step provisionning not finished: %s", err.Error())
			}

			// Secrets for node PKI and certificates must exist
			s = &corev1.Secret{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetSecretNameForPkiTransport(es)}, s); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, s.Data)
			assert.NotEmpty(t, s.OwnerReferences)
			assert.NotEmpty(t, s.Annotations[patch.LastAppliedConfig])

			s = &corev1.Secret{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetSecretNameForTlsTransport(es)}, s); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, s.Data)
			assert.NotEmpty(t, s.OwnerReferences)
			assert.NotEmpty(t, s.Annotations[patch.LastAppliedConfig])

			// Secrets for API PKI and certificates must exist
			s = &corev1.Secret{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetSecretNameForPkiApi(es)}, s); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, s.Data)
			assert.NotEmpty(t, s.OwnerReferences)
			assert.NotEmpty(t, s.Annotations[patch.LastAppliedConfig])

			s = &corev1.Secret{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetSecretNameForTlsApi(es)}, s); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, s.Data)
			assert.NotEmpty(t, s.OwnerReferences)
			assert.NotEmpty(t, s.Annotations[patch.LastAppliedConfig])

			// Secrets for internal credentials must exist
			s = &corev1.Secret{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetSecretNameForCredentials(es)}, s); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, s.Data)
			assert.NotEmpty(t, s.OwnerReferences)
			assert.NotEmpty(t, s.Annotations[patch.LastAppliedConfig])

			// Services must exists
			svc = &corev1.Service{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetGlobalServiceName(es)}, svc); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, svc.OwnerReferences)
			assert.NotEmpty(t, svc.Annotations[patch.LastAppliedConfig])

			for _, nodeGroup := range es.Spec.NodeGroups {
				svc = &corev1.Service{}
				if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetNodeGroupServiceName(es, nodeGroup.Name)}, svc); err != nil {
					t.Fatal(err)
				}
				assert.NotEmpty(t, svc.OwnerReferences)
				assert.NotEmpty(t, svc.Annotations[patch.LastAppliedConfig])

				svc = &corev1.Service{}
				if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetNodeGroupServiceNameHeadless(es, nodeGroup.Name)}, svc); err != nil {
					t.Fatal(err)
				}
				assert.NotEmpty(t, svc.OwnerReferences)
				assert.NotEmpty(t, svc.Annotations[patch.LastAppliedConfig])
			}

			// Load balancer must exist
			svc = &corev1.Service{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetLoadBalancerName(es)}, svc); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, svc.OwnerReferences)
			assert.NotEmpty(t, svc.Annotations[patch.LastAppliedConfig])

			// Ingress must exist
			i = &networkingv1.Ingress{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetIngressName(es)}, i); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, i.OwnerReferences)
			assert.NotEmpty(t, i.Annotations[patch.LastAppliedConfig])

			// ConfigMaps must exist
			for _, nodeGroup := range es.Spec.NodeGroups {
				cm = &corev1.ConfigMap{}
				if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetNodeGroupConfigMapName(es, nodeGroup.Name)}, cm); err != nil {
					t.Fatal(err)
				}
				assert.NotEmpty(t, cm.OwnerReferences)
				assert.NotEmpty(t, cm.Annotations[patch.LastAppliedConfig])
			}

			// PDB must exist
			for _, nodeGroup := range es.Spec.NodeGroups {
				pdb = &policyv1.PodDisruptionBudget{}
				if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetNodeGroupPDBName(es, nodeGroup.Name)}, pdb); err != nil {
					t.Fatal(err)
				}
				assert.NotEmpty(t, pdb.OwnerReferences)
				assert.NotEmpty(t, pdb.Annotations[patch.LastAppliedConfig])
			}

			// Statefulset musts exist
			for _, nodeGroup := range es.Spec.NodeGroups {
				sts = &appv1.StatefulSet{}
				if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetNodeGroupName(es, nodeGroup.Name)}, sts); err != nil {
					t.Fatal(err)
				}
				assert.NotEmpty(t, sts.OwnerReferences)
				assert.NotEmpty(t, sts.Annotations[patch.LastAppliedConfig])
			}

			return nil
		},
	}
}

func doUpdateElasticsearchStep() test.TestStep {
	return test.TestStep{
		Name: "update",
		Do: func(c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			logrus.Infof("=== Update Elasticsearch cluster %s/%s ===", key.Namespace, key.Name)

			if o == nil {
				return errors.New("Elasticsearch is null")
			}
			es := o.(*elasticsearchapi.Elasticsearch)

			// Add labels must force to update all resources
			es.Labels = map[string]string{
				"test": "fu",
			}

			data["lastUpdate"] = condition.FindStatusCondition(es.Status.Conditions, ElasticsearchCondition).LastTransitionTime

			if err = c.Update(context.Background(), es); err != nil {
				return err
			}

			return nil
		},
		Check: func(t *testing.T, c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {

			return nil
		},
	}
}

func doDeleteElasticsearchStep() test.TestStep {
	return test.TestStep{
		Name: "delete",
		Do: func(c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			logrus.Infof("=== Delete Elasticsearch cluster %s/%s ===", key.Namespace, key.Name)

			if o == nil {
				return errors.New("Centreon serviceGroup is null")
			}
			es := o.(*elasticsearchapi.Elasticsearch)

			wait := int64(0)
			if err = c.Delete(context.Background(), es, &client.DeleteOptions{GracePeriodSeconds: &wait}); err != nil {
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
