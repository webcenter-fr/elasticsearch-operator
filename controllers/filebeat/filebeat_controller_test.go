package filebeat

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/disaster37/k8s-objectmatcher/patch"
	"github.com/disaster37/operator-sdk-extra/pkg/test"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/apis/beat/v1alpha1"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1alpha1"
	"github.com/webcenter-fr/elasticsearch-operator/apis/shared"
	elasticsearchcontrollers "github.com/webcenter-fr/elasticsearch-operator/controllers/elasticsearch"
	localhelper "github.com/webcenter-fr/elasticsearch-operator/pkg/helper"
	localtest "github.com/webcenter-fr/elasticsearch-operator/pkg/test"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	condition "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (t *FilebeatControllerTestSuite) TestFilebeatController() {
	key := types.NamespacedName{
		Name:      "t-ls-" + localhelper.RandomString(10),
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
			logrus.Infof("=== Add new Filebeat %s/%s ===", key.Namespace, key.Name)

			// First, create Elasticsearch
			es := &elasticsearchcrd.Elasticsearch{
				ObjectMeta: metav1.ObjectMeta{
					Name:      key.Name,
					Namespace: key.Namespace,
				},
				Spec: elasticsearchcrd.ElasticsearchSpec{
					Version: "8.6.0",
					NodeGroups: []elasticsearchcrd.NodeGroupSpec{
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
				if condition.FindStatusCondition(es.Status.Conditions, elasticsearchcontrollers.ElasticsearchCondition) != nil && condition.FindStatusCondition(es.Status.Conditions, elasticsearchcontrollers.ElasticsearchCondition).Reason != "Initialize" {
					return nil
				}

				return errors.New("Not yet created")

			}, time.Second*30, time.Second*1)

			if err != nil || isTimeout {
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
					ElasticsearchRef: shared.ElasticsearchRef{
						ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
							Name: es.Name,
						},
					},
					Deployment: beatcrd.DeploymentSpec{
						Replicas: 1,
					},
					Config: map[string]string{
						"filebeat.yml": `
pipeline.workers: 2
queue.type: persisted
`,
					},
					Module: map[string]string{
						"test.conf": "test",
					},
					Ingresses: []beatcrd.Ingress{
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
				s   *corev1.Secret
				svc *corev1.Service
				i   *networkingv1.Ingress
				cm  *corev1.ConfigMap
				pdb *policyv1.PodDisruptionBudget
				sts *appv1.StatefulSet
			)

			isTimeout, err := localtest.RunWithTimeout(func() error {
				if err := c.Get(context.Background(), key, fb); err != nil {
					t.Fatal("Filebeat not found")
				}

				// In envtest, no kubelet
				// So the Filebeat condition never set as true
				if condition.FindStatusCondition(fb.Status.Conditions, FilebeatCondition) != nil && condition.FindStatusCondition(fb.Status.Conditions, FilebeatCondition).Reason != "Initialize" {
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
			assert.NotEmpty(t, fb.Status.Phase)

			return nil
		},
	}
}

func doUpdateFilebeatStep() test.TestStep {
	return test.TestStep{
		Name: "update",
		Do: func(c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			logrus.Infof("=== Update Filebeat cluster %s/%s ===", key.Namespace, key.Name)

			if o == nil {
				return errors.New("Filebeat is null")
			}
			fb := o.(*beatcrd.Filebeat)

			// Add labels must force to update all resources
			fb.Labels = map[string]string{
				"test": "fu",
			}

			data["lastVersion"] = fb.ResourceVersion

			if err = c.Update(context.Background(), fb); err != nil {
				return err
			}

			time.Sleep(5 * time.Second)

			return nil
		},
		Check: func(t *testing.T, c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			fb := &beatcrd.Filebeat{}

			var (
				s   *corev1.Secret
				svc *corev1.Service
				i   *networkingv1.Ingress
				cm  *corev1.ConfigMap
				pdb *policyv1.PodDisruptionBudget
				sts *appv1.StatefulSet
			)

			lastVersion := data["lastVersion"].(string)

			isTimeout, err := localtest.RunWithTimeout(func() error {
				if err := c.Get(context.Background(), key, fb); err != nil {
					t.Fatal("Filebeat not found")
				}

				// In envtest, no kubelet
				// So the Filebeat condition never set as true
				if lastVersion != fb.ResourceVersion && (fb.Status.Phase == FilebeatPhaseStarting) {
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

			// Services for ingress must exist
			svc = &corev1.Service{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetServiceName(fb, "syslog")}, svc); err != nil {
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
			assert.NotEmpty(t, fb.Status.Phase)

			return nil
		},
	}
}

func doDeleteFilebeatStep() test.TestStep {
	return test.TestStep{
		Name: "delete",
		Do: func(c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			logrus.Infof("=== Delete Filebeat cluster %s/%s ===", key.Namespace, key.Name)

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

			// In envtest, no kubelet
			// So the cascading children delation not works

			return nil
		},
	}
}
