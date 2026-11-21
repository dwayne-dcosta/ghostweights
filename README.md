# GhostWeights 👻⚖️

**Ephemeral Edge LLM Optimizer & Zero-Copy Virtual Block Storage Daemon**

GhostWeights eliminates NVMe SSD storage bloat for local edge LLMs (GGUF / Safetensors) on macOS and Linux. Instead of extracting multi-gigabyte model files to disk or relying on slow `.zip` archives, GhostWeights slices model weights into block-indexed `.gw` archives, mounts them virtually in volatile RAM on demand, and executes an automated **Ghost Wipe** the moment your inference process terminates.

```text
                  [ Compressed Model File (.gw) ]
                                 │
                        (ghostweights run)
                                 ▼
┌─────────────────────────────────────────────────────────────────┐
│         Layer 1: Block-Indexed Offset Reader (GW01 Format)      │
└────────────────────────────────┬────────────────────────────────┘
                                 │
             (On-Demand 4MB Chunk Decompression in RAM)
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

## Key Features
* **Block-Indexed `.gw` Format (`GW01`):** Slices raw weights into independent 4MB compressed blocks with a binary Header Index Table for microsecond random-access tensor reads.
* **Volatile LRU RAM Cache:** Keeps active tensor blocks in memory and serves arbitrary byte ranges via `io.ReaderAt` and HTTP Range streaming without writing unpacked data to your NVMe SSD.
* **Automated Ghost Wipe:** Traps `SIGINT` / `SIGTERM` and subprocess exit events to forcefully unmount virtual endpoints, overwrite cached RAM blocks with zeroes, and delete transient `/tmp` session directories.
* **Zero-Dependency Core & Python SDK:** Written in pure standard-library Go (`compress/zlib`, `net/http`, `os/signal`) paired with a drop-in Python context manager (`with ghostweights.load(...)`).

## Quickstart

### 1. Build & Verify
```bash
make build
make test
```

### 2. Pack & Run via CLI
```bash
# Compress a local model file into block-indexed .gw format
./bin/ghostweights pack llama-3-8b.gguf llama-3-8b.gw

# Inspect header & storage savings
./bin/ghostweights inspect llama-3-8b.gw

# Mount virtually and run an ephemeral client session
./bin/ghostweights run llama-3-8b.gw -- bash -c 'curl -s -I $GHOSTWEIGHTS_URL | head -n 5'
```

### 3. Python Context Manager SDK
```python
from sdk.python import load, pack

pack("llama-3-8b.gguf", "llama-3-8b.gw")

with load("llama-3-8b.gw") as model:
    # Stream exact tensor byte slices directly from volatile RAM
    header_bytes = model.read_range(0, 4095)
# Ghost Wipe triggers automatically here
```
