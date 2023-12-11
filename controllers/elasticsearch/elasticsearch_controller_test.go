package elasticsearch

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
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	beatcrd "github.com/webcenter-fr/elasticsearch-operator/apis/beat/v1"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
	elasticsearchapicrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearchapi/v1"
	"github.com/webcenter-fr/elasticsearch-operator/apis/shared"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	condition "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (t *ElasticsearchControllerTestSuite) TestElasticsearchController() {
	key := types.NamespacedName{
		Name:      "t-es-" + helper.RandomString(10),
		Namespace: "default",
	}
	es := &elasticsearchcrd.Elasticsearch{}
	data := map[string]any{}

	testCase := test.NewTestCase(t.T(), t.k8sClient, key, es, 5*time.Second, data)
	testCase.Steps = []test.TestStep{
		doCreateElasticsearchStep(),
		doUpdateElasticsearchStep(),
		doUpdateElasticsearchIncreaseNodeGroupStep(),
		doUpdateElasticsearchDecreaseNodeGroupStep(),
		doUpdateElasticsearchAddLicenseStep(),
		doUpdateElasticsearchAddKeystoreStep(),
		doDeleteElasticsearchStep(),
	}

	testCase.Run()
}

func doCreateElasticsearchStep() test.TestStep {
	return test.TestStep{
		Name: "create",
		Do: func(c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			logrus.Infof("=== Add new Elasticsearch cluster %s/%s ===", key.Namespace, key.Name)

			es := &elasticsearchcrd.Elasticsearch{
				ObjectMeta: metav1.ObjectMeta{
					Name:      key.Name,
					Namespace: key.Namespace,
				},
				Spec: elasticsearchcrd.ElasticsearchSpec{
					Endpoint: elasticsearchcrd.ElasticsearchEndpointSpec{
						Ingress: &elasticsearchcrd.ElasticsearchIngressSpec{
							EndpointIngressSpec: shared.EndpointIngressSpec{
								Enabled: true,
								Host:    "test.cluster.local",
								SecretRef: &corev1.LocalObjectReference{
									Name: "test-tls",
								},
							},
						},
						LoadBalancer: &elasticsearchcrd.ElasticsearchLoadBalancerSpec{
							EndpointLoadBalancerSpec: shared.EndpointLoadBalancerSpec{
								Enabled: true,
							},
						},
					},
					Monitoring: shared.MonitoringSpec{
						Prometheus: &shared.MonitoringPrometheusSpec{
							Enabled: true,
						},
						Metricbeat: &shared.MonitoringMetricbeatSpec{
							Enabled: true,
							ElasticsearchRef: shared.ElasticsearchRef{
								ManagedElasticsearchRef: &shared.ElasticsearchManagedRef{
									Name:      "elastic",
									Namespace: "monitoring",
								},
							},
						},
					},
					NodeGroups: []elasticsearchcrd.ElasticsearchNodeGroupSpec{
						{
							Name: "test",
							Roles: []string{
								"master",
								"data",
								"ingest",
							},
							Deployment: shared.Deployment{
								Replicas: 2,
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
				},
			}

			if err = c.Create(context.Background(), es); err != nil {
				return err
			}

			return nil
		},
		Check: func(t *testing.T, c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			es := &elasticsearchcrd.Elasticsearch{}
			var (
				s          *corev1.Secret
				svc        *corev1.Service
				i          *networkingv1.Ingress
				np         *networkingv1.NetworkPolicy
				cm         *corev1.ConfigMap
				pdb        *policyv1.PodDisruptionBudget
				sts        *appv1.StatefulSet
				dpl        *appv1.Deployment
				user       *elasticsearchapicrd.User
				pm         *monitoringv1.PodMonitor
				metricbeat *beatcrd.Metricbeat
			)

			isTimeout, err := test.RunWithTimeout(func() error {
				if err := c.Get(context.Background(), key, es); err != nil {
					t.Fatal("Elasticsearch not found")
				}

				// In envtest, no kubelet
				// So the Elasticsearch condition never set as true
				if condition.FindStatusCondition(es.Status.Conditions, controller.ReadyCondition.String()) != nil && condition.FindStatusCondition(es.Status.Conditions, controller.ReadyCondition.String()).Reason != "Initialize" {
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

			// Network policy must exist
			np = &networkingv1.NetworkPolicy{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetNetworkPolicyName(es)}, np); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, np.OwnerReferences)
			assert.NotEmpty(t, np.Annotations[patch.LastAppliedConfig])

			// Statefulset musts exist
			for _, nodeGroup := range es.Spec.NodeGroups {
				sts = &appv1.StatefulSet{}
				if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetNodeGroupName(es, nodeGroup.Name)}, sts); err != nil {
					t.Fatal(err)
				}
				assert.NotEmpty(t, sts.OwnerReferences)
				assert.NotEmpty(t, sts.Annotations[patch.LastAppliedConfig])
			}

			// Users musts exist
			userList := []string{
				GetUserSystemName(es, "kibana_system"),
				GetUserSystemName(es, "logstash_system"),
				GetUserSystemName(es, "beats_system"),
				GetUserSystemName(es, "apm_system"),
				GetUserSystemName(es, "remote_monitoring_user"),
			}
			for _, name := range userList {
				user = &elasticsearchapicrd.User{}
				if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: name}, user); err != nil {
					t.Fatal(err)
				}
				assert.NotEmpty(t, user.OwnerReferences)
				assert.NotEmpty(t, user.Annotations[patch.LastAppliedConfig])
			}

			// Exporter must exist
			dpl = &appv1.Deployment{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetExporterDeployementName(es)}, dpl); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, dpl.OwnerReferences)
			assert.NotEmpty(t, dpl.Annotations[patch.LastAppliedConfig])

			// Pod monitor must exist
			pm = &monitoringv1.PodMonitor{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetPodMonitorName(es)}, pm); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, pm.OwnerReferences)
			assert.NotEmpty(t, pm.Annotations[patch.LastAppliedConfig])

			// Metricbeat must exist
			metricbeat = &beatcrd.Metricbeat{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetMetricbeatName(es)}, metricbeat); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, metricbeat.OwnerReferences)
			assert.NotEmpty(t, metricbeat.Annotations[patch.LastAppliedConfig])

			// Status must be update
			assert.NotEmpty(t, es.Status.Health)
			assert.NotEmpty(t, es.Status.PhaseName)
			assert.NotEmpty(t, es.Status.Url)
			assert.NotNil(t, es.Status.CredentialsRef)
			assert.False(t, *es.Status.IsOnError)

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
			es := o.(*elasticsearchcrd.Elasticsearch)

			// Add labels must force to update all resources
			es.Labels = map[string]string{
				"test": "fu",
			}

			data["lastVersion"] = es.ResourceVersion

			if err = c.Update(context.Background(), es); err != nil {
				return err
			}

			time.Sleep(5 * time.Second)

			return nil
		},
		Check: func(t *testing.T, c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			es := &elasticsearchcrd.Elasticsearch{}

			var (
				s          *corev1.Secret
				svc        *corev1.Service
				i          *networkingv1.Ingress
				np         *networkingv1.NetworkPolicy
				cm         *corev1.ConfigMap
				pdb        *policyv1.PodDisruptionBudget
				sts        *appv1.StatefulSet
				dpl        *appv1.Deployment
				user       *elasticsearchapicrd.User
				pm         *monitoringv1.PodMonitor
				metricbeat *beatcrd.Metricbeat
			)

			lastVersion := data["lastVersion"].(string)

			isTimeout, err := test.RunWithTimeout(func() error {
				if err := c.Get(context.Background(), key, es); err != nil {
					t.Fatal("Elasticsearch not found")
				}

				// In envtest, no kubelet
				// So the Elasticsearch condition never set as true
				if lastVersion != es.ResourceVersion && (es.Status.PhaseName == controller.StartingPhase) {
					return nil
				}

				return errors.New("Not yet updated")

			}, time.Second*30, time.Second*1)
			if err != nil || isTimeout {
				t.Fatalf("All Elasticsearch step upgrading not finished: %s", err.Error())
			}

			// Secrets for node PKI and certificates must exist
			s = &corev1.Secret{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetSecretNameForPkiTransport(es)}, s); err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, "fu", s.Labels["test"])
			assert.NotEmpty(t, s.Data)
			assert.NotEmpty(t, s.OwnerReferences)
			assert.NotEmpty(t, s.Annotations[patch.LastAppliedConfig])

			s = &corev1.Secret{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetSecretNameForTlsTransport(es)}, s); err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, "fu", s.Labels["test"])
			assert.NotEmpty(t, s.Data)
			assert.NotEmpty(t, s.OwnerReferences)
			assert.NotEmpty(t, s.Annotations[patch.LastAppliedConfig])

			// Secrets for API PKI and certificates must exist
			s = &corev1.Secret{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetSecretNameForPkiApi(es)}, s); err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, "fu", s.Labels["test"])
			assert.NotEmpty(t, s.Data)
			assert.NotEmpty(t, s.OwnerReferences)
			assert.NotEmpty(t, s.Annotations[patch.LastAppliedConfig])

			s = &corev1.Secret{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetSecretNameForTlsApi(es)}, s); err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, "fu", s.Labels["test"])
			assert.NotEmpty(t, s.Data)
			assert.NotEmpty(t, s.OwnerReferences)
			assert.NotEmpty(t, s.Annotations[patch.LastAppliedConfig])

			// Secrets for internal credentials must exist
			s = &corev1.Secret{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetSecretNameForCredentials(es)}, s); err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, "fu", s.Labels["test"])
			assert.NotEmpty(t, s.Data)
			assert.NotEmpty(t, s.OwnerReferences)
			assert.NotEmpty(t, s.Annotations[patch.LastAppliedConfig])

			// Services must exists
			svc = &corev1.Service{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetGlobalServiceName(es)}, svc); err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, "fu", svc.Labels["test"])
			assert.NotEmpty(t, svc.OwnerReferences)
			assert.NotEmpty(t, svc.Annotations[patch.LastAppliedConfig])

			for _, nodeGroup := range es.Spec.NodeGroups {
				svc = &corev1.Service{}
				if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetNodeGroupServiceName(es, nodeGroup.Name)}, svc); err != nil {
					t.Fatal(err)
				}
				assert.Equal(t, "fu", svc.Labels["test"])
				assert.NotEmpty(t, svc.OwnerReferences)
				assert.NotEmpty(t, svc.Annotations[patch.LastAppliedConfig])

				svc = &corev1.Service{}
				if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetNodeGroupServiceNameHeadless(es, nodeGroup.Name)}, svc); err != nil {
					t.Fatal(err)
				}
				assert.Equal(t, "fu", svc.Labels["test"])
				assert.NotEmpty(t, svc.OwnerReferences)
				assert.NotEmpty(t, svc.Annotations[patch.LastAppliedConfig])
			}

			// Load balancer must exist
			svc = &corev1.Service{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetLoadBalancerName(es)}, svc); err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, "fu", svc.Labels["test"])
			assert.NotEmpty(t, svc.OwnerReferences)
			assert.NotEmpty(t, svc.Annotations[patch.LastAppliedConfig])

			// Ingress must exist
			i = &networkingv1.Ingress{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetIngressName(es)}, i); err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, "fu", i.Labels["test"])
			assert.NotEmpty(t, i.OwnerReferences)
			assert.NotEmpty(t, i.Annotations[patch.LastAppliedConfig])

			// ConfigMaps must exist
			for _, nodeGroup := range es.Spec.NodeGroups {
				cm = &corev1.ConfigMap{}
				if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetNodeGroupConfigMapName(es, nodeGroup.Name)}, cm); err != nil {
					t.Fatal(err)
				}
				assert.Equal(t, "fu", cm.Labels["test"])
				assert.NotEmpty(t, cm.OwnerReferences)
				assert.NotEmpty(t, cm.Annotations[patch.LastAppliedConfig])
			}

			// PDB must exist
			for _, nodeGroup := range es.Spec.NodeGroups {
				pdb = &policyv1.PodDisruptionBudget{}
				if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetNodeGroupPDBName(es, nodeGroup.Name)}, pdb); err != nil {
					t.Fatal(err)
				}
				assert.Equal(t, "fu", pdb.Labels["test"])
				assert.NotEmpty(t, pdb.OwnerReferences)
				assert.NotEmpty(t, pdb.Annotations[patch.LastAppliedConfig])
			}

			// Network policy must exist
			np = &networkingv1.NetworkPolicy{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetNetworkPolicyName(es)}, np); err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, "fu", np.Labels["test"])
			assert.NotEmpty(t, np.OwnerReferences)
			assert.NotEmpty(t, np.Annotations[patch.LastAppliedConfig])

			// Statefulset musts exist
			for _, nodeGroup := range es.Spec.NodeGroups {
				sts = &appv1.StatefulSet{}
				if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetNodeGroupName(es, nodeGroup.Name)}, sts); err != nil {
					t.Fatal(err)
				}
				assert.Equal(t, "fu", sts.Labels["test"])
				assert.NotEmpty(t, sts.OwnerReferences)
				assert.NotEmpty(t, sts.Annotations[patch.LastAppliedConfig])
			}

			// Users musts exist
			userList := []string{
				GetUserSystemName(es, "kibana_system"),
				GetUserSystemName(es, "logstash_system"),
				GetUserSystemName(es, "beats_system"),
				GetUserSystemName(es, "apm_system"),
				GetUserSystemName(es, "remote_monitoring_user"),
			}
			for _, name := range userList {
				user = &elasticsearchapicrd.User{}
				if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: name}, user); err != nil {
					t.Fatal(err)
				}
				assert.Equal(t, "fu", user.Labels["test"])
				assert.NotEmpty(t, user.OwnerReferences)
				assert.NotEmpty(t, user.Annotations[patch.LastAppliedConfig])
			}

			// Exporter must exist
			dpl = &appv1.Deployment{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetExporterDeployementName(es)}, dpl); err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, "fu", dpl.Labels["test"])
			assert.NotEmpty(t, dpl.OwnerReferences)
			assert.NotEmpty(t, dpl.Annotations[patch.LastAppliedConfig])

			// Pod monitor must exist
			pm = &monitoringv1.PodMonitor{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetPodMonitorName(es)}, pm); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, pm.OwnerReferences)
			assert.NotEmpty(t, pm.Annotations[patch.LastAppliedConfig])
			assert.Equal(t, "fu", pm.Labels["test"])

			// Metricbeat must exist
			metricbeat = &beatcrd.Metricbeat{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetMetricbeatName(es)}, metricbeat); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, metricbeat.OwnerReferences)
			assert.NotEmpty(t, metricbeat.Annotations[patch.LastAppliedConfig])
			assert.Equal(t, "fu", metricbeat.Labels["test"])

			// Status must be update
			assert.NotEmpty(t, es.Status.Health)
			assert.NotEmpty(t, es.Status.PhaseName)
			assert.NotEmpty(t, es.Status.Url)
			assert.NotNil(t, es.Status.CredentialsRef)
			assert.False(t, *es.Status.IsOnError)

			return nil
		},
	}
}

