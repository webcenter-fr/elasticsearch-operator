# Elastic operator

This Elastic operator permit to deploy Elastic stack (elasticsearch, kibana, logstash, filebeat, metricbeat en crebro) and manage some configurations like role, user, ILM, SLM, repositories, etc ...


Like you say, there are on official elastic operator called [ECK](https://github.com/elastic/cloud-on-k8s) that can work fine and better than here.

Benefit of this operator:
- This operator permit to work with all license (basic, platinium and enterprise)
- Only one operator to manage different license on each cluster
- You can configure Elasticsearch after deploy it: role, user, license, component, index template, ILM, SLM, repositories, role mappings.
- You can configure Kibana after deploy it: logstash pipeline, role, user space
- You can add automatically host entry in Cerebro (UI to manage Elasticsearch)
- It reconcile when you update external secrets and external configMaps
- It manage automatically the transport TLS certificats and API TLS certificates
- It rolling update Elaticsearch cluster pod by pod.


## Deploy operator

For moment, it only support the deployment by OLM with private catalog.


So, you need to add catalog on `olm` namespace:
```yaml
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: elasticsearch-operator
  namespace: olm
spec:
  sourceType: grpc
  image: quay.io/webcenter/elasticsearch-operator-catalog:v0.0.38
```

Then, you need to add the subscriptions:
```yaml
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: elasticsearch-operator-subscription
spec:
  channel: alpha
  name: elasticsearch-operator
  source: elasticsearch-operator
  sourceNamespace: olm
  config:
    env:
      - name: LOG_LEVEL
        value: INFO
```

> To upgrade operator, you just need to update the image version on `elasticsearch-operator` catalog source

## Deploy Elasticsearch cluster

To deploy Elasticsearch cluster, you need to set a custom resource of type `Elasticsearch`.
To do that, you need to set:
  - main settings: Elasticsearch version, cluster name
  - global settings share between node groups: 
  - nodes groups settings: the topology of elasticsearch cluster
  - endpoint to access on Elasticsearch
  - TLS to encrypt network andf trust nodes
  - License if platinium or enterprise is needed
  - Monitoring to monitor elasticsearch from prometheus/graphana or with dedicated Elasticsearch/kibana cluster
  - Generate automatically system passwords

You can read some samples:
  - [Single node cluster sample](documentations/elasticsearch/single-node-sample.md)
  - [Hot / warm cluster sample](documentations/elasticsearch/hot-warm-sample.md)
  - [Monitoring cluster sample](documentations/elasticsearch/monitoring-sample.md)


You can read complete documentation per sub section
  - [Main settings](documentations/elasticsearch/main-settings.md)
  - [Global settings share between node groups](documentations/elasticsearch/global-settings.md)
  - [Node group settings](documentations/elasticsearch/node-group-settings.md)
  - [Endpoint settings](documentations/elasticsearch/endpoint-settings.md)
  - [TLS settings](documentations/elasticsearch/tls-settings.md)
  - [Monitoring settings](documentations/elasticsearch/monitoring-settings.md)
  - [License settings](documentations/elasticsearch/license-settings.md)


## Manage Elasticsearch cluster

## Deploy Kibana

## Manage Kibana

## Deploy Logstash

## Deploy Filebeat

## Deploy Metricbeat

## Deploy Cerebro

