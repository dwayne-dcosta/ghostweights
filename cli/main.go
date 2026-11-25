package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
)

func argAt(slice []string, target int) string {
	for idx, val := range slice {
		if idx == target {
			return val
		}
	}
	return ""
}

func printUsage() {
	fmt.Println("==================================================================")
	fmt.Println("      GHOSTWEIGHTS (v1.1.0) // EPHEMERAL EDGE LLM OPTIMIZER      ")
	fmt.Println("==================================================================")
	fmt.Println("Usage:")
	fmt.Println("  ghostweights pack <input-model> <output.gw>               Compress weights into .gw (keeps original)")
	fmt.Println("  ghostweights shrink <input-model> <output.gw>             Compress into .gw, verify, and delete original")
	fmt.Println("  ghostweights run <model.gw> [-- <cmd...>]                 Mount model virtually & run local process")
	fmt.Println("  ghostweights container <model.gw> <image> [-- <cmd...>]   Spin up ephemeral container & wipe on exit")
	fmt.Println("  ghostweights inspect <model.gw>                           Inspect .gw header & block index metadata")
}

func runEphemeralSession(gwPath string, clientCmd []string, containerMode bool) error {
	reader, err := OpenGhostModel(gwPath)
	if err != nil {
		return err
	}

	baseName := strings.TrimSuffix(filepath.Base(gwPath), ".gw")
	if baseName == "" {
		baseName = "model.bin"
	}

	vm, err := MountGhostVolume(reader, baseName)
	if err != nil {
		_ = reader.WipeMemory()
		return err
	}

	cleanedUp := false
	ghostWipe := func(reason string) {
		if cleanedUp {
			return
		}
		cleanedUp = true
		fmt.Printf("\n🚨 [GhostWeights] %s — Executing Volatile Ghost Wipe...\n", reason)
		if err := vm.UnmountAndDestroy(); err != nil {
			fmt.Printf("⚠️  [GhostWeights] Wipe warning: %v\n", err)
		} else {
			fmt.Println("💥 [GhostWeights] Virtual mount destroyed & volatile RAM zeroed out.")
			fmt.Println("✨ [GhostWeights] NVMe SSD footprint restored to zero bytes.")
		}
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	if len(clientCmd) > 0 {
		streamURL := vm.EndpointURL
		if containerMode {
			streamURL = strings.Replace(streamURL, "127.0.0.1", "host.docker.internal", 1)
		}

		cmd := exec.Command(argAt(clientCmd, 0), clientCmd[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		cmd.Env = append(os.Environ(),
			"GHOSTWEIGHTS_URL="+streamURL,
			"GHOSTWEIGHTS_MOUNT="+vm.MountPoint,
		)

		fmt.Printf("🚀 [Sandbox] Launching ephemeral workload: %s\n", strings.Join(clientCmd, " "))
		if err := cmd.Start(); err != nil {
			ghostWipe("CLIENT LAUNCH FAILED")
			return fmt.Errorf("failed starting wrapped process: %w", err)
		}

		done := make(chan error, 1)
		go func() {
			done <- cmd.Wait()
		}()

		select {
		case sig := <-sigChan:
			_ = cmd.Process.Signal(sig)
			<-done
			ghostWipe(fmt.Sprintf("SIGNAL RECEIVED (%v)", sig))
		case waitErr := <-done:
			ghostWipe("WORKLOAD TERMINATED")
			if waitErr != nil {
				return fmt.Errorf("workload exited with error: %w", waitErr)
			}
		}
		return nil
	}

	fmt.Println("💡 [Daemon Active] Press Ctrl+C to terminate session and trigger the Ghost Wipe.")
	sig := <-sigChan
	ghostWipe(fmt.Sprintf("SIGNAL RECEIVED (%v)", sig))
	return nil
}

func inspectModel(gwPath string) error {
	reader, err := OpenGhostModel(gwPath)
	if err != nil {
		return err
	}
	defer reader.WipeMemory()

	stat, err := os.Stat(gwPath)
	if err != nil {
		return err
	}

	saved := (1.0 - (float64(stat.Size()) / float64(reader.OriginalSize))) * 100
	fmt.Printf("🔍 [Inspect] File: %s\n", gwPath)
	fmt.Printf("   • Original Uncompressed Size : %d bytes\n", reader.OriginalSize)
	fmt.Printf("   • Packed .gw Disk Size       : %d bytes (%.2f%% saved)\n", stat.Size(), saved)
	fmt.Printf("   • Block Chunk Size           : %d bytes\n", reader.ChunkSize)
	fmt.Printf("   • Total Indexed Blocks       : %d\n", len(reader.Indices))
	return nil
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	subcmd := argAt(os.Args, 1)
	switch subcmd {
	case "pack":
		if len(os.Args) < 4 {
			fmt.Println("Error: pack requires <input-model> <output.gw>")
			os.Exit(1)
		}
		if err := PackModel(argAt(os.Args, 2), argAt(os.Args, 3), false); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Pack failed: %v\n", err)
			os.Exit(1)
		}

	case "shrink":
		if len(os.Args) < 4 {
			fmt.Println("Error: shrink requires <input-model> <output.gw>")
			os.Exit(1)
		}
		if err := PackModel(argAt(os.Args, 2), argAt(os.Args, 3), true); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Shrink failed: %v\n", err)
			os.Exit(1)
		}

	case "run":
		if len(os.Args) < 3 {
			fmt.Println("Error: run requires <model.gw>")
			os.Exit(1)
		}
		gwPath := argAt(os.Args, 2)
		var clientCmd []string
		for idx, val := range os.Args {
			if idx >= 3 && val == "--" && idx+1 < len(os.Args) {
				clientCmd = os.Args[idx+1:]
				break
			}
		}
		if err := runEphemeralSession(gwPath, clientCmd, false); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Run failed: %v\n", err)
			os.Exit(1)
		}

	case "container":
		if len(os.Args) < 4 {
			fmt.Println("Error: container requires <model.gw> <docker-image> [-- <cmd...>]")
			os.Exit(1)
		}
		gwPath := argAt(os.Args, 2)
		imageName := argAt(os.Args, 3)
		var innerCmd []string
		for idx, val := range os.Args {
			if idx >= 4 && val == "--" && idx+1 < len(os.Args) {
				innerCmd = os.Args[idx+1:]
				break
			}
		}

		dockerArgs := []string{
			"run", "--rm",
			"--add-host=host.docker.internal:host-gateway",
			"-e", "GHOSTWEIGHTS_URL",
			imageName,
		}
		dockerArgs = append(dockerArgs, innerCmd...)

		if err := runEphemeralSession(gwPath, append([]string{"docker"}, dockerArgs...), true); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Container session failed: %v\n", err)
			os.Exit(1)
		}

	case "inspect":
		if len(os.Args) < 3 {
			fmt.Println("Error: inspect requires <model.gw>")
			os.Exit(1)
		}
		if err := inspectModel(argAt(os.Args, 2)); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Inspect failed: %v\n", err)
			os.Exit(1)
		}

	default:
		printUsage()
		os.Exit(1)
	}
}
