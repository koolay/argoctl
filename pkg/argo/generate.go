package argo

import (
	"context"
	"fmt"
	"log"
	"time"

	argoappv1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	repoapiclient "github.com/argoproj/argo-cd/v2/reposerver/apiclient"
	"github.com/argoproj/argo-cd/v2/reposerver/cache"
	"github.com/argoproj/argo-cd/v2/reposerver/metrics"
	"github.com/argoproj/argo-cd/v2/reposerver/repository"
	"github.com/argoproj/argo-cd/v2/util/argo"
	cacheutil "github.com/argoproj/argo-cd/v2/util/cache"
	"github.com/argoproj/argo-cd/v2/util/git"
)

type Generater struct{}

var parallelismLimit = 100

func (g *Generater) FromHelm(
	ctx context.Context,
	appName, revision, namespace, chart, repoURL string,
) ([]string, error) {
	var (
		// k8s cluster version
		kubeVersion string
		// APIVersions contains list of API versions supported by the cluster
		apiVersions []string
		// add labels for every rs
		appLabelKey = "application"
		storePath   = "/Users/huwl/tmp/gitops"
	)
	ms := metrics.NewMetricsServer()
	cs := cache.NewCache(
		cacheutil.NewCache(cacheutil.NewInMemoryCache(20*time.Minute)),
		10*time.Minute,
		10*time.Minute,
	)

	s := repository.NewService(ms,
		cs,
		repository.RepoServerInitConstants{ParallelismLimit: int64(parallelismLimit)},
		argo.NewResourceTracking(),
		&git.NoopCredsStore{},
		storePath,
	)

	if err := s.Init(); err != nil {
		log.Fatal(err)
	}

	log.Println("repoURL", repoURL, "chart", chart)

	res, err := s.GenerateManifest(ctx, &repoapiclient.ManifestRequest{
		Revision:    revision,
		AppLabelKey: appLabelKey,
		AppName:     appName,
		Namespace:   namespace,
		// we don't need git repo
		Repo: &argoappv1.Repository{
			Repo: repoURL,
		},
		ApplicationSource: &argoappv1.ApplicationSource{
			RepoURL:        repoURL,
			Chart:          chart,
			Path:           "",
			TargetRevision: revision,
			Helm: &argoappv1.ApplicationSourceHelm{
				Version: "",
				// ValueFiles:      []string{"values.yaml"},
				ReleaseName:     appName,
				PassCredentials: false,
			},
		},
		KustomizeOptions: nil,
		KubeVersion:      kubeVersion,
		ApiVersions:      apiVersions,
		Plugins:          nil,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate manifests of helm: %s, error: %w", chart, err)
	}

	return res.Manifests, err
}
