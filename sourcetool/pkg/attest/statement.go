package attest

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"

	spb "github.com/in-toto/attestation/go/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

type BundleReader struct {
	reader   *bufio.Reader
	verifier Verifier
}

func NewBundleReader(reader *bufio.Reader, verifier Verifier) *BundleReader {
	return &BundleReader{reader: reader, verifier: verifier}
}

func (br BundleReader) convertLineToStatement(line string) (*spb.Statement, error) {
	// Is this a sigstore bundle with a statement?
	vr, err := br.verifier.Verify(line)
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

type StatementMatcher func(*spb.Statement) bool

func MatchesTypeAndCommit(predicateType, commit string) StatementMatcher {
	return func(statement *spb.Statement) bool {
		if statement.PredicateType != predicateType {
			log.Printf("statement predicate type (%s) doesn't match %s", statement.PredicateType, predicateType)
			return false
		}
		if !DoesSubjectIncludeCommit(statement, commit) {
			log.Printf("statement \n%v\n does not match commit %s", StatementToString(statement), commit)
			return false
		}
		return true
	}
}

// Reads all the statements that:
// 1. Have the specified predicate type.
// 2. Have a subject that matches the specified git commit.
func (br *BundleReader) ReadStatement(matcher StatementMatcher) (*spb.Statement, error) {
	// Read until we get a statement or end of file.
	for {
		line, err := br.reader.ReadString('\n')
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
		statement, err := br.convertLineToStatement(line)
		if err != nil {
			return nil, fmt.Errorf("problem converting line to statement line: '%s', error: %w", line, err)
		}
		if statement == nil {
			// Not sure what this is, just continue
			continue
		}
		if matcher(statement) {
			return statement, nil
		}
		// If we loop again it's because that line didn't have a matching statement
	}
	return nil, nil
}

func DoesSubjectIncludeCommit(statement *spb.Statement, commit string) bool {
	return GetSubjectForCommit(statement, commit) != nil
}

// Returns the _first_ subject that includes the commit.
// TODO: add support for multiple subjects...
func GetSubjectForCommit(statement *spb.Statement, commit string) *spb.ResourceDescriptor {
	for _, subject := range statement.Subject {
		if subject.Digest["gitCommit"] == commit {
			return subject
		}
	}
	return nil
}

// Just make this easy for logging...
func StatementToString(stmt *spb.Statement) string {
	if stmt == nil {
		return "<nil>"
	}

	options := protojson.MarshalOptions{
		Multiline:     true,
		Indent:        " ",
		AllowPartial:  true,
		UseProtoNames: false,
	}

	jsonBytes, err := options.Marshal(stmt)
	if err != nil {
		return fmt.Sprintf("%v", err)
	}
	return string(jsonBytes)
}
