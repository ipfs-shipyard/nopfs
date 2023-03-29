package nopfs

import (
	"bufio"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-path"
	"github.com/multiformats/go-multicodec"
	"github.com/multiformats/go-multihash"
	"go.uber.org/multierr"
	"gopkg.in/yaml.v3"
)

// ErrHeaderNotFound is returned when no header can be Decoded.
var ErrHeaderNotFound = errors.New("header not found")

// DenylistHeader represents the header of a Denylist file.
type DenylistHeader struct {
	Version     int
	Name        string
	Description string
	Author      string
	Hints       map[string]string

	headerBytes []byte
	headerLines uint64
}

// Decode decodes a DenlistHeader from a reader. Per the specification, the
// maximum size of a header is 1KiB. If no header is found, ErrHeaderNotFound
// is returned.
func (h *DenylistHeader) Decode(r io.Reader) error {
	limRdr := &io.LimitedReader{
		R: r,
		N: 1 << 10, // 1KiB per spec
	}
	buf := bufio.NewReader(limRdr)

	h.headerBytes = nil
	h.headerLines = 0

	for {
		line, err := buf.ReadBytes('\n')
		if err == io.EOF {
			h.headerBytes = nil
			h.headerLines = 0
			return ErrHeaderNotFound
		}
		if err != nil {
			return err
		}
		h.headerLines++
		if string(line) == "---\n" {
			break
		}
		h.headerBytes = append(h.headerBytes, line...)
	}

	err := yaml.Unmarshal(h.headerBytes, h)
	if err != nil {
		logger.Error(err)
		return err
	}
	return nil
}

// String provides a short string summary of the Header.
func (h DenylistHeader) String() string {
	return fmt.Sprintf("%s (%s) by %s", h.Name, h.Description, h.Author)
}

// A Denylist represents a denylist file and its rules. It can parse and
// follow a denylist file, and can answer questions about blocked or allowed
// items in this denylist.
type Denylist struct {
	Header   DenylistHeader
	Filename string

	Entries Entries

	IPFSBlocksDB       *BlocksDB
	IPNSBlocksDB       *BlocksDB
	DoubleHashBlocksDB map[uint64]*BlocksDB // codec -> blocks using that codec
	PathBlocksDB       *BlocksDB
	PathPrefixBlocks   Entries
	// MimeBlocksDB

	f       io.ReadSeekCloser
	watcher *fsnotify.Watcher
}

// NewDenylist opens a denylist file and processes it (parses all its entries).
//
// If follow is false, the file handle is closed.
//
// If follow is true, the denylist file will be followed upon return. Any
// appended rules will be processed live-updated in the
// denylist. Denylist.Close() should be used when the Denylist or the
// following is no longer needed.
func NewDenylist(filepath string, follow bool) (*Denylist, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}

	dl := Denylist{
		Filename:           filepath,
		f:                  f,
		IPFSBlocksDB:       &BlocksDB{},
		IPNSBlocksDB:       &BlocksDB{},
		PathBlocksDB:       &BlocksDB{},
		DoubleHashBlocksDB: make(map[uint64]*BlocksDB),
	}

	err = dl.parseAndFollow(follow)
	return &dl, err
}

// NewDenylistReader processes a denylist from the given reader (parses all
// its entries).
func NewDenylistReader(r io.ReadSeekCloser) (*Denylist, error) {
	dl := Denylist{
		Filename:           "",
		f:                  r,
		IPFSBlocksDB:       &BlocksDB{},
		IPNSBlocksDB:       &BlocksDB{},
		PathBlocksDB:       &BlocksDB{},
		DoubleHashBlocksDB: make(map[uint64]*BlocksDB),
	}

	err := dl.parseAndFollow(false)
	return &dl, err
}

// read the header and make sure the reader is in the right position for
// further processing. In case of no header a default one is used.
func (dl *Denylist) readHeader() error {
	err := dl.Header.Decode(dl.f)
	if err == ErrHeaderNotFound {
		dl.Header.Version = 1
		dl.Header.Name = filepath.Base(dl.Filename)
		dl.Header.Description = "No header found"
		dl.Header.Author = "unknown"
		// reset the reader
		_, err = dl.f.Seek(0, 0)
		if err != nil {
			logger.Error(err)
			return err
		}
		logger.Warnf("Opening %s: empty header", dl.Filename)
		logger.Infof("Processing %s: %s", dl.Filename, dl.Header)
		return nil
	} else if err != nil {
		return err
	}

	logger.Infof("Processing %s: %s", dl.Filename, dl.Header)

	// We have to deal with the buffered reader reading beyond the header.
	_, err = dl.f.Seek(int64(len(dl.Header.headerBytes)+4), 0)
	if err != nil {
		logger.Error(err)
		return err
	}
	// The reader should be set at the line after ---\n now.
	// Reader to parse the rest of lines.

	return nil
}

