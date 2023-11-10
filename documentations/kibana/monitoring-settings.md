# Monitoring settings for Kibana

It's a good idea to monitor performance and health of your Kibana instances. You have 2 way to that:
  - Use kubernetes tools: You monitor Kibana from Prometheus / Graphana. Metrics exposed by Kibana expoter. It use a Kibana plugin.
  - Use Elastic tools: You deploy dedicated Elasticsearch / kibana for monitor. Metrics is collected and send by metricbeat.

## Monitor with Prometheus / Graphana

To use it, you need to have deployed Prometheus / Graphana stack on your kubernetes cluster with prometheus operator (to handle podMonitor resource).

You can use the following setting:
- **enabled** (boolean): Set to true to enable prometheus monitoring
- **url** (string): Prometheus exporter plugin. Default to `https://github.com/pjhampton/kibana-prometheus-exporter/releases`

**kibana.yaml**:
```yaml
apiVersion: kibana.k8s.webcenter.fr/v1
kind: Kibana
metadata:
  name: kibana
  namespace: cluster-dev
spec:
  monitoring:
    prometheus:
      enabled: true
      url: https://github.com/pjhampton/kibana-prometheus-exporter/releases/download/8.6.0/kibanaPrometheusExporter-8.6.0.zip
```

## Monitor cluster with Elastic tools

To use it, you need to have deployed [Elasticsearch cluster dedicated for monitoring](monitoring-sample.md).


You can use the following setting:
- **enabled** (boolean): Set to true to enable prometheus monitoring
- **elasticsearchRef** (object / required): The monitoring Elasticsearch cluster ref
  - **managed** (object): Use it if monitoring cluster is deployed with this operator
    - **name** (string / required): The name of elasticsearch resource.
    - **namespace** (string): The namespace where monitoring cluster is deployed on. Not needed if is on same namespace.
    - **targetNodeGroup** (string): The node group where to stream metrics. Default is used all node groups.
  - **external** (object): Use it if monitoring cluster is not deployed with this operator.
    - **addresses** (slice of string): The list of IPs, DNS, URL to access on monitoring Elasticseatch cluster
    - **secretRef** (object): The secret ref that store the credentials to write metrics on Elasticsearch from metricbeat. It need to contain the keys `username` and `password`
      - **name** (string / require): The secret name.
  - **elasticsearchCASecretRef** (object). It's the secret that store custom CA to connect on monitoring Elasticsearch cluster.
    - **name** (string / require): The secret name
- **resources** (object): The resources to set on metricbeat pod. Default to `{"requests": {"cpu": "100m", "memory": "100Mi"}, "limits: {"cpu": "300m", "memory": "200Mi"}}`. Read the [official doc to know the properties](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/)
- **refreshPeriod** (string): The refresh period used by Metricbeat. Default to `10s`
- **version** (string): The version of metricbeat to use. Default it use the same version of current Elasticsearch.


**elasticsearch.yaml**:
```yaml
apiVersion: elasticsearch.k8s.webcenter.fr/v1
kind: Elasticsearch
metadata:
  labels:
    socle: cluster-dev
  name: elasticsearch
  namespace: cluster-dev
spec:
  monitoring:
    metricbeat:
      enabled: true
      elasticsearchRef:
        managed:
          name: elasticsearch
          namespace: cluster-monitoring
          targetNodeGroup: all
        external:
          addresses:
            - https://cluster-monitoring.domain.local
          secretRef:
            name: monitoring-credentials
        elasticsearchCASecretRef:
          name: custom-ca-monitoring
      resources:
        limits:
          cpu: '1'
          memory: 1Gi
        requests:
          cpu: '300m'
          memory: 256Mi
```

**custom-ca-monitoring-secret.yaml**:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: custom-ca-monitoring
  namespace: cluster-dev
type: Opaque
data:
  ca.crt: ++++++++
```

**monitoring-credentials.yaml**:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: monitoring-credentials
  namespace: cluster-dev
type: Opaque
data:
  username: ++++++++
  password: ++++++++
```