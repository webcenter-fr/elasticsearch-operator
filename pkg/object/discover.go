package object

import (
	"github.com/disaster37/operator-sdk-extra/v3/pkg/object"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DiscoverObject is the interface that all discover objects must implement
type DiscoverObject interface {
	client.Object

	// GetDiscoverStatus returns the status of the discover object
	GetDiscoverStatus() DiscoverObjectStatus
}

// DiscoverObjectStatus is the interface that all discover object statuses must implement
type DiscoverObjectStatus interface {
	object.MultiPhaseObjectStatus

	// GetSecretFileRef returns the name of the secret where file-based information is stored
	GetSecretFileRef() *string

	// GetSecretEnvRef returns the name of the secret where env-based information is stored
	GetSecretEnvRef() *string
}
