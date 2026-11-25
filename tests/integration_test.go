package tests

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestGhostWeightsEndToEnd(t *testing.T) {
	tmpDir := t.TempDir()
	binName := "ghostweights"
	if runtime.GOOS == "windows" {
		binName = "ghostweights.exe"
	}
	binaryPath := filepath.Join(tmpDir, binName)

	// 1. Build the Go CLI binary from ../cli
	buildCmd := exec.Command("go", "build", "-o", binaryPath, "../cli")
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build ghostweights binary: %v\nOutput: %s", err, string(out))
	}

	// 2. Generate a 9MB deterministic sample model file (spans 3 blocks: 4MB + 4MB + 1MB)
	rawModelPath := filepath.Join(tmpDir, "test_model.bin")
	rawSize := 9 * 1024 * 1024
	rawData := make([]byte, rawSize)
	for i := range rawData {
		rawData[i] = byte((i / 4096) % 256)
	}
	if err := os.WriteFile(rawModelPath, rawData, 0600); err != nil {
		t.Fatalf("failed writing raw sample model: %v", err)
	}

	// 3. Test `ghostweights pack`
	gwPath := filepath.Join(tmpDir, "test_model.gw")
	packCmd := exec.Command(binaryPath, "pack", rawModelPath, gwPath)
	packOut, err := packCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("pack command failed: %v\nOutput: %s", err, string(packOut))
	}
	if !strings.Contains(string(packOut), "Packed") {
		t.Errorf("unexpected pack output: %s", string(packOut))
	}

	// 4. Test `ghostweights inspect`
	inspectCmd := exec.Command(binaryPath, "inspect", gwPath)
	inspectOut, err := inspectCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("inspect command failed: %v\nOutput: %s", err, string(inspectOut))
	}
	if !strings.Contains(string(inspectOut), "Total Indexed Blocks       : 3") {
		t.Errorf("expected 3 indexed blocks in inspect output, got:\n%s", string(inspectOut))
	}

	// 5. Test `ghostweights run` with a wrapped bash script that fetches a specific byte range
	fetchedBytesPath := filepath.ToSlash(filepath.Join(tmpDir, "fetched_slice.bin"))
	mountDirRecordPath := filepath.ToSlash(filepath.Join(tmpDir, "mount_dir.txt"))

	script := fmt.Sprintf(`
		printf '%%s' "$GHOSTWEIGHTS_MOUNT" > %q
		curl -s -H "Range: bytes=4194000-4194999" "$GHOSTWEIGHTS_URL" -o %q
	`, mountDirRecordPath, fetchedBytesPath)

	runCmd := exec.Command(binaryPath, "run", gwPath, "--", "bash", "-c", script)
	runOut, err := runCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run command failed: %v\nOutput: %s", err, string(runOut))
	}

	if !strings.Contains(string(runOut), "NVMe SSD footprint restored to zero bytes") {
		t.Errorf("expected Ghost Wipe confirmation in output, got:\n%s", string(runOut))
	}

	// 6. Verify the byte range streamed via HTTP Range matches the exact original bytes
	fetchedSlice, err := os.ReadFile(fetchedBytesPath)
	if err != nil {
		t.Fatalf("failed reading fetched byte slice: %v", err)
	}
	expectedSlice := rawData[4194000:4195000]
	if !bytes.Equal(fetchedSlice, expectedSlice) {
		t.Errorf("streamed byte slice mismatch! Got %d bytes, expected %d bytes", len(fetchedSlice), len(expectedSlice))
	}

	// 7. Verify the ephemeral mount directory was completely destroyed (zero footprint)
	mountDirBytes, err := os.ReadFile(mountDirRecordPath)
	if err != nil {
		t.Fatalf("failed reading recorded mount dir: %v", err)
	}
	mountDir := strings.TrimSpace(string(mountDirBytes))
	if _, err := os.Stat(mountDir); !os.IsNotExist(err) {
		t.Errorf("Ghost Wipe failed: ephemeral mount directory %s still exists on disk!", mountDir)
	}

	// 8. Verify the local HTTP endpoint is shut down and unreachable after termination
	if strings.Contains(string(runOut), "http://127.0.0.1:") {
		lines := strings.Split(string(runOut), "\n")
		for _, line := range lines {
			if idx := strings.Index(line, "http://127.0.0.1:"); idx != -1 {
				url := strings.TrimSpace(line[idx:])
				resp, err := http.Get(url)
				if err == nil {
					io.Copy(io.Discard, resp.Body)
					resp.Body.Close()
					t.Errorf("Ghost Wipe failed: endpoint %s is still accepting connections!", url)
				}
			}
		}
	}
}
