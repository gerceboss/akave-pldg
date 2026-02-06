# Project Plan: PyTorch + Akave O3 Integration

**Status**: Planning  
**Contributor**: Aryan Jain  
**Date**: February 2026

---

## 1. Technical Architecture

The integration leverages the **Akave Python SDK (`akavesdk-py`)** to provide a high-performance, native interface for PyTorch training pipelines. This replaces the generic S3 wrapper with a protocol-native implementation.

### 1.1 Core Components

#### 1. `O3Client` (SDK Adapter & Resiliency Layer)
A wrapper around `akavesdk.SDK` that manages session persistence, authentication, and **fault tolerance**.
- **Responsibilities**:
  - Initializes `SDKConfig` with `AKAVE_PRIVATE_KEY` and node endpoints.
  - **Automatic Retries**: Implements exponential backoff for `IPC` and `Streaming` calls to handle transient network limits/failures.
  - Exposes `sdk.ipc()` and `sdk.streaming_api()`.
- **Why**: Centralizes session management and protects the training loop from fragile network conditions.

#### 2. `O3Dataset` (High-Performance Data Loading)
A custom PyTorch `Dataset` fully optimizing Akave's **Streaming API**.
- **Inherits**: `torch.utils.data.Dataset`.
- **Responsibilities**:
  - **Smart Caching**:
    - *Metadata*: Caches bucket file lists locally to avoid repeated blockchain lookups on initialization.
    - *Data*: Persists downloaded chunks (LRU) to reduce bandwidth usage across epochs.
  - **Parallelism**: Exposes `max_concurrency` config to tune download threads matches CPU cores.
  - **Lazy Loading**: Uses `create_range_file_download` to fetch exact byte ranges on-demand.
- **Signature**: `class O3Dataset(bucket_name, prefix, sdk_instance, cache_dir, concurrency=4)`

#### 3. `O3CheckpointManager` (Persistence Layer)
Handles model state serialization and storage using Akave's IPC interface.
- **Responsibilities**:
  - `save_checkpoint(model, optimizer, epoch)`: Uses `ipc().upload()` to store model buffers.
  - `load_checkpoint(run_id, epoch)`: Uses `ipc().download()` to restore training state.
  - **Metadata tracking**: Automatically tags uploads with `root_cid` for version integrity verification.

### 1.2 Data Flow
1.  **Init**: `O3Client` connects; `O3Dataset` fetches bucket listing and caches it locally (`metadata.json`).
2.  **Training**:
    - `DataLoader` worker requests index `i`.
    - `O3Dataset` checks local chunk cache -> if missing, uses `StreamingAPI` (w/ Retry) -> loads Tensor.
3.  **Persistence**:
    - At checkpoint interval, model is serialized.
    - `O3CheckpointManager` uploads via IPC and logs the resulting CID.

---

## 2. Implementation Milestones

### Milestone 1: Setup & Native SDK Integration
**Goal**: Establish environment and verify IPC connectivity.
- [ ] Initialize Python environment and `requirements.txt` (incl. `akavesdk`).
- [ ] Implement `O3Client` setup and connection validation script.
- [ ] Verify bucket creation and listing via `sdk.ipc()`.

### Milestone 2: O3Dataset with Range-Streaming
**Goal**: Enable partial file downloads for efficient data loading.
- [ ] Implement `O3Dataset` using `sdk.streaming_api()`.
- [ ] Implement chunk-level local caching.
- [ ] Benchmark data loading speed against standard local loading.

### Milestone 3: Checkpoint & Recovery
**Goal**: Implement decentralized state persistence.
- [ ] Implement `O3CheckpointManager` with `ipc` upload/download logic.
- [ ] Add auto-resume functionality based on the latest uploaded CID in a bucket.
- [ ] Verify model weight consistency after an O3 round-trip.

### Milestone 4: End-to-End Reference & Docs
**Goal**: Finalize deliverables and user guidance.
- [ ] Create `examples/train_mnist.py` using the full O3 stack.
- [ ] Document `AKAVE_PRIVATE_KEY` and node endpoint configuration.
- [ ] Verify training recovery after simulated process termination.

## 3. Success Criteria
- **Native performance**: Range-streaming minimizes time-to-first-batch.
- **Verifiability**: Every checkpoint should be uniquely identified by its Akave CID.
- **Usability**: Drop-in compatibility with standard PyTorch `DataLoader`.
