# Analysis: Slow Environment Startup in Scriberr

## Problem Statement
The application startup time is significantly delayed (by approximately 8.5 seconds) due to environment readiness checks performed by model adapters. These checks verify that the Python virtual environments are correctly set up and that necessary libraries can be imported before processing starts.

## Measurements
Measured on current hardware using `uv run` for various environment checks:

| Check / Adapter | Command | Duration |
|---|-|---|
| **NVIDIA (NeMo)** | `python -c "import nemo.collections.asr"` | **~8.3s** |
| **WhisperX** | `python -c "import whisperx"` | ~0.9s |
| **PyAnnote** | `from pyannote.audio import Pipeline` | ~2.5s |
| **Baseline** | `uv run ... python --version` | ~0.6s |

## Root Cause
The `CheckEnvironmentReady` function in `internal/transcription/adapters/base_adapter.go` executes a full Python interpreter launch and imports heavy libraries (like `torch` and `nemo`) to verify the environment. 

While these checks are parallelized in `ModelRegistry.InitializeModels`, the total startup delay is governed by the slowest component (NVIDIA adapters), leading to a mandatory ~8s wait every time the server starts.

## Environment Variable Tuning
We tested several environment variables to see if they could suppress slow initialization logic (like CUDA discovery or network checks):

| Environment Variables | Effect on NeMo Import | Result |
|---|---|---|
| `TRANSFORMERS_OFFLINE=1` `HF_HUB_OFFLINE=1` | Skip network checks for models | No significant change (~8.2s) |
| `CUDA_VISIBLE_DEVICES=""` | Skip CUDA/GPU discovery and initialization | **-1.1s** (~7.1s) |
| `NEMO_DISABLE_IMPORT_CHECKS=1` | Skip internal NeMo dependency verification | **-1.5s** (~6.7s) |

**Conclusion**: While disabling CUDA and import checks helps, the core overhead remains high due to the sheer size of the `nemo.collections.asr` and `torch` submodules.

## Proposed Optimizations

### 1. Sentinel File (Recommended)
Instead of running a Python command, the system can create a sentinel file (e.g., `.scriberr_ready`) inside the environment directory after a successful `uv sync` and initial model download.
- **Benefit**: Reduces check time to nearly 0ms.
- **Implementation**: Adapters check for the file's existence. If missing, they run the full `PrepareEnvironment` logic and create the file upon success.

### 2. Lighter Check (Implemented)
Change the `importStatement` check to a simple top-level package import (e.g., `import nemo` instead of `import nemo.collections.asr`). 
- **Benefit**: Reduces check time from 8.3s to **~0.6s**.
- **Reasoning**: Top-level packages in these libraries often have very light `__init__.py` files that don't trigger the loading of heavy submodules like Torch or CUDA.

### 3. Asynchronous Initialization (Implemented)
Ensured `InitializeModels` runs in the background and does not block the main API from becoming ready. 
- **Effect**: Server startup is now instantaneous (from the Go perspective), while models continue to prepare their environments in the background.
- **Verification**: `registry.InitializeModels` now returns immediately after launching background goroutines.

## Progress Made
- **Optimized Readiness Checks**: Updated NVIDIA, Sortformer, and PyAnnote adapters to use top-level package imports, reducing initialization time by ~14x.
- **Background Initialization**: Refactored `ModelRegistry` to perform environment preparation asynchronously.
- **Added Timing Logs**: `base_adapter.go` now logs the duration of every environment check.
- **Added Regression Test**: `TestParakeetPrepareEnvironment` in `internal/transcription/adapters_test.go` can be used to monitor startup performance.

