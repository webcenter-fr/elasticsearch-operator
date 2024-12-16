package kibana

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/disaster37/k8s-objectmatcher/patch"
	"github.com/disaster37/operator-sdk-extra/pkg/helper"
	"github.com/disaster37/operator-sdk-extra/pkg/test"
	routev1 "github.com/openshift/api/route/v1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/apis/kibana/v1"
	"github.com/webcenter-fr/elasticsearch-operator/apis/shared"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (t *KibanaControllerTestSuite) TestKibanaController() {
	key := types.NamespacedName{
		Name:      "t-kb-" + helper.RandomString(10),
		Namespace: "default",
	}
	kb := &kibanacrd.Kibana{}
	data := map[string]any{}

	testCase := test.NewTestCase(t.T(), t.k8sClient, key, kb, 5*time.Second, data)
	testCase.Steps = []test.TestStep{
		doCreateKibanaStep(),
		doUpdateKibanaStep(),
		doDeleteKibanaStep(),
	}

	testCase.Run()
}

func doCreateKibanaStep() test.TestStep {
	return test.TestStep{
		Name: "create",
		Do: func(c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			logrus.Infof("=== Add new Kibana %s/%s ===\n\n", key.Namespace, key.Name)

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
							Name: "all",
							Roles: []string{
								"master",
								"client",
								"data",
							},
							Deployment: shared.Deployment{
								Replicas: 1,
							},
						},
					},
				},
			}

			if err = c.Create(context.Background(), es); err != nil {
				return err
			}

			kb := &kibanacrd.Kibana{
				ObjectMeta: metav1.ObjectMeta{
					Name:      key.Name,
					Namespace: key.Namespace,
				},
				Spec: kibanacrd.KibanaSpec{
					Version: "8.6.0",
					Endpoint: shared.EndpointSpec{
						Ingress: &shared.EndpointIngressSpec{
							Enabled: true,
							Host:    "test.cluster.local",
							SecretRef: &corev1.LocalObjectReference{
								Name: "test-tls",
							},
						},
						Route: &shared.EndpointRouteSpec{
							Enabled:    true,
							Host:       "test.cluster.local",
							TlsEnabled: ptr.To(true),
						},
						LoadBalancer: &shared.EndpointLoadBalancerSpec{
							Enabled: true,
						},
					},
					ElasticsearchRef: shared.ElasticsearchRef{
						ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
							Name: es.Name,
						},
					},
					Deployment: kibanacrd.KibanaDeploymentSpec{
						Deployment: shared.Deployment{
							Replicas: 2,
						},
					},
					Monitoring: shared.MonitoringSpec{
						Prometheus: &shared.MonitoringPrometheusSpec{
							Enabled: ptr.To(true),
						},
					},
				},
			}

			if err = c.Create(context.Background(), kb); err != nil {
				return err
			}

			return nil
		},
		Check: func(t *testing.T, c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			kb := &kibanacrd.Kibana{}
			var (
				s              *corev1.Secret
				svc            *corev1.Service
				i              *networkingv1.Ingress
				cm             *corev1.ConfigMap
				pdb            *policyv1.PodDisruptionBudget
				dpl            *appv1.Deployment
				np             *networkingv1.NetworkPolicy
				pm             *monitoringv1.PodMonitor
				route          *routev1.Route
				serviceAccount *corev1.ServiceAccount
				roleBinding    *rbacv1.RoleBinding
			)

			isTimeout, err := test.RunWithTimeout(func() error {
				if err := c.Get(context.Background(), key, kb); err != nil {
					t.Fatal("Kibana not found")
				}

				if kb.GetStatus().GetObservedGeneration() > 0 {
					return nil
				}

				return errors.New("Not yet created")
			}, time.Second*30, time.Second*1)
			if err != nil || isTimeout {
				t.Fatalf("All Kibana step provisionning not finished: %s", err.Error())
			}

			// Secrets for PKI and certificates must exist
			s = &corev1.Secret{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetSecretNameForPki(kb)}, s); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, s.Data)
			assert.NotEmpty(t, s.OwnerReferences)
			assert.NotEmpty(t, s.Annotations[patch.LastAppliedConfig])

			s = &corev1.Secret{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetSecretNameForTls(kb)}, s); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, s.Data)
			assert.NotEmpty(t, s.OwnerReferences)
			assert.NotEmpty(t, s.Annotations[patch.LastAppliedConfig])

			// Secrets for CA Elasticsearch
			s = &corev1.Secret{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetSecretNameForCAElasticsearch(kb)}, s); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, s.Data)
			assert.NotEmpty(t, s.OwnerReferences)
			assert.NotEmpty(t, s.Annotations[patch.LastAppliedConfig])

			// Secrets for credentials must exist
			s = &corev1.Secret{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetSecretNameForCredentials(kb)}, s); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, s.Data)
			assert.NotEmpty(t, s.OwnerReferences)
			assert.NotEmpty(t, s.Annotations[patch.LastAppliedConfig])

			// Services must exists
			svc = &corev1.Service{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetServiceName(kb)}, svc); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, svc.OwnerReferences)
			assert.NotEmpty(t, svc.Annotations[patch.LastAppliedConfig])

			// Load balancer must exist
			svc = &corev1.Service{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetLoadBalancerName(kb)}, svc); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, svc.OwnerReferences)
			assert.NotEmpty(t, svc.Annotations[patch.LastAppliedConfig])

			// Ingress must exist
			i = &networkingv1.Ingress{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetIngressName(kb)}, i); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, i.OwnerReferences)
			assert.NotEmpty(t, i.Annotations[patch.LastAppliedConfig])

			// Route must exist
			route = &routev1.Route{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetIngressName(kb)}, route); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, route.OwnerReferences)
			assert.NotEmpty(t, route.Annotations[patch.LastAppliedConfig])

			// Service Account must exist
			serviceAccount = &corev1.ServiceAccount{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetServiceAccountName(kb)}, serviceAccount); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, serviceAccount.OwnerReferences)
			assert.NotEmpty(t, serviceAccount.Annotations[patch.LastAppliedConfig])

			// roleBinding must exist
			roleBinding = &rbacv1.RoleBinding{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetServiceAccountName(kb)}, roleBinding); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, roleBinding.OwnerReferences)
			assert.NotEmpty(t, roleBinding.Annotations[patch.LastAppliedConfig])

			// ConfigMaps must exist
			cm = &corev1.ConfigMap{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetConfigMapName(kb)}, cm); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, cm.OwnerReferences)
			assert.NotEmpty(t, cm.Annotations[patch.LastAppliedConfig])

			// PDB must exist
			pdb = &policyv1.PodDisruptionBudget{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetPDBName(kb)}, pdb); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, pdb.OwnerReferences)
			assert.NotEmpty(t, pdb.Annotations[patch.LastAppliedConfig])

			// Network policy exist
			np = &networkingv1.NetworkPolicy{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetNetworkPolicyName(kb)}, np); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, np.OwnerReferences)
			assert.NotEmpty(t, np.Annotations[patch.LastAppliedConfig])

			// Deployment musts exist
			dpl = &appv1.Deployment{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetDeploymentName(kb)}, dpl); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, dpl.OwnerReferences)
			assert.NotEmpty(t, dpl.Annotations[patch.LastAppliedConfig])

			// Pod monitor must exist
			pm = &monitoringv1.PodMonitor{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetPodMonitorName(kb)}, pm); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, pm.OwnerReferences)
			assert.NotEmpty(t, pm.Annotations[patch.LastAppliedConfig])

			// Status must be update
			assert.NotEmpty(t, kb.Status.PhaseName)
			assert.NotEmpty(t, kb.Status.Url)
			assert.False(t, *kb.Status.IsOnError)

			return nil
		},
	}
}

