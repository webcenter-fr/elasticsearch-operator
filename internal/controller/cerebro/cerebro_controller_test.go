package cerebro

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/disaster37/operator-sdk-extra/v3/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/v3/pkg/helper"
	"github.com/disaster37/operator-sdk-extra/v3/pkg/test"
	routev1 "github.com/openshift/api/route/v1"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	cerebrocrd "github.com/webcenter-fr/elasticsearch-operator/api/cerebro/v1"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearch/v1"
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	condition "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (t *CerebroControllerTestSuite) TestCerebroController() {
	key := types.NamespacedName{
		Name:      "t-cb-" + helper.RandomString(10),
		Namespace: "default",
	}
	data := map[string]any{}

	testCase := test.NewTestCase[*cerebrocrd.Cerebro](t.T(), t.k8sClient, key, 5*time.Second, data)
	testCase.Steps = []test.TestStep[*cerebrocrd.Cerebro]{
		doCreateCerebroStep(),
		doUpdateCerebroStep(),
		doAddHostStep(),
		doMigrateLegacySecretStep(),
		doDeleteCerebroStep(),
	}

	testCase.Run()
}

func doCreateCerebroStep() test.TestStep[*cerebrocrd.Cerebro] {
	return test.TestStep[*cerebrocrd.Cerebro]{
		Name: "create",
		Do: func(c client.Client, key types.NamespacedName, o *cerebrocrd.Cerebro, data map[string]any) (err error) {
			logrus.Infof("=== Add new Cerebro %s/%s ===\n\n", key.Namespace, key.Name)

			cb := &cerebrocrd.Cerebro{
				ObjectMeta: metav1.ObjectMeta{
					Name:      key.Name,
					Namespace: key.Namespace,
				},
				Spec: cerebrocrd.CerebroSpec{
					Version: "v0.1.0",
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
							TlsEnabled: ptr.To[bool](true),
						},
						LoadBalancer: &shared.EndpointLoadBalancerSpec{
							Enabled: true,
						},
					},
					Deployment: cerebrocrd.CerebroDeploymentSpec{
						Deployment: shared.Deployment{
							Replicas: 1,
						},
					},
				},
			}

			if err = c.Create(context.Background(), cb); err != nil {
				return err
			}

			return nil
		},
		Check: func(t *testing.T, c client.Client, key types.NamespacedName, o *cerebrocrd.Cerebro, data map[string]any) (err error) {
			cb := &cerebrocrd.Cerebro{}
			var (
				s     *corev1.Secret
				svc   *corev1.Service
				i     *networkingv1.Ingress
				cm    *corev1.ConfigMap
				dpl   *appv1.Deployment
				route *routev1.Route
			)

			isTimeout, err := test.RunWithTimeout(func() error {
				if err := c.Get(context.Background(), key, cb); err != nil {
					t.Fatal("Cerebro not found")
				}

				if cb.GetStatus().GetObservedGeneration() > 0 {
					return nil
				}

				return errors.New("Not yet created")
			}, time.Second*30, time.Second*1)
			if err != nil || isTimeout {
				t.Fatalf("All Cerebro step provisionning not finished: %s", err.Error())
			}

			// Secrets must exist
			s = &corev1.Secret{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetSecretNameForApplication(cb)}, s); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, s.Data)
			assert.NotEmpty(t, s.OwnerReferences)

			// Services must exists
			svc = &corev1.Service{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetServiceName(cb)}, svc); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, svc.OwnerReferences)

			// Load balancer must exist
			svc = &corev1.Service{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetLoadBalancerName(cb)}, svc); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, svc.OwnerReferences)

			// Ingress must exist
			i = &networkingv1.Ingress{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetIngressName(cb)}, i); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, i.OwnerReferences)

			// Route must exist
			route = &routev1.Route{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetIngressName(cb)}, route); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, route.OwnerReferences)

			// ConfigMaps must exist
			cm = &corev1.ConfigMap{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetConfigMapName(cb)}, cm); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, cm.OwnerReferences)

			// Deployment musts exist
			dpl = &appv1.Deployment{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetDeploymentName(cb)}, dpl); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, dpl.OwnerReferences)

			// Status must be update
			assert.NotEmpty(t, cb.Status.PhaseName)
			assert.NotEmpty(t, cb.Status.Url)
			assert.False(t, *cb.Status.IsOnError)

			return nil
		},
	}
}

