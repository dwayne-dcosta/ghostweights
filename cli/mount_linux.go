//go:build linux

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

// VirtualMount represents an active, read-only volatile memory mount on Linux (/dev/shm)
type VirtualMount struct {
	Reader      *GhostReader
	MountPoint  string
	EndpointURL string
	server      *http.Server
	listener    net.Listener
}

// MountGhostVolume exposes the compressed .gw model using Linux /dev/shm volatile RAM storage
func MountGhostVolume(reader *GhostReader, modelName string) (*VirtualMount, error) {
	baseTmp := "/dev/shm"
	if _, err := os.Stat(baseTmp); os.IsNotExist(err) {
		baseTmp = os.TempDir()
	}

	sessionDir := filepath.Join(baseTmp, fmt.Sprintf("ghostweights-%d", os.Getpid()))
	if err := os.MkdirAll(sessionDir, 0700); err != nil {
		return nil, fmt.Errorf("failed creating /dev/shm session dir: %w", err)
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

	fmt.Printf("🐧 [Linux /dev/shm Mount] Mounted %s (%d bytes virtual size)\n", modelName, reader.OriginalSize)
	fmt.Printf("🔗 [Zero-Copy Stream] Active at %s\n", endpointURL)

	return &VirtualMount{
		Reader:      reader,
		MountPoint:  sessionDir,
		EndpointURL: endpointURL,
		server:      srv,
		listener:    listener,
	}, nil
}

// UnmountAndDestroy executes the volatile Ghost Wipe on Linux
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
