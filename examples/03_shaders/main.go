//go:build windows

// 03_shaders — uniforms as the CPU↔GPU communication channel.
//
// A fullscreen quad covers the viewport (-1..1 in both axes).
// The vertex shader just passes through positions; all the visual
// work happens in the fragment shader using two uniforms:
//
//   uTime  float  — seconds since start, drives animation
//   uMouse vec2   — cursor in NDC (-1..1), controls the ripple origin
//
// Move the mouse to shift the ripple centre. The concentric rings
// demonstrate how a single uniform value touches every fragment at
// zero per-vertex cost.
//
// Build:
//
//	CGO_ENABLED=0 go build -o 03_shaders.exe .
package main

import (
	"fmt"
	"log"

	gl     "github.com/ClaudioTheobaldo/gl-purego/v2.1/gl"
	glutil "github.com/ClaudioTheobaldo/gl-purego/examples/glutil"
	glfw   "github.com/ClaudioTheobaldo/glfw-purego/v3.3/glfw"
)

// ── shaders ─────────────────────────────────────────────────────────────────

const vertSrc = `#version 330 core
layout(location = 0) in vec2 aPos;
out vec2 vUV;
void main() {
    gl_Position = vec4(aPos, 0.0, 1.0);
    // Remap [-1,1] → [0,1] for a convenient UV coordinate in the frag shader.
    vUV = aPos * 0.5 + 0.5;
}`

const fragSrc = `#version 330 core
in  vec2 vUV;
out vec4 fragColor;

uniform float uTime;
uniform vec2  uMouse; // NDC: x,y in [-1, 1]

void main() {
    // Convert fragment to the same space as uMouse (NDC centred on 0).
    vec2 pos = vUV * 2.0 - 1.0;

    // Distance from the mouse cursor.
    float d = distance(pos, uMouse);

    // Concentric ripple rings: sin wave in space, scrolling with time.
    float rings = sin(d * 18.0 - uTime * 4.0);

    // Map [-1,1] → [0,1] for brightness.
    float brightness = rings * 0.5 + 0.5;

    // Colour varies with UV so we can see coordinate mapping clearly.
    vec3 col = vec3(vUV.x, vUV.y, brightness);
    fragColor = vec4(col, 1.0);
}`

// ── geometry ─────────────────────────────────────────────────────────────────

// Fullscreen quad — two triangles, positions only.
var quadVerts = []float32{
	-1, -1,
	1, -1,
	1, 1,
	-1, -1,
	1, 1,
	-1, 1,
}

// ── state ────────────────────────────────────────────────────────────────────

var (
	winW, winH     = 800, 600
	mouseX, mouseY float64 // NDC
)

func main() {
	if err := glfw.Init(); err != nil {
		log.Fatalf("glfw.Init: %v", err)
	}
	defer glfw.Terminate()

	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 3)
	glfw.WindowHint(glfw.OpenGLProfileHint, int(glfw.CoreProfile))
	glfw.WindowHint(glfw.OpenGLForwardCompatible, 1)

	win, err := glfw.CreateWindow(winW, winH, "03 — Shaders & Uniforms", nil, nil)
	if err != nil {
		log.Fatalf("CreateWindow: %v", err)
	}
	defer win.Destroy()

	win.MakeContextCurrent()
	glfw.SwapInterval(1)

	if err := gl.InitWithProcAddrFunc(glfw.GetProcAddress); err != nil {
		log.Fatalf("gl.Init: %v", err)
	}

	// ── callbacks ──────────────────────────────────────────────────────────

	win.SetFramebufferSizeCallback(func(_ *glfw.Window, w, h int) {
		winW, winH = w, h
		gl.Viewport(0, 0, int32(w), int32(h))
	})
	winW, winH = win.GetFramebufferSize()
	gl.Viewport(0, 0, int32(winW), int32(winH))

	win.SetCursorPosCallback(func(_ *glfw.Window, x, y float64) {
		// Convert window pixel coords → NDC.
		mouseX = (x/float64(winW))*2.0 - 1.0
		mouseY = -((y/float64(winH))*2.0 - 1.0) // flip Y (window Y is top-down)
	})

	win.SetKeyCallback(func(w *glfw.Window, key glfw.Key, _ int, action glfw.Action, _ glfw.ModifierKey) {
		if action == glfw.Press && key == glfw.KeyEscape {
			w.SetShouldClose(true)
		}
	})

	// ── GPU resources ──────────────────────────────────────────────────────

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
	gl.VertexAttribPointer(0, 2, gl.FLOAT, false, 8, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(0)
	gl.BindVertexArray(0)

	// Cache uniform locations.
	uTime  := gl.GetUniformLocation(prog, gl.Str("uTime\x00"))
	uMouse := gl.GetUniformLocation(prog, gl.Str("uMouse\x00"))

	fmt.Println("Move the mouse to shift the ripple origin. ESC to quit.")

	for !win.ShouldClose() {
		gl.ClearColor(0, 0, 0, 1)
		gl.Clear(gl.COLOR_BUFFER_BIT)

		t := float32(glfw.GetTime())

		gl.UseProgram(prog)
		gl.Uniform1f(uTime, t)
		gl.Uniform2f(uMouse, float32(mouseX), float32(mouseY))

		gl.BindVertexArray(vao)
		gl.DrawArrays(gl.TRIANGLES, 0, 6)
		gl.BindVertexArray(0)

		win.SwapBuffers()
		glfw.PollEvents()
	}
}


