package filebeat

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/disaster37/k8s-objectmatcher/patch"
	"github.com/disaster37/operator-sdk-extra/pkg/apis"
	"github.com/disaster37/operator-sdk-extra/pkg/helper"
	"github.com/disaster37/operator-sdk-extra/pkg/test"
	routev1 "github.com/openshift/api/route/v1"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/api/beat/v1"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearch/v1"
	sharedcrd "github.com/webcenter-fr/elasticsearch-operator/api/shared"
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

func (t *FilebeatControllerTestSuite) TestFilebeatController() {
	key := types.NamespacedName{
		Name:      "t-fb-" + helper.RandomString(10),
		Namespace: "default",
	}
	fb := &beatcrd.Filebeat{}
	data := map[string]any{}

	testCase := test.NewTestCase(t.T(), t.k8sClient, key, fb, 5*time.Second, data)
	testCase.Steps = []test.TestStep{
		doCreateFilebeatStep(),
		doUpdateFilebeatStep(),
		doDeleteFilebeatStep(),
	}

	testCase.Run()
}

func doCreateFilebeatStep() test.TestStep {
	return test.TestStep{
		Name: "create",
		Do: func(c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			logrus.Infof("=== Add new Filebeat %s/%s ===\n\n", key.Namespace, key.Name)

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
							Deployment: sharedcrd.Deployment{
								Replicas: 1,
							},
						},
					},
				},
			}

			if err = c.Create(context.Background(), es); err != nil {
				return err
			}

			// Create secret that store credential to connect on Elasticsearch
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      key.Name,
					Namespace: key.Namespace,
				},
				StringData: map[string]string{
					"username": "filebeat",
					"password": "strong password",
				},
			}
			if err = c.Create(context.Background(), secret); err != nil {
				return err
			}

			pathType := networkingv1.PathTypePrefix
			fb := &beatcrd.Filebeat{
				ObjectMeta: metav1.ObjectMeta{
					Name:      key.Name,
					Namespace: key.Namespace,
				},
				Spec: beatcrd.FilebeatSpec{
					Version: "8.6.0",
					ElasticsearchRef: sharedcrd.ElasticsearchRef{
						ManagedElasticsearchRef: &sharedcrd.ElasticsearchManagedRef{
							Name: es.Name,
						},
						SecretRef: &corev1.LocalObjectReference{
							Name: key.Name,
						},
					},
					Deployment: beatcrd.FilebeatDeploymentSpec{
						Deployment: sharedcrd.Deployment{
							Replicas: 2,
						},
					},
					Config: &apis.MapAny{
						Data: map[string]any{
							"pipeline.workers": 2,
							"queue.type":       "persisted",
						},
					},
					Modules: &apis.MapAny{
						Data: map[string]any{
							"module.yaml": map[string]any{
								"foo": "bar",
							},
						},
					},
					Ingresses: []sharedcrd.Ingress{
						{
							Name:                  "syslog",
							ContainerPort:         601,
							ContainerPortProtocol: corev1.ProtocolTCP,
							Spec: networkingv1.IngressSpec{
								Rules: []networkingv1.IngressRule{
									{
										Host: "filebeat.cluster.local",
										IngressRuleValue: networkingv1.IngressRuleValue{
											HTTP: &networkingv1.HTTPIngressRuleValue{
												Paths: []networkingv1.HTTPIngressPath{
													{
														Path:     "/",
														PathType: &pathType,
														Backend:  networkingv1.IngressBackend{},
													},
												},
											},
										},
									},
								},
							},
						},
					},
					Routes: []sharedcrd.Route{
						{
							Name:                  "syslog2",
							ContainerPort:         601,
							ContainerPortProtocol: corev1.ProtocolTCP,
							Spec: routev1.RouteSpec{
								Host: "filebeat.cluster.local",
								Path: "/",
								TLS: &routev1.TLSConfig{
									Termination: routev1.TLSTerminationEdge,
								},
							},
						},
					},
					Pki: beatcrd.FilebeatPkiSpec{
						Enabled: ptr.To[bool](true),
						Tls: map[string]sharedcrd.TlsSelfSignedCertificateSpec{
							"nxlog": {
								AltNames: []string{"*.domain.com"},
							},
						},
					},
				},
			}

			if err = c.Create(context.Background(), fb); err != nil {
				return err
			}

			return nil
		},
		Check: func(t *testing.T, c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			fb := &beatcrd.Filebeat{}
			var (
				s              *corev1.Secret
				svc            *corev1.Service
				i              *networkingv1.Ingress
				cm             *corev1.ConfigMap
				pdb            *policyv1.PodDisruptionBudget
				sts            *appv1.StatefulSet
				route          *routev1.Route
				serviceAccount *corev1.ServiceAccount
				roleBinding    *rbacv1.RoleBinding
			)

			isTimeout, err := test.RunWithTimeout(func() error {
				if err := c.Get(context.Background(), key, fb); err != nil {
					t.Fatal("Filebeat not found")
				}

				if fb.GetStatus().GetObservedGeneration() > 0 {
					return nil
				}

				return errors.New("Not yet created")
			}, time.Second*30, time.Second*1)
			if err != nil || isTimeout {
				t.Fatalf("All Filebeat step provisionning not finished: %s", err.Error())
			}

			// Secrets for CA Elasticsearch
			s = &corev1.Secret{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetSecretNameForCAElasticsearch(fb)}, s); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, s.Data)
			assert.NotEmpty(t, s.OwnerReferences)
			assert.NotEmpty(t, s.Annotations[patch.LastAppliedConfig])

			// Secrets for Pki
			s = &corev1.Secret{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetSecretNameForPki(fb)}, s); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, s.Data)
			assert.NotEmpty(t, s.OwnerReferences)
			assert.NotEmpty(t, s.Annotations[patch.LastAppliedConfig])

			// Secrets for certificates
			s = &corev1.Secret{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetSecretNameForTls(fb)}, s); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, s.Data)
			assert.NotEmpty(t, s.OwnerReferences)
			assert.NotEmpty(t, s.Annotations[patch.LastAppliedConfig])

			// Secrets for credentials must exist
			s = &corev1.Secret{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetSecretNameForCredentials(fb)}, s); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, s.Data)
			assert.NotEmpty(t, s.OwnerReferences)
			assert.NotEmpty(t, s.Annotations[patch.LastAppliedConfig])

			// Services for ingress must exist
			svc = &corev1.Service{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetServiceName(fb, "syslog")}, svc); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, svc.OwnerReferences)
			assert.NotEmpty(t, svc.Annotations[patch.LastAppliedConfig])

			// Services for route must exist
			svc = &corev1.Service{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetServiceName(fb, "syslog2")}, svc); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, svc.OwnerReferences)
			assert.NotEmpty(t, svc.Annotations[patch.LastAppliedConfig])

			// Global service must exist
			svc = &corev1.Service{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetGlobalServiceName(fb)}, svc); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, svc.OwnerReferences)
			assert.NotEmpty(t, svc.Annotations[patch.LastAppliedConfig])

			// Ingress must exist
			i = &networkingv1.Ingress{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetIngressName(fb, "syslog")}, i); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, i.OwnerReferences)
			assert.NotEmpty(t, i.Annotations[patch.LastAppliedConfig])

			// Route must exist
			route = &routev1.Route{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetIngressName(fb, "syslog2")}, route); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, route.OwnerReferences)
			assert.NotEmpty(t, route.Annotations[patch.LastAppliedConfig])

			// Service Account must exist
			serviceAccount = &corev1.ServiceAccount{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetServiceAccountName(fb)}, serviceAccount); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, serviceAccount.OwnerReferences)
			assert.NotEmpty(t, serviceAccount.Annotations[patch.LastAppliedConfig])

			// roleBinding must exist
			roleBinding = &rbacv1.RoleBinding{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetServiceAccountName(fb)}, roleBinding); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, roleBinding.OwnerReferences)
			assert.NotEmpty(t, roleBinding.Annotations[patch.LastAppliedConfig])

			// ConfigMaps must exist
			cm = &corev1.ConfigMap{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetConfigMapConfigName(fb)}, cm); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, cm.OwnerReferences)
			assert.NotEmpty(t, cm.Annotations[patch.LastAppliedConfig])

			cm = &corev1.ConfigMap{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetConfigMapModuleName(fb)}, cm); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, cm.OwnerReferences)
			assert.NotEmpty(t, cm.Annotations[patch.LastAppliedConfig])

			// PDB must exist
			pdb = &policyv1.PodDisruptionBudget{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetPDBName(fb)}, pdb); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, pdb.OwnerReferences)
			assert.NotEmpty(t, pdb.Annotations[patch.LastAppliedConfig])

			// Statefulset musts exist
			sts = &appv1.StatefulSet{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetStatefulsetName(fb)}, sts); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, sts.OwnerReferences)
			assert.NotEmpty(t, sts.Annotations[patch.LastAppliedConfig])

			// Status must be update
			assert.NotEmpty(t, fb.Status.PhaseName)
			assert.False(t, *fb.Status.IsOnError)

			return nil
		},
	}
}

