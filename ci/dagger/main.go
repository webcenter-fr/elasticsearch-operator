// A generated module for ElasticsearchOperator functions
//
// This module has been generated via dagger init and serves as a reference to
// basic module structure as you get started with Dagger.
//
// Two functions have been pre-created. You can modify, delete, or add to them,
// as needed. They demonstrate usage of arguments and return types using simple
// echo and grep commands. The functions can be called from the dagger CLI or
// from one of the SDKs.
//
// The first line in this comment block is a short description line and the
// rest is a long description with more detail on the module's purpose or usage,
// if appropriate. All modules should have a short description.

package main

import (
	"context"
	"fmt"
	"strings"

	"dagger/elasticsearch-operator/internal/dagger"

	"emperror.dev/errors"
	"github.com/disaster37/dagger-library-go/lib/helper"
)

const (
	kubeVersion                 = "1.36.0"
	sdkVersion                  = "v1.39.2"
	controllerGenVersion        = "v0.18.0"
	kustomizeVersion            = "v5.4.3"
	cleanCrdVersion             = "v0.1.9"
	opmVersion                  = "v1.48.0"
	registry                    = "quay.io"
	repository                  = "webcenter/elasticsearch-operator"
	gitUsername          string = "github"
	gitEmail             string = "github@localhost"
	name                        = "elasticsearch-operator"
)

type ElasticsearchOperator struct {
	// +private
	Src *dagger.Directory

	// +private
	*dagger.OperatorSDK
}

func New(
	// The source directory
	// +required
	src *dagger.Directory,
) *ElasticsearchOperator {
	// Pre-install a recent govulncheck. The upstream golang module installs the
	// latest GitHub release tag (v1.1.4) which panics on generic code
	// ("Substituting types.Signatures with generic functions are currently
	// unsupported"). It is installed in /usr/local/bin because /go/bin is
	// shadowed by a cache mount in the golang module. The PATH is overridden so
	// /usr/local/bin is checked before the (cached) /go/bin binaries.
	golangCtr := dag.Container().From("golang:1.27.0").
		WithEnvVariable("PATH", "/usr/local/bin:/go/bin:/usr/local/go/bin:/usr/local/sbin:/usr/sbin:/usr/bin:/sbin:/bin").
		WithEnvVariable("GOBIN", "/usr/local/bin").
		WithExec([]string{"go", "install", "golang.org/x/vuln/cmd/govulncheck@v1.8.0"}).
		WithEnvVariable("GOBIN", "")

	return &ElasticsearchOperator{
		Src: src,
		OperatorSDK: dag.OperatorSDK(
			src.WithoutDirectory("ci"),
			name,
			dagger.OperatorSDKOpts{
				Container: golangCtr,
			},
		),
	}
}

// k3sEntrypoint setup the cgroup nesting because k3s only does it when running
// as PID 1, which doesn't happen in Dagger given that we're using our custom
// shim.
// Copied from https://github.com/moby/moby/blob/ed89041433a031cafc0a0f19cfe573c31688d377/hack/dind#L28-L37
const k3sEntrypoint = `#!/bin/sh

set -o errexit
set -o nounset

if [ -f /sys/fs/cgroup/cgroup.controllers ]; then
  echo "[$(date -Iseconds)] [CgroupV2 Fix] Evacuating Root Cgroup ..."
  # move the processes from the root group to the /init group,
  # otherwise writing subtree_control fails with EBUSY.
  mkdir -p /sys/fs/cgroup/init
  xargs -rn1 < /sys/fs/cgroup/cgroup.procs > /sys/fs/cgroup/init/cgroup.procs || :
  # enable controllers
  sed -e 's/ / +/g' -e 's/^/+/' <"/sys/fs/cgroup/cgroup.controllers" >"/sys/fs/cgroup/cgroup.subtree_control"
  echo "[$(date -Iseconds)] [CgroupV2 Fix] Done"
fi

exec "$@"
`

