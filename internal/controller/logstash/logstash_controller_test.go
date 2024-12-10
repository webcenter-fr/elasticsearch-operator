package logstash

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/disaster37/k8s-objectmatcher/patch"
	"github.com/disaster37/operator-sdk-extra/pkg/helper"
	"github.com/disaster37/operator-sdk-extra/pkg/test"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/apis/logstash/v1"
	"github.com/webcenter-fr/elasticsearch-operator/apis/shared"
	sharedcrd "github.com/webcenter-fr/elasticsearch-operator/apis/shared"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (t *LogstashControllerTestSuite) TestLogstashController() {
	key := types.NamespacedName{
		Name:      "t-ls-" + helper.RandomString(10),
		Namespace: "default",
	}
	ls := &logstashcrd.Logstash{}
	data := map[string]any{}

	testCase := test.NewTestCase(t.T(), t.k8sClient, key, ls, 5*time.Second, data)
	testCase.Steps = []test.TestStep{
		doCreateLogstashStep(),
		doUpdateLogstashStep(),
		doDeleteLogstashStep(),
	}

	testCase.Run()
}

func doCreateLogstashStep() test.TestStep {
	return test.TestStep{
		Name: "create",
		Do: func(c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			logrus.Infof("=== Add new Logstash %s/%s ===\n\n", key.Namespace, key.Name)

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

			pathType := networkingv1.PathTypePrefix
			ls := &logstashcrd.Logstash{
				ObjectMeta: metav1.ObjectMeta{
					Name:      key.Name,
					Namespace: key.Namespace,
				},
				Spec: logstashcrd.LogstashSpec{
					Version: "8.6.0",
					ElasticsearchRef: sharedcrd.ElasticsearchRef{
						ManagedElasticsearchRef: &sharedcrd.ElasticsearchManagedRef{
							Name: es.Name,
						},
					},
					Deployment: logstashcrd.LogstashDeploymentSpec{
						Deployment: sharedcrd.Deployment{
							Replicas: 2,
						},
					},
					Config: map[string]string{
						"logstash.yml": `
pipeline.workers: 2
queue.type: persisted
`,
					},
					Pipeline: map[string]string{
						"test.conf": "test",
					},
					Pattern: map[string]string{
						"pattern.conf": "test",
					},
					Ingresses: []shared.Ingress{
						{
							Name:                  "filebeat",
							ContainerPort:         5003,
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
					Pki: logstashcrd.LogstashPkiSpec{
						Enabled: ptr.To[bool](true),
						Tls: map[string]logstashcrd.LogstashTlsSpec{
							"filebeat": {
								Consumer: "filebeat",
								TlsSelfSignedCertificateSpec: sharedcrd.TlsSelfSignedCertificateSpec{
									AltNames: []string{"*.domain.com"},
								},
							},
						},
					},
				},
			}

			if err = c.Create(context.Background(), ls); err != nil {
				return err
			}

			return nil
		},
		Check: func(t *testing.T, c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			ls := &logstashcrd.Logstash{}
			var (
				s   *corev1.Secret
				svc *corev1.Service
				i   *networkingv1.Ingress
				cm  *corev1.ConfigMap
				pdb *policyv1.PodDisruptionBudget
				sts *appv1.StatefulSet
			)

			isTimeout, err := test.RunWithTimeout(func() error {
				if err := c.Get(context.Background(), key, ls); err != nil {
					t.Fatal("Logstash not found")
				}

				if ls.GetStatus().GetObservedGeneration() > 0 {
					return nil
				}

				return errors.New("Not yet created")

			}, time.Second*30, time.Second*1)
			if err != nil || isTimeout {
				t.Fatalf("All Logstash step provisionning not finished: %s", err.Error())
			}

			// Secrets for CA Elasticsearch
			s = &corev1.Secret{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetSecretNameForCAElasticsearch(ls)}, s); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, s.Data)
			assert.NotEmpty(t, s.OwnerReferences)
			assert.NotEmpty(t, s.Annotations[patch.LastAppliedConfig])

			// Secrets for credentials must exist
			s = &corev1.Secret{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetSecretNameForCredentials(ls)}, s); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, s.Data)
			assert.NotEmpty(t, s.OwnerReferences)
			assert.NotEmpty(t, s.Annotations[patch.LastAppliedConfig])

			// Secrets for Pki
			s = &corev1.Secret{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetSecretNameForPki(ls)}, s); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, s.Data)
			assert.NotEmpty(t, s.OwnerReferences)
			assert.NotEmpty(t, s.Annotations[patch.LastAppliedConfig])

			// Secrets for certificates
			s = &corev1.Secret{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetSecretNameForTls(ls)}, s); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, s.Data)
			assert.NotEmpty(t, s.OwnerReferences)
			assert.NotEmpty(t, s.Annotations[patch.LastAppliedConfig])

			// Services for ingress must exist
			svc = &corev1.Service{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetServiceName(ls, "filebeat")}, svc); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, svc.OwnerReferences)
			assert.NotEmpty(t, svc.Annotations[patch.LastAppliedConfig])

			// Global service must exist
			svc = &corev1.Service{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetGlobalServiceName(ls)}, svc); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, svc.OwnerReferences)
			assert.NotEmpty(t, svc.Annotations[patch.LastAppliedConfig])

			// Ingress must exist
			i = &networkingv1.Ingress{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetIngressName(ls, "filebeat")}, i); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, i.OwnerReferences)
			assert.NotEmpty(t, i.Annotations[patch.LastAppliedConfig])

			// ConfigMaps must exist
			cm = &corev1.ConfigMap{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetConfigMapConfigName(ls)}, cm); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, cm.OwnerReferences)
			assert.NotEmpty(t, cm.Annotations[patch.LastAppliedConfig])

			cm = &corev1.ConfigMap{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetConfigMapPipelineName(ls)}, cm); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, cm.OwnerReferences)
			assert.NotEmpty(t, cm.Annotations[patch.LastAppliedConfig])

			cm = &corev1.ConfigMap{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetConfigMapPatternName(ls)}, cm); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, cm.OwnerReferences)
			assert.NotEmpty(t, cm.Annotations[patch.LastAppliedConfig])

			// PDB must exist
			pdb = &policyv1.PodDisruptionBudget{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetPDBName(ls)}, pdb); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, pdb.OwnerReferences)
			assert.NotEmpty(t, pdb.Annotations[patch.LastAppliedConfig])

			// Statefulset musts exist
			sts = &appv1.StatefulSet{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetStatefulsetName(ls)}, sts); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, sts.OwnerReferences)
			assert.NotEmpty(t, sts.Annotations[patch.LastAppliedConfig])

			// Status must be update
			assert.NotEmpty(t, ls.Status.PhaseName)
			assert.False(t, *ls.Status.IsOnError)

			return nil
		},
	}
}