func doUpdateFilebeatStep() test.TestStep {
	return test.TestStep{
		Name: "update",
		Do: func(c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			logrus.Infof("=== Update Filebeat cluster %s/%s ===\n\n", key.Namespace, key.Name)

			if o == nil {
				return errors.New("Filebeat is null")
			}
			fb := o.(*beatcrd.Filebeat)

			// Add labels must force to update all resources
			fb.Labels = map[string]string{
				"test": "fu",
			}
			// Change spec to track generation
			fb.Spec.Deployment.Labels = map[string]string{
				"test": "fu",
			}

			data["lastGeneration"] = fb.GetStatus().GetObservedGeneration()

			if err = c.Update(context.Background(), fb); err != nil {
				return err
			}

			return nil
		},
		Check: func(t *testing.T, c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			fb := &beatcrd.Filebeat{}

			var (
				s              *corev1.Secret
				svc            *corev1.Service
				i              *networkingv1.Ingress
				cm             *corev1.ConfigMap
				pdb            *policyv1.PodDisruptionBudget
				sts            *appv1.StatefulSet
				route          *routev1.Route
				serviceAccount *corev1.ServiceAccount
				roleBinding    *rbacv1.RoleBinding
			)

			lastGeneration := data["lastGeneration"].(int64)

			isTimeout, err := test.RunWithTimeout(func() error {
				if err := c.Get(context.Background(), key, fb); err != nil {
					t.Fatal("Filebeat not found")
				}

				if lastGeneration < fb.GetStatus().GetObservedGeneration() {
					return nil
				}

				return errors.New("Not yet updated")
			}, time.Second*30, time.Second*1)
			if err != nil || isTimeout {
				t.Fatalf("All Filebeat step upgrading not finished: %s", err.Error())
			}

			// Secrets for CA Elasticsearch
			s = &corev1.Secret{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetSecretNameForCAElasticsearch(fb)}, s); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, s.Data)
			assert.NotEmpty(t, s.OwnerReferences)
			assert.NotEmpty(t, s.Annotations[patch.LastAppliedConfig])
			assert.Equal(t, "fu", s.Labels["test"])

			// Secrets for credentials must exist
			s = &corev1.Secret{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetSecretNameForCredentials(fb)}, s); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, s.Data)
			assert.NotEmpty(t, s.OwnerReferences)
			assert.NotEmpty(t, s.Annotations[patch.LastAppliedConfig])
			assert.Equal(t, "fu", s.Labels["test"])

			// Secrets for Pki
			s = &corev1.Secret{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetSecretNameForPki(fb)}, s); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, s.Data)
			assert.NotEmpty(t, s.OwnerReferences)
			assert.NotEmpty(t, s.Annotations[patch.LastAppliedConfig])
			assert.Equal(t, "fu", s.Labels["test"])

			// Secrets for certificates
			s = &corev1.Secret{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetSecretNameForTls(fb)}, s); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, s.Data)
			assert.NotEmpty(t, s.OwnerReferences)
			assert.NotEmpty(t, s.Annotations[patch.LastAppliedConfig])
			assert.Equal(t, "fu", s.Labels["test"])

			// Services for ingress must exist
			svc = &corev1.Service{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetServiceName(fb, "syslog")}, svc); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, svc.OwnerReferences)
			assert.NotEmpty(t, svc.Annotations[patch.LastAppliedConfig])
			assert.Equal(t, "fu", svc.Labels["test"])

			// Services for route must exist
			svc = &corev1.Service{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetServiceName(fb, "syslog2")}, svc); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, svc.OwnerReferences)
			assert.NotEmpty(t, svc.Annotations[patch.LastAppliedConfig])
			assert.Equal(t, "fu", svc.Labels["test"])

			// Global service must exist
			svc = &corev1.Service{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetGlobalServiceName(fb)}, svc); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, svc.OwnerReferences)
			assert.NotEmpty(t, svc.Annotations[patch.LastAppliedConfig])
			assert.Equal(t, "fu", svc.Labels["test"])

			// Ingress must exist
			i = &networkingv1.Ingress{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetIngressName(fb, "syslog")}, i); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, i.OwnerReferences)
			assert.NotEmpty(t, i.Annotations[patch.LastAppliedConfig])
			assert.Equal(t, "fu", i.Labels["test"])

			// Route must exist
			route = &routev1.Route{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetIngressName(fb, "syslog2")}, route); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, route.OwnerReferences)
			assert.NotEmpty(t, route.Annotations[patch.LastAppliedConfig])
			assert.Equal(t, "fu", route.Labels["test"])

			// Service Account must exist
			serviceAccount = &corev1.ServiceAccount{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetServiceAccountName(fb)}, serviceAccount); err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, "fu", serviceAccount.Labels["test"])
			assert.NotEmpty(t, serviceAccount.OwnerReferences)
			assert.NotEmpty(t, serviceAccount.Annotations[patch.LastAppliedConfig])

			// roleBinding must exist
			roleBinding = &rbacv1.RoleBinding{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetServiceAccountName(fb)}, roleBinding); err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, "fu", roleBinding.Labels["test"])
			assert.NotEmpty(t, roleBinding.OwnerReferences)
			assert.NotEmpty(t, roleBinding.Annotations[patch.LastAppliedConfig])

			// ConfigMaps must exist
			cm = &corev1.ConfigMap{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetConfigMapConfigName(fb)}, cm); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, cm.OwnerReferences)
			assert.NotEmpty(t, cm.Annotations[patch.LastAppliedConfig])
			assert.Equal(t, "fu", cm.Labels["test"])

			cm = &corev1.ConfigMap{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetConfigMapModuleName(fb)}, cm); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, cm.OwnerReferences)
			assert.NotEmpty(t, cm.Annotations[patch.LastAppliedConfig])
			assert.Equal(t, "fu", cm.Labels["test"])

			// PDB must exist
			pdb = &policyv1.PodDisruptionBudget{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetPDBName(fb)}, pdb); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, pdb.OwnerReferences)
			assert.NotEmpty(t, pdb.Annotations[patch.LastAppliedConfig])
			assert.Equal(t, "fu", pdb.Labels["test"])

			// Statefulset musts exist
			sts = &appv1.StatefulSet{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetStatefulsetName(fb)}, sts); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, sts.OwnerReferences)
			assert.NotEmpty(t, sts.Annotations[patch.LastAppliedConfig])
			assert.Equal(t, "fu", sts.Labels["test"])

			// Status must be update
			assert.NotEmpty(t, fb.Status.PhaseName)
			assert.False(t, *fb.Status.IsOnError)

			return nil
		},
	}
}

func doDeleteFilebeatStep() test.TestStep {
	return test.TestStep{
		Name: "delete",
		Do: func(c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			logrus.Infof("=== Delete Filebeat cluster %s/%s ===\n\n", key.Namespace, key.Name)

			if o == nil {
				return errors.New("Filebeat is null")
			}
			ls := o.(*beatcrd.Filebeat)

			wait := int64(0)
			if err = c.Delete(context.Background(), ls, &client.DeleteOptions{GracePeriodSeconds: &wait}); err != nil {
				return err
			}

			return nil
		},
		Check: func(t *testing.T, c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			fb := &beatcrd.Filebeat{}
			isDeleted := false

			// In envtest, no kubelet
			// So the cascading children delation not works
			isTimeout, err := test.RunWithTimeout(func() error {
				if err = c.Get(context.Background(), key, fb); err != nil {
					if k8serrors.IsNotFound(err) {
						isDeleted = true
						return nil
					}
					t.Fatal(err)
				}

				return errors.New("Not yet deleted")
			}, time.Second*30, time.Second*1)
			if err != nil || isTimeout {
				t.Fatalf("Filebeat stil exist: %s", err.Error())
			}

			assert.True(t, isDeleted)

			return nil
		},
	}
}
