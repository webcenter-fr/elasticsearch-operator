package v1

import (
	"regexp"
	"strings"

	"k8s.io/apimachinery/pkg/util/validation/field"
)

// discoverNamePattern restrict the logical name to characters that are safe
// both as environment variable suffix and as mount path directory name.
// It can't contain '/', '..', space or any other path or shell special
// character.
var discoverNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

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
		if discoverRef == nil {
			continue
		}
		if discoverRef.GetName() == *name {
			return discoverRef
		}
	}

	return nil
}

// ValidateName permit to validate the discover logical name.
// It is used to compute the environment variable suffix and the mount path
// directory, so it must not contain path traversal sequence or characters
// that break environment variable naming.
func (h Discover) ValidateName() *field.Error {
	if h.Name == nil || *h.Name == "" {
		return nil
	}

	if !discoverNamePattern.MatchString(*h.Name) || strings.Contains(*h.Name, "..") {
		return field.Invalid(field.NewPath("spec").Child("name"), *h.Name, "Name must start with an alphanumeric character, must only contain alphanumeric characters, '-', '_' or '.', and must not contain '..'")
	}

	return nil
}

// ValidateField permit to validate a DiscoverRef entry.
// Exactly one discover type must be referenced.
func (h DiscoverRef) ValidateField(fldPath *field.Path) *field.Error {
	nbRefs := 0
	if h.Kafka != nil && h.Kafka.Name != "" {
		nbRefs++
	}
	if h.Logstash != nil && h.Logstash.Name != "" {
		nbRefs++
	}
	if h.Elasticsearch != nil && h.Elasticsearch.Name != "" {
		nbRefs++
	}

	if nbRefs == 0 {
		return field.Required(fldPath, "You need to provide one discover reference")
	}
	if nbRefs > 1 {
		return field.Invalid(fldPath, h, "You can't provide many discover references on same entry")
	}

	return nil
}

// ValidateDiscoverRefs permit to validate all entries of a discoverRef list
func ValidateDiscoverRefs(discoverRefs []*DiscoverRef, fldPath *field.Path) field.ErrorList {
	allErrs := make(field.ErrorList, 0)
	for i, discoverRef := range discoverRefs {
		path := fldPath.Index(i)
		if discoverRef == nil {
			allErrs = append(allErrs, field.Required(path, "Discover reference must not be null"))
			continue
		}
		if err := discoverRef.ValidateField(path); err != nil {
			allErrs = append(allErrs, err)
		}
	}

	return allErrs
}
