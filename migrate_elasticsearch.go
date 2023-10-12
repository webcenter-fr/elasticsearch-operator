package main

import (
	"context"
	"fmt"

	"emperror.dev/errors"
	"github.com/disaster37/operator-sdk-extra/pkg/helper"
	"github.com/sirupsen/logrus"
	elasticsearchcrd "github.com/webcenter-fr/elasticsearch-operator/apis/elasticsearch/v1"
	"github.com/webcenter-fr/elasticsearch-operator/controllers/elasticsearch"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// migrateElasticsearch permit to migrate existing Elasticsearch cluster
func migrateElasticsearch(ctx context.Context, clientDynamic dynamic.Interface, clientStd kubernetes.Interface, log *logrus.Entry) (err error) {
	esList := make([]elasticsearchcrd.Elasticsearch, 0)
	var uList *unstructured.UnstructuredList

	// Read all Elasticsearch cluster on k8s
	if uList, err = clientDynamic.Resource(elasticsearchcrd.GroupVersion.WithResource("elasticsearches")).List(ctx, v1.ListOptions{}); err != nil {
		return errors.Wrap(err, "Error when read all Elasticsearch cluster to patch them")
	}

	for _, item := range uList.Items {
		es := &elasticsearchcrd.Elasticsearch{}
		if err = runtime.DefaultUnstructuredConverter.FromUnstructured(item.Object, es); err != nil {
			return errors.Wrap(err, "Error when convert from unstructured to Elasticsearch object")
		}
		esList = append(esList, *es)

	}

	log.Debugf("Found %d Elasticsearch clusters", len(esList))

	return addAnnotationsOnTLSSecrets(ctx, clientStd, esList, log)

}

// addAnnotationsOnTLSSecrets permit to set needed annotation on existing Elasticsearch cluster to manage in right way the cluster
func addAnnotationsOnTLSSecrets(ctx context.Context, clientStd kubernetes.Interface, esList []elasticsearchcrd.Elasticsearch, log *logrus.Entry) (err error) {

	var secret *corev1.Secret
	listSecretName := make([]string, 0, 2)

	for _, esCluster := range esList {
		listSecretName = append(listSecretName, elasticsearch.GetSecretNameForTlsTransport(&esCluster))
		listSecretName = append(listSecretName, elasticsearch.GetSecretNameForTlsApi(&esCluster))
		for _, secretName := range listSecretName {
			if secret, err = clientStd.CoreV1().Secrets(esCluster.Namespace).Get(ctx, secretName, v1.GetOptions{}); err != nil {
				if !k8serrors.IsNotFound(err) {
					return errors.Wrapf(err, "Error when read existing secret %s/%s", esCluster.Namespace, secretName)
				}
				continue
			}

			if secret.Annotations[fmt.Sprintf("%s/sequence", elasticsearchcrd.ElasticsearchAnnotationKey)] == "" {
				secret.Annotations[fmt.Sprintf("%s/sequence", elasticsearchcrd.ElasticsearchAnnotationKey)] = helper.RandomString(64)
				if _, err = clientStd.CoreV1().Secrets(secret.Namespace).Update(ctx, secret, v1.UpdateOptions{}); err != nil {
					return errors.Wrapf(err, "Error when upgrade secret %s:%s", esCluster.Namespace, secretName)
				}
				log.Infof("Successfully migrate secret %s/%s", esCluster.Namespace, secretName)
			}
		}
	}

	return nil
}