func doUpdateCerebroStep() test.TestStep[*cerebrocrd.Cerebro] {
	return test.TestStep[*cerebrocrd.Cerebro]{
		Name: "update",
		Do: func(c client.Client, key types.NamespacedName, o *cerebrocrd.Cerebro, data map[string]any) (err error) {
			logrus.Infof("=== Update Cerebro cluster %s/%s ===\n\n", key.Namespace, key.Name)

			if o == nil {
				return errors.New("Cerebro is null")
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
		Check: func(t *testing.T, c client.Client, key types.NamespacedName, o *cerebrocrd.Cerebro, data map[string]any) (err error) {
			cb := &cerebrocrd.Cerebro{}

			var (
				s     *corev1.Secret
				svc   *corev1.Service
				i     *networkingv1.Ingress
				cm    *corev1.ConfigMap
				dpl   *appv1.Deployment
				route *routev1.Route
			)

			lastGeneration := data["lastGeneration"].(int64)

			isTimeout, err := test.RunWithTimeout(func() error {
				if err := c.Get(context.Background(), key, cb); err != nil {
					t.Fatal("Cerebro not found")
				}

				if lastGeneration < cb.GetStatus().GetObservedGeneration() {
					return nil
				}

				return errors.New("Not yet updated")
			}, time.Second*30, time.Second*1)
			if err != nil || isTimeout {
				t.Fatalf("All Cerebro step upgrading not finished: %s", err.Error())
			}

			// Secrets must exist
			s = &corev1.Secret{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetSecretNameForApplication(cb)}, s); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, s.Data)
			assert.NotEmpty(t, s.OwnerReferences)
			assert.Equal(t, "fu", s.Labels["test"])

			// Services must exists
			svc = &corev1.Service{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetServiceName(cb)}, svc); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, svc.OwnerReferences)
			assert.Equal(t, "fu", svc.Labels["test"])

			// Load balancer must exist
			svc = &corev1.Service{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetLoadBalancerName(cb)}, svc); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, svc.OwnerReferences)
			assert.Equal(t, "fu", svc.Labels["test"])

			// Ingress must exist
			i = &networkingv1.Ingress{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetIngressName(cb)}, i); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, i.OwnerReferences)
			assert.Equal(t, "fu", i.Labels["test"])

			// Route must exist
			route = &routev1.Route{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetIngressName(cb)}, route); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, route.OwnerReferences)
			assert.Equal(t, "fu", route.Labels["test"])

			// ConfigMaps must exist
			cm = &corev1.ConfigMap{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetConfigMapName(cb)}, cm); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, cm.OwnerReferences)
			assert.Equal(t, "fu", cm.Labels["test"])

			// Deployment musts exist
			dpl = &appv1.Deployment{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetDeploymentName(cb)}, dpl); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, dpl.OwnerReferences)
			assert.Equal(t, "fu", dpl.Labels["test"])

			// Status must be update
			assert.NotEmpty(t, cb.Status.PhaseName)
			assert.NotEmpty(t, cb.Status.Url)
			assert.False(t, *cb.Status.IsOnError)

			return nil
		},
	}
}

func doAddHostStep() test.TestStep[*cerebrocrd.Cerebro] {
	return test.TestStep[*cerebrocrd.Cerebro]{
		Name: "addHost",
		Do: func(c client.Client, key types.NamespacedName, o *cerebrocrd.Cerebro, data map[string]any) (err error) {
			logrus.Infof("=== Add Cerebro host %s/%s ===\n\n", key.Namespace, key.Name)

			if o == nil {
				return errors.New("Cerebro is null")
			}

			cm := &corev1.ConfigMap{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetConfigMapName(o)}, cm); err != nil {
				return err
			}

			// Add elasticsearch cluster
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

			isTimeout, err := test.RunWithTimeout(func() error {
				if err := c.Get(context.Background(), key, es); err != nil {
					return err
				}

				// In envtest, no kubelet
				// So the Elasticsearch condition never set as true
				if condition.FindStatusCondition(es.Status.Conditions, controller.ReadyCondition.String()) != nil && condition.FindStatusCondition(es.Status.Conditions, controller.ReadyCondition.String()).Reason != "Initialize" {
					return nil
				}

				return errors.New("Not yet created")
			}, time.Second*30, time.Second*1)

			if err != nil || isTimeout {
				panic(err)
				// return err
			}

			// Add host must reconcile the settings
			host := &cerebrocrd.Host{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: o.Namespace,
				},
				Spec: cerebrocrd.HostSpec{
					CerebroRef: cerebrocrd.HostCerebroRef{
						Name:      o.Name,
						Namespace: o.Namespace,
					},
					ElasticsearchRef: cerebrocrd.ElasticsearchRef{
						ManagedElasticsearchRef: &corev1.LocalObjectReference{
							Name: key.Name,
						},
					},
				},
			}

			data["lastVersion"] = cm.ResourceVersion

			if err = c.Create(context.Background(), host); err != nil {
				return err
			}

			logrus.Infof("Cerebro Host %s/%s added", host.Namespace, host.Name)

			time.Sleep(5 * time.Second)

			return nil
		},
		Check: func(t *testing.T, c client.Client, key types.NamespacedName, o *cerebrocrd.Cerebro, data map[string]any) (err error) {
			cb := &cerebrocrd.Cerebro{}
			cm := &corev1.ConfigMap{}

			lastVersion := data["lastVersion"].(string)

			if err := c.Get(context.Background(), key, cb); err != nil {
				t.Fatal("Cerebro not found")
			}

			isTimeout, err := test.RunWithTimeout(func() error {
				if err := c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetConfigMapName(cb)}, cm); err != nil {
					t.Fatal("Cerebro not found")
				}

				// In envtest, no kubelet
				// So the condition never set as true
				if lastVersion != cm.ResourceVersion {
					return nil
				}

				return errors.New("Not yet updated")
			}, time.Second*30, time.Second*1)
			if err != nil || isTimeout {
				t.Fatalf("All Cerebro step upgrading not finished: %s", err.Error())
			}

			// ConfigMaps must exist
			assert.Contains(t, cm.Data["application.yaml"], fmt.Sprintf("name: %s", key.Name))
			assert.Contains(t, cm.Data["application.yaml"], fmt.Sprintf("host: https://%s-es.%s.svc:9200", key.Name, key.Namespace))

			return nil
		},
	}
}

