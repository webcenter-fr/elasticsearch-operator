# Single cluster node sample

In this sample, we will deploy an Elasticsearch cluster with single node on namespace `cluster-dev`

Some infos:
- It will get the data, master and ingest role.
- It will be accessible by ingress with https://elasticsearch-cluster-dev.domain.local
- It use platinium license to get Active directory auth
- It attach PVC of type NFS to store snapshot
- It set soft anti affinity (not really need on single node)
- It inject custom CA on java cacerts
- It inject some secrets on java keystore

> we need to create some resources here (secret, pvc) because of we use platinium license, we use Active directory auth and we add NFS volume to store snapshot. We also inject custom CA on cacerts to access on-premise S3 storage.

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
  clusterName: cluster-dev
  endpoint:
    ingress:
      enabled: true
      annotations:
        nginx.ingress.kubernetes.io/proxy-body-size: 512M
        nginx.ingress.kubernetes.io/proxy-connect-timeout: '600'
        nginx.ingress.kubernetes.io/proxy-read-timeout: '600'
        nginx.ingress.kubernetes.io/proxy-send-timeout: '600'
        nginx.ingress.kubernetes.io/ssl-redirect: 'true'
      host: elasticsearch-cluster-dev.domain.local
      secretRef:
        name: es-tls
  globalNodeGroup:
    additionalVolumes:
      - mountPath: /mnt/snapshot
        name: snapshot
        persistentVolumeClaim:
          claimName: pvc-elasticsearch-snapshot
    antiAffinity:
      type: soft
    caSecretRef:
      name: custom-ca
    config:
      action.destructive_requires_name: true
      gateway.recover_after_time: 5m
      http.cors.allow-credentials: true
      http.cors.allow-headers: X-Requested-With,X-Auth-Token,Content-Type,
      Content-Length, Authorization
      http.cors.allow-origin: /.*/
      http.cors.enabled: true
      http.max_content_length: 500mb
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
      cluster.routing.allocation.disk.watermark.flood_stage: 1gb
      cluster.routing.allocation.disk.watermark.high: 1gb
      cluster.routing.allocation.disk.watermark.low: 2gb
      gateway.expected_data_nodes: 1
      gateway.recover_after_data_nodes: 1
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
  nodeGroups:
    - name: all
      persistence:
        volumeClaim:
          accessModes:
            - ReadWriteOnce
          resources:
            requests:
              storage: 20Gi
          storageClassName: local-path
      replicas: 1
      resources:
        limits:
          cpu: 1000m
          memory: 4Gi
        requests:
          cpu: 250m
          memory: 4Gi
      roles:
        - master
        - data_hot
        - data_content
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