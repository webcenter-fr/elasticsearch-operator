# Endpoint setting for Kibana

When you deploy Kibana, you need to have a public endpoint for user. To do that, you can use:
  - **Ingress endpoint**: Access to Kibana from standard kubernetes ingress (reverse proxy)
  - **Route endpoint**: Access Kibana via an OpenShift route (reverse proxy).
  - **Load balancer endpoint**: Access to Kibana from kubernetes Load balancer (not supported on all kubernetes cluster)

## Ingress endpoint

You can use the following settings to configure an ingress endpoint:
- **enabled** (boolean): Set to `true` to enable ingress endpoint. Default is `false`.
- **host** (string / required): The hostname to access Kibana. No default value.
- **tlsEnabled** (boolean): Set to `false` to disable TLS. Default is `true`.
- **secretRef** (object): Specifies the secret containing a custom certificate (`tls.key` and `tls.crt`).
  - **name** (string / required): The name of the secret.
- **labels** (map of string): Labels to add to the ingress. Default: `empty`.
- **annotations** (map of string): Annotations to add to the ingress. Default: `empty`.
- **ingressSpec** (object): Additional ingress properties. Refer to the [official Kubernetes documentation](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#ingressspec-v1-networking-k8s-io) for details.


**kibana.yaml**:
```yaml
apiVersion: kibana.k8s.webcenter.fr/v1
kind: Kibana
metadata:
  name: kibana
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
      host: kibana-dev.domain.local
      secretRef:
        name: kb-tls
      ingressSpec:
        ingressClassName: nginx-example
```

## Route Endpoint

### Configuration Options:
- **`enabled`** (boolean): Set to `true` to enable the route endpoint. Default: `false`.
- **`host`** (string / required): The hostname to access Kibana. No default value.
- **`tlsEnabled`** (boolean): Set to `false` to disable TLS. Default: `true`.
- **`secretRef`** (object): Specifies the secret containing a custom certificate (`tls.key` and `tls.crt`).
  - **`name`** (string / required): The name of the secret.
- **`labels`** (map of string): Labels to add to the route. Default: `empty`.
- **`annotations`** (map of string): Annotations to add to the route. Default: `empty`.
- **`routeSpec`** (object): Additional route properties. Refer to the [official OpenShift documentation](https://docs.openshift.com/container-platform/4.11/networking/routes/route-configuration.html) for details.

### Example: **dashboard.yaml**
```yaml
apiVersion: kibana.k8s.webcenter.fr/v1
kind: Kibana
metadata:
  name: dashboard
  namespace: cluster-dev
spec:
  endpoint:
    route:
      annotations:
        haproxy.router.openshift.io/timeout: '600s'
      enabled: true
      host: dashboard-dev.domain.local
      routeSpec:
        path: "/"
```

## Load balancer endpoint

The kubernertes cluster must be support load balancer service. On cloud provider it can work out of the box. Is not the case on onprmise cluster. You need to deploy extra stack like metalLB.

You can use the following settings:
- **enabled** (bool): Set to true to enable load balancer. Default to `false`.

> To get Load balancer IP, you need to get the IP on status service.

**kibana.yaml**:
```yaml
apiVersion: kibana.k8s.webcenter.fr/v1
kind: Kibana
metadata:
  name: kibana
  namespace: cluster-dev
spec:
  endpoint:
    loadBalancer:
      enabled: true
```