// TestOlmOperator start a k3s cluster, install OLM on it and deploy the
// operator using the given catalog image.
// It return the kube service and the kubeconfig file.
func (h *ElasticsearchOperator) testOlmOperator(
	ctx context.Context,

	// The catalog image to install
	// +required
	catalogImage string,

	// The operator name
	// +required
	name string,

	// The channel of the operator to install
	// +optional
	channel string,
) (*dagger.Service, *dagger.File, error) {
	if channel == "" {
		channel = "stable"
	}

	// Build the k3s server container. The containerd data and the kubelet
	// state are mounted on cache volumes (disk) instead of tmpfs because the
	// OLM catalog images are too large for a tmpfs mount on the runner: the
	// containerd process get killed and the cluster dies.
	configCache := dag.CacheVolume(fmt.Sprintf("k3s_config_%s", name))
	k3sCtr := dag.Container().
		From("rancher/k3s:v1.34.3-k3s1").
		WithNewFile("/usr/bin/entrypoint.sh", k3sEntrypoint, dagger.ContainerWithNewFileOpts{
			Permissions: 0o755,
		}).
		WithEntrypoint([]string{"entrypoint.sh"}).
		WithMountedCache("/etc/rancher/k3s", configCache).
		WithMountedCache("/var/lib/rancher/k3s", dag.CacheVolume(fmt.Sprintf("k3s_data_%s", name))).
		WithMountedCache("/var/lib/kubelet", dag.CacheVolume(fmt.Sprintf("k3s_kubelet_%s", name))).
		WithMountedTemp("/var/log").
		WithExec([]string{"sh", "-c", `
cat <<EOF > /etc/rancher/k3s/registries.yaml
configs:
  "*":
    tls:
      insecure_skip_verify: true
EOF`}).
		WithExposedPort(6443)

	service := k3sCtr.AsService(dagger.ContainerAsServiceOpts{
		UseEntrypoint: true,
		Args: []string{
			"sh", "-c",
			"k3s server --bind-address $(ip route | grep src | awk '{print $NF}') --disable traefik --disable metrics-server --egress-selector-mode=disabled --cluster-cidr=10.44.0.0/16 --service-cidr=10.45.0.0/16",
		},
		InsecureRootCapabilities: true,
	})

	// Get the kubeconfig from the config cache. Wait until the k3s server
	// wrote it.
	kubeconfig := dag.Container().
		From("alpine").
		WithMountedCache("/cache/k3s", configCache).
		WithExec([]string{"sh", "-c", "n=0; until [ -f /cache/k3s/k3s.yaml ] || [ $n -ge 180 ]; do sleep 1; n=$((n+1)); done; cp /cache/k3s/k3s.yaml /kubeconfig.yaml"}).
		File("/kubeconfig.yaml")

	if _, err := service.Start(ctx); err != nil {
		return nil, nil, errors.Wrap(err, "Error when start k3s cluster")
	}

	// Kubectl container
	kubectl := dag.Container().
		From("bitnami/kubectl").
		WithoutEntrypoint().
		WithFile("/.kube/config", kubeconfig, dagger.ContainerWithFileOpts{Owner: "1001"}).
		WithUser("1001")

	// Wait that kube is ready
	if _, err := kubectl.
		WithExec([]string{"sh", "-c", "n=0; until kubectl get --raw /readyz >/dev/null 2>&1 || [ $n -ge 240 ]; do sleep 1; n=$((n+1)); done; kubectl get --raw /readyz"}).
		Stdout(ctx); err != nil {
		return nil, nil, errors.Wrap(err, "Error when wait k3s cluster ready")
	}

	// Install OLM
	if _, err := h.SDK().Container().
		WithFile("/kubeconfig", kubeconfig).
		WithEnvVariable("KUBECONFIG", "/kubeconfig").
		WithExec(helper.ForgeCommand("operator-sdk olm install")).
		Stdout(ctx); err != nil {
		return nil, nil, errors.Wrap(err, "Error when install OLM")
	}

	// Install Prometheus CRD needed by the operator
	if _, err := kubectl.
		WithExec([]string{"kubectl", "apply", "--server-side=true", "-f", "https://raw.githubusercontent.com/prometheus-community/helm-charts/refs/heads/main/charts/kube-prometheus-stack/charts/crds/crds/crd-servicemonitors.yaml"}).
		WithExec([]string{"kubectl", "apply", "--server-side=true", "-f", "https://raw.githubusercontent.com/prometheus-community/helm-charts/refs/heads/main/charts/kube-prometheus-stack/charts/crds/crds/crd-podmonitors.yaml"}).
		Stdout(ctx); err != nil {
		return nil, nil, errors.Wrap(err, "Error when install ServiceMonitor / PodMonitor CRD")
	}

	catalogYaml := fmt.Sprintf(`apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: test
  namespace: olm
spec:
  sourceType: grpc
  image: %s
`, catalogImage)

	subscriptionYaml := fmt.Sprintf(`apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: test
  namespace: operators
spec:
  catalogSource: test
  catalogSourceNamespace: olm
  channel: %s
  installPlanApproval: Automatic
  package: %s
`, channel, name)

	// Install catalog and subscription
	if _, err := kubectl.
		WithNewFile("/tmp/catalog.yaml", catalogYaml).
		WithNewFile("/tmp/subscription.yaml", subscriptionYaml).
		WithExec([]string{"kubectl", "apply", "--server-side=true", "-f", "/tmp/catalog.yaml"}).
		WithExec([]string{"kubectl", "apply", "--server-side=true", "-f", "/tmp/subscription.yaml"}).
		Stdout(ctx); err != nil {
		return nil, nil, errors.Wrap(err, "Error when install catalog and subscription")
	}

	// Wait the time it install OLM operator
	kubeCtn := kubectl.WithExec([]string{"sleep", "120"})

	// Get some trace to troubleshooting if needed
	_, _ = kubeCtn.
		WithExec([]string{"kubectl", "-n", "olm", "get", "pods"}).
		WithExec([]string{"kubectl", "-n", "olm", "describe", "catalogSource", "test"}).
		WithExec([]string{"kubectl", "-n", "operators", "describe", "subscription", "test"}).
		WithExec([]string{"kubectl", "-n", "operators", "describe", "installplan"}).
		WithExec([]string{"kubectl", "-n", "operators", "describe", "clusterServiceVersion"}).
		WithExec([]string{"kubectl", "-n", "operators", "get", "all"}).
		Stdout(ctx)

	// Check that the deployment operator is ready
	if _, err := kubeCtn.WithExec([]string{"kubectl", "-n", "operators", "wait", "--for=condition=Available=True", "--all", "deployment", "--timeout=120s"}).Stdout(ctx); err != nil {
		return nil, nil, errors.Wrap(err, "Operator not ready")
	}

	return service, kubeconfig, nil
}

