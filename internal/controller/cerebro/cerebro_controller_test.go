package cerebro

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/disaster37/k8s-objectmatcher/patch"
	"github.com/disaster37/operator-sdk-extra/pkg/controller"
	"github.com/disaster37/operator-sdk-extra/pkg/helper"
	"github.com/disaster37/operator-sdk-extra/pkg/test"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	cerebrocrd "github.com/webcenter-fr/elasticsearch-operator/apis/cerebro/v1"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
	"github.com/webcenter-fr/elasticsearch-operator/apis/shared"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	condition "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (t *CerebroControllerTestSuite) TestCerebroController() {
	key := types.NamespacedName{
		Name:      "t-cb-" + helper.RandomString(10),
		Namespace: "default",
	}
	cb := &cerebrocrd.Cerebro{}
	data := map[string]any{}

	testCase := test.NewTestCase(t.T(), t.k8sClient, key, cb, 5*time.Second, data)
	testCase.Steps = []test.TestStep{
		doCreateCerebroStep(),
		doUpdateCerebroStep(),
		doAddHostStep(),
		doDeleteCerebroStep(),
	}

	testCase.Run()
}

func doCreateCerebroStep() test.TestStep {
	return test.TestStep{
		Name: "create",
		Do: func(c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			logrus.Infof("=== Add new Cerebro %s/%s ===\n\n", key.Namespace, key.Name)

			cb := &cerebrocrd.Cerebro{
				ObjectMeta: metav1.ObjectMeta{
					Name:      key.Name,
					Namespace: key.Namespace,
				},
				Spec: cerebrocrd.CerebroSpec{
					Version: "0.9.4",
					Endpoint: shared.EndpointSpec{
						Ingress: &shared.EndpointIngressSpec{
							Enabled: true,
							Host:    "test.cluster.local",
							SecretRef: &corev1.LocalObjectReference{
								Name: "test-tls",
							},
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
		Check: func(t *testing.T, c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			cb := &cerebrocrd.Cerebro{}
			var (
				s   *corev1.Secret
				svc *corev1.Service
				i   *networkingv1.Ingress
				cm  *corev1.ConfigMap
				dpl *appv1.Deployment
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
			assert.NotEmpty(t, s.Annotations[patch.LastAppliedConfig])

			// Services must exists
			svc = &corev1.Service{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetServiceName(cb)}, svc); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, svc.OwnerReferences)
			assert.NotEmpty(t, svc.Annotations[patch.LastAppliedConfig])

			// Load balancer must exist
			svc = &corev1.Service{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetLoadBalancerName(cb)}, svc); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, svc.OwnerReferences)
			assert.NotEmpty(t, svc.Annotations[patch.LastAppliedConfig])

			// Ingress must exist
			i = &networkingv1.Ingress{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetIngressName(cb)}, i); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, i.OwnerReferences)
			assert.NotEmpty(t, i.Annotations[patch.LastAppliedConfig])

			// ConfigMaps must exist
			cm = &corev1.ConfigMap{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetConfigMapName(cb)}, cm); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, cm.OwnerReferences)
			assert.NotEmpty(t, cm.Annotations[patch.LastAppliedConfig])

			// Deployment musts exist
			dpl = &appv1.Deployment{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetDeploymentName(cb)}, dpl); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, dpl.OwnerReferences)
			assert.NotEmpty(t, dpl.Annotations[patch.LastAppliedConfig])

			// Status must be update
			assert.NotEmpty(t, cb.Status.PhaseName)
			assert.NotEmpty(t, cb.Status.Url)
			assert.False(t, *cb.Status.IsOnError)

			return nil
		},
	}
}

