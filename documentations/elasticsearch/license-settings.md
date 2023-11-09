# License setting for Elasticsearch

Per default, the basic license is used. But some time you should to use platinium or enterprise license for advance usage or to get official support. To to that, you only need to create secret with key `license` and the license as contend.

Then you need to link the secret name to `licenseSecretRef`.

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
  licenseSecretRef:
    name: elasticsearch-license
```