package v1

// GetSecretFileRef returns the name of the secret where file-based information is stored
func (h DiscoverStatus) GetSecretFileRef() *string {
	return h.SecretFileRef
}

// GetSecretEnvRef returns the name of the secret where env-based information is stored
func (h DiscoverStatus) GetSecretEnvRef() *string {
	return h.SecretEnvRef
}

// GetType returns the type of discover referenced
func (h DiscoverRef) GetType() (discoverType DiscoverType) {
	if h.Kafka != nil && h.Kafka.Name != "" {
		return DiscoverTypeKafka
	} else if h.Logstash != nil && h.Logstash.Name != "" {
		return DiscoverTypeLogstash
	} else if h.Elasticsearch != nil && h.Elasticsearch.Name != "" {
		return DiscoverTypeElasticsearch
	}

	return discoverType
}

// GetName returns the name of the discover referenced
func (h DiscoverRef) GetName() string {
	switch h.GetType() {
	case DiscoverTypeKafka:
		return h.Kafka.Name
	case DiscoverTypeLogstash:
		return h.Logstash.Name
	case DiscoverTypeElasticsearch:
		return h.Elasticsearch.Name
	}

	return ""
}

// FindDiscoverRef permit to find a discover ref by its name
func FindDiscoverRef(discoverRefs []*DiscoverRef, name *string) *DiscoverRef {
	if name == nil || *name == "" {
		return nil
	}
	for _, discoverRef := range discoverRefs {
		if discoverRef.GetName() == *name {
			return discoverRef
		}
	}

	return nil
}
