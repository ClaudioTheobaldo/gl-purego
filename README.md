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
| Linux GLES | `libGL.so` + EGL via purego | 🚧 |

## Supported GL versions

| Package | API |
|---------|-----|
| `v2.1/gl` | OpenGL 2.1 |
| `v3.3-core/gl` | OpenGL 3.3 Core |
| `v4.1-core/gl` | OpenGL 4.1 Core |
| `v4.6-core/gl` | OpenGL 4.6 Core |
| `v3.0/gles2` | OpenGL ES 2.0 |
| `v3.1/gles2` | OpenGL ES 3.1 |

> Additional versions generated from the Khronos OpenGL XML registry via `cmd/glgen`.

## Usage

```go
import "github.com/ClaudioTheobaldo/gl-purego/v2.1/gl"

// After creating an OpenGL context:
if err := gl.Init(); err != nil {
    log.Fatal(err)
}

// Or supply your own proc-address resolver (e.g. from glfw-purego):
gl.InitWithProcAddrFunc(func(name string) unsafe.Pointer {
    return window.GetProcAddress(name)
})
```

The rest of the API is identical to `github.com/go-gl/gl/v2.1/gl`.

## Drop-in replacement

```go
// Before
import "github.com/go-gl/gl/v2.1/gl"

// After
import "github.com/ClaudioTheobaldo/gl-purego/v2.1/gl"
```

Or via `go.mod` replace directive:

```
replace github.com/go-gl/gl => github.com/ClaudioTheobaldo/gl-purego
```

## Acknowledgements

This repository was built in collaboration with [Claude Code](https://claude.ai/claude-code) (Anthropic Claude Sonnet 4.6).
