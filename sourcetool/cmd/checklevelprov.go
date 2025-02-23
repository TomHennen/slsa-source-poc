/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/slsa-framework/slsa-source-poc/sourcetool/pkg/attest"
	"github.com/slsa-framework/slsa-source-poc/sourcetool/pkg/gh_control"
	"github.com/slsa-framework/slsa-source-poc/sourcetool/pkg/policy"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/spf13/cobra"
)

type CheckLevelProvArgs struct {
	prevBundlePath       string
	commit               string
	prevCommit           string
	owner                string
	repo                 string
	branch               string
	outputUnsignedBundle string
	outputSignedBundle   string
	expectedIssuer       string
	expectedSan          string
}

// checklevelprovCmd represents the checklevelprov command
var (
	checkLevelProvArgs CheckLevelProvArgs

	checklevelprovCmd = &cobra.Command{
		Use:   "checklevelprov",
		Short: "Checks the given commit against policy using & creating provenance",
		Run: func(cmd *cobra.Command, args []string) {
			doCheckLevelProv(checkLevelProvArgs)
		},
	}
)

func doCheckLevelProv(checkLevelProvArgs CheckLevelProvArgs) {
	gh_connection :=
		gh_control.NewGhConnection(checkLevelProvArgs.owner, checkLevelProvArgs.repo, checkLevelProvArgs.branch)
	ctx := context.Background()

	ver_options := attest.DefaultVerifierOptions
	if checkLevelProvArgs.expectedIssuer != "" {
		ver_options.ExpectedIssuer = checkLevelProvArgs.expectedIssuer
	}
	if checkLevelProvArgs.expectedSan != "" {
		ver_options.ExpectedSan = checkLevelProvArgs.expectedSan
	}

	pa := attest.NewProvenanceAttestor(gh_connection, ver_options)
	p, err := pa.CreateSourceProvenance(ctx, checkLevelProvArgs.prevBundlePath, checkLevelProvArgs.commit, checkLevelProvArgs.prevCommit)
	if err != nil {
		log.Fatal(err)
	}

	// check p against policy
	level, err := policy.EvaluateProv(ctx, gh_connection, p)
	if err != nil {
		log.Fatal(err)
	}

	// create vsa
	unsignedVsa, err := attest.CreateUnsignedSourceVsa(gh_connection, checkLevelProvArgs.commit, level)
	if err != nil {
		log.Fatal(err)
	}

	unsignedProv, err := protojson.Marshal(p)
	if err != nil {
		log.Fatal(err)
	}

	// Store both the unsigned provenance and vsa
	if checkLevelProvArgs.outputUnsignedBundle != "" {
		f, err := os.OpenFile(checkLevelProvArgs.outputUnsignedBundle, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()

		f.WriteString(string(unsignedProv))
		f.WriteString("\n")
		f.WriteString(unsignedVsa)
		f.WriteString("\n")
	} else if checkLevelProvArgs.outputSignedBundle != "" {
		f, err := os.OpenFile(checkLevelProvArgs.outputSignedBundle, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()

		signedProv, err := attest.Sign(string(unsignedProv))
		if err != nil {
			log.Fatal(err)
		}

		signedVsa, err := attest.Sign(unsignedVsa)
		if err != nil {
			log.Fatal(err)
		}

		f.WriteString(signedProv)
		f.WriteString("\n")
		f.WriteString(signedVsa)
		f.WriteString("\n")
	} else {
		// Just output to the screen
		fmt.Println(string(unsignedProv))
		fmt.Println(string(unsignedVsa))
	}
}

func init() {
	rootCmd.AddCommand(checklevelprovCmd)

	checklevelprovCmd.Flags().StringVar(&checkLevelProvArgs.prevBundlePath, "prev_bundle_path", "", "Path to the file with the attestations for the previous commit (as an in-toto bundle).")
	checklevelprovCmd.Flags().StringVar(&checkLevelProvArgs.commit, "commit", "", "The commit to check.")
	checklevelprovCmd.Flags().StringVar(&checkLevelProvArgs.prevCommit, "prev_commit", "", "The commit to check.")
	checklevelprovCmd.Flags().StringVar(&checkLevelProvArgs.owner, "owner", "", "The GitHub repository owner - required.")
	checklevelprovCmd.Flags().StringVar(&checkLevelProvArgs.repo, "repo", "", "The GitHub repository name - required.")
	checklevelprovCmd.Flags().StringVar(&checkLevelProvArgs.branch, "branch", "", "The branch within the repository - required.")
	checklevelprovCmd.Flags().StringVar(&checkLevelProvArgs.outputUnsignedBundle, "output_unsigned_bundle", "", "The path to write a bundle of unsigned attestations.")
	checklevelprovCmd.Flags().StringVar(&checkLevelProvArgs.outputSignedBundle, "output_signed_bundle", "", "The path to write a bundle of signed attestations.")
	checklevelprovCmd.Flags().StringVar(&checkLevelProvArgs.expectedIssuer, "expected_issuer", "", "The expected issuer of attestations.")
	checklevelprovCmd.Flags().StringVar(&checkLevelProvArgs.expectedSan, "expected_san", "", "The expect san of attestations.")

}
