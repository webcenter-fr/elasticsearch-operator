package kibana

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/disaster37/operator-sdk-extra/v3/pkg/helper"
	"github.com/disaster37/operator-sdk-extra/v3/pkg/test"
	routev1 "github.com/openshift/api/route/v1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearch/v1"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/api/kibana/v1"
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
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
	data := map[string]any{}

	testCase := test.NewTestCase[*kibanacrd.Kibana](t.T(), t.k8sClient, key, 5*time.Second, data)
	testCase.Steps = []test.TestStep[*kibanacrd.Kibana]{
		doCreateKibanaStep(),
		doUpdateKibanaStep(),
		doDeleteKibanaStep(),
	}

	testCase.Run()
}

func doCreateKibanaStep() test.TestStep[*kibanacrd.Kibana] {
	return test.TestStep[*kibanacrd.Kibana]{
		Name: "create",
		Do: func(c client.Client, key types.NamespacedName, o *kibanacrd.Kibana, data map[string]any) (err error) {
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
		Check: func(t *testing.T, c client.Client, key types.NamespacedName, o *kibanacrd.Kibana, data map[string]any) (err error) {
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
			assert.NotEmpty(t, s.Data["ca.crt"])
			assert.NotEmpty(t, s.Data["ca.key"])

			s = &corev1.Secret{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetSecretNameForTls(kb)}, s); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, s.Data)
			assert.NotEmpty(t, s.Data["tls.crt"])
			assert.NotEmpty(t, s.Data["tls.key"])

			// Secrets for CA Elasticsearch
			s = &corev1.Secret{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetSecretNameForCAElasticsearch(kb)}, s); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, s.Data)
			assert.NotEmpty(t, s.OwnerReferences)

			// Secrets for credentials must exist
			s = &corev1.Secret{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetSecretNameForCredentials(kb)}, s); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, s.Data)
			assert.NotEmpty(t, s.OwnerReferences)

			// Services must exists
			svc = &corev1.Service{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetServiceName(kb)}, svc); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, svc.OwnerReferences)

			// Load balancer must exist
			svc = &corev1.Service{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetLoadBalancerName(kb)}, svc); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, svc.OwnerReferences)

			// Ingress must exist
			i = &networkingv1.Ingress{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetIngressName(kb)}, i); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, i.OwnerReferences)

			// Route must exist
			route = &routev1.Route{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetIngressName(kb)}, route); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, route.OwnerReferences)

			// Service Account must exist
			serviceAccount = &corev1.ServiceAccount{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetServiceAccountName(kb)}, serviceAccount); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, serviceAccount.OwnerReferences)

			// roleBinding must exist
			roleBinding = &rbacv1.RoleBinding{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetServiceAccountName(kb)}, roleBinding); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, roleBinding.OwnerReferences)

			// ConfigMaps must exist
			cm = &corev1.ConfigMap{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetConfigMapName(kb)}, cm); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, cm.OwnerReferences)

			// PDB must exist
			pdb = &policyv1.PodDisruptionBudget{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetPDBName(kb)}, pdb); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, pdb.OwnerReferences)

			// Network policy exist
			np = &networkingv1.NetworkPolicy{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetNetworkPolicyName(kb)}, np); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, np.OwnerReferences)

			// Deployment musts exist
			dpl = &appv1.Deployment{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetDeploymentName(kb)}, dpl); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, dpl.OwnerReferences)

			// Pod monitor must exist
			pm = &monitoringv1.PodMonitor{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetPodMonitorName(kb)}, pm); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, pm.OwnerReferences)

			// Status must be update
			assert.NotEmpty(t, kb.Status.PhaseName)
			assert.NotEmpty(t, kb.Status.Url)
			assert.False(t, *kb.Status.IsOnError)

			// TLS workflow status must converge to empty (saga completed)
			assert.Empty(t, kb.Status.TlsWorkflowStatus.CurrentPhase)

			return nil
		},
	}
}

