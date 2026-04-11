//go:build windows

// 04_textures — sampling textures in a fragment shader.
//
// A unit quad is drawn with two procedurally generated textures:
//
//   Texture 0 — UV gradient: red = U, green = V, blue = 0
//   Texture 1 — checkerboard: black and white 8×8 pattern
//
// A mix uniform blends between the two.  Press LEFT / RIGHT to
// adjust the blend ratio; the current value is shown in the title bar.
//
// Key concepts:
//   - Texture objects, texture units, sampler2D uniforms
//   - UV coordinates as a vertex attribute passed to the fragment shader
//   - mix(a, b, t) for blending multiple textures
//
// Build:
//
//	CGO_ENABLED=0 go build -o 04_textures.exe .
package main

import (
	"fmt"
	"log"

	gl     "github.com/ClaudioTheobaldo/gl-purego/v2.1/gl"
	glutil "github.com/ClaudioTheobaldo/gl-purego/examples/glutil"
	glfw   "github.com/ClaudioTheobaldo/glfw-purego/v3.3/glfw"
)

// ── shaders ──────────────────────────────────────────────────────────────────

const vertSrc = `#version 330 core
layout(location = 0) in vec2 aPos;
layout(location = 1) in vec2 aUV;
out vec2 vUV;
void main() {
    gl_Position = vec4(aPos, 0.0, 1.0);
    vUV = aUV;
}`

const fragSrc = `#version 330 core
in  vec2 vUV;
out vec4 fragColor;

uniform sampler2D uTex0;   // UV gradient
uniform sampler2D uTex1;   // checkerboard
uniform float     uMix;    // 0.0 = tex0 only, 1.0 = tex1 only

void main() {
    vec4 c0 = texture(uTex0, vUV);
    vec4 c1 = texture(uTex1, vUV);
    fragColor = mix(c0, c1, uMix);
}`

// ── geometry ─────────────────────────────────────────────────────────────────

// Two triangles covering the [-0.8, 0.8] range so we can see the background.
// Layout: [X, Y, U, V]
var quadVerts = []float32{
	-0.8, -0.8, 0, 0,
	0.8, -0.8, 1, 0,
	0.8, 0.8, 1, 1,
	-0.8, -0.8, 0, 0,
	0.8, 0.8, 1, 1,
	-0.8, 0.8, 0, 1,
}

// ── state ─────────────────────────────────────────────────────────────────────

var (
	winW, winH = 800, 600
	mixVal     = float32(0.0)
)

const mixStep = float32(0.05)

