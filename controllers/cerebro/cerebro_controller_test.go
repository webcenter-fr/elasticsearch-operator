package cerebro

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/disaster37/k8s-objectmatcher/patch"
	"github.com/disaster37/operator-sdk-extra/pkg/test"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	cerebrocrd "github.com/webcenter-fr/elasticsearch-operator/apis/cerebro/v1alpha1"
	localhelper "github.com/webcenter-fr/elasticsearch-operator/pkg/helper"
	localtest "github.com/webcenter-fr/elasticsearch-operator/pkg/test"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	condition "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (t *CerebroControllerTestSuite) TestCerebroController() {
	key := types.NamespacedName{
		Name:      "t-cb-" + localhelper.RandomString(10),
		Namespace: "default",
	}
	cb := &cerebrocrd.Cerebro{}
	data := map[string]any{}

	testCase := test.NewTestCase(t.T(), t.k8sClient, key, cb, 5*time.Second, data)
	testCase.Steps = []test.TestStep{
		doCreateCerebroStep(),
		doUpdateCerebroStep(),
		doDeleteCerebroStep(),
	}

	testCase.Run()
}

func doCreateCerebroStep() test.TestStep {
	return test.TestStep{
		Name: "create",
		Do: func(c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			logrus.Infof("=== Add new Cerebro %s/%s ===", key.Namespace, key.Name)

			cb := &cerebrocrd.Cerebro{
				ObjectMeta: metav1.ObjectMeta{
					Name:      key.Name,
					Namespace: key.Namespace,
				},
				Spec: cerebrocrd.CerebroSpec{
					Version: "0.9.4",
					Endpoint: cerebrocrd.CerebroEndpointSpec{
						Ingress: &cerebrocrd.CerebroIngressSpec{
							Enabled: true,
							Host:    "test.cluster.local",
							SecretRef: &corev1.LocalObjectReference{
								Name: "test-tls",
							},
						},
						LoadBalancer: &cerebrocrd.CerebroLoadBalancerSpec{
							Enabled: true,
						},
					},
					Deployment: cerebrocrd.CerebroDeploymentSpec{
						Replicas: 1,
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

			isTimeout, err := localtest.RunWithTimeout(func() error {
				if err := c.Get(context.Background(), key, cb); err != nil {
					t.Fatal("Cerebro not found")
				}

				// In envtest, no kubelet
				// So the condition never set as true
				if condition.FindStatusCondition(cb.Status.Conditions, CerebroCondition) != nil && condition.FindStatusCondition(cb.Status.Conditions, CerebroCondition).Reason != "Initialize" {
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
			assert.NotEmpty(t, cb.Status.Phase)
			assert.NotEmpty(t, cb.Status.Url)
			assert.False(t, *cb.Status.IsError)

			return nil
		},
	}
}

func doUpdateCerebroStep() test.TestStep {
	return test.TestStep{
		Name: "update",
		Do: func(c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			logrus.Infof("=== Update Cerebro cluster %s/%s ===", key.Namespace, key.Name)

			if o == nil {
				return errors.New("Cerebro is null")
			}
			cb := o.(*cerebrocrd.Cerebro)

			// Add labels must force to update all resources
			cb.Labels = map[string]string{
				"test": "fu",
			}

			data["lastVersion"] = cb.ResourceVersion

			if err = c.Update(context.Background(), cb); err != nil {
				return err
			}

			time.Sleep(5 * time.Second)

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

			lastVersion := data["lastVersion"].(string)

			isTimeout, err := localtest.RunWithTimeout(func() error {
				if err := c.Get(context.Background(), key, cb); err != nil {
					t.Fatal("Cerebro not found")
				}

				// In envtest, no kubelet
				// So the condition never set as true
				if lastVersion != cb.ResourceVersion && (cb.Status.Phase == CerebroPhaseStarting) {
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
			assert.NotEmpty(t, cb.Status.Phase)
			assert.NotEmpty(t, cb.Status.Url)
			assert.False(t, *cb.Status.IsError)

			return nil
		},
	}
}

func doDeleteCerebroStep() test.TestStep {
	return test.TestStep{
		Name: "delete",
		Do: func(c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			logrus.Infof("=== Delete Cerebro cluster %s/%s ===", key.Namespace, key.Name)

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

			// In envtest, no kubelet
			// So the cascading children delation not works

			return nil
		},
	}
}