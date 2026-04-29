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
| 15 | `15_instancing` | GL 3.3 core | 100 cubes in one draw call via `glDrawElementsInstanced` + `glVertexAttribDivisor` |
| 16 | `16_compute_julia` | GL 4.3 | Animated Julia set fractal computed entirely on the GPU via compute shader + `imageStore` |
| 17 | `17_gles_triangle` | GLES 3.0 | Coloured triangle via the new EGL backend (ANGLE); first OpenGL ES example |
| 18 | `18_gles_textures` | GLES 3.0 | Textured quad — checkerboard generated in Go, `GL_TEXTURE_2D`, `sampler2D` |
| 19 | `19_gles_compute`  | GLES 3.1 | Compute shader writes UV-gradient into a 512×512 image, fullscreen display |
| 20 | `20_gles32_geometry` | GLES 3.2 | Geometry shader emits normal-visualisation lines per triangle edge |

> Examples 17–20 require [ANGLE](https://chromium.googlesource.com/angle/angle)
> (`libGLESv2.dll` + `libEGL.dll` + `vulkan-1.dll` + `vk_swiftshader.dll`) alongside the executable.
> Examples 17–18 also work with browser-shipped ANGLE (copy from Brave/Chrome —
> see `examples/17_gles_triangle/copy_angle.ps1`).
>
> Examples 19 and 20 require a **Vulkan-enabled ANGLE build** (GLES 3.1 / 3.2).
> ANGLE's D3D11 backend is hardcapped at GLES 3.0 regardless of GPU; the Vulkan
> backend lifts this limit. Both examples automatically set
> `ANGLE_DEFAULT_PLATFORM=vulkan` and exit with a clear message if GLES 3.1/3.2
> is unavailable. Build ANGLE from source with `angle_enable_vulkan=true`, or run
> on Linux with Mesa.

## Controls (camera examples)

| Key | Action |
|-----|--------|
| `W` `A` `S` `D` | Move forward / left / back / right |
| `E` / `Q` | Move up / down |
| Hold **RMB** | Look around |
| Mouse wheel | Adjust movement speed |
| `ESC` | Quit |