func doUpdateElasticsearchIncreaseNodeGroupStep() test.TestStep {
	return test.TestStep{
		Name: "update",
		Do: func(c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			logrus.Infof("=== Increase NodeGroup on Elasticsearch cluster %s/%s ===", key.Namespace, key.Name)

			if o == nil {
				return errors.New("Elasticsearch is null")
			}
			es := o.(*elasticsearchcrd.Elasticsearch)

			// Add labels must force to update all resources
			es.Spec.NodeGroups = append(es.Spec.NodeGroups, elasticsearchcrd.ElasticsearchNodeGroupSpec{
				Name: "data",
				Roles: []string{
					"data",
				},
				Deployment: shared.Deployment{
					Replicas: 2,
				},
			})

			data["lastVersion"] = es.ResourceVersion

			if err = c.Update(context.Background(), es); err != nil {
				return err
			}

			time.Sleep(5 * time.Second)

			return nil
		},
		Check: func(t *testing.T, c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			es := &elasticsearchcrd.Elasticsearch{}

			var (
				s          *corev1.Secret
				svc        *corev1.Service
				i          *networkingv1.Ingress
				np         *networkingv1.NetworkPolicy
				cm         *corev1.ConfigMap
				pdb        *policyv1.PodDisruptionBudget
				sts        *appv1.StatefulSet
				dpl        *appv1.Deployment
				user       *elasticsearchapicrd.User
				pm         *monitoringv1.PodMonitor
				metricbeat *beatcrd.Metricbeat
			)

			lastVersion := data["lastVersion"].(string)

			isTimeout, err := test.RunWithTimeout(func() error {
				if err := c.Get(context.Background(), key, es); err != nil {
					t.Fatal("Elasticsearch not found")
				}

				// In envtest, no kubelet
				// So the Elasticsearch condition never set as true
				if lastVersion != es.ResourceVersion && (es.Status.PhaseName == controller.StartingPhase) {
					return nil
				}

				return errors.New("Not yet updated")

			}, time.Second*30, time.Second*1)
			if err != nil || isTimeout {
				t.Fatalf("All Elasticsearch step upgrading not finished: %s", err.Error())
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

			// Network policy must exist
			np = &networkingv1.NetworkPolicy{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetNetworkPolicyName(es)}, np); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, np.OwnerReferences)
			assert.NotEmpty(t, np.Annotations[patch.LastAppliedConfig])

			// Statefulset musts exist
			for _, nodeGroup := range es.Spec.NodeGroups {
				sts = &appv1.StatefulSet{}
				if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetNodeGroupName(es, nodeGroup.Name)}, sts); err != nil {
					t.Fatal(err)
				}
				assert.NotEmpty(t, sts.OwnerReferences)
				assert.NotEmpty(t, sts.Annotations[patch.LastAppliedConfig])
			}

			// Users musts exist
			userList := []string{
				GetUserSystemName(es, "kibana_system"),
				GetUserSystemName(es, "logstash_system"),
				GetUserSystemName(es, "beats_system"),
				GetUserSystemName(es, "apm_system"),
				GetUserSystemName(es, "remote_monitoring_user"),
			}
			for _, name := range userList {
				user = &elasticsearchapicrd.User{}
				if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: name}, user); err != nil {
					t.Fatal(err)
				}
				assert.NotEmpty(t, user.OwnerReferences)
				assert.NotEmpty(t, user.Annotations[patch.LastAppliedConfig])
			}

			// Exporter must exist
			dpl = &appv1.Deployment{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetExporterDeployementName(es)}, dpl); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, dpl.OwnerReferences)
			assert.NotEmpty(t, dpl.Annotations[patch.LastAppliedConfig])

			// Pod monitor must exist
			pm = &monitoringv1.PodMonitor{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetPodMonitorName(es)}, pm); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, pm.OwnerReferences)
			assert.NotEmpty(t, pm.Annotations[patch.LastAppliedConfig])

			// Metricbeat must exist
			metricbeat = &beatcrd.Metricbeat{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetMetricbeatName(es)}, metricbeat); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, metricbeat.OwnerReferences)
			assert.NotEmpty(t, metricbeat.Annotations[patch.LastAppliedConfig])

			// Status must be update
			assert.NotEmpty(t, es.Status.Health)
			assert.NotEmpty(t, es.Status.PhaseName)
			assert.NotEmpty(t, es.Status.Url)
			assert.NotNil(t, es.Status.CredentialsRef)
			assert.False(t, *es.Status.IsOnError)

			return nil
		},
	}
}

