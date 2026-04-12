//go:build windows

// 16_compute_julia — GL 4.3 compute shader: animated Julia set fractal.
//
// A compute shader writes every pixel of a texture each frame by iterating
// the Julia set formula z → z² + c. A fullscreen triangle then samples that
// texture — no vertex buffer, no rasterised geometry, just pure compute.
//
// Key GL 4.3 features demonstrated:
//
//   - COMPUTE_SHADER stage (glCreateShader(GL_COMPUTE_SHADER))
//   - Image store: layout(rgba8, binding=0) writeonly image2D
//   - glBindImageTexture — bind texture as read/write image unit
//   - glDispatchCompute — launch 16×16 work groups across the texture
//   - glMemoryBarrier(GL_SHADER_IMAGE_ACCESS_BARRIER_BIT) — sync before sample
//
// Requires a driver that supports OpenGL 4.3 (AMD/NVIDIA on Windows/Linux;
// use v4.1 + extensions on macOS, which does not support compute shaders).
//
// The parameter c traces a slow orbit around the complex plane, cycling
// through dozens of distinct fractal shapes every ~20 seconds.
//
// Controls:
//
//	ESC — quit
//
// Build:
//
//	CGO_ENABLED=0 go build -o 16_compute_julia.exe .
package main

import (
	"fmt"
	"log"
	"math"
	"unsafe"

	gl   "github.com/ClaudioTheobaldo/gl-purego/v4.6/gl"
	glfw "github.com/ClaudioTheobaldo/glfw-purego/v3.3/glfw"
)

// ── shaders ───────────────────────────────────────────────────────────────────

// Compute shader: one invocation per pixel, writes Julia set colour to image.
const computeSrc = `#version 430 core
layout(local_size_x = 16, local_size_y = 16) in;

// The output texture bound as an image unit (write-only, RGBA8).
layout(rgba8, binding = 0) uniform writeonly image2D uOutput;

uniform vec2 uResolution; // texture dimensions in pixels
uniform vec2 uC;          // Julia parameter c  (animated)

// Smooth, hue-cycling colouring based on the escape iteration count.
vec3 palette(float t) {
    return 0.5 + 0.5 * cos(vec3(0.0, 2.094, 4.189) + t * 6.28318);
}

void main() {
    ivec2 coord = ivec2(gl_GlobalInvocationID.xy);
    // Guard: dispatch groups are rounded up so some threads land out of bounds.
    if (coord.x >= int(uResolution.x) || coord.y >= int(uResolution.y)) return;

    // Map pixel to complex plane [-1.75, 1.75] × [-1.75, 1.75].
    vec2 z;
    z.x = (float(coord.x) / uResolution.x - 0.5) * 3.5;
    z.y = (float(coord.y) / uResolution.y - 0.5) * 3.5;

    const int maxIter = 256;
    int iter = 0;
    while (iter < maxIter && dot(z, z) < 4.0) {
        float zx = z.x * z.x - z.y * z.y + uC.x;
        z.y  = 2.0 * z.x * z.y + uC.y;
        z.x  = zx;
        iter++;
    }

    vec4 colour;
    if (iter == maxIter) {
        colour = vec4(0.0, 0.0, 0.0, 1.0); // inside the set → black
    } else {
        // Smooth colouring: remap iteration count to [0,1] and palette it.
        float t = float(iter) / float(maxIter);
        colour = vec4(palette(t), 1.0);
    }

    imageStore(uOutput, coord, colour);
}
` + "\x00"

// Vertex shader: generates a fullscreen triangle from gl_VertexID alone —
// no vertex buffer required. The three vertices cover the entire NDC clip space.
const vertSrc = `#version 430 core
out vec2 vUV;
void main() {
    // Vertices at (-1,-1), (3,-1), (-1,3) form a triangle that covers [-1,1]².
    vec2 pos = vec2(
        (gl_VertexID == 1) ?  3.0 : -1.0,
        (gl_VertexID == 2) ?  3.0 : -1.0
    );
    vUV = pos * 0.5 + 0.5;
    gl_Position = vec4(pos, 0.0, 1.0);
}
` + "\x00"

const fragSrc = `#version 430 core
in  vec2      vUV;
uniform sampler2D uTex;
out vec4      FragColor;
void main() { FragColor = texture(uTex, vUV); }
` + "\x00"

// ── shader helpers (v4.6/gl) ──────────────────────────────────────────────────

func buildComputeProgram(src string) (uint32, error) {
	sh, err := compileShader(src, gl.COMPUTE_SHADER)
	if err != nil {
		return 0, fmt.Errorf("compute: %w", err)
	}
	p := gl.CreateProgram()
	gl.AttachShader(p, sh)
	gl.LinkProgram(p)
	gl.DeleteShader(sh)
	var status int32
	gl.GetProgramiv(p, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var n int32
		gl.GetProgramiv(p, gl.INFO_LOG_LENGTH, &n)
		buf := make([]uint8, n+1)
		gl.GetProgramInfoLog(p, n, nil, &buf[0])
		gl.DeleteProgram(p)
		return 0, fmt.Errorf("link: %s", buf)
	}
	return p, nil
}

