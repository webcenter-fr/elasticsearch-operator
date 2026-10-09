package cerebro

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	cerebrocrd "github.com/webcenter-fr/elasticsearch-operator/api/cerebro/v1"
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
	v1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

func TestComputeCerebroUrl(t *testing.T) {
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
		secretRef  *v1.LocalObjectReference
		want       string
	}{
		// Ingress branch
		{name: "ingress default tls", ingress: true, want: "https://my-cerebro.example.com"},
		{name: "ingress default tls with secretRef", ingress: true, secretRef: cert, want: "https://my-cerebro.example.com"},
		{name: "ingress tls true", ingress: true, ingressTLS: enabled, want: "https://my-cerebro.example.com"},
		{name: "ingress tls true with secretRef", ingress: true, ingressTLS: enabled, secretRef: cert, want: "https://my-cerebro.example.com"},
		{name: "ingress tls false", ingress: true, ingressTLS: disabled, want: "http://my-cerebro.example.com"},
		{name: "ingress tls false with secretRef", ingress: true, ingressTLS: disabled, secretRef: cert, want: "http://my-cerebro.example.com"},
		// Route branch
		{name: "route default tls", route: true, want: "https://my-cerebro-route.example.com"},
		{name: "route tls true", route: true, routeTLS: enabled, want: "https://my-cerebro-route.example.com"},
		{name: "route tls false", route: true, routeTLS: disabled, want: "http://my-cerebro-route.example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cb := &cerebrocrd.Cerebro{
				Spec: cerebrocrd.CerebroSpec{
					Endpoint: shared.EndpointSpec{
						Ingress: &shared.EndpointIngressSpec{
							Enabled:    tt.ingress,
							Host:       "my-cerebro.example.com",
							TlsEnabled: tt.ingressTLS,
							SecretRef:  tt.secretRef,
						},
						Route: &shared.EndpointRouteSpec{
							Enabled:    tt.route,
							Host:       "my-cerebro-route.example.com",
							TlsEnabled: tt.routeTLS,
							SecretRef:  tt.secretRef,
						},
					},
				},
			}

			h := &CerebroReconciler{}
			got, err := h.computeCerebroUrl(context.Background(), cb)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