func doUpdateKibanaStep() test.TestStep[*kibanacrd.Kibana] {
	return test.TestStep[*kibanacrd.Kibana]{
		Name: "update",
		Do: func(c client.Client, key types.NamespacedName, o *kibanacrd.Kibana, data map[string]any) (err error) {
			logrus.Infof("=== Update Kibana cluster %s/%s ===\n\n", key.Namespace, key.Name)

			if o == nil {
				return errors.New("Kibana is null")
			}

			// Add labels must force to update all resources
			o.Labels = map[string]string{
				"test": "fu",
			}
			// Change spec to track generation
			o.Spec.Deployment.Labels = map[string]string{
				"test": "fu",
			}

			data["lastGeneration"] = o.GetStatus().GetObservedGeneration()

			if err = c.Update(context.Background(), o); err != nil {
				return err
			}

			return nil
		},
		Check: func(t *testing.T, c client.Client, key types.NamespacedName, o *kibanacrd.Kibana, data map[string]any) (err error) {
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
			assert.NotEmpty(t, s.Data["ca.crt"])
			assert.NotEmpty(t, s.Data["ca.key"])
			assert.Equal(t, "fu", s.Labels["test"])

			s = &corev1.Secret{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetSecretNameForTls(kb)}, s); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, s.Data)
			assert.NotEmpty(t, s.Data["tls.crt"])
			assert.NotEmpty(t, s.Data["tls.key"])
			assert.Equal(t, "fu", s.Labels["test"])

			// Secrets for CA Elasticsearch
			s = &corev1.Secret{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetSecretNameForCAElasticsearch(kb)}, s); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, s.Data)
			assert.NotEmpty(t, s.OwnerReferences)
			assert.Equal(t, "fu", s.Labels["test"])

			// Secrets for credentials must exist
			s = &corev1.Secret{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetSecretNameForCredentials(kb)}, s); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, s.Data)
			assert.NotEmpty(t, s.OwnerReferences)
			assert.Equal(t, "fu", s.Labels["test"])

			// Services must exists
			svc = &corev1.Service{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetServiceName(kb)}, svc); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, svc.OwnerReferences)
			assert.Equal(t, "fu", svc.Labels["test"])

			// Load balancer must exist
			svc = &corev1.Service{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetLoadBalancerName(kb)}, svc); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, svc.OwnerReferences)
			assert.Equal(t, "fu", svc.Labels["test"])

			// Ingress must exist
			i = &networkingv1.Ingress{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetIngressName(kb)}, i); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, i.OwnerReferences)
			assert.Equal(t, "fu", i.Labels["test"])

			// Route must exist
			route = &routev1.Route{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetIngressName(kb)}, route); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, route.OwnerReferences)
			assert.Equal(t, "fu", route.Labels["test"])

			// Service Account must exist
			serviceAccount = &corev1.ServiceAccount{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetServiceAccountName(kb)}, serviceAccount); err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, "fu", serviceAccount.Labels["test"])
			assert.NotEmpty(t, serviceAccount.OwnerReferences)

			// roleBinding must exist
			roleBinding = &rbacv1.RoleBinding{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetServiceAccountName(kb)}, roleBinding); err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, "fu", roleBinding.Labels["test"])
			assert.NotEmpty(t, roleBinding.OwnerReferences)

			// ConfigMaps must exist
			cm = &corev1.ConfigMap{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetConfigMapName(kb)}, cm); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, cm.OwnerReferences)
			assert.Equal(t, "fu", cm.Labels["test"])

			// PDB must exist
			pdb = &policyv1.PodDisruptionBudget{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetPDBName(kb)}, pdb); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, pdb.OwnerReferences)
			assert.Equal(t, "fu", pdb.Labels["test"])

			// Deployment musts exist
			dpl = &appv1.Deployment{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetDeploymentName(kb)}, dpl); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, dpl.OwnerReferences)
			assert.Equal(t, "fu", dpl.Labels["test"])

			// Network policy exist
			np = &networkingv1.NetworkPolicy{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetNetworkPolicyName(kb)}, np); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, np.OwnerReferences)
			assert.Equal(t, "fu", np.Labels["test"])

			// Pod monitor must exist
			pm = &monitoringv1.PodMonitor{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetPodMonitorName(kb)}, pm); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, pm.OwnerReferences)

			// Status must be update
			assert.NotEmpty(t, kb.Status.PhaseName)
			assert.NotEmpty(t, kb.Status.Url)
			assert.False(t, *kb.Status.IsOnError)

			return nil
		},
	}
}