func (h *ElasticsearchOperator) Test(
	ctx context.Context,
	// if only short running tests should be executed
	// +optional
	short bool,
	// if the tests should be executed out of order
	// +optional
	shuffle bool,
	// run select tests only, defined using a regex
	// +optional
	run string,
	// skip select tests, defined using a regex
	// +optional
	skip string,
	// Run test with gotestsum
	// +optional
	withGotestsum bool,
	// Path to test
	// +optional
	path string,
) *dagger.File {
	return h.Golang().Test(dagger.OperatorSDKGolangTestOpts{
		Short:           short,
		Shuffle:         shuffle,
		Run:             run,
		Skip:            skip,
		WithGotestsum:   withGotestsum,
		Path:            path,
		WithKubeversion: kubeVersion,
	})
}

// Bundle generate the bundle
func (h *ElasticsearchOperator) GenerateBundle(
	ctx context.Context,

	// The current version
	// +required
	version string,

	// The channels
	// +optional
	channels string,

	// The previous version
	// +optional
	previousVersion string,
) *dagger.Directory {
	return h.SDK().GenerateBundle(
		fmt.Sprintf("%s:%s", registry, repository),
		version,
		dagger.OperatorSDKSDKGenerateBundleOpts{
			Channels:        channels,
			PreviousVersion: previousVersion,
		},
	)
}

