# Canary Transcription Optimization Report

## Objective
Investigate and implement optimizations to speed up NVIDIA Canary transcription on Apple Silicon (Mac M1/M2/M3).

## Findings

### 1. PyTorch Installation Issue
The existing `pyproject.toml` contained a configuration that explicitly forced the installation of `pytorch-cpu` on macOS (`sys_platform == 'darwin'`).

```toml
torch = [
    { index = "pytorch-cpu", marker = "sys_platform == 'darwin'" },
    ...
]
```

This prevents the installation of the standard PyTorch wheel from PyPI, which includes support for the Metal Performance Shaders (MPS) backend required for GPU acceleration on Apple Silicon.

**Action Taken:** Modified `internal/transcription/adapters/py/nvidia/pyproject.toml` to remove this constraint, allowing `uv` to install the correct GPU-accelerated version of PyTorch for macOS.

### 2. Lack of Explicit MPS Device Support
The `canary_transcribe.py` script relied on default device placement, which typically defaults to CPU unless CUDA is available. It did not check for or utilize the `mps` device available on macOS.

**Action Taken:** Updated `internal/transcription/adapters/py/nvidia/canary_transcribe.py` to:
*   Detect if `torch.backends.mps.is_available()` is true.
*   Select `mps` device if available (and CUDA is not).
*   Pass `map_location=device` when loading the model to ensure tensors are allocated on the correct device.
*   Explicitly move the model to the device using `asr_model.to(device)`.
*   Set `PYTORCH_ENABLE_MPS_FALLBACK=1` when MPS is used, as recommended by NVIDIA NeMo documentation to handle operations not yet implemented on MPS.

### 3. Verification
A test script `test_canary_mock.py` was created to simulate the environment. It verified that:
*   When `torch.backends.mps.is_available()` is mocked to return `True`, the script selects "MPS" device.
*   When it returns `False`, it falls back to "CPU".

### 4. Profiling
Integrated `torch.profiler` profiling to enable detailed performance analysis.
*   **Usage:** Run the script with `--profile` to generate a Chrome trace file (default: `trace.json`).
*   **Visualization:** Open `chrome://tracing` (or `edge://tracing`) in a Chrome/Edge browser and load the JSON file to inspect CPU/GPU timeline execution, operation duration, and bottlenecks.

## Further Recommendations

### Half-Precision (FP16/BF16)
While MPS supports Float16, it can sometimes be numerically unstable depending on the model layers. The current change uses default precision (Float32). If further speedup is needed, one could try converting the model to half precision:

```python
if device_type == "mps":
    asr_model = asr_model.half()
```

However, this requires verification that the Canary model (which is an Encoder-Decoder model) produces accurate results in FP16 on MPS.

### Batch Processing
The current script processes a single audio file at a time. If the use case involves processing many small files, batching them into a single `transcribe` call (providing a list of paths) would significantly improve throughput.

### torch.compile
Usage of `torch.compile` (introduced in PyTorch 2.0) on Apple Silicon (MPS backend) is currently experimental and not recommended for this specific use case.

**Detailed Technical Context:**
*   **Backend Limitations:** The default `inductor` backend, which provides the most significant speedups on NVIDIA GPUs (via Triton code generation), does not yet natively support MPS. To use compilation on Mac, one must use the `aot_eager` backend (`torch.compile(model, backend="aot_eager")`), which offers minimal to no performance gain compared to standard eager mode.
*   **NeMo Compatibility:** Complex Encoder-Decoder models like Canary often utilize dynamic control flow and shapes (e.g., varying audio lengths, beam search decoding). These are historically challenging for graph capture mechanisms (TorchDynamo) to optimize effectively without significant "graph breaks," which can negate performance gains or cause crashes.
*   **Stability:** Current community reports and issues indicate frequent compilation failures (`BackendCompilerFailed`) on MPS when attempting to compile complex neural network graphs.

**Recommendation:** Stick to standard Eager Mode with MPS acceleration (as implemented). Re-evaluate `torch.compile` support when PyTorch releases stable MPS support for the `inductor` backend or a specialized Metal backend.