func doUpdateElasticsearchDecreaseNodeGroupStep() test.TestStep {
	return test.TestStep{
		Name: "update",
		Do: func(c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			logrus.Infof("=== Decrease nodeGroup on Elasticsearch cluster %s/%s ===", key.Namespace, key.Name)

			if o == nil {
				return errors.New("Elasticsearch is null")
			}
			es := o.(*elasticsearchcrd.Elasticsearch)

			data["lastVersion"] = es.ResourceVersion
			data["oldES"] = es.DeepCopy()

			// Add labels must force to update all resources
			es.Spec.NodeGroups = []elasticsearchcrd.ElasticsearchNodeGroupSpec{
				es.Spec.NodeGroups[0],
			}

			if err = c.Update(context.Background(), es); err != nil {
				return err
			}

			time.Sleep(5 * time.Second)

			return nil
		},
		Check: func(t *testing.T, c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			es := &elasticsearchcrd.Elasticsearch{}

			var (
				s          *corev1.Secret
				svc        *corev1.Service
				i          *networkingv1.Ingress
				np         *networkingv1.NetworkPolicy
				cm         *corev1.ConfigMap
				pdb        *policyv1.PodDisruptionBudget
				sts        *appv1.StatefulSet
				dpl        *appv1.Deployment
				user       *elasticsearchapicrd.User
				pm         *monitoringv1.PodMonitor
				metricbeat *beatcrd.Metricbeat
			)

			lastVersion := data["lastVersion"].(string)
			oldES := data["oldES"].(*elasticsearchcrd.Elasticsearch)

			isTimeout, err := test.RunWithTimeout(func() error {
				if err := c.Get(context.Background(), key, es); err != nil {
					t.Fatal("Elasticsearch not found")
				}

				// In envtest, no kubelet
				// So the Elasticsearch condition never set as true
				if lastVersion != es.ResourceVersion && (es.Status.PhaseName == controller.StartingPhase) {
					return nil
				}

				return errors.New("Not yet updated")

			}, time.Second*30, time.Second*1)
			if err != nil || isTimeout {
				t.Fatalf("All Elasticsearch step upgrading not finished: %s", err.Error())
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
			for _, nodeGroup := range oldES.Spec.NodeGroups {
				for _, nodeName := range GetNodeGroupNodeNames(oldES, nodeGroup.Name) {
					if nodeGroup.Name == "data" {
						assert.Empty(t, s.Data[fmt.Sprintf("%s.crt", nodeName)])
						assert.Empty(t, s.Data[fmt.Sprintf("%s.key", nodeName)])
					} else {
						assert.NotEmpty(t, s.Data[fmt.Sprintf("%s.crt", nodeName)])
						assert.NotEmpty(t, s.Data[fmt.Sprintf("%s.key", nodeName)])
					}
				}
			}

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

			for _, nodeGroup := range oldES.Spec.NodeGroups {
				if nodeGroup.Name == "data" {
					svc = &corev1.Service{}
					if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetNodeGroupServiceName(es, nodeGroup.Name)}, svc); err != nil {
						if !k8serrors.IsNotFound(err) {
							t.Fatal(err)
						}
					}

					svc = &corev1.Service{}
					if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetNodeGroupServiceNameHeadless(es, nodeGroup.Name)}, svc); err != nil {
						if !k8serrors.IsNotFound(err) {
							t.Fatal(err)
						}
					}
				} else {
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
				if nodeGroup.Name == "data" {
					cm = &corev1.ConfigMap{}
					if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetNodeGroupConfigMapName(es, nodeGroup.Name)}, cm); err != nil {
						if !k8serrors.IsNotFound(err) {
							t.Fatal(err)
						}
					}
				} else {
					cm = &corev1.ConfigMap{}
					if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetNodeGroupConfigMapName(es, nodeGroup.Name)}, cm); err != nil {
						t.Fatal(err)
					}
					assert.NotEmpty(t, cm.OwnerReferences)
					assert.NotEmpty(t, cm.Annotations[patch.LastAppliedConfig])
				}

			}

			// PDB must exist
			for _, nodeGroup := range es.Spec.NodeGroups {
				if nodeGroup.Name == "data" {
					pdb = &policyv1.PodDisruptionBudget{}
					if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetNodeGroupPDBName(es, nodeGroup.Name)}, pdb); err != nil {
						if !k8serrors.IsNotFound(err) {
							t.Fatal(err)
						}
					}
				} else {
					pdb = &policyv1.PodDisruptionBudget{}
					if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetNodeGroupPDBName(es, nodeGroup.Name)}, pdb); err != nil {
						t.Fatal(err)
					}
					assert.NotEmpty(t, pdb.OwnerReferences)
					assert.NotEmpty(t, pdb.Annotations[patch.LastAppliedConfig])
				}

			}

			// Network policy must exist
			np = &networkingv1.NetworkPolicy{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetNetworkPolicyName(es)}, np); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, np.OwnerReferences)
			assert.NotEmpty(t, np.Annotations[patch.LastAppliedConfig])

			// Statefulset musts exist
			for _, nodeGroup := range es.Spec.NodeGroups {
				if nodeGroup.Name == "data" {
					sts = &appv1.StatefulSet{}
					if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetNodeGroupName(es, nodeGroup.Name)}, sts); err != nil {
						if !k8serrors.IsNotFound(err) {
							t.Fatal(err)
						}
					}
				} else {
					sts = &appv1.StatefulSet{}
					if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetNodeGroupName(es, nodeGroup.Name)}, sts); err != nil {
						t.Fatal(err)
					}
					assert.NotEmpty(t, sts.OwnerReferences)
					assert.NotEmpty(t, sts.Annotations[patch.LastAppliedConfig])
				}

			}

			// Users musts exist
			userList := []string{
				GetUserSystemName(es, "kibana_system"),
				GetUserSystemName(es, "logstash_system"),
				GetUserSystemName(es, "beats_system"),
				GetUserSystemName(es, "apm_system"),
				GetUserSystemName(es, "remote_monitoring_user"),
			}
			for _, name := range userList {
				user = &elasticsearchapicrd.User{}
				if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: name}, user); err != nil {
					t.Fatal(err)
				}
				assert.Equal(t, "fu", user.Labels["test"])
				assert.NotEmpty(t, user.OwnerReferences)
				assert.NotEmpty(t, user.Annotations[patch.LastAppliedConfig])
			}

			// Exporter must exist
			dpl = &appv1.Deployment{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetExporterDeployementName(es)}, dpl); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, dpl.OwnerReferences)
			assert.NotEmpty(t, dpl.Annotations[patch.LastAppliedConfig])

			// Pod monitor must exist
			pm = &monitoringv1.PodMonitor{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetPodMonitorName(es)}, pm); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, pm.OwnerReferences)
			assert.NotEmpty(t, pm.Annotations[patch.LastAppliedConfig])

			// Metricbeat must exist
			metricbeat = &beatcrd.Metricbeat{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetMetricbeatName(es)}, metricbeat); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, metricbeat.OwnerReferences)
			assert.NotEmpty(t, metricbeat.Annotations[patch.LastAppliedConfig])

			// Status must be update
			assert.NotEmpty(t, es.Status.Health)
			assert.NotEmpty(t, es.Status.PhaseName)
			assert.NotEmpty(t, es.Status.Url)
			assert.NotNil(t, es.Status.CredentialsRef)
			assert.False(t, *es.Status.IsOnError)

			return nil
		},
	}
}

