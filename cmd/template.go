/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"context"
	"fmt"
	"log"

	"github.com/koolay/quickstart-deploy/pkg/argo"
	"github.com/spf13/cobra"
)

// templateCmd represents the template command
var templateCmd = &cobra.Command{
	Use:   "template",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("template called")

		var (
			appName      = "nginx"
			appNamespace = "nginx-system"
			chart        = "nginx"
			// add labels for every rs
			revision = "10.2.1"
			repoURL  = "https://charts.bitnami.com/bitnami"
		)

		gen := argo.Generater{}

		manifests, err := gen.FromHelm(context.Background(), appName, revision, appNamespace, chart, repoURL)
		if err != nil {
			log.Fatal(err)
		}

		for _, manifest := range manifests {
			fmt.Println("----------")
			fmt.Println(manifest)
		}
	},
}

func init() {
	rootCmd.AddCommand(templateCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// templateCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// templateCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