// All closing on error is performed here.
func (dl *Denylist) parseAndFollow(follow bool) error {
	if err := dl.readHeader(); err != nil {
		dl.Close()
		return err
	}

	// we will update N as we go after every line.
	// Fixme: this is going to play weird as the buffered reader will
	// read-ahead and consume N.
	limRdr := &io.LimitedReader{
		R: dl.f,
		N: 2 << 20, // 2MiB per spec
	}
	r := bufio.NewReader(limRdr)

	lineNumber := dl.Header.headerLines
	for {
		line, err := r.ReadString('\n')
		// limit reader exhausted
		if err == io.EOF && len(line) >= 2<<20 {
			err = fmt.Errorf("line too long. %s:%d", dl.Filename, lineNumber+1)
			logger.Error(err)
			dl.Close()
			return err
		}
		// keep waiting for data
		if err == io.EOF {
			break
		}
		if err != nil {
			logger.Error(err)
			return err
		}

		lineNumber++
		if err := dl.parseLine(line, lineNumber); err != nil {
			logger.Error(err)
		}
		limRdr.N = 2 << 20 // reset

	}
	// we finished reading the file as it EOF'ed.
	if !follow {
		return nil
	}
	// We now wait for new lines.

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		dl.Close()
		return err
	}
	dl.watcher = watcher
	err = watcher.Add(dl.Filename)
	if err != nil {
		dl.Close()
		return err
	}

	waitForWrite := func() error {
		for {
			select {
			case event := <-dl.watcher.Events:
				if event.Op&fsnotify.Write == fsnotify.Write {
					return nil
				}
			case err := <-dl.watcher.Errors:
				// TODO: log
				return err
			}
		}
	}

	// Is this the right way of tailing a file? Pretty sure there are a
	// bunch of gotchas. It seems to work when saving on top of a file
	// though. Also, important that the limitedReader is there to avoid
	// parsing a huge lines.  Also, this could be done by just having
	// watchers on the folder, but requires a small refactoring.
	go func() {
		line := ""
		limRdr.N = 2 << 20 // reset

		for {
			partialLine, err := r.ReadString('\n')
			line += partialLine

			// limit reader exhausted
			if err == io.EOF && limRdr.N == 0 {
				err = fmt.Errorf("line too long. %s:%d", dl.Filename, lineNumber+1)
				logger.Error(err)
				dl.Close()
				return
			}
			// keep waiting for data
			if err == io.EOF {
				err := waitForWrite()
				if err != nil {
					logger.Error(err)
					dl.Close()
					return
				}
				continue
			}
			if err != nil {
				logger.Error(err)
				dl.Close()
				return
			}

			lineNumber++
			// we have read up to \n
			if err := dl.parseLine(line, lineNumber); err != nil {
				logger.Error(err)
				// log error and continue with next line

			}
			// reset for next line
			line = ""
			limRdr.N = 2 << 20 // reset
		}
	}()
	return nil
}

