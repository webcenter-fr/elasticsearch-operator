# Endpoint setting for Kibana

When you deploy Kibana, you need to have a public endpoint for user. To do that, you can use:
  - Ingress endpoint: Access to Kibana from standard kubernetes ingress (reverse proxy)
  - Load balancer endpoint: Access to Kibana from kubernetes Load balancer (not supported on all kubernetes cluster)

## Ingress endpoint

You can use the following settings:
- **enabled** (boolean): Set to true to enable ingress endpoint. Default to `false`
- **host** (string / required): The hostname to access on Kibana. No default value
- **secretRef** (object): If you set it, it will enable https. If secret exist, it will use the certificate provided, else it will used the default ingress certificate
  - **name** (string / required): the secret name
- **labels** (map of string): The list of labels to add on ingress. Default to `empty`
- **annotations** (map of string): The list of annotations to add on ingress. Default to `empty`
- **ingressSpec** (object): You can set any other ingress properties. Read the [official doc to know the properties](https://kubernetes.io/docs/concepts/services-networking/ingress/)


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