func doDeleteKibanaStep() test.TestStep[*kibanacrd.Kibana] {
	return test.TestStep[*kibanacrd.Kibana]{
		Name: "delete",
		Do: func(c client.Client, key types.NamespacedName, o *kibanacrd.Kibana, data map[string]any) (err error) {
			logrus.Infof("=== Delete Kibana cluster %s/%s ===\n\n", key.Namespace, key.Name)

			if o == nil {
				return errors.New("Kibana is null")
			}

			wait := int64(0)
			if err = c.Delete(context.Background(), o, &client.DeleteOptions{GracePeriodSeconds: &wait}); err != nil {
				return err
			}

			return nil
		},
		Check: func(t *testing.T, c client.Client, key types.NamespacedName, o *kibanacrd.Kibana, data map[string]any) (err error) {
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

// TestKibanaControllerCARotation asserts that a forced CA rotation via
// AnnotationForceRenewTLS changes the CA secret's ca.crt and clears the annotation.
func (t *KibanaControllerTestSuite) TestKibanaControllerCARotation() {
	ctx := context.Background()
	c := t.k8sClient
	key := types.NamespacedName{Name: "t-kb-ca-" + helper.RandomString(8), Namespace: "default"}

	// Create Elasticsearch first (required dependency)
	es := &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Name:      key.Name,
			Namespace: key.Namespace,
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			Version: "8.6.0",
			NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				{Name: "all", Roles: []string{"master", "client", "data"}, Deployment: shared.Deployment{Replicas: 1}},
			},
		},
	}
	assert.NoError(t.T(), c.Create(ctx, es))

	kb := &kibanacrd.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Name:      key.Name,
			Namespace: key.Namespace,
		},
		Spec: kibanacrd.KibanaSpec{
			Version: "8.6.0",
			ElasticsearchRef: shared.ElasticsearchRef{
				ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{Name: es.Name},
			},
			Deployment: kibanacrd.KibanaDeploymentSpec{
				Deployment: shared.Deployment{Replicas: 1},
			},
		},
	}
	assert.NoError(t.T(), c.Create(ctx, kb))

	// Wait for bootstrap to complete (saga must converge to "")
	waitKibanaSagaConverged(t.T(), c, key)
	waitKibanaObservedGeneration(t.T(), c, key, 0)

	// Read the CA secret before rotation
	kb = &kibanacrd.Kibana{}
	assert.NoError(t.T(), c.Get(ctx, key, kb))
	caBefore := &corev1.Secret{}
	assert.NoError(t.T(), c.Get(ctx, types.NamespacedName{Namespace: key.Namespace, Name: GetSecretNameForPki(kb)}, caBefore))
	assert.NotEmpty(t.T(), caBefore.Data["ca.crt"])

	// Set force-renew annotation
	kb = &kibanacrd.Kibana{}
	assert.NoError(t.T(), c.Get(ctx, key, kb))
	if kb.Annotations == nil {
		kb.Annotations = map[string]string{}
	}
	kb.Annotations[AnnotationForceRenewTLS] = "true"
	lastGen := kb.GetStatus().GetObservedGeneration()
	assert.NoError(t.T(), c.Update(ctx, kb))

	// Wait for reconciliation
	waitKibanaObservedGeneration(t.T(), c, key, lastGen)

	// Assert CA cert changed and annotation cleared
	kb = &kibanacrd.Kibana{}
	assert.NoError(t.T(), c.Get(ctx, key, kb))
	caAfter := &corev1.Secret{}
	assert.NoError(t.T(), c.Get(ctx, types.NamespacedName{Namespace: key.Namespace, Name: GetSecretNameForPki(kb)}, caAfter))
	assert.NotEmpty(t.T(), caAfter.Data["ca.crt"])
	assert.NotEqual(t.T(), string(caBefore.Data["ca.crt"]), string(caAfter.Data["ca.crt"]),
		"CA cert must change after forced rotation")
	assert.Empty(t.T(), kb.Annotations[AnnotationForceRenewTLS],
		"force-renew annotation must be cleared after rotation")
}