// parseLine processes every full-line read and puts it into the BlocksDB etc.
// so that things can be queried later. It turns lines into Entry objects.
//
// Note (hector): I'm using B58Encoded-multihash strings as keys for IPFS,
// DoubleHash BlocksDB. Why? We could be using the multihash bytes directly.
// Some reasons (this should be changed if there are better reasons to do so):
//
//   - B58Encoded-strings produce readable keys. These will be readable keys
//     in a database, readable keys in the debug logs etc. Having readable
//     multihashes instead of raw bytes is nice.
//
//   - If we assume IPFS mostly deals in CIDv0s (raw multihashes), we can
//     avoid parsing the Qmxxx multihash in /ipfs/Qmxxx/path and just use the
//     string directly for lookups.
//
//   - If we used raw bytes, we would have to decode every cidV0, but we would
//     not have to b58-encode multihashes for lookups. Chances that this is
//     better but well (decide before going with permanent storage!).
func (dl *Denylist) parseLine(line string, number uint64) error {
	line = strings.TrimSuffix(line, "\n")
	if len(line) == 0 || line[0] == '#' {
		return nil
	}

	e := Entry{
		Line:     number,
		RawValue: line,
	}

	// the rule is always field-0. Anything else is hints.
	splitFields := strings.Fields(line)
	rule := splitFields[0]
	if len(splitFields) > 1 { // we have hints
		hintSlice := splitFields[1:]
		for _, kv := range hintSlice {
			key, value, ok := strings.Cut(kv, "=")
			if !ok {
				continue
			}
			e.Hints[key] = value
		}
	}

	// We treat +<rule> and -<rule> the same. Both serve to declare and
	// allow-rule.
	if unprefixed, found := cutPrefix(rule, "-"); found {
		e.AllowRule = true
		rule = unprefixed
	} else if unprefixed, found := cutPrefix(rule, "+"); found {
		e.AllowRule = true
		rule = unprefixed
	}

	switch {
	case strings.HasPrefix(rule, "//"):
		// Double-hash rule.
		// It can be a Multihash (CIDv0) or a sha256-hex-encoded string.

		var mhType uint64
		// attempt to parse CID
		rule = strings.TrimPrefix(rule, "//")
		c, err := cid.Decode(rule)
		if err == nil {
			prefix := c.Prefix()
			if prefix.Version != 0 {
				return fmt.Errorf("double-hash is not a raw-multihash (cidv0) (%s:%d)", dl.Filename, number)
			}
			e.Multihash = c.Hash()
			// we use the multihash codec to group double-hashes
			// with the same hashing function.
			mhType = c.Prefix().MhType
		} else { // Assume a hex-encoded sha256 string
			bs, err := hex.DecodeString(rule)
			if err != nil {
				return fmt.Errorf("double-hash is not a multihash nor a hex-encoded string (%s:%d): %w", dl.Filename, number, err)
			}
			// We have a hex-encoded string and assume it is a
			// SHA2_256. TODO: could support hints here to use
			// different functions.
			mhBytes, err := multihash.Encode(bs, multihash.SHA2_256)
			if err != nil {
				return err
			}
			e.Multihash = multihash.Multihash(mhBytes)
			mhType = multihash.SHA2_256
		}
		bpath, _ := NewBlockedPath("")
		e.Path = bpath

		// Store it in the appropiate BlocksDB (per mhtype).
		key := e.Multihash.B58String()
		if blocks := dl.DoubleHashBlocksDB[mhType]; blocks == nil {
			dl.DoubleHashBlocksDB[mhType] = &BlocksDB{}
		}
		dl.DoubleHashBlocksDB[mhType].Store(key, e)
		logger.Debugf("%s:%d: Double-hash rule. Func: %s. Key: %s. Entry: %s", filepath.Base(dl.Filename), number, multicodec.Code(mhType).String(), key, e)

	case strings.HasPrefix(rule, "/ipfs/"), strings.HasPrefix(rule, "/ipld/"):
		// ipfs/ipld rule. We parse the CID and use the
		// b58-encoded-multihash as key to the Entry.

		rule = strings.TrimPrefix(rule, "/ipfs/")
		rule = strings.TrimPrefix(rule, "/ipld/")
		cidStr, subPath, _ := strings.Cut(rule, "/")

		c, err := cid.Decode(cidStr)
		if err != nil {
			return fmt.Errorf("error extracting cid %s (%s:%d): %w", cidStr, dl.Filename, number, err)
		}
		e.Multihash = c.Hash()

		blockedPath, err := NewBlockedPath(subPath)
		if err != nil {
			return err
		}
		e.Path = blockedPath

		// Add to IPFS by component multihash
		key := e.Multihash.B58String()
		dl.IPFSBlocksDB.Store(key, e)
		logger.Debugf("%s:%d: IPFS rule. Key: %s. Entry: %s", filepath.Base(dl.Filename), number, key, e)
	case strings.HasPrefix(rule, "/ipns/"):
		// ipns rule. If it carries anything parseable as a CID, we
		// store indexed by the b58-multihash. Otherwise assume it is
		// a domain name and store that directly.
		rule, _ = cutPrefix(rule, "/ipns/")
		key, subPath, _ := strings.Cut(rule, "/")
		c, err := cid.Decode(key)
		if err == nil { // CID key handling.
			key = c.Hash().B58String()
		}
		blockedPath, err := NewBlockedPath(subPath)
		if err != nil {
			return err
		}
		e.Path = blockedPath

		// Add to IPFS by component multihash
		dl.IPNSBlocksDB.Store(key, e)
		logger.Debugf("%s:%d: IPNS rule. Key: %s. Entry: %s", filepath.Base(dl.Filename), number, key, e)
	default:
		// Blocked by path only. We store non-prefix paths directly.
		// We store prefixed paths separately as every path request
		// will have to loop them.
		blockedPath, err := NewBlockedPath(rule)
		if err != nil {
			return err
		}
		e.Path = blockedPath

		key := rule
		if blockedPath.Prefix {
			dl.PathPrefixBlocks = append(dl.PathPrefixBlocks, e)
		} else {
			dl.PathBlocksDB.Store(key, e)
		}
		logger.Debugf("%s:%d: Path rule. Key: %s. Entry: %s", filepath.Base(dl.Filename), number, key, e)
	}

	dl.Entries = append(dl.Entries, e)
	return nil

}

