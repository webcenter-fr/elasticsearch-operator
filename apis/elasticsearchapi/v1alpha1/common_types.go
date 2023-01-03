package v1alpha1

type ElasticsearchRefSpec struct {
	// Name is the Elasticsearch name object
	// If empty, it use Adresses and secretName to connect on external elasticsearch (not managed by operator)
	Name string `json:"name,omitempty"`

	// Addresses is the list of Elasticsearch addresses
	Addresses []string `json:"addresses,omitempty"`

	// SecretName is the secret that contain the setting to connect on Elasticsearch that is not managed by ECK.
	// It need to contain only one entry. The user is the key, and the password is the data
	SecretName string `json:"secretName,omitempty"`
}

// IsManagedByECK permit to know if Elasticsearch is managed by ECK
func (h ElasticsearchRefSpec) IsManagedByOperator() bool {
	return h.Name != ""
}
