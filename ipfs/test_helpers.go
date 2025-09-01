package ipfs

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/ipfs-shipyard/nopfs"
	"github.com/ipfs/boxo/blockstore"
	"github.com/ipfs/boxo/exchange"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multihash"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Test CIDs and blocks for testing
var (
	// Test CIDs
	testCID1    = mustParseCID("bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi")
	testCID2    = mustParseCID("bafkreifjjcie6lypi6ny7amxnfftagclbuxndqonfipmb64f2km2devei4")
	blockedCID1 = mustParseCID("bafkreihubvg2gobtafahmkqrrcqsernhdprw2pumiana3ombkhckfl5ztq")
	blockedCID2 = mustParseCID("bafybeihsf4562gmmyoya2eh4fimpgzpjkh5jistrscvcfpxuznbknufwtq")

	// Test blocks
	testBlock1    = mustCreateBlock(testCID1, []byte("test content 1"))
	testBlock2    = mustCreateBlock(testCID2, []byte("test content 2"))
	blockedBlock1 = mustCreateBlock(blockedCID1, []byte("blocked content 1"))
	blockedBlock2 = mustCreateBlock(blockedCID2, []byte("blocked content 2"))
)

func mustParseCID(cidStr string) cid.Cid {
	c, err := cid.Parse(cidStr)
	if err != nil {
		panic(err)
	}
	return c
}

func mustCreateBlock(c cid.Cid, data []byte) blocks.Block {
	// Create a basic block with the given CID and data
	// Note: In real scenarios, the CID should match the hash of the data
	blk, err := blocks.NewBlockWithCid(data, c)
	if err != nil {
		panic(err)
	}
	return blk
}

// createTestBlock creates a proper block where CID matches the data hash
func createTestBlock(data []byte) (blocks.Block, error) {
	mh, err := multihash.Sum(data, multihash.SHA2_256, -1)
	if err != nil {
		return nil, err
	}
	c := cid.NewCidV1(cid.Raw, mh)
	return blocks.NewBlockWithCid(data, c)
}

// createTestBlocker creates a test Blocker with predefined blocked CIDs
func createTestBlocker(blockedCIDs ...cid.Cid) *nopfs.Blocker {
	blocker := &nopfs.Blocker{
		Denylists: make(map[string]*nopfs.Denylist),
	}

	// Create a simple denylist with the blocked CIDs
	dl := &nopfs.Denylist{
		Entries:            make(nopfs.Entries, 0, len(blockedCIDs)),
		IPFSBlocksDB:       &nopfs.BlocksDB{},
		DoubleHashBlocksDB: make(map[uint64]*nopfs.BlocksDB),
	}

	for i, c := range blockedCIDs {
		mh := c.Hash()
		entry := nopfs.Entry{
			Line:      uint64(i + 1),
			Multihash: mh,
		}
		dl.Entries = append(dl.Entries, entry)
		// Also store in the BlocksDB for efficient lookup
		dl.IPFSBlocksDB.Store(mh.B58String(), entry)

		// Initialize double hash blocks DB for this multihash code if needed
		mhCode := uint64(mh[0])
		if _, ok := dl.DoubleHashBlocksDB[mhCode]; !ok {
			dl.DoubleHashBlocksDB[mhCode] = &nopfs.BlocksDB{}
		}
	}

	blocker.Denylists["test"] = dl
	return blocker
}

// MockBlockstore is a mock implementation of blockstore.Blockstore
type MockBlockstore struct {
	mock.Mock
	blocks map[string]blocks.Block
	mu     sync.RWMutex
}

func NewMockBlockstore() *MockBlockstore {
	return &MockBlockstore{
		blocks: make(map[string]blocks.Block),
	}
}

func (m *MockBlockstore) Get(ctx context.Context, c cid.Cid) (blocks.Block, error) {
	args := m.Called(ctx, c)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(blocks.Block), args.Error(1)
}

func (m *MockBlockstore) GetSize(ctx context.Context, c cid.Cid) (int, error) {
	args := m.Called(ctx, c)
	return args.Int(0), args.Error(1)
}

func (m *MockBlockstore) Has(ctx context.Context, c cid.Cid) (bool, error) {
	args := m.Called(ctx, c)
	return args.Bool(0), args.Error(1)
}

func (m *MockBlockstore) Put(ctx context.Context, b blocks.Block) error {
	args := m.Called(ctx, b)
	return args.Error(0)
}

func (m *MockBlockstore) PutMany(ctx context.Context, bs []blocks.Block) error {
	args := m.Called(ctx, bs)
	return args.Error(0)
}

func (m *MockBlockstore) DeleteBlock(ctx context.Context, c cid.Cid) error {
	args := m.Called(ctx, c)
	return args.Error(0)
}

func (m *MockBlockstore) AllKeysChan(ctx context.Context) (<-chan cid.Cid, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(<-chan cid.Cid), args.Error(1)
}

func (m *MockBlockstore) HashOnRead(enabled bool) {
	m.Called(enabled)
}

// MockExchange is a mock implementation of exchange.Interface
type MockExchange struct {
	mock.Mock
}

func NewMockExchange() *MockExchange {
	return &MockExchange{}
}

func (m *MockExchange) GetBlock(ctx context.Context, c cid.Cid) (blocks.Block, error) {
	args := m.Called(ctx, c)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(blocks.Block), args.Error(1)
}

