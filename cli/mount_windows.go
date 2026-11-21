//go:build windows

package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// VirtualMount represents an active, read-only volatile memory mount on Windows
type VirtualMount struct {
	Reader      *GhostReader
	MountPoint  string
	EndpointURL string
	server      *http.Server
	listener    net.Listener
}

// MountGhostVolume exposes the compressed .gw model as a zero-disk-footprint virtual stream on Windows
func MountGhostVolume(reader *GhostReader, modelName string) (*VirtualMount, error) {
	sessionDir := filepath.Join(os.TempDir(), fmt.Sprintf("ghostweights-%d", os.Getpid()))
	if err := os.MkdirAll(sessionDir, 0700); err != nil {
		return nil, fmt.Errorf("failed creating volatile session dir: %w", err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		os.RemoveAll(sessionDir)
		return nil, fmt.Errorf("failed binding virtual mount socket: %w", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/"+modelName, func(w http.ResponseWriter, r *http.Request) {
		section := io.NewSectionReader(reader, 0, int64(reader.OriginalSize))
		http.ServeContent(w, r, modelName, time.Now(), section)
	})

	srv := &http.Server{Handler: mux}
	go func() {
		_ = srv.Serve(listener)
	}()

	endpointURL := fmt.Sprintf("http://%s/%s", listener.Addr().String(), modelName)
	_ = os.WriteFile(filepath.Join(sessionDir, "mount.uri"), []byte(endpointURL), 0600)

	fmt.Printf("🪟 [Windows Virtual Mount] Mounted %s (%d bytes virtual size)\n", modelName, reader.OriginalSize)
	fmt.Printf("🔗 [Zero-Copy Stream] Active at %s\n", endpointURL)

	return &VirtualMount{
		Reader:      reader,
		MountPoint:  sessionDir,
		EndpointURL: endpointURL,
		server:      srv,
		listener:    listener,
	}, nil
}

// UnmountAndDestroy executes the volatile Ghost Wipe on Windows
func (vm *VirtualMount) UnmountAndDestroy() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if vm.server != nil {
		_ = vm.server.Shutdown(ctx)
	}
	if vm.Reader != nil {
		_ = vm.Reader.WipeMemory()
	}
	if vm.MountPoint != "" {
		return os.RemoveAll(vm.MountPoint)
	}
	return nil
}