// Release permit to release to operator version
func (h *ElasticsearchOperator) CI(
	ctx context.Context,

	// The version to release
	// +required
	version string,

	// Set true to run tests
	// +optional
	ci bool,

	// Set true if current build is a tag
	// It will use the stable and alpha channel
	// alpha channel only instead
	// +optional
	isTag bool,

	// Set true if current build is a Pull request
	// +optional
	isPullRequest bool,

	// The git branch where you should to push
	// You need to provide it when you are on PullRequest or on Tag
	// +optional
	gitBranch string,

	// Set true to skip test
	// +optional
	skipTest bool,

	// The registry username
	// +optional
	registryUsername *dagger.Secret,

	// The registry password
	// +optional
	registryPassword *dagger.Secret,

	// The git token
	// +optional
	gitToken *dagger.Secret,

	// The codeCov token
	// +optional
	codeCoveToken *dagger.Secret,
) (*dagger.Directory, error) {
	var channels string
	var err error

	// Compute channel
	if isTag {
		channels = "stable"
	} else {
		channels = "alpha"
	}

	// Compute username registry
	var username string
	if ci {
		username, err = registryUsername.Plaintext(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "Error when get registry username")
		}
	}

	version, err = h.GetVersion(
		ctx,
		version,
		dagger.OperatorSDKGetVersionOpts{
			IsBuildNumber: !isTag,
		},
	)
	if err != nil {
		return nil, errors.Wrap(err, "Error when get the target version")
	}

	h.OperatorSDK = h.Release(
		version,
		registry,
		repository,
		dagger.OperatorSDKReleaseOpts{
			Channels:                     channels,
			KubeVersion:                  kubeVersion,
			WithTest:                     !skipTest,
			WithPublish:                  ci,
			RegistryUsername:             username,
			RegistryPassword:             registryPassword,
			PublishLast:                  false,
			SkipBuildFromPreviousVersion: !isTag,
		},
	)

	// Put ci folder to not lost it
	dir := h.OperatorSDK.GetSource().WithDirectory("ci", h.Src.Directory("ci"))

	// Test the OLM operator
	if ci {

		// codecov
		if _, err := dag.Codecov().Upload(
			ctx,
			dir,
			codeCoveToken,
			dagger.CodecovUploadOpts{
				Files: []string{"coverage.out"},
			},
		); err != nil {
			return nil, errors.Wrap(err, "Error when upload report on CodeCov")
		}

		catalogName, err := h.OperatorSDK.GetCatalogName(ctx, registry, repository)
		if err != nil {
			return nil, errors.Wrap(err, "Error when get catalog name")
		}
		service, kubeconfig, err := h.testOlmOperator(
			ctx,
			fmt.Sprintf("%s:%s", catalogName, version),
			name,
			strings.TrimSpace(strings.Split(channels, ",")[0]),
		)
		if err != nil {
			return nil, errors.Wrap(err, "Error when test OLM operator")
		}
		defer service.Stop(ctx)

		// Deploy Elasticsearch cluster to test the operator
		kubeCtr := dag.Container().
			From("bitnami/kubectl").
			WithoutEntrypoint().
			WithFile("/.kube/config", kubeconfig, dagger.ContainerWithFileOpts{Owner: "1001"}).
			WithUser("1001").
			WithDirectory("/src", dir).
			WithWorkdir("/src")

		_, err = kubeCtr.
			WithExec(helper.ForgeCommand("kubectl apply -n default --server-side=true -f config/samples/elasticsearch_v1_elasticsearch.yaml")).
			WithExec(helper.ForgeCommand("kubectl -n default wait --for=condition=Ready=True --all elasticsearch --timeout=180s")).
			Stdout(ctx)

		// Get operators logs and Elasticsearch logs
		_, _ = kubeCtr.
			WithExec(helper.ForgeScript("kubectl get -n operators pods -o name | xargs -I {} kubectl logs -n operators {}")).
			WithExec(helper.ForgeScript("kubectl get -n default pods -o name | xargs -I {} kubectl logs -n default {}")).
			Stdout(ctx)

		if err != nil {
			return nil, errors.Wrap(err, "Error when deploy Elasticsearch cluster for testing operator")
		}

		// Publish latest image
		if isTag {
			if _, err = h.OperatorSDK.Oci().PublishCatalog(ctx, fmt.Sprintf("%s:latest", catalogName)); err != nil {
				return nil, errors.Wrap(err, "Error when publish the latest catalog image")
			}
		}

		if !isTag {
			// keep original version file
			versionFile, err := h.Src.File("VERSION").Sync(ctx)
			if err == nil {
				dir = dir.WithFile("VERSION", versionFile)
			} else {
				dir = dir.WithoutFile("VERSION")
			}
		}
		git := dag.GitModule(dir, dagger.GitModuleOpts{Ci: "github"}).
			SetConfig(dagger.GitModuleSetConfigOpts{
				Username: gitUsername,
				Email:    gitEmail,
			})

		if _, err = git.CommitAndPush(
			ctx,
			gitToken,
			dagger.GitModuleCommitAndPushOpts{
				BranchName: gitBranch,
				GitRepoURL: "https://github.com/webcenter-fr/elasticsearch-operator.git",
				Message:    "Commit from CI",
			},
		); err != nil {
			return nil, errors.Wrap(err, "Error when commit and push files change")
		}

	}

	return dir, nil
}
