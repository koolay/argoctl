/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/argoproj/gitops-engine/pkg/cache"
	"github.com/argoproj/gitops-engine/pkg/engine"
	"github.com/argoproj/gitops-engine/pkg/sync"
	"github.com/argoproj/gitops-engine/pkg/utils/kube"
	"github.com/koolay/quickstart-deploy/pkg/argo"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2/klogr"
)

const (
	annotationGCMark = "gitops-agent.argoproj.io/gc-mark"
)

var (
	repoPath string
	// Directory path with-in repository
	paths []string
)

type resourceInfo struct {
	gcMark string
}

// syncCmd represents the sync command
var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		var (
			appName   = "grafana"
			namespace = "grafana-system"
			kubconfig string

			chart = "grafana"
			// add labels for every rs
			revision = "7.6.34"
			repoURL  = "https://charts.bitnami.com/bitnami"
		)

		var prune bool
		logger := klogr.New()

		if kubconfig == "" {
			kubconfig = os.Getenv("KUBECONFIG")
		}

		restConfig, err := clientcmd.BuildConfigFromFlags("", kubconfig)
		if err != nil {
			log.Fatalf("failed to build Kubernet, error: %v", err)
		}

		ctx := context.Background()
		namespaces := []string{namespace}

		gen := argo.Generater{}
		res, err := gen.FromHelm(ctx, appName, revision, namespace, chart, repoURL)
		if err != nil {
			log.Fatal(err)
		}

		var objs []*unstructured.Unstructured
		for _, r := range res {
			var obj unstructured.Unstructured
			if err := json.Unmarshal([]byte(r), &obj); err != nil {
				log.Fatal(err)
			}
			objs = append(objs, &obj)
		}

		clusterCache := cache.NewClusterCache(
			restConfig,
			cache.SetNamespaces(namespaces),
			cache.SetLogr(logger),
			cache.SetPopulateResourceInfoHandler(
				func(un *unstructured.Unstructured, isRoot bool) (info interface{}, cacheManifest bool) {
					// store gc mark of every resource
					gcMark := un.GetAnnotations()[annotationGCMark]
					info = &resourceInfo{gcMark: un.GetAnnotations()[annotationGCMark]}
					// cache resources that has that mark to improve performance
					cacheManifest = gcMark != ""
					return
				},
			),
		)

		gitOpsEngine := engine.NewEngine(restConfig, clusterCache, engine.WithLogr(logger))
		cleanup, err := gitOpsEngine.Run()
		if err != nil {
			log.Fatalf("failed to run gitops engine: %v", err)
		}

		defer cleanup()

		result, err := gitOpsEngine.Sync(context.Background(), objs, func(r *cache.Resource) bool {
			return true
			// return r.Info.(*resourceInfo).gcMark == getGCMark(r.ResourceKey())
		}, revision, namespace, sync.WithPrune(prune), sync.WithLogr(logger))
		if err != nil {
			log.Fatal(err)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintf(w, "RESOURCE\tRESULT\n")
		for _, res := range result {
			_, _ = fmt.Fprintf(w, "%s\t%s\n", res.ResourceKey.String(), res.Message)
		}
		_ = w.Flush()

		fmt.Println("sync called")
	},
}

func GetGCMark(key kube.ResourceKey) string {
	h := sha256.New()
	_, _ = h.Write([]byte(fmt.Sprintf("%s/%s", repoPath, strings.Join(paths, ","))))
	_, _ = h.Write([]byte(strings.Join([]string{key.Group, key.Kind, key.Name}, "/")))
	return "sha256." + base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

func init() {
	rootCmd.AddCommand(syncCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// syncCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// syncCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