func doUpdateKibanaStep() test.TestStep {
	return test.TestStep{
		Name: "update",
		Do: func(c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			logrus.Infof("=== Update Kibana cluster %s/%s ===\n\n", key.Namespace, key.Name)

			if o == nil {
				return errors.New("Kibana is null")
			}
			kb := o.(*kibanacrd.Kibana)

			// Add labels must force to update all resources
			kb.Labels = map[string]string{
				"test": "fu",
			}
			// Change spec to track generation
			kb.Spec.Deployment.Labels = map[string]string{
				"test": "fu",
			}

			data["lastGeneration"] = kb.GetStatus().GetObservedGeneration()

			if err = c.Update(context.Background(), kb); err != nil {
				return err
			}

			return nil
		},
		Check: func(t *testing.T, c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			kb := &kibanacrd.Kibana{}

			var (
				s              *corev1.Secret
				svc            *corev1.Service
				i              *networkingv1.Ingress
				cm             *corev1.ConfigMap
				pdb            *policyv1.PodDisruptionBudget
				dpl            *appv1.Deployment
				np             *networkingv1.NetworkPolicy
				pm             *monitoringv1.PodMonitor
				route          *routev1.Route
				serviceAccount *corev1.ServiceAccount
				roleBinding    *rbacv1.RoleBinding
			)

			lastGeneration := data["lastGeneration"].(int64)

			isTimeout, err := test.RunWithTimeout(func() error {
				if err := c.Get(context.Background(), key, kb); err != nil {
					t.Fatal("Kibana not found")
				}

				if lastGeneration < kb.GetStatus().GetObservedGeneration() {
					return nil
				}

				return errors.New("Not yet updated")
			}, time.Second*30, time.Second*1)
			if err != nil || isTimeout {
				t.Fatalf("All Kibana step upgrading not finished: %s", err.Error())
			}

			// Secrets for PKI and certificates must exist
			s = &corev1.Secret{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetSecretNameForPki(kb)}, s); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, s.Data)
			assert.NotEmpty(t, s.OwnerReferences)
			assert.NotEmpty(t, s.Annotations[patch.LastAppliedConfig])
			assert.Equal(t, "fu", s.Labels["test"])

			s = &corev1.Secret{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetSecretNameForTls(kb)}, s); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, s.Data)
			assert.NotEmpty(t, s.OwnerReferences)
			assert.NotEmpty(t, s.Annotations[patch.LastAppliedConfig])
			assert.Equal(t, "fu", s.Labels["test"])

			// Secrets for CA Elasticsearch
			s = &corev1.Secret{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetSecretNameForCAElasticsearch(kb)}, s); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, s.Data)
			assert.NotEmpty(t, s.OwnerReferences)
			assert.NotEmpty(t, s.Annotations[patch.LastAppliedConfig])
			assert.Equal(t, "fu", s.Labels["test"])

			// Secrets for credentials must exist
			s = &corev1.Secret{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetSecretNameForCredentials(kb)}, s); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, s.Data)
			assert.NotEmpty(t, s.OwnerReferences)
			assert.NotEmpty(t, s.Annotations[patch.LastAppliedConfig])
			assert.Equal(t, "fu", s.Labels["test"])

			// Services must exists
			svc = &corev1.Service{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetServiceName(kb)}, svc); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, svc.OwnerReferences)
			assert.NotEmpty(t, svc.Annotations[patch.LastAppliedConfig])
			assert.Equal(t, "fu", svc.Labels["test"])

			// Load balancer must exist
			svc = &corev1.Service{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetLoadBalancerName(kb)}, svc); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, svc.OwnerReferences)
			assert.NotEmpty(t, svc.Annotations[patch.LastAppliedConfig])
			assert.Equal(t, "fu", svc.Labels["test"])

			// Ingress must exist
			i = &networkingv1.Ingress{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetIngressName(kb)}, i); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, i.OwnerReferences)
			assert.NotEmpty(t, i.Annotations[patch.LastAppliedConfig])
			assert.Equal(t, "fu", i.Labels["test"])

			// Route must exist
			route = &routev1.Route{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetIngressName(kb)}, route); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, route.OwnerReferences)
			assert.NotEmpty(t, route.Annotations[patch.LastAppliedConfig])
			assert.Equal(t, "fu", route.Labels["test"])

			// Service Account must exist
			serviceAccount = &corev1.ServiceAccount{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetServiceAccountName(kb)}, serviceAccount); err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, "fu", serviceAccount.Labels["test"])
			assert.NotEmpty(t, serviceAccount.OwnerReferences)
			assert.NotEmpty(t, serviceAccount.Annotations[patch.LastAppliedConfig])

			// roleBinding must exist
			roleBinding = &rbacv1.RoleBinding{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetServiceAccountName(kb)}, roleBinding); err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, "fu", roleBinding.Labels["test"])
			assert.NotEmpty(t, roleBinding.OwnerReferences)
			assert.NotEmpty(t, roleBinding.Annotations[patch.LastAppliedConfig])

			// ConfigMaps must exist
			cm = &corev1.ConfigMap{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetConfigMapName(kb)}, cm); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, cm.OwnerReferences)
			assert.NotEmpty(t, cm.Annotations[patch.LastAppliedConfig])
			assert.Equal(t, "fu", cm.Labels["test"])

			// PDB must exist
			pdb = &policyv1.PodDisruptionBudget{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetPDBName(kb)}, pdb); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, pdb.OwnerReferences)
			assert.NotEmpty(t, pdb.Annotations[patch.LastAppliedConfig])
			assert.Equal(t, "fu", pdb.Labels["test"])

			// Deployment musts exist
			dpl = &appv1.Deployment{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetDeploymentName(kb)}, dpl); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, dpl.OwnerReferences)
			assert.NotEmpty(t, dpl.Annotations[patch.LastAppliedConfig])
			assert.Equal(t, "fu", dpl.Labels["test"])

			// Network policy exist
			np = &networkingv1.NetworkPolicy{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetNetworkPolicyName(kb)}, np); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, np.OwnerReferences)
			assert.NotEmpty(t, np.Annotations[patch.LastAppliedConfig])
			assert.Equal(t, "fu", np.Labels["test"])

			// Pod monitor must exist
			pm = &monitoringv1.PodMonitor{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetPodMonitorName(kb)}, pm); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, pm.OwnerReferences)
			assert.NotEmpty(t, pm.Annotations[patch.LastAppliedConfig])

			// Status must be update
			assert.NotEmpty(t, kb.Status.PhaseName)
			assert.NotEmpty(t, kb.Status.Url)
			assert.False(t, *kb.Status.IsOnError)

			return nil
		},
	}
}

func doDeleteKibanaStep() test.TestStep {
	return test.TestStep{
		Name: "delete",
		Do: func(c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			logrus.Infof("=== Delete Kibana cluster %s/%s ===\n\n", key.Namespace, key.Name)

			if o == nil {
				return errors.New("Kibana is null")
			}
			kb := o.(*kibanacrd.Kibana)

			wait := int64(0)
			if err = c.Delete(context.Background(), kb, &client.DeleteOptions{GracePeriodSeconds: &wait}); err != nil {
				return err
			}

			return nil
		},
		Check: func(t *testing.T, c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			kb := &kibanacrd.Kibana{}
			isDeleted := false

			// In envtest, no kubelet
			// So the cascading children delation not works
			isTimeout, err := test.RunWithTimeout(func() error {
				if err = c.Get(context.Background(), key, kb); err != nil {
					if k8serrors.IsNotFound(err) {
						isDeleted = true
						return nil
					}
					t.Fatal(err)
				}

				return errors.New("Not yet deleted")
			}, time.Second*30, time.Second*1)
			if err != nil || isTimeout {
				t.Fatalf("Dashboard stil exist: %s", err.Error())
			}

			assert.True(t, isDeleted)

			return nil
		},
	}
}