func doUpdateCerebroStep() test.TestStep {
	return test.TestStep{
		Name: "update",
		Do: func(c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			logrus.Infof("=== Update Cerebro cluster %s/%s ===\n\n", key.Namespace, key.Name)

			if o == nil {
				return errors.New("Cerebro is null")
			}
			cb := o.(*cerebrocrd.Cerebro)

			// Add labels must force to update all resources
			cb.Labels = map[string]string{
				"test": "fu",
			}
			// Change spec to track generation
			cb.Spec.Deployment.Labels = map[string]string{
				"test": "fu",
			}

			data["lastGeneration"] = cb.GetStatus().GetObservedGeneration()

			if err = c.Update(context.Background(), cb); err != nil {
				return err
			}

			return nil
		},
		Check: func(t *testing.T, c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			cb := &cerebrocrd.Cerebro{}

			var (
				s   *corev1.Secret
				svc *corev1.Service
				i   *networkingv1.Ingress
				cm  *corev1.ConfigMap
				dpl *appv1.Deployment
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
			assert.NotEmpty(t, s.Annotations[patch.LastAppliedConfig])
			assert.Equal(t, "fu", s.Labels["test"])

			// Services must exists
			svc = &corev1.Service{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetServiceName(cb)}, svc); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, svc.OwnerReferences)
			assert.NotEmpty(t, svc.Annotations[patch.LastAppliedConfig])
			assert.Equal(t, "fu", svc.Labels["test"])

			// Load balancer must exist
			svc = &corev1.Service{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetLoadBalancerName(cb)}, svc); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, svc.OwnerReferences)
			assert.NotEmpty(t, svc.Annotations[patch.LastAppliedConfig])
			assert.Equal(t, "fu", svc.Labels["test"])

			// Ingress must exist
			i = &networkingv1.Ingress{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetIngressName(cb)}, i); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, i.OwnerReferences)
			assert.NotEmpty(t, i.Annotations[patch.LastAppliedConfig])
			assert.Equal(t, "fu", i.Labels["test"])

			// ConfigMaps must exist
			cm = &corev1.ConfigMap{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetConfigMapName(cb)}, cm); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, cm.OwnerReferences)
			assert.NotEmpty(t, cm.Annotations[patch.LastAppliedConfig])
			assert.Equal(t, "fu", cm.Labels["test"])

			// Deployment musts exist
			dpl = &appv1.Deployment{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetDeploymentName(cb)}, dpl); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, dpl.OwnerReferences)
			assert.NotEmpty(t, dpl.Annotations[patch.LastAppliedConfig])
			assert.Equal(t, "fu", dpl.Labels["test"])

			// Status must be update
			assert.NotEmpty(t, cb.Status.PhaseName)
			assert.NotEmpty(t, cb.Status.Url)
			assert.False(t, *cb.Status.IsOnError)

			return nil
		},
	}
}

func doAddHostStep() test.TestStep {
	return test.TestStep{
		Name: "addHost",
		Do: func(c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			logrus.Infof("=== Add Cerebro host %s/%s ===\n\n", key.Namespace, key.Name)

			if o == nil {
				return errors.New("Cerebro is null")
			}
			cb := o.(*cerebrocrd.Cerebro)

			cm := &corev1.ConfigMap{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetConfigMapName(cb)}, cm); err != nil {
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
					Namespace: cb.Namespace,
				},
				Spec: cerebrocrd.HostSpec{
					CerebroRef: cerebrocrd.HostCerebroRef{
						Name:      cb.Name,
						Namespace: cb.Namespace,
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
		Check: func(t *testing.T, c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
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
			assert.Contains(t, cm.Data["application.conf"], fmt.Sprintf("name = \"%s\"", key.Name))
			assert.Contains(t, cm.Data["application.conf"], fmt.Sprintf("host = \"https://%s-es.%s.svc:9200\"", key.Name, key.Namespace))

			return nil
		},
	}
}

func doDeleteCerebroStep() test.TestStep {
	return test.TestStep{
		Name: "delete",
		Do: func(c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			logrus.Infof("=== Delete Cerebro cluster %s/%s ===\n\n", key.Namespace, key.Name)

			if o == nil {
				return errors.New("Cerebro is null")
			}
			cb := o.(*cerebrocrd.Cerebro)

			wait := int64(0)
			if err = c.Delete(context.Background(), cb, &client.DeleteOptions{GracePeriodSeconds: &wait}); err != nil {
				return err
			}

			return nil
		},
		Check: func(t *testing.T, c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
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
