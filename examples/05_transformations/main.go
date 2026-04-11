//go:build windows

// 05_transformations — model matrices: translate, rotate, scale.
//
// Three identical quads, each with its own model matrix sent as a uniform:
//
//   Left   — translation: orbits around the origin
//   Centre — rotation:    spins in place
//   Right  — scale:       pulses (breathes) in and out
//
// Each quad uses the same VAO / shader; only the uModel uniform changes
// between draw calls.  This is the classic "one mesh, many placements"
// pattern that is the foundation of every real 3-D renderer.
//
// Build:
//
//	CGO_ENABLED=0 go build -o 05_transformations.exe .
package main

import (
	"fmt"
	"log"
	"math"

	gl     "github.com/ClaudioTheobaldo/gl-purego/v2.1/gl"
	glutil "github.com/ClaudioTheobaldo/gl-purego/examples/glutil"
	glfw   "github.com/ClaudioTheobaldo/glfw-purego/v3.3/glfw"
)

// ── shaders ──────────────────────────────────────────────────────────────────

const vertSrc = `#version 330 core
layout(location = 0) in vec2 aPos;
layout(location = 1) in vec3 aColor;
out vec3 vColor;
uniform mat4 uModel;
void main() {
    gl_Position = uModel * vec4(aPos, 0.0, 1.0);
    vColor = aColor;
}`

const fragSrc = `#version 330 core
in  vec3 vColor;
out vec4 fragColor;
void main() {
    fragColor = vec4(vColor, 1.0);
}`

// ── geometry ─────────────────────────────────────────────────────────────────

// Small coloured quad centred at the origin.  Layout: [X, Y, R, G, B]
var quadVerts = []float32{
	-0.2, -0.2, 1.0, 0.4, 0.2,
	0.2, -0.2, 0.2, 1.0, 0.4,
	0.2, 0.2, 0.4, 0.2, 1.0,
	-0.2, -0.2, 1.0, 0.4, 0.2,
	0.2, 0.2, 0.4, 0.2, 1.0,
	-0.2, 0.2, 1.0, 1.0, 0.2,
}

func main() {
	if err := glfw.Init(); err != nil {
		log.Fatalf("glfw.Init: %v", err)
	}
	defer glfw.Terminate()

	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 3)
	glfw.WindowHint(glfw.OpenGLProfileHint, int(glfw.CoreProfile))
	glfw.WindowHint(glfw.OpenGLForwardCompatible, 1)

	win, err := glfw.CreateWindow(800, 600, "05 — Transformations: translate · rotate · scale", nil, nil)
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
		gl.Viewport(0, 0, int32(w), int32(h))
	})
	w, h := win.GetFramebufferSize()
	gl.Viewport(0, 0, int32(w), int32(h))

	win.SetKeyCallback(func(win *glfw.Window, key glfw.Key, _ int, action glfw.Action, _ glfw.ModifierKey) {
		if action == glfw.Press && key == glfw.KeyEscape {
			win.SetShouldClose(true)
		}
	})

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
	const stride = int32(5 * 4)
	gl.VertexAttribPointer(0, 2, gl.FLOAT, false, stride, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(1, 3, gl.FLOAT, false, stride, gl.PtrOffset(8))
	gl.EnableVertexAttribArray(1)
	gl.BindVertexArray(0)

	uModel := gl.GetUniformLocation(prog, gl.Str("uModel\x00"))

	fmt.Println("Watch translate (left), rotate (centre), scale (right). ESC to quit.")

	for !win.ShouldClose() {
		gl.ClearColor(0.08, 0.08, 0.12, 1)
		gl.Clear(gl.COLOR_BUFFER_BIT)

		t := float32(glfw.GetTime())

		gl.UseProgram(prog)
		gl.BindVertexArray(vao)

		// ── left quad: translate (orbit) ──────────────────────────────────
		{
			tx := float32(math.Cos(float64(t))) * 0.55
			ty := float32(math.Sin(float64(t))) * 0.55
			m := translate(tx, ty)
			gl.UniformMatrix4fv(uModel, 1, false, &m[0])
			gl.DrawArrays(gl.TRIANGLES, 0, 6)
		}

		// ── centre quad: rotate in place ──────────────────────────────────
		{
			m := glutil.RotZ(t * 1.5)
			gl.UniformMatrix4fv(uModel, 1, false, &m[0])
			gl.DrawArrays(gl.TRIANGLES, 0, 6)
		}

		// ── right quad: scale (pulse) ─────────────────────────────────────
		{
			s := float32(0.6 + 0.4*math.Sin(float64(t)*2.0))
			m := scale(s, s)
			// Anchor it to the right side of the screen.
			m = glutil.MatMul(translate(0.55, 0), m)
			gl.UniformMatrix4fv(uModel, 1, false, &m[0])
			gl.DrawArrays(gl.TRIANGLES, 0, 6)
		}

		gl.BindVertexArray(0)
		win.SwapBuffers()
		glfw.PollEvents()
	}
}

// ── matrix helpers (column-major 4×4) ────────────────────────────────────────

func identity() [16]float32 {
	return [16]float32{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}
}

func translate(tx, ty float32) [16]float32 {
	m := identity()
	m[12] = tx
	m[13] = ty
	return m
}

func scale(sx, sy float32) [16]float32 {
	m := identity()
	m[0] = sx
	m[5] = sy
	return m
}
