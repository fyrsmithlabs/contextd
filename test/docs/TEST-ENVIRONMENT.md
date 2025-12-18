# Test Environment Architecture

This document details the test environment used for persona simulation.

---

## Container Architecture

```mermaid
graph TB
    subgraph "Host System"
        H1[contextd Repository]
        H2[make build-all]
        H3[Docker Engine]
    end

    subgraph "Build Artifacts"
        B1[contextd binary<br/>36MB, CGO enabled]
        B2[ctxd binary<br/>29MB]
    end

    subgraph "Docker Image: contextd-user-sim"
        subgraph "Base Layer"
            D1[Ubuntu 22.04]
            D2[apt packages]
        end

        subgraph "User Layer"
            D3[developer user]
            D4[sudo access]
        end

        subgraph "Application Layer"
            D5[~/.local/bin/contextd]
            D6[~/.local/bin/ctxd]
            D7[~/contextd-docs/]
        end

        subgraph "Test Data Layer"
            D8[~/projects/my-app/]
            D9[Git initialized]
        end
    end

    H1 --> H2
    H2 --> B1
    H2 --> B2
    B1 --> D5
    B2 --> D6
    H3 --> D1

    style B1 fill:#c8e6c9
    style B2 fill:#c8e6c9
    style D5 fill:#bbdefb
    style D6 fill:#bbdefb
```

---

## Dockerfile Breakdown

```mermaid
flowchart TD
    subgraph "Stage 1: Base Image"
        A1[FROM ubuntu:22.04]
        A2[ENV DEBIAN_FRONTEND=noninteractive]
    end

    subgraph "Stage 2: System Packages"
        B1[apt-get update]
        B2[Install curl, wget, git]
        B3[Install jq, sudo, ca-certificates]
        B4[Install gnupg, vim, build-essential]
        B5[Cleanup apt cache]
    end

    subgraph "Stage 3: User Setup"
        C1[useradd -m developer]
        C2[Add to sudoers NOPASSWD]
        C3[Switch to developer user]
        C4[Set WORKDIR /home/developer]
    end

    subgraph "Stage 4: Environment"
        D1[mkdir ~/.local/bin]
        D2[mkdir ~/.config]
        D3[mkdir ~/.claude]
        D4[Update PATH]
    end

    subgraph "Stage 5: Application"
        E1[COPY contextd]
        E2[COPY ctxd]
        E3[COPY README.md]
        E4[COPY docs/]
        E5[chmod +x binaries]
    end

    subgraph "Stage 6: Test Data"
        F1[mkdir ~/projects/my-app]
        F2[git init]
        F3[git config user]
        F4[Create README.md]
        F5[Initial commit]
    end

    A1 --> A2 --> B1 --> B2 --> B3 --> B4 --> B5
    B5 --> C1 --> C2 --> C3 --> C4
    C4 --> D1 --> D2 --> D3 --> D4
    D4 --> E1 --> E2 --> E3 --> E4 --> E5
    E5 --> F1 --> F2 --> F3 --> F4 --> F5
```

---

## Directory Structure

```mermaid
graph TD
    subgraph "Container Filesystem"
        ROOT["/"]

        ROOT --> HOME["/home/developer/"]

        HOME --> LOCAL[".local/"]
        HOME --> CONFIG[".config/"]
        HOME --> CLAUDE[".claude/"]
        HOME --> PROJECTS["projects/"]
        HOME --> DOCS["contextd-docs/"]

        LOCAL --> BIN["bin/"]
        BIN --> CONTEXTD["contextd ⚡"]
        BIN --> CTXD["ctxd ⚡"]

        CONFIG --> CONTEXTD_CFG["contextd/"]
        CONTEXTD_CFG --> VECTORSTORE["vectorstore/"]
        CONTEXTD_CFG --> LIB["lib/"]
        CONTEXTD_CFG --> CFG_YAML["config.yaml"]

        CLAUDE --> SETTINGS["settings.json"]

        PROJECTS --> MYAPP["my-app/"]
        MYAPP --> DOTGIT[".git/"]
        MYAPP --> APP_README["README.md"]

        DOCS --> DOC_README["README.md"]
        DOCS --> DOC_DOCS["docs/"]
    end

    style CONTEXTD fill:#4caf50,color:#fff
    style CTXD fill:#4caf50,color:#fff
    style SETTINGS fill:#ff9800,color:#fff
    style VECTORSTORE fill:#2196f3,color:#fff
```

