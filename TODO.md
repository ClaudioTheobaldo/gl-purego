# TODO

## Medium

- ~~**GL 3.3 example**~~ ✅ `examples/15_instancing` — 10×10 wave of cubes, single instanced draw call

- ~~**GL 4.x example**~~ ✅ `examples/16_compute_julia` — animated Julia set via compute shader (GL 4.3)
  - Remaining 4.x topics for future examples:
    - Direct State Access (DSA) — `glNamedBufferData`, `glVertexArrayAttribBinding`, etc.
    - Geometry shader — normal visualisation or shadow volumes

- **`go generate` regression test** — run `go run ./cmd/glgen/` for each version and assert
  the output is byte-for-byte identical to what is committed; catches silent regressions
  in gl.xml parsing or type-mapping logic

## Bigger

- **GLES bindings** (`v3.0/gles2`, `v3.1/gles2`, `v3.2/gles2`):
  - The generator already understands `api="gles2"` in gl.xml; needs `-api gles2` flag
  - Different procaddr files: EGL instead of WGL/GLX
  - Different platform constraints (`//go:build android || linux`)
  - Separate module or subdirectory mirroring the go-gl layout

- **GL 3.3 / 4.x init tests** — same mock-resolver pattern as `v2.1/gl/init_test.go`
  but with the correct optional set for each version (v3.3+ has almost no optional
  extras; everything in optionalFuncs is core-required there)
