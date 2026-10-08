package elasticsearch

import (
	"testing"

	"github.com/stretchr/testify/assert"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/api/elasticsearch/v1"
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestBaseTLSSpecCARenewalDays(t *testing.T) {
	o := &elasticsearchcrd.Elasticsearch{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
	}
	spec := baseTLSSpec(o, "secret", "cn", "ca-cn", "ou")
	assert.Zero(t, spec.CARenewalDays)

	o.Spec.Tls = shared.TlsSpec{CaRenewalDays: ptr.To(90)}
	spec = baseTLSSpec(o, "secret", "cn", "ca-cn", "ou")
	assert.Equal(t, 90, spec.CARenewalDays)
}
