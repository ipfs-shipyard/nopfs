// Package tester provides an implementation-agnostic way to test a Blocker.
package tester

import (
	"bytes"
	_ "embed" // allow embedding test.deny
	"errors"
	"fmt"
	"io"

	"github.com/ipfs/boxo/path"
	"github.com/ipfs/go-cid"
)

//go:embed test.deny
var denylistFile []byte

// Blocker defines the minimal interface that a blocker should support
// to be tested.
type Blocker interface {
	ReadHeader(r io.Reader) (Header, error)
	ReadDenylist(r io.ReadSeekCloser) error
	IsPathBlocked(p path.Path) bool
	IsCidBlocked(c cid.Cid) bool
}

// Header represents a denylist header.
type Header interface {
	Name() string
	Hints() map[string]string
}

// Suite repesents the test suite and different test types can be
// enabled/disabled to match what the Blocker implementation supports.
type Suite struct {
	TestHeader           bool
	TestCID              bool
	TestCIDPath          bool
	TestIPNSPath         bool
	TestDoubleHashLegacy bool
	TestDoubleHash       bool

	b Blocker
}

type tReader struct {
	*bytes.Reader
}

func (r *tReader) Close() error {
	return nil
}

// Run performs blocker-validation tests based on test.deny using the
// given blocker. Only the enabled tests in the suite are performed.
func (s *Suite) Run(b Blocker) error {
	s.b = b

	if s.TestHeader {
		if err := s.testHeader(); err != nil {
			return err
		}
	}

	br := bytes.NewReader(denylistFile)
	rdr := &tReader{Reader: br}

	if err := b.ReadDenylist(rdr); err != nil {
		return fmt.Errorf("error reading/parsing denylist: %w", err)
	}

	if s.TestCID {
		if err := s.testCID(); err != nil {
			return err
		}
	}

	if s.TestCIDPath {
		if err := s.testCIDPath(); err != nil {
			return err
		}
	}

	if s.TestIPNSPath {
		if err := s.testIPNSPath(); err != nil {
			return err
		}
	}

	if s.TestDoubleHashLegacy {
		if err := s.testDoubleHashLegacy(); err != nil {
			return err
		}
	}

	if s.TestDoubleHash {
		if err := s.testDoubleHash(); err != nil {
			return err
		}
	}

	return nil
}

func (s *Suite) testHeader() error {
	listWithHeader := bytes.NewBufferString(`version: 1
name: test
hints:
  a: b
---
# empty
`)

	listWithoutHeader2 := bytes.NewBufferString(`---
/ipfs/bafybeihvvulpp4evxj7x7armbqcyg6uezzuig6jp3lktpbovlqfkuqeuoq
`)
	h, err := s.b.ReadHeader(listWithHeader)
	if err != nil {
		return errors.New("error parsing header")
	}
	if h.Name() != "test" {
		return errors.New("header not parsed correctly")
	}
	if hints := h.Hints(); len(hints) != 1 || hints["a"] != "b" {
		return errors.New("header hints not parsed correctly")
	}

	if _, err := s.b.ReadHeader(listWithoutHeader2); err != nil {
		return fmt.Errorf("error parsing list with just --- separator: %w", err)
	}
	return nil
}

func (s *Suite) testCID() error {
	// rule1
	c1 := cid.MustParse("bafybeihvvulpp4evxj7x7armbqcyg6uezzuig6jp3lktpbovlqfkuqeuoq")
	c2 := cid.MustParse("bafkreihvvulpp4evxj7x7armbqcyg6uezzuig6jp3lktpbovlqfkuqeuoq")
	c3 := cid.MustParse("QmesfgDQ3q6prBy2Kg2gKbW4MAGuWiRP2DVuGA5MZSERLo")

	if !s.b.IsCidBlocked(c1) {
		return errors.New("testCID: c1 should be blocked (rule1)")
	}

	if !s.b.IsCidBlocked(c2) {
		return errors.New("testCID: c2 should be blocked (rule1)")
	}

	if !s.b.IsCidBlocked(c3) {
		return errors.New("testCID: c3 should be blocked (rule1)")
	}

	return nil
}

