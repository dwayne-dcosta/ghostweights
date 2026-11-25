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
	MagicBytes    = "GW02"
	ChunkSize     = 4 * 1024 * 1024
	FlagRaw       = 0
	FlagBytePlane = 1
)

type BlockIndex struct {
	UncompressedOffset uint64
	CompressedOffset   uint64
	CompressedSize     uint32
	UncompressedSize   uint32
	Flags              uint32
}

func splitBytePlanes(src []byte) []byte {
	n := len(src)
	dst := make([]byte, n)
	half := (n + 1) / 2
	for i := 0; i < n; i++ {
		if i%2 == 0 {
			dst[i/2] = src[i]
		} else {
			dst[half+(i/2)] = src[i]
		}
	}
	return dst
}

func unsplitBytePlanes(src []byte) []byte {
	n := len(src)
	dst := make([]byte, n)
	half := (n + 1) / 2
	for i := 0; i < n; i++ {
		if i%2 == 0 {
			dst[i] = src[i/2]
		} else {
			dst[i] = src[half+(i/2)]
		}
	}
	return dst
}

func compressZlib(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	zw, err := zlib.NewWriterLevel(&buf, zlib.BestSpeed)
	if err != nil {
		return nil, err
	}
	if _, err := zw.Write(data); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func PackModel(inputPath, outputPath string, replaceSource bool) error {
	inFile, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("failed to open input file: %w", err)
	}

	stat, err := inFile.Stat()
	if err != nil {
		inFile.Close()
		return fmt.Errorf("failed to stat input file: %w", err)
	}
	originalSize := uint64(stat.Size())

	totalBlocks := uint32((originalSize + ChunkSize - 1) / ChunkSize)
	if totalBlocks == 0 {
		totalBlocks = 1
	}

	outFile, err := os.Create(outputPath)
	if err != nil {
		inFile.Close()
		return fmt.Errorf("failed to create output .gw file: %w", err)
	}

	headerSize := int64(20 + (totalBlocks * 28))
	if _, err := outFile.Seek(headerSize, io.SeekStart); err != nil {
		inFile.Close()
		outFile.Close()
		return fmt.Errorf("failed to seek past header: %w", err)
	}

	indices := make([]BlockIndex, 0, totalBlocks)
	buf := make([]byte, ChunkSize)
	var currentUncompressedOffset uint64 = 0
	currentCompressedOffset := uint64(headerSize)

	for {
		n, readErr := io.ReadFull(inFile, buf)
		if n > 0 {
			rawChunk := buf[:n]
			shuffled := splitBytePlanes(rawChunk)
			compShuffled, err := compressZlib(shuffled)
			if err != nil {
				inFile.Close()
				outFile.Close()
				return fmt.Errorf("zlib shuffled error: %w", err)
			}

			chosenBytes := compShuffled
			chosenFlag := uint32(FlagBytePlane)

			if len(compShuffled) >= n-1024 {
				compRaw, err := compressZlib(rawChunk)
				if err != nil {
					inFile.Close()
					outFile.Close()
					return fmt.Errorf("zlib raw error: %w", err)
				}
				if len(compRaw) < len(compShuffled) {
					chosenBytes = compRaw
					chosenFlag = uint32(FlagRaw)
				}
			}

			written, err := outFile.Write(chosenBytes)
			if err != nil {
				inFile.Close()
				outFile.Close()
				return fmt.Errorf("failed writing compressed block: %w", err)
			}

			indices = append(indices, BlockIndex{
				UncompressedOffset: currentUncompressedOffset,
				CompressedOffset:   currentCompressedOffset,
				CompressedSize:     uint32(written),
				UncompressedSize:   uint32(n),
				Flags:              chosenFlag,
			})

			currentUncompressedOffset += uint64(n)
			currentCompressedOffset += uint64(written)
		}

		if readErr == io.EOF || readErr == io.ErrUnexpectedEOF {
			break
		}
		if readErr != nil {
			inFile.Close()
			outFile.Close()
			return fmt.Errorf("error reading input file: %w", readErr)
		}
	}

	if _, err := outFile.Seek(0, io.SeekStart); err != nil {
		inFile.Close()
		outFile.Close()
		return fmt.Errorf("failed to rewind for header write: %w", err)
	}

	if _, err := outFile.Write([]byte(MagicBytes)); err != nil {
		inFile.Close()
		outFile.Close()
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
		binary.Write(outFile, binary.LittleEndian, idx.Flags)
	}

	inFile.Close()
	if err := outFile.Close(); err != nil {
		return fmt.Errorf("failed closing output file: %w", err)
	}

	ratio := (1.0 - (float64(currentCompressedOffset) / float64(originalSize))) * 100
	fmt.Printf("📦 [GhostWeights] Packed %s -> %s\n", inputPath, outputPath)
	fmt.Printf("📊 [Stats] Original: %d bytes | Compressed: %d bytes | Saved: %.2f%% (%d blocks)\n",
		originalSize, currentCompressedOffset, ratio, len(indices))

	if replaceSource {
		verifyReader, err := OpenGhostModel(outputPath)
		if err != nil {
			return fmt.Errorf("verification failed; keeping original file %s: %w", inputPath, err)
		}
		_ = verifyReader.WipeMemory()

		if err := os.Remove(inputPath); err != nil {
			return fmt.Errorf("packed successfully, but failed to remove original file %s: %w", inputPath, err)
		}
		fmt.Printf("🗑️  [Storage Reclaimed] Verified %s and removed uncompressed source (%d bytes freed)\n",
			outputPath, originalSize)
	}

	return nil
}
