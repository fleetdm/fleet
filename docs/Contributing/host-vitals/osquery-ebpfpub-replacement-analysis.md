# osquery ebpfpub replacement analysis

## Critical sequencing: BPF must be fixed first

**The toolchain update CANNOT happen until BPF is fixed:**
1. **Current state**: Stuck on LLVM 9 because ebpfpub uses LLVM 9 APIs
2. **Blocker**: ebpfpub won't compile with LLVM 14+
3. **Solution**: Replace ebpfpub with libbpf (which doesn't depend on LLVM APIs)
4. **Then**: Update toolchain to LLVM 14+
5. **Result**: Can build with Mac SDK 15

## Current state analysis

### What ebpfpub currently provides
1. **Process Events** (`bpf_process_events` table)
   - exec, fork, exit syscalls
   - Process tree tracking
   - Command line arguments

2. **Network Events** (`bpf_socket_events` table)
   - connect, accept, bind syscalls
   - Socket state tracking
   - Connection metadata

### Why it must be replaced
- Blocks LLVM upgrade (9 → 14+) needed for SDK 15
- Uses deprecated LLVM IR APIs
- Performance issues (drops events, high CPU usage)
- Unmaintained codebase

## Option 1: libclang + BTF

### Architecture
```
Source: BPF C code → libclang API → Compile at runtime → Load into kernel
                         ↓
                    BTF for portability
```

### Implementation Approach
**Runtime Compilation**
  - Ship BPF source code with osquery
  - Use libclang API to compile at startup
  - Leverage BTF for kernel structure compatibility

### Pros
- Full control over compilation process
- Can optimize for specific kernel at runtime
- Similar to BCC (BPF Compiler Collection) approach (well-tested pattern)
- No pre-compilation needed

### Cons
- **Large dependency** (libclang is ~100MB)
- Slower startup (compilation overhead)
- Complex error handling
- Requires clang headers at runtime

### Effort estimate
- Large effort
- Why larger than libbpf:
  - **Cannot reuse ebpfpub's IR generation code** - ebpfpub uses internal LLVM APIs, libclang uses different API layer
  - **Must write new compiler integration** - Runtime compilation infrastructure from scratch
  - **Complex error handling** - Runtime compilation failures are harder to debug and recover from
  - **Build system complexity** - libclang dependency management is difficult (~100MB dependency)
  - **Runtime debugging** - Harder to debug BPF programs that fail to compile at runtime
- Can still reuse: ProcessContextFactory, filesystem helpers, and other osquery utilities
- Complex build system changes
- Extensive testing required

## Option 2: libbpf + Precompiled Probes (Recommended)

### Key Concepts
- **BPF Programs**: C code that runs in the kernel to monitor events
- **Skeletons**: Auto-generated C++ headers that provide easy APIs to load BPF programs
- **CO-RE**: "Compile Once, Run Everywhere" - BPF programs that work across kernel versions

### Architecture
```
Build time: BPF C code → clang -target bpf → .bpf.o files
                              ↓
                         BTF embedded
Runtime: Load .bpf.o → libbpf CO-RE → Kernel
```

### Pros
- **Lightweight runtime** (~200KB libbpf)
- Fast startup (no compilation)
- Industry standard (used by Facebook, Netflix, Cloudflare)
- Excellent tooling (bpftool, libbpf-bootstrap)
- CO-RE ensures portability

### Cons
- Requires build-time BPF compilation
- Need to ship .bpf.o files
- Minimum kernel 4.18+ (BTF requirement)

### Effort estimate
- Medium effort
- Why faster than libclang+BTF:
  - **Clean separation** - BPF C code is completely separate from osquery C++ code
  - **Mature tooling** - bpftool, libbpf-bootstrap provide working templates
  - **Build-time errors** - Compilation errors caught during build, not at runtime
  - **Simple integration** - Skeleton headers provide clean C++ API
  - **Can reuse osquery utilities** - ProcessContextFactory, filesystem helpers, etc.
- Straightforward implementation
- Good examples available (see libbpf-bootstrap, BCC libbpf-tools)

## Implementation steps (libbpf approach)

### Critical dependency chain
```
Mac SDK 15 Build Fix
    └── Requires: Boost Update
        └── Requires: C++20 Support
            └── Requires: Linux Toolchain Update (LLVM 9 → 14+)
                └── BLOCKED BY: ebpfpub using LLVM 9 APIs
                    └── SOLUTION: Replace ebpfpub with libbpf
```

We will keep ebpfpub working during the replacement to do A/B testing.

### Phase 1: Core infrastructure
1. Add libbpf alongside ebpfpub:
   - Keep "Linux:ebpfpub" in CMakeLists.txt
   - Add "Linux:libbpf" as additional library
   - Both libraries coexist during development
2. Plan for shared utilities:
   - Will reuse many existing utilities from osquery/events/linux/bpf
3. Add libbpf to build system
   - Create `/libraries/cmake/source/modules/Findlibbpf.cmake`:
     ```cmake
     # Find libbpf using pkg-config
     find_package(PkgConfig REQUIRED)
     pkg_check_modules(LIBBPF REQUIRED libbpf>=1.6.2) # or whatever the latest is

     # Also find bpftool for skeleton generation
     find_program(BPFTOOL bpftool REQUIRED)
     ```
   - Add libbpf to `/CMakeLists.txt` library list ("Linux:libbpf")
   - Update `/osquery/events/linux/bpf/CMakeLists.txt` to use LIBBPF variables
4. Set up BPF build infrastructure
   - Configure CMake to compile .bpf.c → .bpf.o using clang -target bpf
   - Set up bpftool for skeleton generation (.bpf.o → .skel.h)
   - Create CMake rules for BPF program compilation
   - Reference examples: https://github.com/libbpf/libbpf-bootstrap

### Phase 2: Process events
1. Write process tracking BPF programs
   - Create `bpf_process_events.bpf.c` with tracepoints for:
     - execve, execveat (process execution)
     - fork, vfork, clone (process creation)
     - exit, exit_group (process termination)
   - Use ring buffer for efficient event passing
2. Create new publisher using libbpf (parallel to bpfeventpublisher.cpp)
   - New file: `bpf_process_events_v2_publisher.cpp`
   - Registers table `bpf_process_events_v2`
3. **Reuse existing utilities**:
   - Use `ProcessContextFactory` to gather process info from /proc
   - Use `IFilesystem` abstraction for file operations
   - Leverage existing process context structures
4. Create ring buffer handler for BPF events
5. **Create NEW v2 table** (parallel implementation):
   - New table: `bpf_process_events_v2`
   - Same schema as `bpf_process_events` for easy comparison
   - Both tables run simultaneously for testing
6. Add unit tests

### Phase 3: Network events
1. Write socket tracking BPF programs
   - Create `bpf_socket_events.bpf.c` with tracepoints for:
     - connect, accept, accept4 (connections)
     - bind, listen (server sockets)
   - Track both IPv4 and IPv6
2. Handle multiple network namespaces
3. **Map to existing table schema** (same approach as process events)
   - Maintain `bpf_socket_events` table compatibility
   - Keep all existing columns
4. Add integration tests

### Phase 4: Testing and polish
1. Test on different Linux kernels/OSes
2. Test in Docker containers
3. Add documentation

### Phase 5: Remove ebpfpub
1. **Remove ebpfpub** usage/mentions.
2. **Rename new tables** to production names:
   - `bpf_process_events_v2` → `bpf_process_events`
   - `bpf_socket_events_v2` → `bpf_socket_events`

## Recommendation

**Choose libbpf + precompiled probes** because:

1. **Proven Technology**: Used by major companies in production
2. **Better Performance**: Lower runtime overhead
3. **Easier Maintenance**: Standard tooling and patterns
4. **Faster Development**: Quicker to implement
5. **Smaller Footprint**: ~200KB vs ~100MB dependency
