package v1

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// SetupLogstashIndexer setup indexer for Logstash
func SetupLogstashIndexer(k8sManager manager.Manager) (err error) {
	if err = k8sManager.GetFieldIndexer().IndexField(context.Background(), &Logstash{}, "spec.logstashRef.managed.fullname", func(o client.Object) []string {
		p := o.(*Logstash)
		if p.Spec.LogstashRef.IsManaged() {
			if p.Spec.LogstashRef.ManagedLogstashRef.Namespace != "" {
				return []string{fmt.Sprintf("%s/%s", p.Spec.LogstashRef.ManagedLogstashRef.Namespace, p.Spec.LogstashRef.ManagedLogstashRef.Name)}
			}
			return []string{fmt.Sprintf("%s/%s", p.Namespace, p.Spec.LogstashRef.ManagedLogstashRef.Name)}
		}
		return []string{}
	}); err != nil {
		return err
	}

	if err = k8sManager.GetFieldIndexer().IndexField(context.Background(), &Logstash{}, "spec.logstashRef.userRef.fullname", func(o client.Object) []string {
		p := o.(*Logstash)

		if p.Spec.LogstashRef.IsManaged() && p.Spec.LogstashRef.ManagedLogstashRef.UserRef != nil {
			ns := p.Namespace
			if p.Spec.LogstashRef.ManagedLogstashRef.UserRef.Namespace != "" {
				ns = p.Spec.LogstashRef.ManagedLogstashRef.UserRef.Namespace
			}
			return []string{fmt.Sprintf("%s/%s", ns, p.Spec.LogstashRef.ManagedLogstashRef.UserRef.Name)}
		}

		return []string{}
	}); err != nil {
		return err
	}

	if err = k8sManager.GetFieldIndexer().IndexField(context.Background(), &Logstash{}, "spec.logstashRef.userSecretRef.fullname", func(o client.Object) []string {
		p := o.(*Logstash)

		if p.Spec.LogstashRef.IsManaged() && p.Status.LogstashUserSecretRef != nil {
			ns := p.Namespace
			if p.Spec.LogstashRef.ManagedLogstashRef.UserRef != nil && p.Spec.LogstashRef.ManagedLogstashRef.UserRef.Namespace != "" {
				ns = p.Spec.LogstashRef.ManagedLogstashRef.UserRef.Namespace
			}
			return []string{fmt.Sprintf("%s/%s", ns, *p.Status.LogstashUserSecretRef)}
		}

		if p.Spec.LogstashRef.IsExternal() && p.Spec.LogstashRef.ExternalLogstashRef.UserSecretRef != nil {
			return []string{fmt.Sprintf("%s/%s", p.Namespace, p.Spec.LogstashRef.ExternalLogstashRef.UserSecretRef.Name)}
		}

		return []string{}
	}); err != nil {
		return err
	}

	if err = k8sManager.GetFieldIndexer().IndexField(context.Background(), &Logstash{}, "spec.logstashRef.logstashCASecretRef.fullname", func(o client.Object) []string {
		p := o.(*Logstash)
		res := make([]string, 0, 1)

		if p.Spec.LogstashRef.LogstashCaSecretRef != nil {
			res = append(res, fmt.Sprintf("%s/%s", p.Namespace, p.Spec.LogstashRef.LogstashCaSecretRef.Name))
		}

		if p.Spec.LogstashRef.IsManaged() && p.Status.LogstashCASecretRef != nil {
			ns := p.Spec.LogstashRef.ManagedLogstashRef.Namespace
			if p.Spec.LogstashRef.ManagedLogstashRef.Namespace == "" {
				ns = p.Namespace
			}
			res = append(res, fmt.Sprintf("%s/%s", ns, *p.Status.LogstashCASecretRef))
		}

		return res
	}); err != nil {
		return err
	}

	return nil
}
