# Examples

Each example is a standalone Go module. All examples are Windows-only for now
(the `//go:build windows` constraint); Linux/macOS support tracks glfw-purego
platform progress.

## Setup

The examples share a `go.work` workspace at the repository root. Since
`go.work` is gitignored (Go tooling convention), recreate it after cloning:

```bash
cd gl-purego
go run ./scripts/setup_workspace.go
```

Or manually:

```bash
cd gl-purego
go work init .
go work use ../glfw-purego
go work use $(ls -d examples/*/ | tr '\n' ' ')
```

Then build and run any example:

```bash
cd examples/08_lighting
CGO_ENABLED=0 go run .
```

## Example list

### Core concepts (v2.1)

| # | Directory | Topic |
|---|-----------|-------|
| 01 | `01_hello_window` | Window creation, clear colour, event loop |
| 02 | `02_triangle` | First triangle: VBO, VAO, vertex shader, fragment shader |
| 03 | `03_shaders` | Uniforms: CPU↔GPU communication, animated colour |
| 04 | `04_textures` | Texture sampling, multiple texture units, blending |
| 05 | `05_transformations` | Model matrices: translate, rotate, scale |
| 06 | `06_coordinate_systems` | MVP pipeline: model / view / projection |
| 07 | `07_camera` | Free-fly first-person camera (mouse + keyboard) |
| 08 | `08_lighting` | Phong shading: ambient + diffuse + specular |
| 09 | `09_materials` | GLSL material and light structs |
| 10 | `10_light_casters` | Directional light, point light, spotlight |
| 11 | `11_depth_testing` | Depth buffer visualisation, depth functions |
| 12 | `12_stencil_testing` | Stencil buffer: object outlining |
| 13 | `13_blending` | Alpha blending, transparency sorting |
| 14 | `14_framebuffers` | Render-to-texture, post-processing effects |

### Bonus (v2.1)

| Directory | Topic |
|-----------|-------|
| `cube` | Coloured rotating cube |
| `platonic` | Animated platonic solids (tetrahedron → cube → octahedron → …) |
| `polygon` | Regular N-gon, adjustable vertex count |
| `sphere` | UV sphere with normal-derived colouring |

### Modern GL features

| # | Directory | GL | Topic |
|---|-----------|-----|-------|
| 15 | `15_instancing` | 3.3 core | 100 cubes in one draw call via `glDrawElementsInstanced` + `glVertexAttribDivisor` |
| 16 | `16_compute_julia` | 4.3 | Animated Julia set fractal computed entirely on the GPU via compute shader + `imageStore` |

## Controls (camera examples)

| Key | Action |
|-----|--------|
| `W` `A` `S` `D` | Move forward / left / back / right |
| `E` / `Q` | Move up / down |
| Hold **RMB** | Look around |
| Mouse wheel | Adjust movement speed |
| `ESC` | Quit |