func (s *Suite) testPaths(paths []string, testName, testRule string, allow bool) error {
	for _, p := range paths {
		ppath, err := path.NewPath(p)
		if err != nil {
			return err
		}
		blocked := s.b.IsPathBlocked(ppath)
		if !blocked && !allow {
			return fmt.Errorf("%s: path %s should be blocked (%s)", testName, p, testRule)
		}
		if blocked && allow {
			return fmt.Errorf("%s: path %s should be allowed (%s)", testName, p, testRule)
		}
	}
	return nil
}

func (s *Suite) testCIDPath() error {
	n := "testCIDPath"

	// rule1
	rule1 := []string{
		"/ipfs/bafybeihvvulpp4evxj7x7armbqcyg6uezzuig6jp3lktpbovlqfkuqeuoq",
		"/ipfs/bafkreihvvulpp4evxj7x7armbqcyg6uezzuig6jp3lktpbovlqfkuqeuoq",
		"/ipfs/QmesfgDQ3q6prBy2Kg2gKbW4MAGuWiRP2DVuGA5MZSERLo",
	}
	rule1allowed := []string{
		"/ipfs/bafybeihvvulpp4evxj7x7armbqcyg6uezzuig6jp3lktpbovlqfkuqeuoq/sub2",
		"/ipfs/bafkreihvvulpp4evxj7x7armbqcyg6uezzuig6jp3lktpbovlqfkuqeuoq/sub3",
		"/ipfs/QmesfgDQ3q6prBy2Kg2gKbW4MAGuWiRP2DVuGA5MZSERLo/sub4",
	}
	if err := s.testPaths(rule1, n, "rule1", false); err != nil {
		return err
	}
	if err := s.testPaths(rule1allowed, n, "rule1", true); err != nil {
		return err
	}

	// rule2
	rule2 := []string{
		"/ipfs/QmdWFA9FL52hx3j9EJZPQP1ZUH8Ygi5tLCX2cRDs6knSf8",
		"/ipfs/QmdWFA9FL52hx3j9EJZPQP1ZUH8Ygi5tLCX2cRDs6knSf8/a/b",
		"/ipfs/QmdWFA9FL52hx3j9EJZPQP1ZUH8Ygi5tLCX2cRDs6knSf8/z/",
		"/ipfs/QmdWFA9FL52hx3j9EJZPQP1ZUH8Ygi5tLCX2cRDs6knSf8/z",
	}
	if err := s.testPaths(rule2, n, "rule2", false); err != nil {
		return err
	}

	// rule3
	rule3 := []string{
		"/ipfs/Qmah2YDTfrox4watLCr3YgKyBwvjq8FJZEFdWY6WtJ3Xt2/test",
		"/ipfs/Qmah2YDTfrox4watLCr3YgKyBwvjq8FJZEFdWY6WtJ3Xt2/test2",
		"/ipfs/Qmah2YDTfrox4watLCr3YgKyBwvjq8FJZEFdWY6WtJ3Xt2/test/one",
	}
	rule3allowed := []string{
		"/ipfs/Qmah2YDTfrox4watLCr3YgKyBwvjq8FJZEFdWY6WtJ3Xt2/tes",
		"/ipfs/Qmah2YDTfrox4watLCr3YgKyBwvjq8FJZEFdWY6WtJ3Xt2",
		"/ipfs/Qmah2YDTfrox4watLCr3YgKyBwvjq8FJZEFdWY6WtJ3Xt2/one/test",
	}
	if err := s.testPaths(rule3, n, "rule3", false); err != nil {
		return err
	}
	if err := s.testPaths(rule3allowed, n, "rule3", true); err != nil {
		return err
	}

	// rule4
	rule4 := []string{
		"/ipfs/QmTuvSQbEDR3sarFAN9kAeXBpiBCyYYNxdxciazBba11eC/test",
		"/ipfs/QmTuvSQbEDR3sarFAN9kAeXBpiBCyYYNxdxciazBba11eC/test2",
		"/ipfs/QmTuvSQbEDR3sarFAN9kAeXBpiBCyYYNxdxciazBba11eC/test/one",
	}
	rule4allowed := []string{
		"/ipfs/QmTuvSQbEDR3sarFAN9kAeXBpiBCyYYNxdxciazBba11eC/tes",
		"/ipfs/QmTuvSQbEDR3sarFAN9kAeXBpiBCyYYNxdxciazBba11eC",
		"/ipfs/QmTuvSQbEDR3sarFAN9kAeXBpiBCyYYNxdxciazBba11eC/one/test",
	}
	if err := s.testPaths(rule4, n, "rule4", false); err != nil {
		return err
	}
	if err := s.testPaths(rule4allowed, n, "rule4", true); err != nil {
		return err
	}

	// rule5
	rule5 := []string{
		"/ipfs/QmUboz9UsQBDeS6Tug1U8jgoFkgYxyYood9NDyVURAY9pK/blocked",
		"/ipfs/QmUboz9UsQBDeS6Tug1U8jgoFkgYxyYood9NDyVURAY9pK/blocked2",
	}
	rule5allowed := []string{
		"/ipfs/QmUboz9UsQBDeS6Tug1U8jgoFkgYxyYood9NDyVURAY9pK/blockednot",
		"/ipfs/QmUboz9UsQBDeS6Tug1U8jgoFkgYxyYood9NDyVURAY9pK/blocked/not",
		"/ipfs/QmUboz9UsQBDeS6Tug1U8jgoFkgYxyYood9NDyVURAY9pK/blocked/exceptions",
		"/ipfs/QmUboz9UsQBDeS6Tug1U8jgoFkgYxyYood9NDyVURAY9pK/blocked/exceptions2",
		"/ipfs/QmUboz9UsQBDeS6Tug1U8jgoFkgYxyYood9NDyVURAY9pK/blocked/exceptions/one",
	}
	if err := s.testPaths(rule5, n, "rule5", false); err != nil {
		return err
	}

	if err := s.testPaths(rule5allowed, n, "rule5", true); err != nil {
		return err
	}

	return nil
}

