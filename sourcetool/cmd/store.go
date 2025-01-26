/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"log"

	"github.com/slsa-framework/slsa-source-poc/sourcetool/pkg/store"
	"github.com/spf13/cobra"
)

type StoreArgs struct {
	// Path to a file with the data to store.
	commit string
	// The predicate type of the data.
	predType string
}

var (
	storeArgs StoreArgs
	// storeCmd represents the store command
	storeCmd = &cobra.Command{
		Use:   "store --commit COMMIT --predType TYPE path",
		Short: "Stores data from a file, associated with a commit in refs/slsa/commits",
		Long: `Stores data from a file, associated with a commit in refs/slsa/commits.

The data will be stored in the git repo this command is run in.
Assumes the data is an in-toto attestation (https://github.com/in-toto/attestation).`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			dataPath := args[0]
			storedPath, err := store.Store(storeArgs.commit, storeArgs.predType, dataPath)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Printf("Storing %s in %s\n", dataPath, storedPath)
		},
	}
)

func init() {
	rootCmd.AddCommand(storeCmd)

	// Here you will define your flags and configuration settings.
	storeCmd.Flags().StringVar(&storeArgs.commit, "commit", "", "The commit to associate the data with. Default: current commit.")
	storeCmd.Flags().StringVar(&storeArgs.predType, "pred_type", "https://slsa.dev/verification_summary/v1", "The predicate type of the data being stored.")
}