func doMigrateLegacySecretStep() test.TestStep[*cerebrocrd.Cerebro] {
	return test.TestStep[*cerebrocrd.Cerebro]{
		Name: "migrateLegacySecret",
		Do: func(c client.Client, key types.NamespacedName, o *cerebrocrd.Cerebro, data map[string]any) (err error) {
			logrus.Infof("=== Migrate legacy secret %s/%s ===\n\n", key.Namespace, key.Name)

			if o == nil {
				return errors.New("Cerebro is null")
			}

			// Simulate a legacy secret carrying the value under the `application` key
			s := &corev1.Secret{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetSecretNameForApplication(o)}, s); err != nil {
				return err
			}
			s.Data = map[string][]byte{
				"application": []byte("legacy-value"),
			}
			if err = c.Update(context.Background(), s); err != nil {
				return err
			}

			// Touch the Cerebro spec to trigger a reconcile
			o.Spec.Deployment.Labels = map[string]string{
				"migrate": "true",
			}
			if err = c.Update(context.Background(), o); err != nil {
				return err
			}

			return nil
		},
		Check: func(t *testing.T, c client.Client, key types.NamespacedName, o *cerebrocrd.Cerebro, data map[string]any) (err error) {
			s := &corev1.Secret{}

			isTimeout, err := test.RunWithTimeout(func() error {
				if err := c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetSecretNameForApplication(o)}, s); err != nil {
					t.Fatal(err)
				}

				// The legacy value must be preserved under `session-key` and the old
				// `application` key must be gone
				if string(s.Data["session-key"]) == "legacy-value" && len(s.Data["application"]) == 0 {
					return nil
				}

				return errors.New("Secret not yet migrated")
			}, time.Second*30, time.Second*1)
			if err != nil || isTimeout {
				t.Fatalf("Secret migration not finished: %s", err.Error())
			}

			assert.Equal(t, "legacy-value", string(s.Data["session-key"]))
			assert.NotContains(t, s.Data, "application")

			return nil
		},
	}
}

func doDeleteCerebroStep() test.TestStep[*cerebrocrd.Cerebro] {
	return test.TestStep[*cerebrocrd.Cerebro]{
		Name: "delete",
		Do: func(c client.Client, key types.NamespacedName, o *cerebrocrd.Cerebro, data map[string]any) (err error) {
			logrus.Infof("=== Delete Cerebro cluster %s/%s ===\n\n", key.Namespace, key.Name)

			if o == nil {
				return errors.New("Cerebro is null")
			}

			wait := int64(0)
			if err = c.Delete(context.Background(), o, &client.DeleteOptions{GracePeriodSeconds: &wait}); err != nil {
				return err
			}

			return nil
		},
		Check: func(t *testing.T, c client.Client, key types.NamespacedName, o *cerebrocrd.Cerebro, data map[string]any) (err error) {
			cb := &cerebrocrd.Cerebro{}
			isDeleted := false

			// In envtest, no kubelet
			// So the cascading children delation not works
			isTimeout, err := test.RunWithTimeout(func() error {
				if err = c.Get(context.Background(), key, cb); err != nil {
					if k8serrors.IsNotFound(err) {
						isDeleted = true
						return nil
					}
					t.Fatal(err)
				}

				return errors.New("Not yet deleted")
			}, time.Second*30, time.Second*1)
			if err != nil || isTimeout {
				t.Fatalf("Cerebro stil exist: %s", err.Error())
			}

			assert.True(t, isDeleted)

			return nil
		},
	}
}
