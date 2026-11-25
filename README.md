# GhostWeights 👻⚖️

[![GhostWeights CI](https://github.com/dwayne-dcosta/ghostweights/actions/workflows/go.yml/badge.svg)](https://github.com/dwayne-dcosta/ghostweights/actions/workflows/go.yml)

**Zero-Footprint Ephemeral Runtime & Lossless Tensor Compressor for Edge & Containerized LLMs**

GhostWeights eliminates NVMe SSD storage bloat and container cold-start duplication for local and containerized LLMs (`.safetensors` & `.gguf`). It slices weights into 4MB indexed blocks (`GW02`), applies lossless **BF16/FP16 Byte-Plane Splitting** (separating exponent and mantissa byte planes before compression), mounts models virtually in volatile RAM on demand, and executes an automated **Ghost Wipe** on exit.

```text
                  [ Compressed Model File (.gw) ]
                                 │
              (ghostweights run / ghostweights container)
                                 ▼
┌─────────────────────────────────────────────────────────────────┐
│    Layer 1: GW02 Block Indexer & BF16 Byte-Plane Unshuffler     │
└────────────────────────────────┬────────────────────────────────┘
                                 │
          (On-Demand 4MB Chunk Decompression in ~64MB RAM)
                                 ▼
┌─────────────────────────────────────────────────────────────────┐
│      Layer 2: Zero-Copy Virtual Mount (HTTP Range / mmap)       │
└────────────────────────────────┬────────────────────────────────┘
                                 │
                     (Process Exit / SIGINT)
                                 ▼
┌─────────────────────────────────────────────────────────────────┐
│         Layer 3: Automated Ghost Wipe (Zero Disk Footprint)     │
└─────────────────────────────────────────────────────────────────┘
```

## Verified Hugging Face Benchmarks (Apple Silicon)

Tested on **`Qwen/Qwen2.5-1.5B-Instruct`** (`BF16 .safetensors` and `Q4_K_M .gguf`):

| Model Artifact | Format | Original Size | Packed `.gw` Size | Net Disk Saved | Post-Exit Footprint |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `Qwen2.5-1.5B-Instruct` | `BF16 .safetensors` | 3,087,467,144 B (3.09 GB) | 2,194,548,819 B (2.19 GB) | **28.92% (892.9 MB)** | **0 Bytes** |
| `Qwen2.5-1.5B-Instruct` | `Q4_K_M .gguf` | 1,117,320,736 B (1.12 GB) | 1,112,895,528 B (1.11 GB) | **0.40% (4.43 MB)** | **0 Bytes** |

* **Constant ~64MB RAM Overhead:** The 16-block LRU cache (`16 × 4MB`) bounds memory overhead to ~64MB whether streaming a 1.5B or 70B+ parameter model (up to 16 PB theoretical ceiling).
* **Lossless Byte-Plane Splitting:** Separates interleaved `BF16/FP16` exponent and mantissa bytes per 4MB chunk, delivering **~29% lossless compression** on floating-point tensors while falling back automatically on pre-quantized 4-bit GGUFs.

## CLI Toolkit

### 1. Build & Verify
```bash
make build
make test
```

### 2. Core Commands
```bash
# Non-destructive lossless compression (keeps original file)
./bin/ghostweights pack model.safetensors qwen.gw

# Compress, verify block integrity, and remove uncompressed source to reclaim disk space
./bin/ghostweights shrink model.safetensors qwen.gw

# Inspect GW02 header & compression stats
./bin/ghostweights inspect qwen.gw

# Mount virtually in RAM for local processes (wipes on exit)
./bin/ghostweights run qwen.gw -- bash -c 'curl -s -r 0-4095 $GHOSTWEIGHTS_URL | wc -c'

# Spin up an ephemeral Docker container wired to host RAM stream (destroys container + mount on exit)
./bin/ghostweights container qwen.gw python:3.11-slim -- python3 -c "import os; print(os.environ['GHOSTWEIGHTS_URL'])"
```

### 3. Python Context Manager SDK
```python
from sdk.python import load, pack

pack("model.safetensors", "qwen.gw")

with load("qwen.gw") as model:
    # Stream exact tensor byte slices directly from volatile RAM
    header_bytes = model.read_range(0, 4095)
# Ghost Wipe triggers automatically here
```
