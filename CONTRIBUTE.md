# Contribute

PR are awlays welcome here. Please use the `main` branch to start.

## Getting start

You need to the following tools:
  - [dagger cli](https://docs.dagger.io/install/)
  - [kubectl](https://kubernetes.io/fr/docs/tasks/tools/install-kubectl/)
  - [direnv](https://direnv.net/)

Create a fix or feature branch and then make your stuff on it.
After that, you are ready to make a Pull request. The PR will launch the CI and if all is right, it will publish the new catalog image that you and git owner will test before to merge the PR.

### Get binaries tools on local

```bash
dagger call -m operator-sdk --src . sdk get-cli export --path ./bin
```

### Test operator on local run
To test it, you need to open 2 shell

First shell
```bash
dagger call -m operator-sdk --src . run-operator up
```

Second shell
```bash
# Get the kubeconfig file
dagger call -m operator-sdk --src . kube kubeconfig --local export --path kubeconfig

# Deploy elasticsearch cluster
kubectl apply -f config/samples/elasticsearch_v1_elasticsearch.yaml -n default

# Expose elasticsearch on localhost to work with local operator
# Without that, the local operator can access on elasticsearch API
kubectl port-forward service/elasticsearch-sample-os -n default 9200:9200
```


### Test the OLM build
To test it, you need to open 2 shell

First shell
```bash
# Put the right tag of your image. It change on each CI build

```bash
dagger call -m operator-sdk --src . test-olm-operator --catalog-image quay.io/webcenter/elasticsearch-operator-catalog:0.0.74-pr58 --name elasticsearch-operator --channel alpha up

# If not work, you can use instead
dagger call -m operator-sdk --src . install-olm-operator --catalog-image quay.io/webcenter/elasticsearch-operator-catalog:0.0.74-pr58 --name elasticsearch-operator --channel alpha up


Second shell
```bash
dagger call -m operator-sdk --src . kube kubeconfig --local export --path kubeconfig

# It auto if you have direnv
#export KUBECONFIG=kubeconfig

kubectl config set-context --current --namespace=operators
kubectl logs -f -l control-plane=elasticsearch-operator

# If pod not working like expected, you can test this step
kubectl describe subscription test
kubectl describe installplan test
kubectl describe deployment 

# Deploy elasticsearch cluster
kubectl apply -n default -f config/samples/elasticsearch_v1_elasticsearch.yaml
kubectl logs -f -l control-plane=elasticsearch-operator

```

## CI / tools

We use dagger.io to run local task or to run pipeline on CI.

### Run all step on local (without push image)

```bash
dagger call --src . ci
```

### Format code

```bash
dagger call -m golang --src . format export --path .
```

### Lint Golang project

```bash
dagger call -m golang --src . lint
```

### Vulnerability check

```bash
dagger call -m golang --src . vulncheck
```

### Run local test with envtest

```bash
dagger call --src . test --withGotestsum
```

### Invoke operator-sdk cli

```bash
dagger call --src . sdk run --cmd version stdout
```

### Generate SDK manifests

```bash
dagger call -m operator-sdk --src . sdk generate-manifests export --path .
```


### Generate Bundle manifest

```bash
dagger call --src . generate-bundle --version 0.0.72 export --path .
```


### Run operator localy

**Start k3s cluster**:
```bash
dagger call --src . cluster up
```

**Start operator**:
```bash
dagger call --src . kubeconfig export --path kubeconfig
ENABLE_WEBHOOKS=false LOG_LEVEL=trace LOG_FORMATTER=json go run cmd/main.go
```

**Load samples**:
```bash
kubectl apply -f config/samples/elasticsearch_v1alpha1_elasticsearch.yaml
```