// Close closes the Denylist file handle and stops watching write events on it.
func (dl *Denylist) Close() error {
	var err error
	if dl.watcher != nil {
		err = multierr.Append(err, dl.watcher.Close())
	}
	if dl.f != nil {
		err = multierr.Append(err, dl.f.Close())
	}

	return err
}

// IsSubpathBlocked returns Blocking Status for the given subpath.
func (dl *Denylist) IsSubpathBlocked(subpath string) StatusResponse {
	subpath = strings.TrimPrefix(subpath, "/")

	logger.Debugf("IsSubpathBlocked load path: %s", subpath)
	pathBlockEntries, _ := dl.PathBlocksDB.Load(subpath)
	status, entry := pathBlockEntries.CheckPathStatus(subpath)
	if status != StatusNotFound { // hit
		return StatusResponse{
			Status:   status,
			Filename: dl.Filename,
			Entry:    entry,
		}
	}
	// Check every prefix path.  Note: this is very innefficient, we
	// should have some HAMT that we can traverse with every character if
	// we were to support a large number of subpath-prefix blocks.
	status, entry = dl.PathPrefixBlocks.CheckPathStatus(subpath)
	return StatusResponse{
		Status:   status,
		Filename: dl.Filename,
		Entry:    entry,
	}
}

// IsIPNSPathBlocked returns Blocking Status for a given IPNS name and its
// subpath. The name is NOT an "/ipns/name" path, but just the name.
func (dl *Denylist) IsIPNSPathBlocked(name, subpath string) StatusResponse {
	subpath = strings.TrimPrefix(subpath, "/")

	var p path.Path
	if len(subpath) > 0 {
		p = path.FromString("/ipns/" + name + "/" + subpath)
	} else {
		p = path.FromString("/ipns/" + name)
	}
	key := name
	// Check if it is a CID and use the multihash as key then
	c, err := cid.Decode(key)
	if err == nil {
		key = c.Hash().B58String()
	}
	logger.Debugf("IsIPNSPathBlocked load: %s %s", key, subpath)
	entries, _ := dl.IPNSBlocksDB.Load(key)
	status, entry := entries.CheckPathStatus(subpath)
	if status != StatusNotFound { // hit!
		return StatusResponse{
			Path:     p,
			Status:   status,
			Filename: dl.Filename,
			Entry:    entry,
		}
	}

	// Double-hash blocking
	for codec, blocks := range dl.DoubleHashBlocksDB {
		double, err := multihash.Sum([]byte(p), codec, -1)
		if err != nil {
			logger.Error(err)
			continue
		}
		b58 := double.B58String()
		logger.Debugf("IsPathBlocked load IPNS doublehash: %s", b58)
		entries, _ := blocks.Load(b58)
		status, entry := entries.CheckPathStatus("")
		if status != StatusNotFound { // Hit!
			return StatusResponse{
				Path:     p,
				Status:   status,
				Filename: dl.Filename,
				Entry:    entry,
			}
		}
	}

	// Not found
	return StatusResponse{
		Path:     p,
		Status:   StatusNotFound,
		Filename: dl.Filename,
	}
}

// IsIPFSPathBlocked returns Blocking Status for a given IPFS CID and its
// subpath. The cidStr is NOT an "/ipns/cid" path, but just the cid.
func (dl *Denylist) IsIPFSPathBlocked(cidStr, subpath string) StatusResponse {
	return dl.isIPFSIPLDPathBlocked(cidStr, subpath, "ipfs")
}

