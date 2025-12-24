## 2025-12-15 - [Batch Embedding Optimization]
**Learning:** Found an inconsistency where `ChromemStore.AddDocuments` was generating embeddings sequentially in a loop, whereas `QdrantStore` was using the batched `EmbedDocuments` method. This caused unnecessary network/computation overhead for local vector storage.
**Action:** Always check all implementations of an interface (like `Store`) to ensure performance optimizations (like batching) are applied consistently across all providers.
