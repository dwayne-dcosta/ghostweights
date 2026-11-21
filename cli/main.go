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

func printUsage() {
	fmt.Println("==================================================================")
	fmt.Println("      GHOSTWEIGHTS (v1.0.0) // EPHEMERAL EDGE LLM OPTIMIZER      ")
	fmt.Println("==================================================================")
	fmt.Println("Usage:")
	fmt.Println("  ghostweights pack <input-model> <output.gw>   Compress weights into block-indexed .gw format")
	fmt.Println("  ghostweights run <model.gw> [-- <cmd...>]     Mount model virtually & run ephemeral session")
	fmt.Println("  ghostweights inspect <model.gw>               Inspect .gw header & block index metadata")
}

func runEphemeralSession(gwPath string, clientCmd []string) error {
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

	// Ensure Ghost Wipe always runs on exit
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

	// If a wrapped command was provided (e.g., `ghostweights run llama.gw -- python3 app.py`),
	// pass GHOSTWEIGHTS_URL into its environment and destroy the mount as soon as it exits.
	if len(clientCmd) > 0 {
		cmd := exec.Command(clientCmd[0], clientCmd[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		cmd.Env = append(os.Environ(),
			"GHOSTWEIGHTS_URL="+vm.EndpointURL,
			"GHOSTWEIGHTS_MOUNT="+vm.MountPoint,
		)

		fmt.Printf("🚀 [Sandbox] Launching ephemeral client process: %s\n", strings.Join(clientCmd, " "))
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
			ghostWipe("CLIENT PROCESS TERMINATED")
			if waitErr != nil {
				return fmt.Errorf("client process exited with error: %w", waitErr)
			}
		}
		return nil
	}

	// Standalone daemon mode: stay mounted until user hits Ctrl+C
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

	subcmd := os.Args[1]
	switch subcmd {
	case "pack":
		if len(os.Args) < 4 {
			fmt.Println("Error: pack requires <input-model> and <output.gw>")
			os.Exit(1)
		}
		if err := PackModel(os.Args[2], os.Args[3]); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Pack failed: %v\n", err)
			os.Exit(1)
		}

	case "run":
		if len(os.Args) < 3 {
			fmt.Println("Error: run requires <model.gw>")
			os.Exit(1)
		}
		gwPath := os.Args[2]
		var clientCmd []string
		for i := 3; i < len(os.Args); i++ {
			if os.Args[i] == "--" && i+1 < len(os.Args) {
				clientCmd = os.Args[i+1:]
				break
			}
		}
		if err := runEphemeralSession(gwPath, clientCmd); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Run failed: %v\n", err)
			os.Exit(1)
		}

	case "inspect":
		if len(os.Args) < 3 {
			fmt.Println("Error: inspect requires <model.gw>")
			os.Exit(1)
		}
		if err := inspectModel(os.Args[2]); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Inspect failed: %v\n", err)
			os.Exit(1)
		}

	default:
		printUsage()
		os.Exit(1)
	}
}
