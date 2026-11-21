import os
import re
import subprocess
import time
import urllib.request
from contextlib import contextmanager
from pathlib import Path
from typing import Generator, Optional


class GhostWeightsError(Exception):
    """Raised when GhostWeights fails to pack, mount, or clean up a model."""


class GhostSession:
    """Represents an active ephemeral virtual mount of a .gw model."""

    def __init__(self, endpoint_url: str, mount_dir: str, process: subprocess.Popen):
        self.url = endpoint_url
        self.mount_dir = mount_dir
        self._process = process

    def read_range(self, start: int, end: int) -> bytes:
        """Stream an exact byte range directly from volatile RAM without disk extraction."""
        req = urllib.request.Request(
            self.url,
            headers={"Range": f"bytes={start}-{end}"},
        )
        with urllib.request.urlopen(req) as resp:
            return resp.read()


def _resolve_binary(binary_path: Optional[str] = None) -> str:
    if binary_path:
        return binary_path
    repo_root = Path(__file__).resolve().parents[2]
    local_bin = repo_root / "bin" / "ghostweights"
    if local_bin.exists():
        return str(local_bin)
    return "ghostweights"


def pack(input_path: str, output_path: str, binary_path: Optional[str] = None) -> str:
    """Compress a raw GGUF/Safetensors file into block-indexed .gw format."""
    bin_exec = _resolve_binary(binary_path)
    result = subprocess.run(
        [bin_exec, "pack", input_path, output_path],
        capture_output=True,
        text=True,
    )
    if result.returncode != 0:
        raise GhostWeightsError(f"Pack failed: {result.stderr.strip()}")
    return result.stdout.strip()


@contextmanager
def load(gw_path: str, binary_path: Optional[str] = None) -> Generator[GhostSession, None, None]:
    """
    Context manager that mounts a compressed .gw model virtually in RAM
    and guarantees a volatile Ghost Wipe when the block exits.

    Usage:
        with ghostweights.load("llama-3.gw") as model:
            header_bytes = model.read_range(0, 1023)
    """
    bin_exec = _resolve_binary(binary_path)
    proc = subprocess.Popen(
        [bin_exec, "run", gw_path],
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
        bufsize=1,
    )

    endpoint_url = ""
    mount_dir = ""

    try:
        if proc.stdout is None:
            raise GhostWeightsError("Failed to capture GhostWeights daemon stdout.")

        for _ in range(50):
            line = proc.stdout.readline()
            if not line:
                if proc.poll() is not None:
                    err_out = proc.stderr.read() if proc.stderr else ""
                    raise GhostWeightsError(f"Daemon exited prematurely: {err_out.strip()}")
                time.sleep(0.05)
                continue

            match = re.search(r"(http://127\.0\.0\.1:\d+/\S+)", line)
            if match:
                endpoint_url = match.group(1)
                break

        if not endpoint_url:
            raise GhostWeightsError("Timed out waiting for virtual mount endpoint URL.")

        session = GhostSession(endpoint_url=endpoint_url, mount_dir=mount_dir, process=proc)
        yield session

    finally:
        # Send SIGINT to trigger the Go engine's volatile Ghost Wipe
        if proc.poll() is None:
            proc.send_signal(2)  # SIGINT
            try:
                proc.wait(timeout=5)
            except subprocess.TimeoutExpired:
                proc.kill()