func (s *Suite) testIPNSPath() error {
	n := "testIPNS"
	// rule6
	rule6 := []string{
		"/ipns/domain.example",
	}
	rule6allowed := []string{
		"/ipns/domainaefa.example",
		"/ipns/domain.example/path",
	}

	if err := s.testPaths(rule6, n, "rule6", false); err != nil {
		return err
	}

	if err := s.testPaths(rule6allowed, n, "rule6", true); err != nil {
		return err
	}

	// rule7
	rule7 := []string{
		"/ipns/domain2.example/path",
	}
	rule7allowed := []string{
		"/ipns/domain2.example",
		"/ipns/domain2.example/path2",
	}

	if err := s.testPaths(rule7, n, "rule7", false); err != nil {
		return err
	}

	if err := s.testPaths(rule7allowed, n, "rule7", true); err != nil {
		return err
	}

	// rule8
	rule8 := []string{
		"/ipns/k51qzi5uqu5dhmzyv3zac033i7rl9hkgczxyl81lwoukda2htteop7d3x0y1mf",
		"/ipns/bafzaajaiaejcaotjfs57kieazxny5japcmy5p2pgv2cic77tu6ogghttvurnrufx",
		"/ipns/12D3KooWDkNqEJNmreF3NYYFK1ws7Ra2fuW6cHBTu567SPV3LdYA",
	}
	rule8allowed := []string{
		"/ipns/12D3KooWDkNqEJNmreF3NYYFK1ws7Ra2fuW6cHBTu567SPV3LdYA/path",
	}

	if err := s.testPaths(rule8, n, "rule8", false); err != nil {
		return err
	}

	if err := s.testPaths(rule8allowed, n, "rule8", true); err != nil {
		return err
	}

	return nil
}

