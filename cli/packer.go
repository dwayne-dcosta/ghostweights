package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

const (
	MagicBytes = "GW01"
	ChunkSize  = 4 * 1024 * 1024 // 4MB blocks for fast random-access tensor reads
)

// BlockIndex maps an uncompressed byte offset to its compressed location in the .gw file
type BlockIndex struct {
	UncompressedOffset uint64
	CompressedOffset   uint64
	CompressedSize     uint32
	UncompressedSize   uint32
}

// PackModel slices a raw file into 4MB chunks, compresses each independently via zlib,
// and writes a fast random-access Header Index Table at the start of the .gw file.
func PackModel(inputPath, outputPath string) error {
	inFile, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("failed to open input file: %w", err)
	}
	defer inFile.Close()

	stat, err := inFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat input file: %w", err)
	}
	originalSize := uint64(stat.Size())

	totalBlocks := uint32((originalSize + ChunkSize - 1) / ChunkSize)
	if totalBlocks == 0 {
		totalBlocks = 1
	}

	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output .gw file: %w", err)
	}
	defer outFile.Close()

	// Header layout:
	// [4 bytes Magic "GW01"][8 bytes OriginalSize][4 bytes ChunkSize][4 bytes TotalBlocks]
	// Followed by: [TotalBlocks * 24 bytes Index Table]
	headerSize := int64(20 + (totalBlocks * 24))
	if _, err := outFile.Seek(headerSize, io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek past header: %w", err)
	}

	indices := make([]BlockIndex, 0, totalBlocks)
	buf := make([]byte, ChunkSize)
	var currentUncompressedOffset uint64 = 0
	currentCompressedOffset := uint64(headerSize)

	for {
		n, readErr := io.ReadFull(inFile, buf)
		if n > 0 {
			var compBuf bytes.Buffer
			zw, err := zlib.NewWriterLevel(&compBuf, zlib.BestSpeed)
			if err != nil {
				return fmt.Errorf("zlib init error: %w", err)
			}
			if _, err := zw.Write(buf[:n]); err != nil {
				return fmt.Errorf("zlib write error: %w", err)
			}
			if err := zw.Close(); err != nil {
				return fmt.Errorf("zlib close error: %w", err)
			}

			compBytes := compBuf.Bytes()
			written, err := outFile.Write(compBytes)
			if err != nil {
				return fmt.Errorf("failed writing compressed block: %w", err)
			}

			indices = append(indices, BlockIndex{
				UncompressedOffset: currentUncompressedOffset,
				CompressedOffset:   currentCompressedOffset,
				CompressedSize:     uint32(written),
				UncompressedSize:   uint32(n),
			})

			currentUncompressedOffset += uint64(n)
			currentCompressedOffset += uint64(written)
		}

		if readErr == io.EOF || readErr == io.ErrUnexpectedEOF {
			break
		}
		if readErr != nil {
			return fmt.Errorf("error reading input file: %w", readErr)
		}
	}

	// Rewind to byte 0 and write the Header + Index Table
	if _, err := outFile.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("failed to rewind for header write: %w", err)
	}

	if _, err := outFile.Write([]byte(MagicBytes)); err != nil {
		return err
	}
	binary.Write(outFile, binary.LittleEndian, originalSize)
	binary.Write(outFile, binary.LittleEndian, uint32(ChunkSize))
	binary.Write(outFile, binary.LittleEndian, uint32(len(indices)))

	for _, idx := range indices {
		binary.Write(outFile, binary.LittleEndian, idx.UncompressedOffset)
		binary.Write(outFile, binary.LittleEndian, idx.CompressedOffset)
		binary.Write(outFile, binary.LittleEndian, idx.CompressedSize)
		binary.Write(outFile, binary.LittleEndian, idx.UncompressedSize)
	}

	ratio := (1.0 - (float64(currentCompressedOffset) / float64(originalSize))) * 100
	fmt.Printf("📦 [GhostWeights] Packed %s -> %s\n", inputPath, outputPath)
	fmt.Printf("📊 [Stats] Original: %d bytes | Compressed: %d bytes | Saved: %.2f%% (%d blocks)\n",
		originalSize, currentCompressedOffset, ratio, len(indices))

	return nil
}
