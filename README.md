# Elasticsearch operator

## Todo
- Use https://github.com/banzaicloud/k8s-objectmatcher to diff object
- Add entry to set security plugin like authentification or authorization. Maybee it's secretRef ?
- Gnerate helm template: https://github.com/spectrocloud/kubesplit

## Create new API object

```bash
operator-sdk create api --group kibana --version v1alpha1 --kind Kibana --resource
```