// TestKibanaControllerLeafRenew asserts that a forced leaf regeneration via
// AnnotationForceRenewCertificates changes the leaf tls.crt while the CA ca.crt stays unchanged.
func (t *KibanaControllerTestSuite) TestKibanaControllerLeafRenew() {
	ctx := context.Background()
	c := t.k8sClient
	key := types.NamespacedName{Name: "t-kb-lr-" + helper.RandomString(8), Namespace: "default"}

	es := &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Name:      key.Name,
			Namespace: key.Namespace,
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			Version: "8.6.0",
			NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				{Name: "all", Roles: []string{"master", "client", "data"}, Deployment: shared.Deployment{Replicas: 1}},
			},
		},
	}
	assert.NoError(t.T(), c.Create(ctx, es))

	kb := &kibanacrd.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Name:      key.Name,
			Namespace: key.Namespace,
		},
		Spec: kibanacrd.KibanaSpec{
			Version: "8.6.0",
			ElasticsearchRef: shared.ElasticsearchRef{
				ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{Name: es.Name},
			},
			Deployment: kibanacrd.KibanaDeploymentSpec{
				Deployment: shared.Deployment{Replicas: 1},
			},
		},
	}
	assert.NoError(t.T(), c.Create(ctx, kb))
	waitKibanaSagaConverged(t.T(), c, key)
	waitKibanaObservedGeneration(t.T(), c, key, 0)

	// Read CA and leaf before
	kb = &kibanacrd.Kibana{}
	assert.NoError(t.T(), c.Get(ctx, key, kb))
	caBefore := &corev1.Secret{}
	assert.NoError(t.T(), c.Get(ctx, types.NamespacedName{Namespace: key.Namespace, Name: GetSecretNameForPki(kb)}, caBefore))
	leafBefore := &corev1.Secret{}
	assert.NoError(t.T(), c.Get(ctx, types.NamespacedName{Namespace: key.Namespace, Name: GetSecretNameForTls(kb)}, leafBefore))

	// Set force-renew-certificates annotation
	kb = &kibanacrd.Kibana{}
	assert.NoError(t.T(), c.Get(ctx, key, kb))
	if kb.Annotations == nil {
		kb.Annotations = map[string]string{}
	}
	kb.Annotations[AnnotationForceRenewCertificates] = "true"
	lastGen := kb.GetStatus().GetObservedGeneration()
	assert.NoError(t.T(), c.Update(ctx, kb))
	waitKibanaObservedGeneration(t.T(), c, key, lastGen)

	// Assert leaf changed, CA unchanged, annotation cleared
	kb = &kibanacrd.Kibana{}
	assert.NoError(t.T(), c.Get(ctx, key, kb))
	caAfter := &corev1.Secret{}
	assert.NoError(t.T(), c.Get(ctx, types.NamespacedName{Namespace: key.Namespace, Name: GetSecretNameForPki(kb)}, caAfter))
	leafAfter := &corev1.Secret{}
	assert.NoError(t.T(), c.Get(ctx, types.NamespacedName{Namespace: key.Namespace, Name: GetSecretNameForTls(kb)}, leafAfter))

	assert.Equal(t.T(), string(caBefore.Data["ca.crt"]), string(caAfter.Data["ca.crt"]),
		"CA cert must not change during leaf-only renewal")
	assert.NotEqual(t.T(), string(leafBefore.Data["tls.crt"]), string(leafAfter.Data["tls.crt"]),
		"leaf cert must change after forced leaf renewal")
	assert.Empty(t.T(), kb.Annotations[AnnotationForceRenewCertificates],
		"force-renew-certificates annotation must be cleared after renewal")
}