// IsIPLDPathBlocked returns Blocking Status for a given IPLD CID and its
// subpath. The cidStr is NOT an "/ipld/cid" path, but just the cid.
func (dl *Denylist) IsIPLDPathBlocked(cidStr, subpath string) StatusResponse {
	return dl.isIPFSIPLDPathBlocked(cidStr, subpath, "ipld")
}

func (dl *Denylist) isIPFSIPLDPathBlocked(cidStr, subpath, protocol string) StatusResponse {
	subpath = strings.TrimPrefix(subpath, "/")

	var p path.Path
	if len(subpath) > 0 {
		p = path.FromString("/" + protocol + "/" + cidStr + "/" + subpath)
	} else {
		p = path.FromString("/" + protocol + "/" + cidStr)
	}
	key := cidStr

	// This could be a shortcut to let the work to the
	// blockservice.  Assuming IsCidBlocked() is going to be
	// called later down the stack (by IPFS).
	//
	// TODO: enable this with options.
	// if p.IsJustAKey() {
	// 	return false
	// }

	var c cid.Cid
	var err error
	if len(key) != 46 || key[:2] != "Qm" {
		// Key is not a CIDv0, we need to convert other CIDs.
		// convert to Multihash (cidV0)
		c, err = cid.Decode(key)
		if err != nil {
			logger.Warnf("could not decode %s as CID: %s", key, err)
			return StatusResponse{
				Path:     p,
				Status:   StatusErrored,
				Filename: dl.Filename,
				Error:    err,
			}
		}
		key = c.Hash().B58String()
	}

	logger.Debugf("isIPFSIPLDPathBlocked load: %s %s", key, subpath)
	entries, _ := dl.IPFSBlocksDB.Load(key)
	status, entry := entries.CheckPathStatus(subpath)
	if status != StatusNotFound { // hit!
		return StatusResponse{
			Path:     p,
			Status:   status,
			Filename: dl.Filename,
			Entry:    entry,
		}
	}

	// Check for double-hashed entries. We need to lookup both the
	// multihash+path and the base32-cidv1 + path
	if !c.Defined() { // if we didn't decode before...
		c, err = cid.Decode(cidStr)
		if err != nil {
			logger.Warnf("could not decode %s as CID: %s", key, err)
			return StatusResponse{
				Path:     p,
				Status:   StatusErrored,
				Filename: dl.Filename,
				Error:    err,
			}
		}
	}

	prefix := c.Prefix()
	for codec, blocks := range dl.DoubleHashBlocksDB {
		// <cidv1base32>/<path>
		// TODO: we should be able to disable this part with an Option
		// or a hint for denylists not using it.
		v1b32 := cid.NewCidV1(prefix.Codec, c.Hash()).String() // base32 string
		v1b32path := v1b32
		// badbits appends / on empty subpath. and hashes that
		// https://github.com/protocol/badbits.dwebops.pub/blob/main/badbits-lambda/helpers.py#L17
		v1b32path += "/" + subpath
		doubleLegacy, err := multihash.Sum([]byte(v1b32path), codec, -1)
		if err != nil {
			logger.Error(err)
			continue
		}

		// encode as b58 which is the key we use for the BlocksDB.
		b58 := doubleLegacy.B58String()
		logger.Debugf("IsIPFFSIPLDPathBlocked load IPFS doublehash (legacy): %d %s", codec, b58)
		entries, _ := blocks.Load(b58)
		status, entry := entries.CheckPathStatus("")
		if status != StatusNotFound { // Hit!
			return StatusResponse{
				Path:     p,
				Status:   status,
				Filename: dl.Filename,
				Entry:    entry,
			}
		}

		// <cidv0>/<path>
		v0path := c.Hash().B58String()
		if subpath != "" {
			v0path += "/" + subpath
		}
		double, err := multihash.Sum([]byte(v0path), codec, -1)
		if err != nil {
			logger.Error(err)
			continue
		}
		b58 = double.B58String()
		logger.Debugf("IsPathBlocked load IPFS doublehash: %d %s", codec, b58)
		entries, _ = blocks.Load(b58)
		status, entry = entries.CheckPathStatus("")
		if status != StatusNotFound { // Hit!
			return StatusResponse{
				Path:     p,
				Status:   status,
				Filename: dl.Filename,
				Entry:    entry,
			}
		}
	}
	return StatusResponse{
		Path:     p,
		Status:   StatusNotFound,
		Filename: dl.Filename,
	}
}

