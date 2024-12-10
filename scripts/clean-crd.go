package main

import (
	"context"
	"fmt"
	"os"

	apiextension "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

var (
	versionToStrip = "v1alpha1"
	crds           = []string{
		"elasticsearches.elasticsearch.k8s.webcenter.fr",
		"filebeats.beat.k8s.webcenter.fr",
		"metricbeats.beat.k8s.webcenter.fr",
		"cerebroes.cerebro.k8s.webcenter.fr",
		"hosts.cerebro.k8s.webcenter.fr",
		"componenttemplates.elasticsearchapi.k8s.webcenter.fr",
		"indexlifecyclepolicies.elasticsearchapi.k8s.webcenter.fr",
		"indextemplates.elasticsearchapi.k8s.webcenter.fr",
		"licenses.elasticsearchapi.k8s.webcenter.fr",
		"rolemappings.elasticsearchapi.k8s.webcenter.fr",
		"roles.elasticsearchapi.k8s.webcenter.fr",
		"snapshotlifecyclepolicies.elasticsearchapi.k8s.webcenter.fr",
		"snapshotrepositories.elasticsearchapi.k8s.webcenter.fr",
		"users.elasticsearchapi.k8s.webcenter.fr",
		"watches.elasticsearchapi.k8s.webcenter.fr",
		"kibanas.kibana.k8s.webcenter.fr",
		"logstashpipelines.kibanaapi.k8s.webcenter.fr",
		"roles.kibanaapi.k8s.webcenter.fr",
		"userspaces.kibanaapi.k8s.webcenter.fr",
		"logstashes.logstash.k8s.webcenter.fr",
	}
)

func exitOnErr(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func main() {
	//  create k8s client
	cfg := config.GetConfigOrDie()
	client, err := apiextension.NewForConfig(cfg)
	exitOnErr(err)

	for _, crdName := range crds {
		cleanCRD(client, crdName)
	}
}

func cleanCRD(client *apiextension.Clientset, crdName string) {
	// retrieve CRD
	crd, err := client.ApiextensionsV1().CustomResourceDefinitions().Get(context.Background(), crdName, v1.GetOptions{})
	exitOnErr(err)
	// remove v1alpha1 from its status
	oldStoredVersions := crd.Status.StoredVersions
	newStoredVersions := make([]string, 0, len(oldStoredVersions))
	for _, stored := range oldStoredVersions {
		if stored != versionToStrip {
			newStoredVersions = append(newStoredVersions, stored)
		}
	}
	crd.Status.StoredVersions = newStoredVersions
	// update the status sub-resource
	crd, err = client.ApiextensionsV1().CustomResourceDefinitions().UpdateStatus(context.Background(), crd, v1.UpdateOptions{})
	exitOnErr(err)
	fmt.Printf("updated CRD %s status storedVersions: %v\n", crdName, crd.Status.StoredVersions)
}