---

## Runtime Data Flow

```mermaid
sequenceDiagram
    participant U as User/Persona
    participant S as Shell
    participant C as contextd
    participant E as FastEmbed
    participant V as Vectorstore
    participant F as Filesystem

    Note over U,F: First Run - Auto Download

    U->>S: contextd --mcp --no-http
    S->>C: Start process
    C->>C: Load config
    C->>E: Initialize embeddings

    E->>F: Check ~/.config/contextd/lib/
    F-->>E: ONNX not found

    E->>E: Download ONNX runtime
    E->>F: Save libonnxruntime.so
    E->>E: Download embedding model
    E->>F: Save model files

    E-->>C: Embeddings ready
    C->>V: Initialize chromem
    V->>F: Create vectorstore/
    V-->>C: Store ready

    C->>C: Register MCP tools
    C-->>S: Server running
    S-->>U: Ready for input
```

---

## Network Isolation

```mermaid
graph LR
    subgraph "Container Network"
        C[contextd]
        C -->|localhost:9090| H[HTTP Server]
        C -->|stdio| M[MCP Protocol]
    end

    subgraph "External Network"
        I[Internet]
        O[ONNX Downloads]
        E[Embedding Models]
    end

    subgraph "First Run Only"
        C -.->|HTTPS| O
        C -.->|HTTPS| E
    end

    Note1[/"No external calls<br/>after first run"/]

    style Note1 fill:#e8f5e9
```

---

## Resource Requirements

```mermaid
graph TD
    subgraph "Disk Space"
        D1[Ubuntu Base: ~75MB]
        D2[Dev Packages: ~300MB]
        D3[contextd binary: 36MB]
        D4[ctxd binary: 29MB]
        D5[ONNX Runtime: ~50MB]
        D6[Embedding Model: ~50MB]
        D7[Total: ~540MB]
    end

    subgraph "Memory"
        M1[Base Container: ~50MB]
        M2[contextd Running: ~100MB]
        M3[Embedding Operations: ~200MB peak]
        M4[Total Peak: ~350MB]
    end

    subgraph "CPU"
        C1[Build: Multi-core beneficial]
        C2[Runtime: Single core sufficient]
        C3[Embedding: Brief CPU spikes]
    end

    style D7 fill:#ff9800,color:#fff
    style M4 fill:#ff9800,color:#fff
```

---

## Build Commands

```bash
# From repository root
cd /home/dahendel/contextd

# Build binaries
make build-all

# Copy to test directory
cp contextd ctxd test/persona-simulation/
cp README.md test/persona-simulation/
cp -r docs test/persona-simulation/

# Build Docker image
cd test/persona-simulation
docker build -t contextd-user-sim -f Dockerfile.user-sim .

# Run container interactively
docker run -it --rm contextd-user-sim

# Run specific test
docker run --rm contextd-user-sim contextd --version

# Run with environment override
docker run --rm -e VECTORSTORE_PROVIDER=invalid contextd-user-sim contextd --mcp --no-http
```

---

## Verification Commands

```mermaid
flowchart LR
    subgraph "Binary Verification"
        V1[contextd --version]
        V2[ctxd --version]
    end

    subgraph "Expected Output"
        O1["contextd v0.3.0-rc7..."]
        O2["ctxd v0.3.0-rc7..."]
    end

    subgraph "Startup Verification"
        V3["contextd --mcp --no-http"]
        O3["MCP server initialized..."]
    end

    V1 --> O1
    V2 --> O2
    V3 --> O3

    style O1 fill:#c8e6c9
    style O2 fill:#c8e6c9
    style O3 fill:#c8e6c9
```
