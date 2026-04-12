# TODO

## Medium

- ~~**GL 3.3 example**~~ ✅ `examples/15_instancing` — 10×10 wave of cubes, single instanced draw call

- ~~**GL 4.x example**~~ ✅ `examples/16_compute_julia` — animated Julia set via compute shader (GL 4.3)
  - Remaining 4.x topics for future examples:
    - Direct State Access (DSA) — `glNamedBufferData`, `glVertexArrayAttribBinding`, etc.
    - Geometry shader — normal visualisation or shadow volumes

- ~~**`go generate` regression test**~~ ✅ `cmd/glgen/generate_test.go` — byte-for-byte stability check for all four GL versions

## Bigger

- ~~**GLES bindings**~~ ✅ `gles2/v3.0/gl` and `gles2/v3.1/gl` — generated via `go run ./cmd/glgen/ -api gles2 -ver X.Y`
  - Remaining: `gles2/v3.2/gl` (adds geometry/tessellation shaders to GLES)
  - Remaining: Android procaddr (`dlopen("libGLESv2.so")` without EGL)

- **GL 3.3 / 4.x init tests** — same mock-resolver pattern as `v2.1/gl/init_test.go`
  but with the correct optional set for each version (v3.3+ has almost no optional
  extras; everything in optionalFuncs is core-required there)