func buildProgram(vs, fs string) (uint32, error) {
	v, err := compileShader(vs, gl.VERTEX_SHADER)
	if err != nil {
		return 0, fmt.Errorf("vertex: %w", err)
	}
	f, err := compileShader(fs, gl.FRAGMENT_SHADER)
	if err != nil {
		gl.DeleteShader(v)
		return 0, fmt.Errorf("fragment: %w", err)
	}
	p := gl.CreateProgram()
	gl.AttachShader(p, v)
	gl.AttachShader(p, f)
	gl.LinkProgram(p)
	gl.DeleteShader(v)
	gl.DeleteShader(f)
	var status int32
	gl.GetProgramiv(p, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var n int32
		gl.GetProgramiv(p, gl.INFO_LOG_LENGTH, &n)
		buf := make([]uint8, n+1)
		gl.GetProgramInfoLog(p, n, nil, &buf[0])
		gl.DeleteProgram(p)
		return 0, fmt.Errorf("link: %s", buf)
	}
	return p, nil
}

func compileShader(src string, kind uint32) (uint32, error) {
	sh := gl.CreateShader(kind)
	cstr, free := gl.Strs(src)
	gl.ShaderSource(sh, 1, cstr, nil)
	free()
	gl.CompileShader(sh)
	var status int32
	gl.GetShaderiv(sh, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var n int32
		gl.GetShaderiv(sh, gl.INFO_LOG_LENGTH, &n)
		buf := make([]uint8, n+1)
		gl.GetShaderInfoLog(sh, n, nil, &buf[0])
		gl.DeleteShader(sh)
		return 0, fmt.Errorf("%s", buf)
	}
	return sh, nil
}

// ── main ──────────────────────────────────────────────────────────────────────

const (
	winW = 1024
	winH = 768
)

func main() {
	if err := glfw.Init(); err != nil {
		log.Fatal(err)
	}
	defer glfw.Terminate()

	// GL 4.3 core profile — minimum required for compute shaders.
	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 3)
	glfw.WindowHint(glfw.OpenGLProfileHint, int(glfw.CoreProfile))
	glfw.WindowHint(glfw.OpenGLForwardCompatible, 1)

	win, err := glfw.CreateWindow(winW, winH, "16 — GL 4.3 Compute: Julia Set", nil, nil)
	if err != nil {
		log.Fatal(err)
	}
	win.MakeContextCurrent()
	glfw.SwapInterval(1)

	if err := gl.InitWithProcAddrFunc(func(name string) unsafe.Pointer {
		return glfw.GetProcAddress(name)
	}); err != nil {
		log.Fatal(err)
	}

	// ── Compute program ───────────────────────────────────────────────────────
	computeProg, err := buildComputeProgram(computeSrc)
	if err != nil {
		log.Fatalf("compute shader: %v", err)
	}
	uResolution := gl.GetUniformLocation(computeProg, gl.Str("uResolution\x00"))
	uC := gl.GetUniformLocation(computeProg, gl.Str("uC\x00"))

	// ── Display program ───────────────────────────────────────────────────────
	displayProg, err := buildProgram(vertSrc, fragSrc)
	if err != nil {
		log.Fatalf("display shader: %v", err)
	}
	uTex := gl.GetUniformLocation(displayProg, gl.Str("uTex\x00"))

	// ── Output texture ────────────────────────────────────────────────────────
	// Immutable RGBA8 texture (GL 4.2 TexStorage2D) — one mip level.
	var tex uint32
	gl.GenTextures(1, &tex)
	gl.BindTexture(gl.TEXTURE_2D, tex)
	gl.TexStorage2D(gl.TEXTURE_2D, 1, gl.RGBA8, winW, winH)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, int32(gl.LINEAR))
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, int32(gl.LINEAR))
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, int32(gl.CLAMP_TO_EDGE))
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, int32(gl.CLAMP_TO_EDGE))

	// ── VAO for fullscreen draw ───────────────────────────────────────────────
	// Core profile requires a bound VAO even when no vertex buffers are used.
	var vao uint32
	gl.GenVertexArrays(1, &vao)
	gl.BindVertexArray(vao)

	// Work group counts: ceiling-divide texture size by local group size (16).
	groupsX := uint32(math.Ceil(float64(winW) / 16))
	groupsY := uint32(math.Ceil(float64(winH) / 16))

	for !win.ShouldClose() {
		if win.GetKey(glfw.KeyEscape) == glfw.Press {
			win.SetShouldClose(true)
		}

		t := glfw.GetTime()

		// ── Compute pass ──────────────────────────────────────────────────────
		// Bind the texture as an image unit so the compute shader can call
		// imageStore() to write individual pixels.
		gl.BindImageTexture(0, tex, 0, false, 0, gl.WRITE_ONLY, gl.RGBA8)

		gl.UseProgram(computeProg)
		gl.Uniform2f(uResolution, winW, winH)

		// Animate c along a circular orbit — cycles through many Julia shapes.
		cx := float32(0.7885 * math.Cos(t*0.3))
		cy := float32(0.7885 * math.Sin(t*0.3))
		gl.Uniform2f(uC, cx, cy)

		gl.DispatchCompute(groupsX, groupsY, 1)

		// ── Barrier ───────────────────────────────────────────────────────────
		// Ensure all imageStore() writes are visible before the fragment shader
		// samples the texture.
		gl.MemoryBarrier(gl.SHADER_IMAGE_ACCESS_BARRIER_BIT)

		// ── Display pass ──────────────────────────────────────────────────────
		gl.Clear(gl.COLOR_BUFFER_BIT)

		gl.UseProgram(displayProg)
		gl.ActiveTexture(gl.TEXTURE0)
		gl.BindTexture(gl.TEXTURE_2D, tex)
		gl.Uniform1i(uTex, 0)

		gl.BindVertexArray(vao)
		// Draw the fullscreen triangle — 3 vertices, no index buffer, no VBO.
		gl.DrawArrays(gl.TRIANGLES, 0, 3)

		win.SwapBuffers()
		glfw.PollEvents()
	}
}