func (m *MockExchange) GetBlocks(ctx context.Context, ks []cid.Cid) (<-chan blocks.Block, error) {
	args := m.Called(ctx, ks)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(<-chan blocks.Block), args.Error(1)
}

func (m *MockExchange) NotifyNewBlocks(ctx context.Context, bs ...blocks.Block) error {
	args := m.Called(ctx, bs)
	return args.Error(0)
}

func (m *MockExchange) Close() error {
	args := m.Called()
	return args.Error(0)
}

// MockSessionExchange is a mock that also implements SessionExchange
type MockSessionExchange struct {
	MockExchange
}

func NewMockSessionExchange() *MockSessionExchange {
	return &MockSessionExchange{}
}

func (m *MockSessionExchange) NewSession(ctx context.Context) exchange.Fetcher {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(exchange.Fetcher)
}

// MockFetcher is a mock implementation of exchange.Fetcher
type MockFetcher struct {
	mock.Mock
}

func NewMockFetcher() *MockFetcher {
	return &MockFetcher{}
}

func (m *MockFetcher) GetBlock(ctx context.Context, c cid.Cid) (blocks.Block, error) {
	args := m.Called(ctx, c)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(blocks.Block), args.Error(1)
}

func (m *MockFetcher) GetBlocks(ctx context.Context, ks []cid.Cid) (<-chan blocks.Block, error) {
	args := m.Called(ctx, ks)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(<-chan blocks.Block), args.Error(1)
}

// MockBlockService is a mock implementation of blockservice.BlockService
type MockBlockService struct {
	mock.Mock
	mockBlockstore *MockBlockstore
	mockExchange   *MockExchange
}

func NewMockBlockService() *MockBlockService {
	return &MockBlockService{
		mockBlockstore: NewMockBlockstore(),
		mockExchange:   NewMockExchange(),
	}
}

func (m *MockBlockService) GetBlock(ctx context.Context, c cid.Cid) (blocks.Block, error) {
	args := m.Called(ctx, c)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(blocks.Block), args.Error(1)
}

func (m *MockBlockService) GetBlocks(ctx context.Context, ks []cid.Cid) <-chan blocks.Block {
	args := m.Called(ctx, ks)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(<-chan blocks.Block)
}

func (m *MockBlockService) Blockstore() blockstore.Blockstore {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(blockstore.Blockstore)
}

func (m *MockBlockService) Exchange() exchange.Interface {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(exchange.Interface)
}

func (m *MockBlockService) AddBlock(ctx context.Context, b blocks.Block) error {
	args := m.Called(ctx, b)
	return args.Error(0)
}

func (m *MockBlockService) AddBlocks(ctx context.Context, bs []blocks.Block) error {
	args := m.Called(ctx, bs)
	return args.Error(0)
}

func (m *MockBlockService) DeleteBlock(ctx context.Context, c cid.Cid) error {
	args := m.Called(ctx, c)
	return args.Error(0)
}

func (m *MockBlockService) Close() error {
	args := m.Called()
	return args.Error(0)
}

// Helper error for testing
var ErrTestNotFound = errors.New("block not found")

// Test setup helpers

// setupBlockedBlockstore creates a BlockedBlockstore with mock and blocker configured
func setupBlockedBlockstore(t *testing.T, blockedCIDs ...cid.Cid) (*MockBlockstore, *BlockedBlockstore) {
	mockBS := NewMockBlockstore()
	blocker := createTestBlocker(blockedCIDs...)
	blockedBS := &BlockedBlockstore{
		Blockstore: mockBS,
		blocker:    blocker,
	}
	return mockBS, blockedBS
}

// setupBlockedExchange creates a BlockedExchange with mock and blocker configured
func setupBlockedExchange(t *testing.T, blockedCIDs ...cid.Cid) (*MockExchange, *BlockedExchange) {
	mockEx := NewMockExchange()
	blocker := createTestBlocker(blockedCIDs...)
	blockedEx := &BlockedExchange{
		Interface: mockEx,
		blocker:   blocker,
	}
	return mockEx, blockedEx
}

// setupBlockedService creates a BlockService with mock and blocker configured
func setupBlockedService(t *testing.T, blockedCIDs ...cid.Cid) (*MockBlockService, *BlockService) {
	mockBS := NewMockBlockService()
	blocker := createTestBlocker(blockedCIDs...)

	// Setup default mock expectations for Blockstore and Exchange
	mockBS.On("Blockstore").Return(mockBS.mockBlockstore).Maybe()
	mockBS.On("Exchange").Return(mockBS.mockExchange).Maybe()

	wrapped := WrapBlockService(mockBS, blocker)
	return mockBS, wrapped.(*BlockService)
}

// assertBlocked checks that an error indicates blocked content
func assertBlocked(t *testing.T, err error) {
	t.Helper()
	require.Error(t, err, "expected blocked error")
	assert.Contains(t, err.Error(), "blocked", "error should indicate content is blocked")
}

// collectBlocks collects all blocks from a channel into a slice
func collectBlocks(ch <-chan blocks.Block) []blocks.Block {
	var results []blocks.Block
	for blk := range ch {
		results = append(results, blk)
	}
	return results
}

// makeBlockChannel creates a channel with the given blocks and closes it
func makeBlockChannel(blks ...blocks.Block) <-chan blocks.Block {
	ch := make(chan blocks.Block, len(blks))
	for _, blk := range blks {
		ch <- blk
	}
	close(ch)
	return ch
}
