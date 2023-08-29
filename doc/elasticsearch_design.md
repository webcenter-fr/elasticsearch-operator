# Elasticsearch design

The operator doing the following step when it reconsil `Elasticsearch`:
- Generate secret that store `admin` account. This account is used by operator, and so it never be change by external intervention.
- Generate TLS certificates for internal communication. You can't custom it.
  Under the wood, it will generate internal PKI.
  Ensure certificate not yet expire, else renew it.
  If certificate is renewed, it will restart node on rolling upgrade
- Generate TLS certificates for HTTP endpoints. You can disable it if you use Ingress. You can custom self signed certificate or use your own certificates instead.
  Ensure certificate not yet expire, else renew it.
  If certificate is renewed, it will restart node on rolling upgrade.
- Generate keystore for secret config file
  If contend change, it will restart node on rolling upgrade.
- Generate configMap for security plugin
  If contend change, it will restart node on rolling upgrade
- For each node groups
  - Generate Elasticsearch config as configMap
  - Generate statefullset
  - Generate service
  - Generate pod disruption budget
- Generate Job to setting security, 
  - internal account (admin, dashbord)
  - Authentification
  - Authorization
- Expose cluster
  - Generate Ingress if needed
  - Generate Service as LoadBalancer