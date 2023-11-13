# Endpoint setting for Filebeat

You can use ingress or service endpoint for Filebeat. You can define as many as you like

## Ingresses endpoint

You can define as many as you like. It's the slice of ingresses.
It's permit to create custom ingresses if needed by input or to access on Filebeat API. It will set automatically the service needed by ingress.

You can use the following settings:
- **name** (string / require): the ingress name
  - **spec** (object / require): the ingress spec. Read the [official doc to know the properties](https://kubernetes.io/docs/concepts/services-networking/ingress/).
  - **labels** (map of string): The list of labels to add on ingress. Default to `empty`
  - **annotations** (map of string): The list of annotations to add on ingress. Default to `empty`
  - **containerProtocol** (string): the protocol to set when create service consumed by ingress.`udp` or `tcp`
  - **containerPort** (number /require): the port to set when create service consumed by ingress

**filebeat.yaml**:
```yaml
apiVersion: beat.k8s.webcenter.fr/v1
kind: Filebeat
metadata:
  name: filebeat
  namespace: cluster-dev
spec:
  ingresses:
    - name: api
      spec:
        rules:
          - host: filebeat-api-dev.domain.local
            http:
              paths:
                - backend:
                    service:
                      name: filebeat-api
                      port:
                        number: 5066
                  path: /
                  pathType: Prefix
        tls:
          - hosts:
              - filebeat-api-dev.domain.local
            secretName: ls-tls
      labels:
        label1: my label
      annotations:
        annotation1: my annotation
      containerProtocol: tcp
      containerPort: 5066
```

## Services endpoint

You can define as many as you like. It's the slice of ingresses.
it's permit to create custom services if needed by input.

You can use the following settings:
- **name** (string / require): the service name
  - **spec** (object / require): the service spec.  Read the [official doc to know the properties](https://kubernetes.io/fr/docs/concepts/services-networking/service/)
  - **labels** (map of string): The list of labels to add on service. Default to `empty`
  - **annotations** (map of string): The list of annotations to add on service. Default to `empty`

**filebeat.yaml**:
```yaml
apiVersion: beat.k8s.webcenter.fr/v1
kind: Filebeat
metadata:
  name: filebeat
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
          - name: linux
            nodePort: 30016
            port: 5144
            protocol: TCP
            targetPort: 5144
        type: ClusterIP
```