func doUpdateElasticsearchAddLicenseStep() test.TestStep {
	return test.TestStep{
		Name: "update",
		Do: func(c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			logrus.Infof("=== Add license on Elasticsearch cluster %s/%s ===", key.Namespace, key.Name)

			if o == nil {
				return errors.New("Elasticsearch is null")
			}
			es := o.(*elasticsearchcrd.Elasticsearch)

			data["lastVersion"] = es.ResourceVersion
			data["oldES"] = es.DeepCopy()

			// Add secret that contain license
			s := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "license",
					Namespace: es.Namespace,
				},
				Type: corev1.SecretTypeOpaque,
				Data: map[string][]byte{
					"license": []byte(`{"license": "fake"}`),
				},
			}
			if err = c.Create(context.Background(), s); err != nil {
				return err
			}

			es.Spec.LicenseSecretRef = &corev1.LocalObjectReference{
				Name: "license",
			}

			if err = c.Update(context.Background(), es); err != nil {
				return err
			}

			time.Sleep(5 * time.Second)

			return nil
		},
		Check: func(t *testing.T, c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			es := &elasticsearchcrd.Elasticsearch{}

			var (
				s          *corev1.Secret
				svc        *corev1.Service
				i          *networkingv1.Ingress
				np         *networkingv1.NetworkPolicy
				cm         *corev1.ConfigMap
				pdb        *policyv1.PodDisruptionBudget
				sts        *appv1.StatefulSet
				dpl        *appv1.Deployment
				user       *elasticsearchapicrd.User
				license    *elasticsearchapicrd.License
				pm         *monitoringv1.PodMonitor
				metricbeat *beatcrd.Metricbeat
			)

			lastVersion := data["lastVersion"].(string)
			oldES := data["oldES"].(*elasticsearchcrd.Elasticsearch)

			isTimeout, err := test.RunWithTimeout(func() error {
				if err := c.Get(context.Background(), key, es); err != nil {
					t.Fatal("Elasticsearch not found")
				}

				// In envtest, no kubelet
				// So the Elasticsearch condition never set as true
				if lastVersion != es.ResourceVersion && (es.Status.PhaseName == controller.StartingPhase) {
					return nil
				}

				return errors.New("Not yet updated")

			}, time.Second*30, time.Second*1)
			if err != nil || isTimeout {
				t.Fatalf("All Elasticsearch step upgrading not finished: %s", err.Error())
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
			for _, nodeGroup := range oldES.Spec.NodeGroups {
				for _, nodeName := range GetNodeGroupNodeNames(oldES, nodeGroup.Name) {
					if nodeGroup.Name == "data" {
						assert.Empty(t, s.Data[fmt.Sprintf("%s.crt", nodeName)])
						assert.Empty(t, s.Data[fmt.Sprintf("%s.key", nodeName)])
					} else {
						assert.NotEmpty(t, s.Data[fmt.Sprintf("%s.crt", nodeName)])
						assert.NotEmpty(t, s.Data[fmt.Sprintf("%s.key", nodeName)])
					}
				}
			}

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

			for _, nodeGroup := range oldES.Spec.NodeGroups {
				if nodeGroup.Name == "data" {
					svc = &corev1.Service{}
					if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetNodeGroupServiceName(es, nodeGroup.Name)}, svc); err != nil {
						if !k8serrors.IsNotFound(err) {
							t.Fatal(err)
						}
					}

					svc = &corev1.Service{}
					if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetNodeGroupServiceNameHeadless(es, nodeGroup.Name)}, svc); err != nil {
						if !k8serrors.IsNotFound(err) {
							t.Fatal(err)
						}
					}
				} else {
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
				if nodeGroup.Name == "data" {
					cm = &corev1.ConfigMap{}
					if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetNodeGroupConfigMapName(es, nodeGroup.Name)}, cm); err != nil {
						if !k8serrors.IsNotFound(err) {
							t.Fatal(err)
						}
					}
				} else {
					cm = &corev1.ConfigMap{}
					if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetNodeGroupConfigMapName(es, nodeGroup.Name)}, cm); err != nil {
						t.Fatal(err)
					}
					assert.NotEmpty(t, cm.OwnerReferences)
					assert.NotEmpty(t, cm.Annotations[patch.LastAppliedConfig])
				}

			}

			// PDB must exist
			for _, nodeGroup := range es.Spec.NodeGroups {
				if nodeGroup.Name == "data" {
					pdb = &policyv1.PodDisruptionBudget{}
					if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetNodeGroupPDBName(es, nodeGroup.Name)}, pdb); err != nil {
						if !k8serrors.IsNotFound(err) {
							t.Fatal(err)
						}
					}
				} else {
					pdb = &policyv1.PodDisruptionBudget{}
					if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetNodeGroupPDBName(es, nodeGroup.Name)}, pdb); err != nil {
						t.Fatal(err)
					}
					assert.NotEmpty(t, pdb.OwnerReferences)
					assert.NotEmpty(t, pdb.Annotations[patch.LastAppliedConfig])
				}

			}

			// Network policy must exist
			np = &networkingv1.NetworkPolicy{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetNetworkPolicyName(es)}, np); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, np.OwnerReferences)
			assert.NotEmpty(t, np.Annotations[patch.LastAppliedConfig])

			// Statefulset musts exist
			for _, nodeGroup := range es.Spec.NodeGroups {
				if nodeGroup.Name == "data" {
					sts = &appv1.StatefulSet{}
					if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetNodeGroupName(es, nodeGroup.Name)}, sts); err != nil {
						if !k8serrors.IsNotFound(err) {
							t.Fatal(err)
						}
					}
				} else {
					sts = &appv1.StatefulSet{}
					if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetNodeGroupName(es, nodeGroup.Name)}, sts); err != nil {
						t.Fatal(err)
					}
					assert.NotEmpty(t, sts.OwnerReferences)
					assert.NotEmpty(t, sts.Annotations[patch.LastAppliedConfig])
				}

			}

			// Users musts exist
			userList := []string{
				GetUserSystemName(es, "kibana_system"),
				GetUserSystemName(es, "logstash_system"),
				GetUserSystemName(es, "beats_system"),
				GetUserSystemName(es, "apm_system"),
				GetUserSystemName(es, "remote_monitoring_user"),
			}
			for _, name := range userList {
				user = &elasticsearchapicrd.User{}
				if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: name}, user); err != nil {
					t.Fatal(err)
				}
				assert.NotEmpty(t, user.OwnerReferences)
				assert.NotEmpty(t, user.Annotations[patch.LastAppliedConfig])
			}

			// License must exist
			license = &elasticsearchapicrd.License{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetLicenseName(es)}, license); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, license.OwnerReferences)
			assert.NotEmpty(t, license.Annotations[patch.LastAppliedConfig])

			// Exporter must exist
			dpl = &appv1.Deployment{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetExporterDeployementName(es)}, dpl); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, dpl.OwnerReferences)
			assert.NotEmpty(t, dpl.Annotations[patch.LastAppliedConfig])

			// Pod monitor must exist
			pm = &monitoringv1.PodMonitor{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetPodMonitorName(es)}, pm); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, pm.OwnerReferences)
			assert.NotEmpty(t, pm.Annotations[patch.LastAppliedConfig])

			// Metricbeat must exist
			metricbeat = &beatcrd.Metricbeat{}
			if err = c.Get(context.Background(), types.NamespacedName{Namespace: key.Namespace, Name: GetMetricbeatName(es)}, metricbeat); err != nil {
				t.Fatal(err)
			}
			assert.NotEmpty(t, metricbeat.OwnerReferences)
			assert.NotEmpty(t, metricbeat.Annotations[patch.LastAppliedConfig])

			// Status must be update
			assert.NotEmpty(t, es.Status.Health)
			assert.NotEmpty(t, es.Status.PhaseName)
			assert.NotEmpty(t, es.Status.Url)
			assert.NotNil(t, es.Status.CredentialsRef)
			assert.False(t, *es.Status.IsOnError)

			return nil
		},
	}
}

