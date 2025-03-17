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
	kubeVersion                 = "1.31.0"
	sdkVersion                  = "v1.37.0"
	controllerGenVersion        = "v0.16.1"
	kustomizeVersion            = "v5.4.3"
	cleanCrdVersion             = "v0.1.9"
	opmVersion                  = "v1.48.0"
	registry                    = "quay.io"
	repository                  = "webcenter/elasticsearch-operator"
	gitUsername          string = "github"
	gitEmail             string = "github@localhost"
	name                        = "elasticsearch-operator"
	defaultBranch               = "main"
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
	return &ElasticsearchOperator{
		Src:         src,
		OperatorSDK: dag.OperatorSDK(src.WithoutDirectory("ci"), name),
	}
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

	// Set the current branch name. It's needed because of CI overwrite the branch name by PR
	// +optional
	branchName string,

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
				Files:   []string{"coverage.out"},
				Verbose: true,
			},
		); err != nil {
			return nil, errors.Wrap(err, "Error when upload report on CodeCov")
		}

		catalogName, err := h.OperatorSDK.GetCatalogName(ctx, registry, repository)
		if err != nil {
			return nil, errors.Wrap(err, "Error when get catalog name")
		}
		service := h.OperatorSDK.TestOlmOperator(
			fmt.Sprintf("%s:%s", catalogName, version),
			name,
			dagger.OperatorSDKTestOlmOperatorOpts{
				Channel: strings.TrimSpace(strings.Split(channels, ",")[0]),
			},
		)
		defer service.Stop(ctx)

		// Deploy Elasticsearch operator to look
		kubeCtr := h.OperatorSDK.Kube().Kubectl().
			WithServiceBinding("kube.svc", service)

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

		// Compute the branch and directory
		var branch string
		git := dag.Git().
			SetConfig(gitUsername, gitEmail, dagger.GitSetConfigOpts{BaseRepoURL: "github.com", Token: gitToken})

		if !isTag {
			// keep original version file
			versionFile, err := h.Src.File("VERSION").Sync(ctx)
			if err == nil {
				dir = dir.WithFile("VERSION", versionFile)
			} else {
				dir = dir.WithoutFile("VERSION")
			}

			if branchName == "" {
				return nil, errors.New("You need to provide the branch name")
			}
			branch = branchName
		} else {
			branch = defaultBranch
		}

		if isPullRequest {
			git = git.With(func(r *dagger.Git) *dagger.Git {
				ctr := r.BaseContainer().
					WithDirectory("/project", dir).
					WithWorkdir("/project").
					WithExec(helper.ForgeCommand("git remote -v")).
					WithExec(helper.ForgeCommandf("git fetch origin %s:%s", branch, branch)).
					WithExec(helper.ForgeCommandf("git checkout %s", branch))

				return r.WithCustomContainer(ctr)
			})
		} else {
			git = git.SetRepo(h.Src.WithDirectory(".", dir), dagger.GitSetRepoOpts{Branch: branch})
		}
		if _, err = git.CommitAndPush(ctx, "Commit from CI pipeline"); err != nil {
			return nil, errors.Wrap(err, "Error when commit and push files change")
		}

	}

	return dir, nil
}