// TestKibanaControllerBYO asserts that when Spec.Tls.CertificateSecretRef is set,
// no self-managed TLS secrets are created and no phase writes occur.
func (t *KibanaControllerTestSuite) TestKibanaControllerBYO() {
	ctx := context.Background()
	c := t.k8sClient
	key := types.NamespacedName{Name: "t-kb-byo-" + helper.RandomString(8), Namespace: "default"}

	es := &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Name:      key.Name,
			Namespace: key.Namespace,
		},
		Spec: elasticsearchcrd.ElasticsearchSpec{
			Version: "8.6.0",
			NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				{Name: "all", Roles: []string{"master", "client", "data"}, Deployment: shared.Deployment{Replicas: 1}},
			},
		},
	}
	assert.NoError(t.T(), c.Create(ctx, es))

	kb := &kibanacrd.Kibana{
		ObjectMeta: metav1.ObjectMeta{
			Name:      key.Name,
			Namespace: key.Namespace,
		},
		Spec: kibanacrd.KibanaSpec{
			Version: "8.6.0",
			Tls: shared.TlsSpec{
				CertificateSecretRef: &corev1.LocalObjectReference{Name: "my-byo-secret"},
				Enabled:              ptr.To[bool](true),
			},
			ElasticsearchRef: shared.ElasticsearchRef{
				ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{Name: es.Name},
			},
			Deployment: kibanacrd.KibanaDeploymentSpec{
				Deployment: shared.Deployment{Replicas: 1},
			},
		},
	}
	assert.NoError(t.T(), c.Create(ctx, kb))
	waitKibanaObservedGeneration(t.T(), c, key, 0)

	// Assert no self-managed TLS secrets exist
	kb = &kibanacrd.Kibana{}
	assert.NoError(t.T(), c.Get(ctx, key, kb))

	leafSecret := &corev1.Secret{}
	err := c.Get(ctx, types.NamespacedName{Namespace: key.Namespace, Name: fmt.Sprintf("%s-tls-kb", key.Name)}, leafSecret)
	assert.True(t.T(), k8serrors.IsNotFound(err), "self-managed leaf secret must not exist in BYO mode")

	caSecret := &corev1.Secret{}
	err = c.Get(ctx, types.NamespacedName{Namespace: key.Namespace, Name: fmt.Sprintf("%s-tls-kb-ca", key.Name)}, caSecret)
	assert.True(t.T(), k8serrors.IsNotFound(err), "self-managed CA secret must not exist in BYO mode")

	// Assert no TLS workflow phase writes
	assert.Empty(t.T(), kb.Status.TlsWorkflowStatus.CurrentPhase)
}

// waitKibanaObservedGeneration waits until the Kibana observed generation is
// strictly greater than min, returning the new value.
func waitKibanaObservedGeneration(t *testing.T, c client.Client, key types.NamespacedName, min int64) int64 {
	kb := &kibanacrd.Kibana{}
	var last int64
	isTimeout, err := test.RunWithTimeout(func() error {
		if err := c.Get(context.Background(), key, kb); err != nil {
			t.Fatal(err)
		}
		last = kb.GetStatus().GetObservedGeneration()
		if last > min {
			return nil
		}
		return errors.New("not yet updated")
	}, 30*time.Second, 1*time.Second)
	if err != nil {
		t.Fatalf("Kibana not converged: %s", err.Error())
	}
	if isTimeout {
		t.Fatal("Kibana not converged: timed out waiting for observed generation")
	}
	return last
}

// waitKibanaSagaConverged polls until the TLS workflow saga has completed
// (CurrentPhase == ""), with a timeout. The TLS saga needs 3-4 reconcile
// cycles to complete ("" → Rotate → Converge → "").
func waitKibanaSagaConverged(t *testing.T, c client.Client, key types.NamespacedName) {
	kb := &kibanacrd.Kibana{}
	isTimeout, err := test.RunWithTimeout(func() error {
		if err := c.Get(context.Background(), key, kb); err != nil {
			t.Fatal(err)
		}
		if kb.Status.TlsWorkflowStatus.CurrentPhase == "" {
			return nil
		}
		return errors.New("saga not yet converged")
	}, 30*time.Second, 1*time.Second)
	if err != nil {
		t.Fatalf("Kibana saga not converged: %s", err.Error())
	}
	if isTimeout {
		t.Fatalf("Kibana saga not converged: timed out (phase=%s)", kb.Status.TlsWorkflowStatus.CurrentPhase)
	}
}