func doUpdateLogstashStep() test.TestStep {
	return test.TestStep{
		Name: "update",
		Do: func(c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			logrus.Infof("=== Update Logstash cluster %s/%s ===\n\n", key.Namespace, key.Name)

			if o == nil {
				return errors.New("Logstash is null")
			}
			ls := o.(*logstashcrd.Logstash)

			// Add labels must force to update all resources
			ls.Labels = map[string]string{
				"test": "fu",
			}
			// Change spec to track generation
			ls.Spec.Deployment.Labels = map[string]string{
				"test": "fu",
			}

			data["lastGeneration"] = ls.GetStatus().GetObservedGeneration()

			if err = c.Update(context.Background(), ls); err != nil {
				return err
			}

			return nil
		},
		Check: func(t *testing.T, c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			ls := &logstashcrd.Logstash{}

			var (
				s   *corev1.Secret
				svc *corev1.Service
				i   *networkingv1.Ingress
				cm  *corev1.ConfigMap
				pdb *policyv1.PodDisruptionBudget
				sts *appv1.StatefulSet
			)

			lastGeneration := data["lastGeneration"].(int64)

			isTimeout, err := test.RunWithTimeout(func() error {
				if err := c.Get(context.Background(), key, ls); err != nil {
					t.Fatal("Logstash not found")
				}

				if lastGeneration < ls.GetStatus().GetObservedGeneration() {
					return nil
				}

				return errors.New("Not yet updated")

			}, time.Second*30, time.Second*1)
			if err != nil || isTimeout {
				t.Fatalf("All Logstash step upgrading not finished: %s", err.Error())
			}

			// Secrets for CA Elasticsearch
			s = &corev1.Secret{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetSecretNameForCAElasticsearch(ls)}, s); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, s.Data)
			assert.NotEmpty(t, s.OwnerReferences)
			assert.NotEmpty(t, s.Annotations[patch.LastAppliedConfig])
			assert.Equal(t, "fu", s.Labels["test"])

			// Secrets for credentials must exist
			s = &corev1.Secret{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetSecretNameForCredentials(ls)}, s); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, s.Data)
			assert.NotEmpty(t, s.OwnerReferences)
			assert.NotEmpty(t, s.Annotations[patch.LastAppliedConfig])
			assert.Equal(t, "fu", s.Labels["test"])

			// Secrets for Pki
			s = &corev1.Secret{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetSecretNameForPki(ls)}, s); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, s.Data)
			assert.NotEmpty(t, s.OwnerReferences)
			assert.NotEmpty(t, s.Annotations[patch.LastAppliedConfig])
			assert.Equal(t, "fu", s.Labels["test"])

			// Secrets for certificates
			s = &corev1.Secret{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetSecretNameForTls(ls)}, s); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, s.Data)
			assert.NotEmpty(t, s.OwnerReferences)
			assert.NotEmpty(t, s.Annotations[patch.LastAppliedConfig])
			assert.Equal(t, "fu", s.Labels["test"])

			// Services for ingress must exist
			svc = &corev1.Service{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetServiceName(ls, "filebeat")}, svc); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, svc.OwnerReferences)
			assert.NotEmpty(t, svc.Annotations[patch.LastAppliedConfig])
			assert.Equal(t, "fu", svc.Labels["test"])

			// Global service must exist
			svc = &corev1.Service{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetGlobalServiceName(ls)}, svc); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, svc.OwnerReferences)
			assert.NotEmpty(t, svc.Annotations[patch.LastAppliedConfig])
			assert.Equal(t, "fu", svc.Labels["test"])

			// Ingress must exist
			i = &networkingv1.Ingress{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetIngressName(ls, "filebeat")}, i); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, i.OwnerReferences)
			assert.NotEmpty(t, i.Annotations[patch.LastAppliedConfig])
			assert.Equal(t, "fu", i.Labels["test"])

			// ConfigMaps must exist
			cm = &corev1.ConfigMap{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetConfigMapConfigName(ls)}, cm); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, cm.OwnerReferences)
			assert.NotEmpty(t, cm.Annotations[patch.LastAppliedConfig])
			assert.Equal(t, "fu", cm.Labels["test"])

			cm = &corev1.ConfigMap{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetConfigMapPipelineName(ls)}, cm); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, cm.OwnerReferences)
			assert.NotEmpty(t, cm.Annotations[patch.LastAppliedConfig])
			assert.Equal(t, "fu", cm.Labels["test"])

			cm = &corev1.ConfigMap{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetConfigMapPatternName(ls)}, cm); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, cm.OwnerReferences)
			assert.NotEmpty(t, cm.Annotations[patch.LastAppliedConfig])
			assert.Equal(t, "fu", cm.Labels["test"])

			// PDB must exist
			pdb = &policyv1.PodDisruptionBudget{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetPDBName(ls)}, pdb); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, pdb.OwnerReferences)
			assert.NotEmpty(t, pdb.Annotations[patch.LastAppliedConfig])
			assert.Equal(t, "fu", pdb.Labels["test"])

			// Statefulset musts exist
			sts = &appv1.StatefulSet{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetStatefulsetName(ls)}, sts); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, sts.OwnerReferences)
			assert.NotEmpty(t, sts.Annotations[patch.LastAppliedConfig])
			assert.Equal(t, "fu", sts.Labels["test"])

			// Status must be update
			assert.NotEmpty(t, ls.Status.PhaseName)
			assert.False(t, *ls.Status.IsOnError)

			return nil
		},
	}
}

func doDeleteLogstashStep() test.TestStep {
	return test.TestStep{
		Name: "delete",
		Do: func(c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			logrus.Infof("=== Delete Logstash cluster %s/%s ===\n\n", key.Namespace, key.Name)

			if o == nil {
				return errors.New("Logstash is null")
			}
			ls := o.(*logstashcrd.Logstash)

			wait := int64(0)
			if err = c.Delete(context.Background(), ls, &client.DeleteOptions{GracePeriodSeconds: &wait}); err != nil {
				return err
			}

			return nil
		},
		Check: func(t *testing.T, c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {

			ls := &logstashcrd.Logstash{}
			isDeleted := false

			// In envtest, no kubelet
			// So the cascading children delation not works
			isTimeout, err := test.RunWithTimeout(func() error {
				if err = c.Get(context.Background(), key, ls); err != nil {
					if k8serrors.IsNotFound(err) {
						isDeleted = true
						return nil
					}
					t.Fatal(err)
				}

				return errors.New("Not yet deleted")
			}, time.Second*30, time.Second*1)
			if err != nil || isTimeout {
				t.Fatalf("Logstash stil exist: %s", err.Error())
			}

			assert.True(t, isDeleted)

			return nil
		},
	}
}
