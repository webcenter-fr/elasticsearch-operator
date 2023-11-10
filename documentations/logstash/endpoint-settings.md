# Endpoint setting for Logstash

You can use ingress or service endpoint for Logstash. You can define as many as you like

## Ingresses endpoint

You can define as many as you like. It's the slice of ingresses.
It's permit to create custom ingresses if needed by input or to access on Logstash API. It will set automatically the service needed by ingress.

You can use the following settings:
- **name** (string / require): the ingress name
  - **spec** (object / require): the ingress spec. Read the [official doc to know the properties](https://kubernetes.io/docs/concepts/services-networking/ingress/).
  - **labels** (map of string): The list of labels to add on ingress. Default to `empty`
  - **annotations** (map of string): The list of annotations to add on ingress. Default to `empty`
  - **containerProtocol** (string): the protocol to set when create service consumed by ingress.`udp` or `tcp`
  - **containerPort** (number /require): the port to set when create service consumed by ingress

**logstash.yaml**:
```yaml
apiVersion: logstash.k8s.webcenter.fr/v1
kind: Logstash
metadata:
  name: logstash
  namespace: cluster-dev
spec:
  ingresses:
    - name: api
      spec:
        rules:
          - host: logstash-api-dev.domain.local
            http:
              paths:
                - backend:
                    service:
                      name: logstash-api
                      port:
                        number: 9600
                  path: /
                  pathType: Prefix
        tls:
          - hosts:
              - logstash-api-dev.domain.local
            secretName: ls-tls
      labels:
        label1: my label
      annotations:
        annotation1: my annotation
      containerProtocol: tcp
      containerPort: 9600
```

## Services endpoint

You can define as many as you like. It's the slice of ingresses.
it's permit to create custom services if needed by input.

You can use the following settings:
- **name** (string / require): the service name
  - **spec** (object / require): the service spec.  Read the [official doc to know the properties](https://kubernetes.io/fr/docs/concepts/services-networking/service/)
  - **labels** (map of string): The list of labels to add on service. Default to `empty`
  - **annotations** (map of string): The list of annotations to add on service. Default to `empty`

**logstash.yaml**:
```yaml
apiVersion: logstash.k8s.webcenter.fr/v1
kind: Logstash
metadata:
  name: logstash
  namespace: cluster-dev
spec:
  services:
    - name: beat
      labels:
        label1: my label
      annotations:
        annotation1: my annotation
      spec:
        ports:
          - name: beats
            port: 5003
            protocol: TCP
            targetPort: 5003
        type: ClusterIP
```