// IsPathBlocked provides Blocking Status for a given path.  This is done by
// interpreting the full path and checking for blocked Path, IPFS, IPNS or
// double-hashed items matching it.
//
// Matching is more efficient if:
//
//   - Paths in the form of /ipfs/Qm/... (sha2-256-multihash) are used rather than CIDv1.
//
//   - A single double-hashing pattern is used.
//
//   - A small number of path-only match rules using prefixes are used.
func (dl *Denylist) IsPathBlocked(p path.Path) StatusResponse {
	segments := p.Segments()
	if len(segments) < 2 {
		return StatusResponse{
			Path:     p,
			Status:   StatusErrored,
			Filename: dl.Filename,
			Error:    errors.New("path is too short"),
		}
	}
	proto := segments[0]
	key := segments[1]
	subpath := path.Join(segments[2:])

	// First, check that we are not blocking this subpath in general
	if len(subpath) > 0 {
		if resp := dl.IsSubpathBlocked(subpath); resp.Status != StatusNotFound {
			resp.Path = p
			return resp
		}

	}

	// Second, check that we are not blocking ipfs or ipns paths
	// like this one.

	// ["ipfs", "<cid>", ...]

	switch proto {
	case "ipns":
		return dl.IsIPNSPathBlocked(key, subpath)
	case "ipfs":
		return dl.IsIPFSPathBlocked(key, subpath)
	case "ipld":
		return dl.IsIPLDPathBlocked(key, subpath)
	default:
		return StatusResponse{
			Path:     p,
			Status:   StatusNotFound,
			Filename: dl.Filename,
		}
	}
}

// IsCidBlocked provides Blocking Status for a given CID.  This is done by
// extracting the multihash and checking if it is blocked by any rule.
func (dl *Denylist) IsCidBlocked(c cid.Cid) StatusResponse {
	mh := c.Hash()
	b58 := mh.B58String()
	logger.Debugf("IsCidBlocked load: %s", b58)
	entries, _ := dl.IPFSBlocksDB.Load(b58)
	// Look for an entry with an empty path
	// which means the Mhash itself is blocked.
	status, entry := entries.CheckPathStatus("")
	if status != StatusNotFound { // Hit!
		return StatusResponse{
			Cid:      c,
			Status:   status,
			Filename: dl.Filename,
			Entry:    entry,
		}
	}

	// Now check if a double-hash covers this CID

	// convert cid to v1 base32
	// the double-hash using multhash sha2-256
	// then check that
	sha256blocks := dl.DoubleHashBlocksDB[multihash.SHA2_256]
	if sha256blocks != nil {
		prefix := c.Prefix()
		b32 := cid.NewCidV1(prefix.Codec, c.Hash()).String() + "/" // yes, needed
		logger.Debug("IsCidBlocked cidv1b32 ", b32)
		double, err := multihash.Sum([]byte(b32), multihash.SHA2_256, -1)
		if err != nil {
			logger.Error(err)
			return StatusResponse{
				Cid:      c,
				Status:   StatusErrored,
				Filename: dl.Filename,
				Error:    err,
			}
		}
		b58 := double.B58String()
		logger.Debugf("IsCidBlocked load sha256 doublehash: %s", b58)
		entries, _ := sha256blocks.Load(b58)
		status, entry := entries.CheckPathStatus("")
		if status != StatusNotFound { // Hit!
			return StatusResponse{
				Cid:      c,
				Status:   status,
				Filename: dl.Filename,
				Entry:    entry,
			}
		}
	}

	// Otherwise, double-hash the multihash string with the given codecs for
	// which we have blocks.
	for codec, blocks := range dl.DoubleHashBlocksDB {
		double, err := multihash.Sum([]byte(b58), codec, -1)
		if err != nil {
			logger.Error(err)
			continue
		}
		b58 := double.B58String()
		logger.Debugf("IsCidBlocked load %d doublehash: %s", codec, b58)
		entries, _ := blocks.Load(b58)
		status, entry := entries.CheckPathStatus("")
		if status != StatusNotFound { // Hit!
			return StatusResponse{
				Cid:      c,
				Status:   status,
				Filename: dl.Filename,
				Entry:    entry,
			}
		}
	}

	return StatusResponse{
		Cid:      c,
		Status:   StatusNotFound,
		Filename: dl.Filename,
	}
}
