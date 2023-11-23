package v1

import (
	"testing"

	"github.com/disaster37/operator-sdk-extra/pkg/apis"
	"github.com/stretchr/testify/assert"
	"github.com/webcenter-fr/elasticsearch-operator/apis/shared"
	v1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestGetStatus(t *testing.T) {
	status := ElasticsearchStatus{
		BasicMultiPhaseObjectStatus: apis.BasicMultiPhaseObjectStatus{
			PhaseName: "test",
		},
	}
	o := &Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Status: status,
	}

	assert.Equal(t, &status, o.GetStatus())
}

func TestIsIngressEnabled(t *testing.T) {

	// With default values
	o := &Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: ElasticsearchSpec{},
	}
	assert.False(t, o.IsIngressEnabled())

	// When Ingress is specified but disabled
	o.Spec.Endpoint = ElasticsearchEndpointSpec{
		Ingress: &ElasticsearchIngressSpec{
			EndpointIngressSpec: shared.EndpointIngressSpec{
				Enabled: false,
			},
		},
	}
	assert.False(t, o.IsIngressEnabled())

	// When ingress is enabled
	o.Spec.Endpoint.Ingress.Enabled = true
	assert.True(t, o.IsIngressEnabled())

}

func TestIsLoadBalancerEnabled(t *testing.T) {
	// With default values
	o := &Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: ElasticsearchSpec{},
	}
	assert.False(t, o.IsLoadBalancerEnabled())

	// When Load balancer is specified but disabled
	o.Spec.Endpoint = ElasticsearchEndpointSpec{
		LoadBalancer: &ElasticsearchLoadBalancerSpec{
			EndpointLoadBalancerSpec: shared.EndpointLoadBalancerSpec{
				Enabled: false,
			},
		},
	}
	assert.False(t, o.IsLoadBalancerEnabled())

	// When Load balancer is specified and enabled
	o.Spec.Endpoint.LoadBalancer.Enabled = true
	assert.True(t, o.IsLoadBalancerEnabled())
}

func TestIsSetVMMaxMapCount(t *testing.T) {
	var o *Elasticsearch

	// With default values
	o = &Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: ElasticsearchSpec{},
	}
	assert.True(t, o.IsSetVMMaxMapCount())

	// When enabled
	o = &Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: ElasticsearchSpec{
			SetVMMaxMapCount: ptr.To[bool](true),
		},
	}
	assert.True(t, o.IsSetVMMaxMapCount())

	// When disabled
	o = &Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: ElasticsearchSpec{
			SetVMMaxMapCount: ptr.To[bool](false),
		},
	}
	assert.False(t, o.IsSetVMMaxMapCount())
}

func TestIsPersistence(t *testing.T) {
	var o *ElasticsearchNodeGroupSpec

	// With default value
	o = &ElasticsearchNodeGroupSpec{}
	assert.False(t, o.IsPersistence())

	// When persistence is not enabled
	o = &ElasticsearchNodeGroupSpec{
		Persistence: &ElasticsearchPersistenceSpec{},
	}

	assert.False(t, o.IsPersistence())

	// When claim PVC is set
	o = &ElasticsearchNodeGroupSpec{
		Persistence: &ElasticsearchPersistenceSpec{
			VolumeClaimSpec: &v1.PersistentVolumeClaimSpec{},
		},
	}

	assert.True(t, o.IsPersistence())

	// When volume is set
	o = &ElasticsearchNodeGroupSpec{
		Persistence: &ElasticsearchPersistenceSpec{
			Volume: &v1.VolumeSource{},
		},
	}

	assert.True(t, o.IsPersistence())

}

func TestIsPdb(t *testing.T) {
	var o Elasticsearch

	// When default
	o = Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: ElasticsearchSpec{},
	}
	assert.False(t, o.IsPdb(ElasticsearchNodeGroupSpec{}))

	// When default with replica > 1
	o = Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: ElasticsearchSpec{
			NodeGroups: []ElasticsearchNodeGroupSpec{
				{
					Name:     "test",
					Replicas: 2,
				},
			},
		},
	}
	assert.True(t, o.IsPdb(o.Spec.NodeGroups[0]))

	// When PDB is set on globalGroup
	o = Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: ElasticsearchSpec{
			GlobalNodeGroup: ElasticsearchGlobalNodeGroupSpec{
				PodDisruptionBudgetSpec: &policyv1.PodDisruptionBudgetSpec{},
			},
		},
	}
	assert.True(t, o.IsPdb(ElasticsearchNodeGroupSpec{}))

	// When PDB is set on nodeGroup
	o = Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: ElasticsearchSpec{},
	}
	assert.True(t, o.IsPdb(ElasticsearchNodeGroupSpec{
		PodDisruptionBudgetSpec: &policyv1.PodDisruptionBudgetSpec{},
	}))

}

func TestIsBootstrapping(t *testing.T) {

	// With default values
	o := &Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: ElasticsearchSpec{},
	}
	assert.False(t, o.IsBoostrapping())

	// When is false
	o.Status.IsBootstrapping = ptr.To[bool](false)
	assert.False(t, o.IsBoostrapping())

	// When is true
	o.Status.IsBootstrapping = ptr.To[bool](true)
	assert.True(t, o.IsBoostrapping())

}

func TestNumberOfReplicas(t *testing.T) {
	// With default value
	o := &Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: ElasticsearchSpec{},
	}
	assert.Equal(t, int32(0), o.NumberOfReplicas())

	// When multiple node groups
	o = &Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test",
		},
		Spec: ElasticsearchSpec{
			NodeGroups: []ElasticsearchNodeGroupSpec{
				{
					Name:     "test1",
					Replicas: 3,
				},
				{
					Name:     "test2",
					Replicas: 2,
				},
			},
		},
	}

	assert.Equal(t, int32(5), o.NumberOfReplicas())
}
