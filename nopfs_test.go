package nopfs

import (
	"io"
	"testing"

	"github.com/ipfs-shipyard/nopfs/tester"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"github.com/ipfs/go-path"
)

type testBlocker struct {
	Blocker
}

type testHeader struct {
	DenylistHeader
}

func (th testHeader) Name() string {
	return th.DenylistHeader.Name
}

func (th testHeader) Hints() map[string]string {
	return th.DenylistHeader.Hints
}

func (tb *testBlocker) ReadHeader(r io.Reader) (tester.Header, error) {
	dl := Denylist{}

	err := dl.Header.Decode(r)
	if err != nil {
		return testHeader{}, err
	}
	return testHeader{
		DenylistHeader: dl.Header,
	}, nil
}

func (tb *testBlocker) ReadDenylist(r io.ReadSeekCloser) error {
	tb.Blocker.Denylists = make(map[string]*Denylist)
	dl, err := NewDenylistReader(r)
	if err != nil {
		return err
	}
	dl.Filename = "test"
	tb.Blocker.Denylists["test"] = dl
	return nil
}

func (tb *testBlocker) IsPathBlocked(p path.Path) bool {
	res := tb.Blocker.IsPathBlocked(p)
	return res.Status == StatusBlocked
}

func (tb *testBlocker) IsCidBlocked(c cid.Cid) bool {
	res := tb.Blocker.IsCidBlocked(c)
	return res.Status == StatusBlocked
}

func TestSuite(t *testing.T) {
	logging.SetLogLevel("nopfs", "ERROR")

	tb := testBlocker{
		Blocker: Blocker{},
	}

	suite := &tester.Suite{
		TestHeader:           true,
		TestCID:              true,
		TestCIDPath:          true,
		TestIPNSPath:         true,
		TestMime:             false,
		TestDoubleHashLegacy: true,
		TestDoubleHash:       true,
	}

	err := suite.Run(&tb)
	if err != nil {
		t.Fatal(err)
	}
}
