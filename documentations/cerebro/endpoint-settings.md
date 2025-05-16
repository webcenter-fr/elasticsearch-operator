# Endpoint setting for Cerebro

When you deploy Cerebro, you need to have a public endpoint for user. To do that, you can use:
  - **Ingress endpoint**: Access to Cerebro from standard kubernetes ingress (reverse proxy)
  - **Route endpoint**: Access Cerebro using an OpenShift route (reverse proxy).
  - **Load balancer endpoint**: Access to Cerebro from kubernetes Load balancer (not supported on all kubernetes cluster)

## Ingress endpoint

You can use the following settings to configure an ingress endpoint:
- **enabled** (boolean): Set to `true` to enable ingress endpoint. Default is `false`.
- **host** (string / required): The hostname to access Cerebro. No default value.
- **tlsEnabled** (boolean): Set to `false` to disable TLS. Default is `true`.
- **secretRef** (object): If a secret exists, it will use the certificate provided; otherwise, it will use the default ingress certificate.
  - **name** (string / required): The secret name.
- **labels** (map of string): The list of labels to add on ingress.
- **annotations** (map of string): The list of annotations to add on ingress.
- **ingressSpec** (object): Additional ingress properties. Refer to the [official Kubernetes documentation](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#ingressspec-v1-networking-k8s-io) for details.


**cerebro.yaml**:
```yaml
apiVersion: cerebro.k8s.webcenter.fr/v1
kind: Cerebro
metadata:
  name: cerebro
  namespace: cluster-dev
spec:
  endpoint:
    ingress:
      annotations:
        nginx.ingress.kubernetes.io/proxy-connect-timeout: '600'
        nginx.ingress.kubernetes.io/proxy-read-timeout: '600'
        nginx.ingress.kubernetes.io/proxy-send-timeout: '600'
      enabled: true
      host: cerebro-dev.domain.local
      secretRef:
        name: cerebro-tls
      ingressSpec:
        ingressClassName: nginx-example
```

## Route endpoint

You can use the following settings to configure a route endpoint:
- **enabled** (boolean): Set to `true` to enable the route endpoint. Default is `false`.
- **host** (string / required): The hostname to access Cerebro. No default value.
- **tlsEnabled** (boolean): Set to `false` to disable TLS. Default is `true`.
- **secretRef** (object): The secret containing a custom certificate (`tls.key` and `tls.crt`).
  - **name** (string / required): The name of the secret.
- **labels** (map of string): Labels to add to the route. Default is empty.
- **annotations** (map of string): Annotations to add to the route. Default is empty.
- **routeSpec** (object): Additional route properties. Refer to the [official OpenShift documentation](https://docs.openshift.com/container-platform/4.11/networking/routes/route-configuration.html) for details.

### Example Configuration

**cerebro.yaml**:
```yaml
apiVersion: cerebro.k8s.webcenter.fr/v1
kind: Cerebro
metadata:
  name: cerebro
  namespace: cluster-dev
spec:
  endpoint:
    route:
      annotations:
        haproxy.router.openshift.io/timeout: '600s'
      enabled: true
      host: cerebro-dev.domain.local
      routeSpec:
        path: "/"
```

## Load balancer endpoint

The kubernertes cluster must be support load balancer service. On cloud provider it can work out of the box. Is not the case on onprmise cluster. You need to deploy extra stack like metalLB.

You can use the following settings:
- **enabled** (bool): Set to true to enable load balancer. Default to `false`.

> To get Load balancer IP, you need to get the IP on status service.

**cerebro.yaml**:
```yaml
apiVersion: cerebro.k8s.webcenter.fr/v1
kind: Cerebro
metadata:
  name: cerebro
  namespace: cluster-dev
spec:
  endpoint:
    loadBalancer:
      enabled: true
```