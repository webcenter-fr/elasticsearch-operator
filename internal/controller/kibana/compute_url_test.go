package kibana

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	kibanacrd "github.com/webcenter-fr/elasticsearch-operator/api/kibana/v1"
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
	v1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

func TestComputeKibanaUrl(t *testing.T) {
	var (
		enabled  = ptr.To(true)
		disabled = ptr.To(false)
		cert     = &v1.LocalObjectReference{Name: "cert"}
	)

	tests := []struct {
		name       string
		ingress    bool
		route      bool
		ingressTLS *bool
		routeTLS   *bool
		backendTLS *bool
		secretRef  *v1.LocalObjectReference
		want       string
	}{
		// Ingress branch
		{name: "ingress default tls", ingress: true, want: "https://my-kibana.example.com"},
		{name: "ingress default tls with secretRef", ingress: true, secretRef: cert, want: "https://my-kibana.example.com"},
		{name: "ingress tls nil, backend tls true", ingress: true, backendTLS: enabled, want: "https://my-kibana.example.com"},
		{name: "ingress tls nil, backend tls false", ingress: true, backendTLS: disabled, want: "https://my-kibana.example.com"},
		{name: "ingress tls nil, backend tls false with secretRef", ingress: true, backendTLS: disabled, secretRef: cert, want: "https://my-kibana.example.com"},
		{name: "ingress tls true, backend tls nil", ingress: true, ingressTLS: enabled, want: "https://my-kibana.example.com"},
		{name: "ingress tls true, backend tls false", ingress: true, ingressTLS: enabled, backendTLS: disabled, want: "https://my-kibana.example.com"},
		{name: "ingress tls false, backend tls nil", ingress: true, ingressTLS: disabled, want: "https://my-kibana.example.com"},
		{name: "ingress tls false, backend tls true", ingress: true, ingressTLS: disabled, backendTLS: enabled, want: "https://my-kibana.example.com"},
		{name: "ingress tls false, backend tls false", ingress: true, ingressTLS: disabled, backendTLS: disabled, want: "http://my-kibana.example.com"},
		{name: "ingress tls false, backend tls false with secretRef", ingress: true, ingressTLS: disabled, backendTLS: disabled, secretRef: cert, want: "http://my-kibana.example.com"},
		// Route branch
		{name: "route default tls", route: true, want: "https://my-kibana-route.example.com"},
		{name: "route default tls, backend tls false", route: true, backendTLS: disabled, want: "https://my-kibana-route.example.com"},
		{name: "route tls true, backend tls false", route: true, routeTLS: enabled, backendTLS: disabled, want: "https://my-kibana-route.example.com"},
		{name: "route tls true, backend tls false with secretRef", route: true, routeTLS: enabled, backendTLS: disabled, secretRef: cert, want: "https://my-kibana-route.example.com"},
		{name: "route tls false, backend tls nil", route: true, routeTLS: disabled, want: "https://my-kibana-route.example.com"},
		{name: "route tls false, backend tls true", route: true, routeTLS: disabled, backendTLS: enabled, want: "https://my-kibana-route.example.com"},
		{name: "route tls false, backend tls false", route: true, routeTLS: disabled, backendTLS: disabled, want: "http://my-kibana-route.example.com"},
		{name: "route tls false, backend tls false with secretRef", route: true, routeTLS: disabled, backendTLS: disabled, secretRef: cert, want: "http://my-kibana-route.example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kb := &kibanacrd.Kibana{
				Spec: kibanacrd.KibanaSpec{
					Endpoint: shared.EndpointSpec{
						Ingress: &shared.EndpointIngressSpec{
							Enabled:    tt.ingress,
							Host:       "my-kibana.example.com",
							TlsEnabled: tt.ingressTLS,
							SecretRef:  tt.secretRef,
						},
						Route: &shared.EndpointRouteSpec{
							Enabled:    tt.route,
							Host:       "my-kibana-route.example.com",
							TlsEnabled: tt.routeTLS,
							SecretRef:  tt.secretRef,
						},
					},
					Tls: shared.TlsSpec{Enabled: tt.backendTLS},
				},
			}

			h := &KibanaReconciler{}
			got, err := h.computeKibanaUrl(context.Background(), kb, nil)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
