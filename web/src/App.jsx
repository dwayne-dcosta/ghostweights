import React, { useState } from 'react';
import {
  Terminal, ShieldCheck, Cpu, HardDrive, Play, Square,
  Copy, Check, Code2, Heart, Layers, Container, Zap
} from 'lucide-react';

export default function App() {
  const [copied, setCopied] = useState(false);
  const [activeTab, setActiveTab] = useState('cli');
  const [simState, setSimState] = useState('idle');

  const installCmd = 'go install github.com/dwayne-dcosta/ghostweights/cli@v1.1.0';

  const handleCopy = () => {
    navigator.clipboard.writeText(installCmd);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="min-h-screen bg-[#05080f] text-slate-200 selection:bg-cyan-500/30">
      <header className="sticky top-0 z-50 border-b border-slate-800/80 bg-[#05080f]/80 backdrop-blur-md">
        <div className="mx-auto flex max-w-6xl items-center justify-between px-6 py-4">
          <div className="flex items-center gap-3">
            <div className="flex h-9 w-9 items-center justify-center rounded-lg border border-cyan-500/40 bg-cyan-500/10 text-cyan-400">
              <Cpu className="h-5 w-5" />
            </div>
            <span className="text-lg font-bold tracking-tight text-white">
              Ghost<span className="text-cyan-400">Weights</span>
            </span>
            <span className="rounded-full border border-emerald-500/30 bg-emerald-500/10 px-2.5 py-0.5 text-xs font-medium text-emerald-400">
              v1.1.0
            </span>
          </div>
          <a
            href="https://github.com/dwayne-dcosta/ghostweights"
            target="_blank"
            rel="noreferrer"
            className="flex items-center gap-2 rounded-lg border border-slate-700 bg-slate-900/90 px-3.5 py-2 text-sm font-medium text-slate-200 transition hover:border-slate-500 hover:text-white"
          >
            <Code2 className="h-4 w-4" />
            <span>GitHub</span>
          </a>
        </div>
      </header>

      <section className="relative overflow-hidden px-6 pt-20 pb-24">
        <div className="mx-auto max-w-4xl text-center">
          <div className="mb-6 inline-flex items-center gap-2 rounded-full border border-cyan-500/30 bg-cyan-500/10 px-4 py-1.5 text-xs font-medium text-cyan-300">
            <Zap className="h-3.5 w-3.5" />
            <span>GW02 Engine • Lossless BF16 Byte-Plane Splitting • macOS / Linux / Windows</span>
          </div>
          <h1 className="text-4xl font-extrabold tracking-tight text-white sm:text-6xl">
            Run Multi-Gigabyte LLMs With{' '}
            <span className="bg-gradient-to-r from-cyan-400 via-teal-300 to-emerald-400 bg-clip-text text-transparent">
              Zero NVMe Disk Footprint
            </span>
          </h1>
          <p className="mx-auto mt-6 max-w-2xl text-lg leading-relaxed text-slate-400">
            Compress Hugging Face weights up to <strong className="text-slate-200">28.9% smaller</strong> with lossless BF16 byte-plane indexing, stream 4MB blocks directly in volatile RAM, and execute an automated <strong className="text-cyan-300">Ghost Wipe</strong> on exit.
          </p>
          <div className="mx-auto mt-10 flex max-w-xl items-center justify-between rounded-xl border border-slate-800 bg-slate-950/90 px-4 py-3 font-mono text-sm">
            <div className="flex items-center gap-3 overflow-x-auto text-slate-300">
              <span className="text-cyan-400">$</span>
              <code className="whitespace-nowrap">{installCmd}</code>
            </div>
            <button
              onClick={handleCopy}
              className="ml-3 flex shrink-0 items-center gap-1.5 rounded-lg border border-slate-700 bg-slate-800/80 px-3 py-1.5 text-xs font-sans font-medium text-slate-200 transition hover:border-cyan-500/50 hover:text-cyan-300"
            >
              {copied ? <Check className="h-3.5 w-3.5 text-emerald-400" /> : <Copy className="h-3.5 w-3.5" />}
              <span>{copied ? 'Copied' : 'Copy'}</span>
            </button>
          </div>
        </div>
      </section>
      <section id="simulator" className="border-y border-slate-800/80 bg-slate-950/50 px-6 py-20">
        <div className="mx-auto max-w-5xl">
          <div className="mb-10 text-center">
            <h2 className="text-3xl font-bold text-white">Interactive Ghost Wipe Simulator</h2>
            <p className="mt-2 text-slate-400">
              Test how GhostWeights streams tensor blocks in ~64MB volatile RAM and leaves zero residual disk bytes on exit.
            </p>
          </div>
          <div className="rounded-2xl border border-slate-800 bg-[#090d16] p-6 sm:p-8">
            <div className="flex flex-wrap items-center justify-between gap-4 border-b border-slate-800 pb-6">
              <div className="flex items-center gap-3">
                <Terminal className="h-5 w-5 text-cyan-400" />
                <span className="font-mono text-sm text-slate-300">
                  ghostweights container qwen2.5-1.5b.gw inference-worker
                </span>
              </div>
              <div className="flex items-center gap-3">
                <button
                  onClick={() => setSimState('running')}
                  className="flex items-center gap-2 rounded-lg bg-cyan-500 px-4 py-2 text-sm font-semibold text-slate-950 transition hover:bg-cyan-400"
                >
                  <Play className="h-4 w-4" />
                  <span>Mount &amp; Stream</span>
                </button>
                <button
                  onClick={() => setSimState('wiped')}
                  disabled={simState !== 'running'}
                  className="flex items-center gap-2 rounded-lg border border-rose-500/40 bg-rose-500/10 px-4 py-2 text-sm font-semibold text-rose-300 transition hover:bg-rose-500/20 disabled:opacity-40"
                >
                  <Square className="h-4 w-4" />
                  <span>Send SIGINT (Ghost Wipe)</span>
                </button>
              </div>
            </div>
            <div className="mt-8 grid gap-6 md:grid-cols-3">
              <div className="rounded-xl border border-slate-800 bg-slate-900/60 p-5">
                <div className="flex items-center justify-between text-xs font-medium uppercase text-slate-400">
                  <span>1. Packed Archive (.gw)</span>
                  <HardDrive className="h-4 w-4 text-cyan-400" />
                </div>
                <div className="mt-3 text-2xl font-bold text-white">2.19 GB</div>
                <p className="mt-1 text-xs text-emerald-400">28.92% smaller than 3.09 GB BF16 source</p>
              </div>
              <div className={`rounded-xl border p-5 transition-all ${simState === 'running' ? 'border-cyan-500/60 bg-cyan-950/20' : 'border-slate-800 bg-slate-900/60'}`}>
                <div className="flex items-center justify-between text-xs font-medium uppercase text-slate-400">
                  <span>2. Active Volatile RAM LRU</span>
                  <Layers className="h-4 w-4 text-cyan-400" />
                </div>
                <div className="mt-3 text-2xl font-bold text-white">
                  {simState === 'running' ? '64.0 MB (16 Blocks)' : '0.0 MB'}
                </div>
                <p className="mt-1 text-xs text-slate-400">
                  {simState === 'running' ? 'Streaming HTTP Range slices...' : simState === 'wiped' ? 'Memory buffers zeroed on SIGINT' : 'Waiting for mount command'}
                </p>
              </div>
              <div className={`rounded-xl border p-5 transition-all ${simState === 'wiped' ? 'border-emerald-500/60 bg-emerald-950/20' : 'border-slate-800 bg-slate-900/60'}`}>
                <div className="flex items-center justify-between text-xs font-medium uppercase text-slate-400">
                  <span>3. Unpacked NVMe Footprint</span>
                  <ShieldCheck className="h-4 w-4 text-emerald-400" />
                </div>
                <div className="mt-3 text-2xl font-bold text-emerald-400">0 Bytes</div>
                <p className="mt-1 text-xs text-slate-400">
                  {simState === 'wiped' ? 'Ghost Wipe verified: mount & socket destroyed' : 'Zero temporary extraction on SSD'}
                </p>
              </div>
            </div>
          </div>
        </div>
      </section>
      <section id="benchmarks" className="px-6 py-20">
        <div className="mx-auto max-w-5xl">
          <h2 className="text-3xl font-bold text-white">Verified Hugging Face Benchmarks</h2>
          <p className="mt-2 mb-8 text-slate-400">Measured on Apple Silicon using Qwen/Qwen2.5-1.5B-Instruct.</p>
          <div className="overflow-x-auto rounded-xl border border-slate-800">
            <table className="w-full text-left text-sm">
              <thead className="border-b border-slate-800 bg-slate-900/80 text-xs uppercase text-slate-400">
                <tr>
                  <th className="px-6 py-4">Model</th>
                  <th className="px-6 py-4">Format</th>
                  <th className="px-6 py-4">Original</th>
                  <th className="px-6 py-4">Packed .gw</th>
                  <th className="px-6 py-4">Saved</th>
                  <th className="px-6 py-4">Post-Exit</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-800/80">
                <tr>
                  <td className="px-6 py-4 font-medium text-white">Qwen2.5-1.5B-Instruct</td>
                  <td className="px-6 py-4 font-mono text-cyan-300">BF16 .safetensors</td>
                  <td className="px-6 py-4">3.09 GB</td>
                  <td className="px-6 py-4">2.19 GB</td>
                  <td className="px-6 py-4 font-semibold text-emerald-400">28.92% (892.9 MB)</td>
                  <td className="px-6 py-4 font-mono text-emerald-400">0 Bytes</td>
                </tr>
                <tr>
                  <td className="px-6 py-4 font-medium text-white">Qwen2.5-1.5B-Instruct</td>
                  <td className="px-6 py-4 font-mono text-slate-300">Q4_K_M .gguf</td>
                  <td className="px-6 py-4">1.12 GB</td>
                  <td className="px-6 py-4">1.11 GB</td>
                  <td className="px-6 py-4">0.40% (4.43 MB)</td>
                  <td className="px-6 py-4 font-mono text-emerald-400">0 Bytes</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </section>

      <section id="quickstart" className="border-t border-slate-800/80 bg-slate-950/50 px-6 py-20">
        <div className="mx-auto max-w-5xl">
          <div className="mb-8 flex flex-wrap items-center justify-between gap-4">
            <div>
              <h2 className="text-3xl font-bold text-white">Drop-In CLI &amp; Python SDK</h2>
              <p className="mt-1 text-slate-400">Native binaries for macOS, Linux, and Windows.</p>
            </div>
            <div className="flex rounded-lg border border-slate-800 bg-slate-900 p-1">
              <button onClick={() => setActiveTab('cli')} className={`rounded-md px-4 py-1.5 text-sm font-medium ${activeTab === 'cli' ? 'bg-cyan-500 text-slate-950' : 'text-slate-400'}`}>Go CLI</button>
              <button onClick={() => setActiveTab('python')} className={`rounded-md px-4 py-1.5 text-sm font-medium ${activeTab === 'python' ? 'bg-cyan-500 text-slate-950' : 'text-slate-400'}`}>Python SDK</button>
            </div>
          </div>
          <div className="rounded-2xl border border-slate-800 bg-[#090d16] p-6 font-mono text-sm text-slate-300 overflow-x-auto">
            {activeTab === 'cli' ? (
              <pre>{`ghostweights shrink model.safetensors qwen.gw
ghostweights container qwen.gw python:3.11-slim -- python3 infer.py
ghostweights inspect qwen.gw`}</pre>
            ) : (
              <pre>{`from sdk.python import load, pack
pack("model.safetensors", "qwen.gw")
with load("qwen.gw") as model:
    header = model.read_range(0, 4095)`}</pre>
            )}
          </div>
        </div>
      </section>

      <section id="enterprise" className="border-t border-slate-800/80 px-6 py-20">
        <div className="mx-auto grid max-w-5xl gap-8 md:grid-cols-2">
          <div className="rounded-2xl border border-slate-800 bg-[#090d16] p-8">
            <Container className="h-6 w-6 text-cyan-400" />
            <h3 className="mt-4 text-xl font-bold text-white">Enterprise &amp; Sovereign AI</h3>
            <p className="mt-2 text-sm text-slate-400">Partner with us for AES-GCM block encryption, Kubernetes CSI drivers, and direct VRAM streaming.</p>
            <a href="https://github.com/dwayne-dcosta/ghostweights" target="_blank" rel="noreferrer" className="mt-6 inline-block rounded-lg border border-cyan-500/40 bg-cyan-500/10 px-4 py-2 text-sm font-medium text-cyan-300">Request Design Partnership</a>
          </div>
          <div className="rounded-2xl border border-slate-800 bg-[#090d16] p-8">
            <Heart className="h-6 w-6 text-rose-400" />
            <h3 className="mt-4 text-xl font-bold text-white">Sponsor Open-Source</h3>
            <p className="mt-2 text-sm text-slate-400">MIT-licensed and free for developers. Support ongoing development on GitHub Sponsors.</p>
            <a href="https://github.com/dwayne-dcosta/ghostweights" target="_blank" rel="noreferrer" className="mt-6 inline-block rounded-lg border border-slate-700 bg-slate-800/80 px-4 py-2 text-sm font-medium text-slate-200">Sponsor on GitHub</a>
          </div>
        </div>
      </section>

      <footer className="border-t border-slate-800/80 py-8 text-center text-xs text-slate-500">
        Built by Dwayne D&apos;costa • MIT License • Zero-Persistence Edge AI Infrastructure
      </footer>
    </div>
  );
}
