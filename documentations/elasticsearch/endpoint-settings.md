# Endpoint setting for Elasticsearch

When you deploy Elasticsearch cluster, you need to have a public endpoint for user and remote application access (not hosted on the same kubernetes cluster). To do that, you can use:
  - **Ingress endpoint**: Access to Elasticsearch from standard kubernetes ingress (reverse proxy)
  - **Route endpoint**: Access Elasticsearch through an OpenShift route (reverse proxy).
  - **Load balancer endpoint**: Access to Elasticsearch from kubernetes Load balancer (not supported on all kubernetes cluster)

## Ingress endpoint

You can use the following settings to configure an ingress endpoint:
- **enabled** (boolean): Set to `true` to enable ingress endpoint. Default is `false`.
- **targetNodeGroupName** (string): You can retrieve endpoint traffic to a particular node group, like a client node group. Default is to load balance through all nodes.
- **host** (string / required): The hostname to access the Elasticsearch API. No default value.
- **tlsEnabled** (boolean): Set to `false` to disable TLS. Default is `true`.
- **secretRef** (object): If a secret exists, it will use the certificate provided; otherwise, it will use the default ingress certificate.
  - **name** (string / required): The secret name.
- **labels** (map of strings): A list of labels to add to the ingress.
- **annotations** (map of strings): A list of annotations to add to the ingress.
- **ingressSpec** (object): Additional ingress properties. Refer to the [official Kubernetes documentation](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#ingressspec-v1-networking-k8s-io) for details.


**elasticsearch.yaml**:
```yaml
apiVersion: elasticsearch.k8s.webcenter.fr/v1
kind: Elasticsearch
metadata:
  name: elasticsearch
  namespace: cluster-dev
spec:
  endpoint:
    ingress:
      annotations:
        nginx.ingress.kubernetes.io/proxy-body-size: 512M
        nginx.ingress.kubernetes.io/proxy-connect-timeout: '600'
        nginx.ingress.kubernetes.io/proxy-read-timeout: '600'
        nginx.ingress.kubernetes.io/proxy-send-timeout: '600'
        nginx.ingress.kubernetes.io/ssl-redirect: 'true'
      enabled: true
      host: elasticsearch-cluster-prd.domain.local
      secretRef:
        name: es-tls
      ingressSpec:
        ingressClassName: nginx-example
      targetNodeGroupName: client
```

## Route Endpoint

You can configure the route endpoint with the following settings:

- **enabled** (boolean): Set to `true` to enable the route endpoint. Default: `false`.
- **targetNodeGroupName** (string): Direct endpoint traffic to a specific node group, such as the client node group. By default, traffic is load-balanced across all nodes.
- **host** (string, required): The hostname to access the Elasticsearch API. No default value.
- **tlsEnabled** (boolean): Set to `false` to disable TLS. Default: `true`.
- **secretRef** (object): Specifies the secret containing a custom certificate (`tls.key` and `tls.crt`).
  - **name** (string, required): The name of the secret.
- **labels** (map of strings): A list of labels to add to the route. Default: `empty`.
- **annotations** (map of strings): A list of annotations to add to the route. Default: `empty`.
- **routeSpec** (object): Additional route properties. Refer to the [official OpenShift documentation](https://docs.openshift.com/container-platform/4.11/networking/routes/route-configuration.html) for details.

### Example Configuration

```yaml
apiVersion: elasticsearch.k8s.webcenter.fr/v1
kind: Elasticsearch
metadata:
  name: elasticsearch
  namespace: cluster-dev
spec:
  endpoint:
    route:
      annotations:
        haproxy.router.openshift.io/timeout: '600s'
      enabled: true
      host: elasticsearch-cluster-dev.domain.local
      routeSpec:
        path: "/"
      targetNodeGroupName: client
```

## Load balancer endpoint

The kubernertes cluster must be support load balancer service. On cloud provider it can work out of the box. Is not the case on onprmise cluster. You need to deploy extra stack like metalLB.

You can use the following settings:
- **enabled** (bool): Set to true to enable load balancer. Default to `false`.
- **targetNodeGroupName** (string): You can retrive endpoint traffic to particular node group, like client node group.

> To get Load balancer IP, you need to get the IP on status service.

**elasticsearch.yaml**:
```yaml
apiVersion: elasticsearch.k8s.webcenter.fr/v1
kind: Elasticsearch
metadata:
  name: elasticsearch
  namespace: cluster-dev
spec:
  endpoint:
    loadBalancer:
      enabled: true
      targetNodeGroupName: client
```