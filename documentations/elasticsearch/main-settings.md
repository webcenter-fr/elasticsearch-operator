# Main settings to deploy Elasticsearch

You can use the following main setting to deploy Elasticsearch:
- **image** (string): Elasticsearch image to use. Default to `docker.elastic.co/elasticsearch/elasticsearch`
- **imagePullPolicy** (string): The image pull policy. Default to `IfNotPresent`
- **imagePullSecrets** (string): The image pull secrets to use. Default to `empty`
- **version** (string): The image version to use. Default to `latest`
- **clusterName** (string): The cluster name. Default is use the Elasticsearch custom resource name
- **setVMMaxMapCount** (boolean): Set VMMaxMapCount on kubernetes nodes where Elasticsearch is deployed. Default to `true`
- **pluginsList** (slice of string): The list of plugins to install on runtime (just before run Elasticsearch). Use it for test purpose. For production, please build custom image to embedded your plugins. Default to `empty`


**elasticsearch.yaml**:
```yaml
apiVersion: elasticsearch.k8s.webcenter.fr/v1
kind: Elasticsearch
metadata:
    socle: cluster-dev
  name: elasticsearch
  namespace: cluster-dev
spec:
  image: docker.elastic.co/elasticsearch/elasticsearch
  imagePullPolicy: IfNotPresent
  imagePullSecrets:
    - name: my-pull-secret
  version: 8.7.1
  clusterName: my-cluster
  setVMMaxMapCount: true
  pluginsList:
    - 'analysis-icu'
```