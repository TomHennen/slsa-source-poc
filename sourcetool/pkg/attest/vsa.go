package attest

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"strings"

	vpb "github.com/in-toto/attestation/go/predicates/vsa/v1"
	spb "github.com/in-toto/attestation/go/v1"
	"github.com/slsa-framework/slsa-source-poc/sourcetool/pkg/gh_control"
	"github.com/slsa-framework/slsa-source-poc/sourcetool/pkg/slsa_types"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const VsaPredicateType = "https://slsa.dev/verification_summary/v1"

func CreateUnsignedSourceVsa(repoUri, ref, commit string, verifiedLevels slsa_types.SourceVerifiedLevels, policy string) (string, error) {
	resourceUri := fmt.Sprintf("git+%s", repoUri)
	vsaPred := &vpb.VerificationSummary{
		Verifier: &vpb.VerificationSummary_Verifier{
			Id: "https://github.com/slsa-framework/slsa-source-poc"},
		TimeVerified:       timestamppb.Now(),
		ResourceUri:        resourceUri,
		Policy:             &vpb.VerificationSummary_Policy{Uri: policy},
		VerificationResult: "PASSED",
		VerifiedLevels:     verifiedLevels,
	}

	predJson, err := protojson.Marshal(vsaPred)
	if err != nil {
		return "", err
	}

	// TODO: update to source_refs to match updated spec.
	branchAnnotation := map[string]any{"source_branches": []any{ref}}
	annotationStruct, err := structpb.NewStruct(branchAnnotation)
	if err != nil {
		return "", fmt.Errorf("creating struct from map: %w", err)
	}
	sub := []*spb.ResourceDescriptor{{
		Digest:      map[string]string{"gitCommit": commit},
		Annotations: annotationStruct,
	}}

	var predPb structpb.Struct
	err = protojson.Unmarshal(predJson, &predPb)
	if err != nil {
		return "", err
	}

	statementPb := spb.Statement{
		Type:          spb.StatementTypeUri,
		Subject:       sub,
		PredicateType: VsaPredicateType,
		Predicate:     &predPb,
	}

	statement, err := protojson.Marshal(&statementPb)
	if err != nil {
		return "", err
	}
	return string(statement), nil
}

// Gets provenance for the commit from git notes.
func GetVsa(ctx context.Context, ghc *gh_control.GitHubConnection, verifier Verifier, commit, ref string) (*spb.Statement, *vpb.VerificationSummary, error) {
	notes, err := ghc.GetNotesForCommit(ctx, commit)
	if notes == "" {
		log.Printf("didn't find notes for commit %s", commit)
		return nil, nil, nil
	}

	if err != nil {
		log.Fatal(err)
	}
	return getVsaFromReader(NewBundleReader(bufio.NewReader(strings.NewReader(notes)), verifier), commit, ref)
}

func GetSourceRefsForCommit(vsaStatement *spb.Statement, commit string) ([]string, error) {
	subject := GetSubjectForCommit(vsaStatement, commit)
	if subject == nil {
		return []string{}, fmt.Errorf("statement \n%v\n does not match commit %s", StatementToString(vsaStatement), commit)
	}
	protoRefs := subject.GetAnnotations().Fields["source_branches"].GetListValue()
	stringRefs := []string{}
	for _, ref := range protoRefs.Values {
		stringRefs = append(stringRefs, ref.GetStringValue())
	}
	return stringRefs, nil
}

func getVsaPred(statement *spb.Statement) (*vpb.VerificationSummary, error) {
	predJson, err := protojson.Marshal(statement.Predicate)
	if err != nil {
		return nil, err
	}

	var predStruct vpb.VerificationSummary
	// Using regular json.Unmarshal because this is just a regular struct.
	err = protojson.Unmarshal(predJson, &predStruct)
	if err != nil {
		return nil, err
	}
	return &predStruct, nil
}

func MatchesTypeCommitAndRef(predicateType, commit, targetRef string) StatementMatcher {
	return func(statement *spb.Statement) bool {
		if statement.PredicateType != predicateType {
			log.Printf("statement predicate type (%s) doesn't match %s", statement.PredicateType, predicateType)
			return false
		}
		refs, err := GetSourceRefsForCommit(statement, commit)
		if err != nil {
			log.Printf("statement \n%v\n does not match commit %s: %v", StatementToString(statement), commit, err)
			return false
		}
		for _, ref := range refs {
			if targetRef == gh_control.AnyReference || ref == targetRef {
				log.Printf("statement \n%v\n matches commit '%s' on ref '%s'", StatementToString(statement), commit, targetRef)
				return true
			}
		}
		log.Printf("source_branches (%v) in VSA does not contain %s", refs, targetRef)
		return false
	}
}

func getVsaFromReader(reader *BundleReader, commit, ref string) (*spb.Statement, *vpb.VerificationSummary, error) {
	for {
		stmt, err := reader.ReadStatement(MatchesTypeCommitAndRef(VsaPredicateType, commit, ref))
		if err != nil {
			// Ignore errors, we want to check all the lines.
			log.Printf("error while processing line: %v", err)
			continue
		}

		if stmt == nil {
			// No statements left.
			break
		}

		vsaPred, err := getVsaPred(stmt)
		if err != nil {
			return nil, nil, err
		}

		return stmt, vsaPred, nil
	}

	log.Printf("didn't find commit %s for ref %s", commit, ref)
	return nil, nil, nil
}
