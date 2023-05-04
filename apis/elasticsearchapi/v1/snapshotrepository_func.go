package v1

// GetSnapshotRepositoryName return the snapshot repository name
// If name is empty, it use the ressource name
func (o *SnapshotRepository) GetSnapshotRepositoryName() string {
	if o.Spec.Name == "" {
		return o.Name
	}

	return o.Spec.Name
}
