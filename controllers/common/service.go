package common

import corev1 "k8s.io/api/core/v1"

// CleanServiceToDiff clean computed fields
func CleanServiceToDiff(s corev1.ServiceSpec) corev1.ServiceSpec {

	if s.ClusterIP != "None" {
		s.ClusterIP = ""
	}

	s.ClusterIPs = nil
	s.IPFamilies = nil
	s.IPFamilyPolicy = nil
	s.InternalTrafficPolicy = nil

	return s
}