func main() {
	if err := glfw.Init(); err != nil {
		log.Fatalf("glfw.Init: %v", err)
	}
	defer glfw.Terminate()

	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 3)
	glfw.WindowHint(glfw.OpenGLProfileHint, int(glfw.CoreProfile))
	glfw.WindowHint(glfw.OpenGLForwardCompatible, 1)

	win, err := glfw.CreateWindow(winW, winH, windowTitle(), nil, nil)
	if err != nil {
		log.Fatalf("CreateWindow: %v", err)
	}
	defer win.Destroy()

	win.MakeContextCurrent()
	glfw.SwapInterval(1)

	if err := gl.InitWithProcAddrFunc(glfw.GetProcAddress); err != nil {
		log.Fatalf("gl.Init: %v", err)
	}

	win.SetFramebufferSizeCallback(func(_ *glfw.Window, w, h int) {
		winW, winH = w, h
		gl.Viewport(0, 0, int32(w), int32(h))
	})
	winW, winH = win.GetFramebufferSize()
	gl.Viewport(0, 0, int32(winW), int32(winH))

	win.SetKeyCallback(func(w *glfw.Window, key glfw.Key, _ int, action glfw.Action, _ glfw.ModifierKey) {
		if action != glfw.Press && action != glfw.Repeat {
			return
		}
		switch key {
		case glfw.KeyEscape:
			w.SetShouldClose(true)
		case glfw.KeyRight, glfw.KeyUp:
			if mixVal < 1 {
				mixVal += mixStep
				if mixVal > 1 {
					mixVal = 1
				}
				w.SetTitle(windowTitle())
			}
		case glfw.KeyLeft, glfw.KeyDown:
			if mixVal > 0 {
				mixVal -= mixStep
				if mixVal < 0 {
					mixVal = 0
				}
				w.SetTitle(windowTitle())
			}
		}
	})

	// ── textures ────────────────────────────────────────────────────────────

	tex0 := makeGradientTexture(64, 64)
	tex1 := makeCheckerTexture(64, 64, 8)
	defer gl.DeleteTextures(1, &tex0)
	defer gl.DeleteTextures(1, &tex1)

	// ── shader + VAO ────────────────────────────────────────────────────────

	prog, err := glutil.BuildProgram(vertSrc, fragSrc)
	if err != nil {
		log.Fatalf("shader: %v", err)
	}
	defer gl.DeleteProgram(prog)

	var vao, vbo uint32
	gl.GenVertexArrays(1, &vao)
	gl.GenBuffers(1, &vbo)
	defer func() { gl.DeleteVertexArrays(1, &vao); gl.DeleteBuffers(1, &vbo) }()

	gl.BindVertexArray(vao)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(quadVerts)*4, gl.Ptr(quadVerts), gl.STATIC_DRAW)
	const stride = int32(4 * 4) // 2 pos + 2 uv
	gl.VertexAttribPointer(0, 2, gl.FLOAT, false, stride, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(1, 2, gl.FLOAT, false, stride, gl.PtrOffset(8))
	gl.EnableVertexAttribArray(1)
	gl.BindVertexArray(0)

	// Tell each sampler uniform which texture unit it reads from.
	// This only needs to happen once after linking.
	gl.UseProgram(prog)
	gl.Uniform1i(gl.GetUniformLocation(prog, gl.Str("uTex0\x00")), 0) // unit 0
	gl.Uniform1i(gl.GetUniformLocation(prog, gl.Str("uTex1\x00")), 1) // unit 1

	uMix := gl.GetUniformLocation(prog, gl.Str("uMix\x00"))

	fmt.Println("LEFT/RIGHT to blend between the two textures. ESC to quit.")

	for !win.ShouldClose() {
		gl.ClearColor(0.1, 0.1, 0.15, 1)
		gl.Clear(gl.COLOR_BUFFER_BIT)

		// Bind each texture to its designated unit.
		gl.ActiveTexture(gl.TEXTURE0)
		gl.BindTexture(gl.TEXTURE_2D, tex0)
		gl.ActiveTexture(gl.TEXTURE1)
		gl.BindTexture(gl.TEXTURE_2D, tex1)

		gl.UseProgram(prog)
		gl.Uniform1f(uMix, mixVal)

		gl.BindVertexArray(vao)
		gl.DrawArrays(gl.TRIANGLES, 0, 6)
		gl.BindVertexArray(0)

		win.SwapBuffers()
		glfw.PollEvents()
	}
}

func windowTitle() string {
	return fmt.Sprintf("04 — Textures  |  mix = %.2f  |  LEFT/RIGHT to blend", mixVal)
}

// ── procedural texture generators ────────────────────────────────────────────

// makeGradientTexture creates a w×h RGBA texture where R = U, G = V, B = 0.
func makeGradientTexture(w, h int) uint32 {
	pixels := make([]uint8, w*h*4)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := (y*w + x) * 4
			pixels[i+0] = uint8(float32(x) / float32(w-1) * 255) // R = U
			pixels[i+1] = uint8(float32(y) / float32(h-1) * 255) // G = V
			pixels[i+2] = 0
			pixels[i+3] = 255
		}
	}
	return uploadTexture(int32(w), int32(h), pixels)
}

// makeCheckerTexture creates a w×h RGBA checkerboard with cells of size cells.
func makeCheckerTexture(w, h, cells int) uint32 {
	pixels := make([]uint8, w*h*4)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := (y*w + x) * 4
			cx := (x * cells / w) % 2
			cy := (y * cells / h) % 2
			var v uint8
			if (cx+cy)%2 == 0 {
				v = 255
			}
			pixels[i+0] = v
			pixels[i+1] = v
			pixels[i+2] = v
			pixels[i+3] = 255
		}
	}
	return uploadTexture(int32(w), int32(h), pixels)
}

func uploadTexture(w, h int32, pixels []uint8) uint32 {
	var tex uint32
	gl.GenTextures(1, &tex)
	gl.BindTexture(gl.TEXTURE_2D, tex)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, int32(gl.CLAMP_TO_EDGE))
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, int32(gl.CLAMP_TO_EDGE))
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, int32(gl.NEAREST))
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, int32(gl.NEAREST))
	gl.TexImage2D(gl.TEXTURE_2D, 0, int32(gl.RGBA), w, h, 0,
		gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(pixels))
	return tex
}

