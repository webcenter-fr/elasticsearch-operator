package v1

// GetWatchName return the watch name
// If name is empty, it use the ressource name
func (o *Watch) GetWatchName() string {
	if o.Spec.Name == "" {
		return o.Name
	}

	return o.Spec.Name
}
