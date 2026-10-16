package v1

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// SetupKafkaIndexer setup indexer for Kafka
func SetupKafkaIndexer(k8sManager manager.Manager) (err error) {
	if err = k8sManager.GetFieldIndexer().IndexField(context.Background(), &Kafka{}, "spec.kafkaRef.managed.fullname", func(o client.Object) []string {
		p := o.(*Kafka)
		if p.Spec.KafkaRef.IsManaged() {
			if p.Spec.KafkaRef.ManagedKafkaRef.Namespace != "" {
				return []string{fmt.Sprintf("%s/%s", p.Spec.KafkaRef.ManagedKafkaRef.Namespace, p.Spec.KafkaRef.ManagedKafkaRef.Name)}
			}
			return []string{fmt.Sprintf("%s/%s", p.Namespace, p.Spec.KafkaRef.ManagedKafkaRef.Name)}
		}
		return []string{}
	}); err != nil {
		return err
	}

	if err = k8sManager.GetFieldIndexer().IndexField(context.Background(), &Kafka{}, "spec.kafkaRef.userRef.fullname", func(o client.Object) []string {
		p := o.(*Kafka)

		if p.Spec.KafkaRef.IsManaged() && p.Spec.KafkaRef.ManagedKafkaRef.UserRef != nil {
			ns := p.Spec.KafkaRef.ManagedKafkaRef.Namespace
			if p.Spec.KafkaRef.ManagedKafkaRef.Namespace == "" {
				ns = p.Namespace
			}
			return []string{fmt.Sprintf("%s/%s", ns, p.Spec.KafkaRef.ManagedKafkaRef.UserRef.Name)}
		}

		return []string{}
	}); err != nil {
		return err
	}

	if err = k8sManager.GetFieldIndexer().IndexField(context.Background(), &Kafka{}, "spec.kafkaRef.userSecretRef.fullname", func(o client.Object) []string {
		p := o.(*Kafka)

		if p.Spec.KafkaRef.IsManaged() && p.Status.KafkaUserSecretRef != nil {
			ns := p.Spec.KafkaRef.ManagedKafkaRef.Namespace
			if p.Spec.KafkaRef.ManagedKafkaRef.Namespace == "" {
				ns = p.Namespace
			}
			return []string{fmt.Sprintf("%s/%s", ns, *p.Status.KafkaUserSecretRef)}
		}

		if p.Spec.KafkaRef.IsExternal() && p.Spec.KafkaRef.ExternalKafkaRef.UserSecretRef != nil {
			return []string{fmt.Sprintf("%s/%s", p.Namespace, p.Spec.KafkaRef.ExternalKafkaRef.UserSecretRef.Name)}
		}

		return []string{}
	}); err != nil {
		return err
	}

	if err = k8sManager.GetFieldIndexer().IndexField(context.Background(), &Kafka{}, "spec.kafkaRef.kafkaCASecretRef.fullname", func(o client.Object) []string {
		p := o.(*Kafka)
		res := make([]string, 0, 1)

		if p.Spec.KafkaRef.KafkaCaSecretRef != nil {
			res = append(res, fmt.Sprintf("%s/%s", p.Namespace, p.Spec.KafkaRef.KafkaCaSecretRef.Name))
		}

		if p.Spec.KafkaRef.IsManaged() && p.Status.KafkaCASecretRef != nil {
			ns := p.Spec.KafkaRef.ManagedKafkaRef.Namespace
			if p.Spec.KafkaRef.ManagedKafkaRef.Namespace == "" {
				ns = p.Namespace
			}
			res = append(res, fmt.Sprintf("%s/%s", ns, *p.Status.KafkaCASecretRef))
		}
		return res
	}); err != nil {
		return err
	}

	return nil
}