func (s *Suite) testDoubleHashLegacy() error {
	n := "TestDoubleHashLegacy"
	// rule10
	c := cid.MustParse("bafybeiefwqslmf6zyyrxodaxx4vwqircuxpza5ri45ws3y5a62ypxti42e")
	if !s.b.IsCidBlocked(c) {
		return fmt.Errorf("%s: cid %s should be blocked (rule10)", n, c)
	}

	//rule 11 (and 10)
	rule11 := []string{
		"/ipfs/bafybeiefwqslmf6zyyrxodaxx4vwqircuxpza5ri45ws3y5a62ypxti42e/",
		"/ipfs/bafybeiefwqslmf6zyyrxodaxx4vwqircuxpza5ri45ws3y5a62ypxti42e",
		"/ipfs/bafybeiefwqslmf6zyyrxodaxx4vwqircuxpza5ri45ws3y5a62ypxti42e/path",
	}
	rule11allowed := []string{
		"/ipfs/bafybeiefwqslmf6zyyrxodaxx4vwqircuxpza5ri45ws3y5a62ypxti42e/path2",
	}

	if err := s.testPaths(rule11, n, "rule11", false); err != nil {
		return err
	}

	if err := s.testPaths(rule11allowed, n, "rule11", true); err != nil {
		return err
	}

	return nil
}

func (s *Suite) testDoubleHash() error {
	n := "TestDoubleHash"
	//rule 13
	c1 := cid.MustParse("bafybeidjwik6im54nrpfg7osdvmx7zojl5oaxqel5cmsz46iuelwf5acja")
	c2 := cid.MustParse("QmVTF1yEejXd9iMgoRTFDxBv7HAz9kuZcQNBzHrceuK9HR")
	if !s.b.IsCidBlocked(c1) {
		return fmt.Errorf("%s: cid %s should be blocked (rule12)", n, c1)
	}

	if !s.b.IsCidBlocked(c2) {
		return fmt.Errorf("%s: cid %s should be blocked (rule12)", n, c2)
	}

	// rule13
	rule13 := []string{
		"/ipfs/bafyb4ieqht3b2rssdmc7sjv2cy2gfdilxkfh7623nvndziyqnawkmo266a/path",
		"/ipfs/bafyb4ieqht3b2rssdmc7sjv2cy2gfdilxkfh7623nvndziyqnawkmo266a/path",
		"/ipfs/f01701e20903cf61d46521b05f926ba1634628d0bba8a7ffb5b6d5a3ca310682ca63b5ef0/path",
	}
	rule13allowed := []string{
		"/ipfs/bafyb4ieqht3b2rssdmc7sjv2cy2gfdilxkfh7623nvndziyqnawkmo266a/",
		"/ipfs/bafyb4ieqht3b2rssdmc7sjv2cy2gfdilxkfh7623nvndziyqnawkmo266a/",
		"/ipfs/f01701e20903cf61d46521b05f926ba1634628d0bba8a7ffb5b6d5a3ca310682ca63b5ef0/",
		"/ipfs/bafyb4ieqht3b2rssdmc7sjv2cy2gfdilxkfh7623nvndziyqnawkmo266a/path2",
		"/ipfs/bafyb4ieqht3b2rssdmc7sjv2cy2gfdilxkfh7623nvndziyqnawkmo266a/path2",
		"/ipfs/f01701e20903cf61d46521b05f926ba1634628d0bba8a7ffb5b6d5a3ca310682ca63b5ef0/path2",
	}

	if err := s.testPaths(rule13, n, "rule13", false); err != nil {
		return err
	}

	if err := s.testPaths(rule13allowed, n, "rule13", true); err != nil {
		return err
	}

	return nil
}
