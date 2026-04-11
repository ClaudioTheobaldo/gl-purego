# gl-purego

CGO-less OpenGL bindings for Go — a drop-in replacement for [`github.com/go-gl/gl`](https://github.com/go-gl/gl).

Uses [`github.com/ebitengine/purego`](https://github.com/ebitengine/purego) for dynamic symbol loading instead of CGO, which means:

- **No C compiler required** at build time
- **Cross-compilation works out of the box** (`CGO_ENABLED=0`)
- **Truly static binaries** on Linux

## Supported platforms

| Platform | Backend | Status |
|----------|---------|--------|
| Windows (amd64, arm64) | `opengl32.dll` via `syscall` | ✅ |
| macOS (amd64, arm64) | `OpenGL.framework` via purego | ✅ |
| Linux (amd64, arm64) | `libGL.so` + GLX via purego | ✅ |

## Supported GL versions

All packages live in the root module (`github.com/ClaudioTheobaldo/gl-purego`).

| Import path | API | Functions | Constants | Notes |
|-------------|-----|-----------|-----------|-------|
| `v2.1/gl` | OpenGL 2.1 | 568 | 864 | Full legacy fixed-function pipeline included |
| `v3.3/gl` | OpenGL 3.3 core | 345 | 818 | Deprecated fixed-function removed |
| `v4.1/gl` | OpenGL 4.1 core | 478 | 930 | macOS maximum; recommended for cross-platform |
| `v4.6/gl` | OpenGL 4.6 core | 656 | 1363 | Latest; DSA, SPIR-V, compute |

> The v3.3 package has *fewer* functions than v2.1 — that is correct. The core profile
> drops ~200 deprecated fixed-function commands (`glBegin`/`glEnd`, `glColor*`, `glVertex*`,
> immediate-mode evaluators, etc.) that were part of the old pipeline.

## Usage

```go
import gl "github.com/ClaudioTheobaldo/gl-purego/v3.3/gl"

// After creating an OpenGL context:
if err := gl.Init(); err != nil {
    log.Fatal(err)
}

// Or supply your own proc-address resolver (e.g. from glfw-purego):
gl.InitWithProcAddrFunc(func(name string) unsafe.Pointer {
    return window.GetProcAddress(name)
})
```

## Drop-in replacement for go-gl

```go
// Before
import "github.com/go-gl/gl/v3.3-core/gl"

// After
import gl "github.com/ClaudioTheobaldo/gl-purego/v3.3/gl"
```

## Code generation (`cmd/glgen`)

All binding code (`package.go` and `init.go` in every version package) is generated
from the official [Khronos OpenGL XML registry](https://github.com/KhronosGroup/OpenGL-Registry/blob/main/xml/gl.xml).
**Do not edit these files by hand.**

### Running the generator

```bash
# From the repo root — generates (or regenerates) a specific version:
go run ./cmd/glgen/ -ver 2.1 -out v2.1/gl
go run ./cmd/glgen/ -ver 3.3 -out v3.3/gl
go run ./cmd/glgen/ -ver 4.1 -out v4.1/gl
go run ./cmd/glgen/ -ver 4.6 -out v4.6/gl
```

`gl.xml` is downloaded from Khronos on the first run and cached at
`cmd/glgen/gl.xml` for subsequent runs.

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-ver` | `2.1` | Maximum GL version to include (e.g. `3.3`, `4.6`) |
| `-out` | `v2.1/gl` | Output directory for `package.go` and `init.go` |
| `-xml` | *(auto)* | Path to a local `gl.xml` (skips download) |

### Adding a new version

1. Create the version directory and copy the static files from an existing version:
   ```bash
   mkdir -p vX.Y/gl
   cp v4.6/gl/{conversions,texture,procaddr_windows,procaddr_linux,procaddr_darwin}.go vX.Y/gl/
   ```
2. Write a `vX.Y/gl/doc.go` with the package comment.
3. Run the generator: `go run ./cmd/glgen/ -ver X.Y -out vX.Y/gl`

### What the generator does

- Reads `gl.xml` and walks every `<feature api="gl" number="X.Y">` element for versions ≤ target
- Applies `<remove profile="core">` sections — this is what strips the deprecated
  fixed-function pipeline for the 3.x/4.x core profile packages
- Maps C types to Go types (`GLenum`→`uint32`, `const GLvoid *`→`unsafe.Pointer`, etc.)
- Escapes Go keyword conflicts (`type`→`xtype`, `near`→`zNear`, `string`→`xstring`, …)
- Marks functions as `required=true` if they are part of the core spec for that version,
  or `required=false` for extras included for compatibility
  (e.g. VAO/FBO are `required=false` in v2.1 but `required=true` in v3.3+)
- Emits `package.go` (constants + wrapper functions) and `init.go` (loader + function pointer vars)

## Acknowledgements

This repository was built in collaboration with [Claude Code](https://claude.ai/claude-code) (Anthropic Claude Sonnet 4.6).
