# Hot / warm cluster sample

In this sample, we will deploy an Elasticsearch cluster with hot / warm topology on namespace `cluster-dev`.
So we will create some dedicated nodes (node groups) per roles:
- ingest nodes: it's the endpoint for users and external applications. It run on not dedicated node.
- master nodes: It run on not dedicated node.
- hot nodes: nodes with SSD disk. It run on physical dedicated node. We use toleration and node selector.
- warm nodes: nodes with SATA disk. It run on physical dedicated node. We use toleration and node selector.

Some infos:
- It will be accessible by ingress with https://elasticsearch-cluster-prd.domain.local
- It use platinium license to get Active directory auth
- It attach PVC of type NFS to store snapshot
- It set hard anti affinity and rack awarness
- It inject custom CA on java cacerts
- It inject some secrets on java keystore
- It send metrics on dedicated Elasticsearch cluster via metricbeat. Take a look of official doc: https://www.elastic.co/guide/en/elasticsearch/reference/current/monitor-elasticsearch-cluster.html. And you can look the [Elasticsearch monitoring cluster sample](monitoring-sample.md)

> we need to create some resources here (secret, pvc) because of we use platinium license, we use Active directory auth and we add NFS volume to store snapshot. We also inject custom CA on cacerts to access on-premise S3 storage.

**elasticsearch.yaml**:
```yaml
apiVersion: elasticsearch.k8s.webcenter.fr/v1
kind: Elasticsearch
metadata:
  labels:
    socle: cluster-prd
  name: elasticsearch
  namespace: cluster-prd
spec:
  clusterName: cluster-prd
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
  globalNodeGroup:
    additionalVolumes:
      - mountPath: /mnt/snapshot
        name: snapshot
        persistentVolumeClaim:
          claimName: pvc-elasticsearch-snapshot
    antiAffinity:
      topologyKey: topology.kubernetes.io/zone
      type: hard
    caSecretRef:
      name: custom-ca
    config:
      elasticsearch.yml: |
        action.destructive_requires_name: true
        gateway.recover_after_time: 5m
        http.cors.allow-credentials: true
        http.cors.allow-headers: X-Requested-With,X-Auth-Token,Content-Type,
        Content-Length, Authorization
        http.cors.allow-origin: /.*/
        http.cors.enabled: true
        http.max_content_length: 500mb


        # Rack awarness
        cluster.routing.allocation.awareness.attributes: node_name

        # Repository
        path.repo:
          - /mnt/snapshot

        xpack.security.audit.enabled: true
        xpack.security.audit.logfile.events.exclude:
          - access_granted
        xpack.security.authc:
          anonymous:
            authz_exception: false
            roles: monitoring
            username: anonymous_user
          realms:
            active_directory.active_directory:
              order: 2
              url:
                - "ldap://dc1.domain.local:389"
                - "ldap://dc2.domain.local:389"
              load_balance.type: "failover"
              follow_referrals: true
              bind_dn: "${ELASTICSEARCH_LDAP_USER}"
              timeout.ldap_search: 60s
              domain_name: DOMAIN
              user_search:
                base_dn: "DC=DOMAIN,DC=LOCAL"
                scope: sub_tree
              group_search:
                base_dn: "OU=Users,DC=DOMAIN,DC=LOCAL"
                scope: sub_tree
              unmapped_groups_as_roles: false
        
        # Custom config
        gateway.expected_data_nodes: 3
        gateway.recover_after_data_nodes: 2
        cluster.routing.allocation.disk.watermark.low: 50gb
        cluster.routing.allocation.disk.watermark.high: 20gb
        cluster.routing.allocation.disk.watermark.flood_stage: 10gb
    envFrom:
      - secretRef:
          name: elasticsearch-env
    initContainerResources:
      limits:
        cpu: 500m
        memory: 256Mi
      requests:
        cpu: 25m
        memory: 64Mi
    keystoreSecretRef:
      name: elasticsearch-keystore
  licenseSecretRef:
    name: elasticsearch-license
  monitoring:
    metricbeat:
      elasticsearchRef:
        managed:
          name: elasticsearch
          namespace: logmanagement-monitoring-prd
      enabled: true
    prometheus:
      enabled: false
  nodeGroups:
    - name: master
      persistence:
        volumeClaim:
          accessModes:
            - ReadWriteOnce
          resources:
            requests:
              storage: 2Gi
          storageClassName: openebs-hostpath
      replicas: 3
      resources:
        limits:
          cpu: '2'
          memory: 4Gi
        requests:
          cpu: '2'
          memory: 4Gi
      roles:
        - master
      waitClusterStatus: green
    - name: hot
      nodeSelector:
        project: cluster-prd-hot
      persistence:
        volumeClaim:
          accessModes:
            - ReadWriteOnce
          resources:
            requests:
              storage: 1300Gi
          storageClassName: openebs-hostpath
      replicas: 3
      resources:
        limits:
          cpu: '11'
          memory: 62Gi
        requests:
          cpu: '11'
          memory: 62Gi
      roles:
        - data_hot
        - data_content
      tolerations:
        - effect: NoSchedule
          key: project
          operator: Equal
          value: cluster-prd-hot
      waitClusterStatus: green
    - name: warm
      nodeSelector:
        project: cluster-prd-warm
      persistence:
        volumeClaim:
          accessModes:
            - ReadWriteOnce
          resources:
            requests:
              storage: 3300Gi
          storageClassName: openebs-hostpath
      replicas: 6
      resources:
        limits:
          cpu: '11'
          memory: 62Gi
        requests:
          cpu: '11'
          memory: 62Gi
      roles:
        - data_warm
      tolerations:
        - effect: NoSchedule
          key: project
          operator: Equal
          value: cluster-prd-warm
      waitClusterStatus: green
    - name: client
      persistence:
        volumeClaim:
          accessModes:
            - ReadWriteOnce
          resources:
            requests:
              storage: 2Gi
          storageClassName: openebs-hostpath
      replicas: 2
      resources:
        limits:
          cpu: '2'
          memory: 4Gi
        requests:
          cpu: '2'
          memory: 4Gi
      roles:
        - ingest
      waitClusterStatus: yellow
  setVMMaxMapCount: true
  tls:
    enabled: true
    keySize: 2048
    renewalDays: 365
    validityDays: 1000
  version: 8.7.1

```

**custom-ca-secret.yaml**:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: custom-ca
  namespace: cluster-dev
type: Opaque
data:
  custom-ca.crt: ++++++++
```

**elasticsearch-env-secret.yaml**:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: elasticsearch-env
  namespace: cluster-dev
type: Opaque
data:
  ELASTICSEARCH_LDAP_USER: ++++++++
```

**elasticsearch-keystore-secret.yaml**:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: elasticsearch-keystore
  namespace: cluster-dev
type: Opaque
data:
  xpack.security.authc.realms.active_directory.active_directory.secure_bind_password: ++++++++
```

**elasticsearch-license-secret.yaml**:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: elasticsearch-license
  namespace: cluster-dev
type: Opaque
data:
  license: ++++++++
```

**pvc-elasticsearch-snapshot.yaml**:
```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: pvc-elasticsearch-snapshot
  namespace: cluster-dev
spec:
  accessModes:
    - ReadWriteMany
  resources:
    requests:
      storage: 10Gi
  storageClassName: nfs-client
```