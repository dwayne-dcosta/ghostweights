package main

import (
	"bytes"
	"compress/zlib"
	"container/list"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sync"
)

const MaxCachedBlocks = 16 // Keeps up to 64MB (16 * 4MB) of active tensor blocks in volatile RAM

type cacheEntry struct {
	blockIdx uint32
	data     []byte
}

// GhostReader provides zero-extraction, random-access reads over a packed .gw file
type GhostReader struct {
	file         *os.File
	OriginalSize uint64
	ChunkSize    uint32
	Indices      []BlockIndex

	mu        sync.Mutex
	cache     map[uint32]*list.Element
	lruList   *list.List
	cacheHits uint64
	diskReads uint64
}

// OpenGhostModel reads the GW01 header and offset table into RAM without unpacking payload blocks
func OpenGhostModel(gwPath string) (*GhostReader, error) {
	f, err := os.Open(gwPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open .gw model: %w", err)
	}

	magic := make([]byte, 4)
	if _, err := io.ReadFull(f, magic); err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to read magic bytes: %w", err)
	}
	if string(magic) != MagicBytes {
		f.Close()
		return nil, fmt.Errorf("invalid file format: expected %s, got %s", MagicBytes, string(magic))
	}

	var originalSize uint64
	var chunkSize uint32
	var totalBlocks uint32

	binary.Read(f, binary.LittleEndian, &originalSize)
	binary.Read(f, binary.LittleEndian, &chunkSize)
	binary.Read(f, binary.LittleEndian, &totalBlocks)

	indices := make([]BlockIndex, totalBlocks)
	for i := uint32(0); i < totalBlocks; i++ {
		binary.Read(f, binary.LittleEndian, &indices[i].UncompressedOffset)
		binary.Read(f, binary.LittleEndian, &indices[i].CompressedOffset)
		binary.Read(f, binary.LittleEndian, &indices[i].CompressedSize)
		binary.Read(f, binary.LittleEndian, &indices[i].UncompressedSize)
	}

	return &GhostReader{
		file:         f,
		OriginalSize: originalSize,
		ChunkSize:    chunkSize,
		Indices:      indices,
		cache:        make(map[uint32]*list.Element),
		lruList:      list.New(),
	}, nil
}

// getBlock fetches a single 4MB decompressed block from volatile LRU cache or decompresses it on-the-fly
func (gr *GhostReader) getBlock(blockIdx uint32) ([]byte, error) {
	gr.mu.Lock()
	if elem, ok := gr.cache[blockIdx]; ok {
		gr.lruList.MoveToFront(elem)
		gr.cacheHits++
		data := elem.Value.(*cacheEntry).data
		gr.mu.Unlock()
		return data, nil
	}
	gr.diskReads++
	gr.mu.Unlock()

	if int(blockIdx) >= len(gr.Indices) {
		return nil, io.EOF
	}
	idx := gr.Indices[blockIdx]

	compData := make([]byte, idx.CompressedSize)
	if _, err := gr.file.ReadAt(compData, int64(idx.CompressedOffset)); err != nil {
		return nil, fmt.Errorf("failed reading compressed block %d: %w", blockIdx, err)
	}

	zr, err := zlib.NewReader(bytes.NewReader(compData))
	if err != nil {
		return nil, fmt.Errorf("zlib reader init error on block %d: %w", blockIdx, err)
	}
	defer zr.Close()

	uncompData := make([]byte, idx.UncompressedSize)
	if _, err := io.ReadFull(zr, uncompData); err != nil {
		return nil, fmt.Errorf("zlib decompress error on block %d: %w", blockIdx, err)
	}

	gr.mu.Lock()
	defer gr.mu.Unlock()

	if elem, ok := gr.cache[blockIdx]; ok {
		gr.lruList.MoveToFront(elem)
		return elem.Value.(*cacheEntry).data, nil
	}

	if gr.lruList.Len() >= MaxCachedBlocks {
		oldest := gr.lruList.Back()
		if oldest != nil {
			oldEntry := oldest.Value.(*cacheEntry)
			delete(gr.cache, oldEntry.blockIdx)
			gr.lruList.Remove(oldest)
		}
	}

	elem := gr.lruList.PushFront(&cacheEntry{blockIdx: blockIdx, data: uncompData})
	gr.cache[blockIdx] = elem
	return uncompData, nil
}

// ReadAt implements io.ReaderAt so virtual mounts and mmap proxies can read arbitrary byte ranges
func (gr *GhostReader) ReadAt(p []byte, off int64) (int, error) {
	if off < 0 {
		return 0, fmt.Errorf("negative offset: %d", off)
	}
	if uint64(off) >= gr.OriginalSize {
		return 0, io.EOF
	}

	totalRead := 0
	for totalRead < len(p) && uint64(off) < gr.OriginalSize {
		blockIdx := uint32(uint64(off) / uint64(gr.ChunkSize))
		offsetInBlock := int(uint64(off) % uint64(gr.ChunkSize))

		blockData, err := gr.getBlock(blockIdx)
		if err != nil {
			return totalRead, err
		}

		available := len(blockData) - offsetInBlock
		needed := len(p) - totalRead
		toCopy := available
		if needed < toCopy {
			toCopy = needed
		}

		copy(p[totalRead:totalRead+toCopy], blockData[offsetInBlock:offsetInBlock+toCopy])
		totalRead += toCopy
		off += int64(toCopy)
	}

	if totalRead < len(p) {
		return totalRead, io.EOF
	}
	return totalRead, nil
}

// WipeMemory zeroes out all cached tensor blocks in volatile RAM and closes the file handle
func (gr *GhostReader) WipeMemory() error {
	gr.mu.Lock()
	defer gr.mu.Unlock()

	for k, elem := range gr.cache {
		entry := elem.Value.(*cacheEntry)
		for i := range entry.data {
			entry.data[i] = 0 // Explicitly overwrite volatile RAM bytes
		}
		delete(gr.cache, k)
	}
	gr.lruList.Init()

	if gr.file != nil {
		return gr.file.Close()
	}
	return nil
}
