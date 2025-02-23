package attest

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	spb "github.com/in-toto/attestation/go/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/slsa-framework/slsa-source-poc/sourcetool/pkg/gh_control"
)

type SourceProvenanceProperty struct {
	// The time from which this property has been continuously enforced.
	Since time.Time `json:"since"`
}

const SourceProvPredicateType = "https://github.com/slsa-framework/slsa-source-poc/source-provenance/v1"

// The predicate that encodes source provenance data.
// The git commit this corresponds to is encoded in the surrounding statement.
type SourceProvenancePred struct {
	// The commit preceeding 'Commit' in the current context.
	PrevCommit string `json:"prev_commit"`
	// TODO: What else should we store? The actor that triggered this change?

	// The properties observed for this commit.
	// For now we're just storing the level here, but later we'll add other stuff
	Properties map[string]SourceProvenanceProperty `json:"properties"`
}

type ProvenanceAttestor struct {
	verification_options VerificationOptions
	gh_connection        *gh_control.GitHubConnection
}

func NewProvenanceAttestor(gh_connection *gh_control.GitHubConnection, verification_options VerificationOptions) *ProvenanceAttestor {
	return &ProvenanceAttestor{verification_options: verification_options, gh_connection: gh_connection}
}

func GetProvPred(statement *spb.Statement) (*SourceProvenancePred, error) {
	predJson, err := protojson.Marshal(statement.Predicate)
	if err != nil {
		return nil, err
	}

	var predStruct SourceProvenancePred
	// Using regular json.Unmarshal because this is just a regular struct.
	err = json.Unmarshal(predJson, &predStruct)
	if err != nil {
		return nil, err
	}
	if len(predStruct.Properties) == 0 {
		return nil, fmt.Errorf("expected %v to have non-zero properties", predStruct)
	}
	return &predStruct, nil
}

func addPredToStatement(provPred *SourceProvenancePred, commit string) (*spb.Statement, error) {
	// Using regular json.Marhsal because this is just a regular struct and not from a proto.
	predJson, err := json.Marshal(provPred)
	if err != nil {
		return nil, err
	}

	sub := []*spb.ResourceDescriptor{{
		Digest: map[string]string{"gitCommit": commit},
	}}

	var predPb structpb.Struct
	err = protojson.Unmarshal(predJson, &predPb)
	if err != nil {
		return nil, err
	}

	statementPb := spb.Statement{
		Type:          spb.StatementTypeUri,
		Subject:       sub,
		PredicateType: SourceProvPredicateType,
		Predicate:     &predPb,
	}

	return &statementPb, nil
}

func doesSubjectIncludeCommit(statement *spb.Statement, commit string) bool {
	for _, subject := range statement.Subject {
		if subject.Digest["gitCommit"] == commit {
			return true
		}
	}
	return false
}

func (pa ProvenanceAttestor) createCurrentProvenance(ctx context.Context, commit, prevCommit string) (*spb.Statement, error) {
	controlStatus, err := pa.gh_connection.DetermineSourceLevelControlOnly(ctx, commit)
	if err != nil {
		return nil, err
	}

	levelProp := SourceProvenanceProperty{Since: time.Now()}
	var curProvPred SourceProvenancePred
	curProvPred.PrevCommit = prevCommit
	curProvPred.Properties = make(map[string]SourceProvenanceProperty)
	curProvPred.Properties[controlStatus.ControlLevel] = levelProp

	return addPredToStatement(&curProvPred, commit)
}

func (pa ProvenanceAttestor) convertLineToStatement(line string) (*spb.Statement, error) {
	// Is this a sigstore bundle with a statement?
	vr, err := Verify(line, pa.verification_options)
	if err == nil {
		// This is it.
		return vr.Statement, nil
	} else {
		// We ignore errors because there could be other stuff in the
		// bundle this line came from.
		log.Printf("Line %s failed verification: %v", line, err)
	}

	// TODO: add support for 'regular' DSSEs.

	return nil, errors.New("could not convert line to statement")
}

func (pa ProvenanceAttestor) getPrevProvenance(prevAttPath, prevCommit string) (*spb.Statement, error) {
	if prevAttPath == "" {
		// There is no prior provenance
		return nil, nil
	}

	f, err := os.Open(prevAttPath)
	if err != nil {
		return nil, err
	}
	reader := bufio.NewReader(f)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			// Handle end of file gracefully
			if err != io.EOF {
				return nil, err
			}
			if line == "" {
				// Nothing to see here.
				break
			}
		}
		// Is this source provenance?
		sp, err := pa.convertLineToStatement(line)
		if err == nil {
			if sp.PredicateType != SourceProvPredicateType {
				log.Printf("statement %v isn't source provenance", sp)
				continue
			}
			if doesSubjectIncludeCommit(sp, prevCommit) {
				// Should be good!
				return sp, nil
			} else {
				log.Printf("subject '%v' does not reference previous commit '%s', skipping", sp.Subject, prevCommit)
			}
		}
	}

	log.Printf("didn't find prev commit %s", prevCommit)
	return nil, nil
}

func (pa ProvenanceAttestor) CreateSourceProvenance(ctx context.Context, prevAttPath, commit, prevCommit string) (*spb.Statement, error) {
	// Source provenance is based on
	// 1. The current control situation (we assume 'commit' has _just_ occurred).
	// 2. How long the properties have been enforced according to the previous provenance.

	curProv, err := pa.createCurrentProvenance(ctx, commit, prevCommit)
	if err != nil {
		return nil, err
	}

	prevProv, err := pa.getPrevProvenance(prevAttPath, prevCommit)
	if err != nil {
		return nil, err
	}

	// No prior provenance found, so we just go with current.
	if prevProv == nil {
		log.Printf("No previous provenance found, have to bootstrap\n")
		return curProv, nil
	}

	prevProvPred, err := GetProvPred(prevProv)
	if err != nil {
		return nil, err
	}

	curProvPred, err := GetProvPred(curProv)
	if err != nil {
		return nil, err
	}

	// There was prior provenance, so update the Since field for each property
	// to the oldest encountered.
	for propName, curProp := range curProvPred.Properties {
		prevProp, ok := prevProvPred.Properties[propName]
		if !ok {
			continue
		}
		if prevProp.Since.Before(curProp.Since) {
			curProp.Since = prevProp.Since
			curProvPred.Properties[propName] = curProp
		}
	}

	return addPredToStatement(curProvPred, commit)
}