func doUpdateElasticsearchAddKeystoreStep() test.TestStep {
	return test.TestStep{
		Name: "update",
		Do: func(c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			logrus.Infof("=== Add keystore and cacerts on Elasticsearch cluster %s/%s ===", key.Namespace, key.Name)

			if o == nil {
				return errors.New("Elasticsearch is null")
			}
			es := o.(*elasticsearchcrd.Elasticsearch)

			data["lastVersion"] = es.ResourceVersion
			// Add secret that contain keysyore secret
			s := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "keystore",
					Namespace: es.Namespace,
				},
				Type: corev1.SecretTypeOpaque,
				Data: map[string][]byte{
					"s3.client.default.access_key": []byte(`access_key`),
					"s3.client.default.secret_key": []byte(`secret_key`),
				},
			}
			if err = c.Create(context.Background(), s); err != nil {
				return err
			}

			// Add secret that contain keysyore secret
			s = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "custom-ca",
					Namespace: es.Namespace,
				},
				Type: corev1.SecretTypeOpaque,
				Data: map[string][]byte{
					"custom-ca.crt": []byte(`my-cert`),
				},
			}
			if err = c.Create(context.Background(), s); err != nil {
				return err
			}

			es.Spec.GlobalNodeGroup.KeystoreSecretRef = &corev1.LocalObjectReference{
				Name: "keystore",
			}
			es.Spec.GlobalNodeGroup.CacertsSecretRef = &corev1.LocalObjectReference{
				Name: "custom-ca",
			}

			if err = c.Update(context.Background(), es); err != nil {
				return err
			}

			time.Sleep(5 * time.Second)

			return nil
		},
		Check: func(t *testing.T, c client.Client, key types.NamespacedName, o client.Object, data map[string]any) (err error) {
			es := &elasticsearchcrd.Elasticsearch{}

			lastVersion := data["lastVersion"].(string)

			isTimeout, err := test.RunWithTimeout(func() error {
				if err := c.Get(context.Background(), key, es); err != nil {
					t.Fatal("Elasticsearch not found")
				}

				// In envtest, no kubelet
				// So the Elasticsearch condition never set as true
				if lastVersion != es.ResourceVersion && (es.Status.PhaseName == controller.StartingPhase) {
					return nil
				}

				return errors.New("Not yet updated")

			}, time.Second*30, time.Second*1)
			if err != nil || isTimeout {
				t.Fatalf("All Elasticsearch step upgrading not finished: %s", err.Error())
			}

			// Status must be update
			assert.NotEmpty(t, es.Status.Health)
			assert.NotEmpty(t, es.Status.PhaseName)
			assert.NotEmpty(t, es.Status.Url)
			assert.NotNil(t, es.Status.CredentialsRef)
			assert.False(t, *es.Status.IsOnError)

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
				return errors.New("Elasticsearch is null")
			}
			es := o.(*elasticsearchcrd.Elasticsearch)

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
