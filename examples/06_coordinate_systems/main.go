//go:build windows

// 06_coordinate_systems — the Model / View / Projection (MVP) pipeline.
//
// Ten coloured cubes sit on a grid in world space.  A fixed camera looks at
// them from above and to the side.  Each cube has its own model matrix built
// from a different world-space position.
//
// The shader receives three separate matrices — uModel, uView, uProjection —
// and multiplies them in the correct order (Projection × View × Model × pos).
// Keeping them separate makes it easy to see which part of the pipeline each
// one belongs to, and is the standard pattern before combining them into a
// single precomputed MVP on the CPU.
//
// Key concepts taught:
//   - Model matrix    — places an object in world space (position, rotation, scale)
//   - View matrix     — positions and orients the camera (lookAt)
//   - Projection matrix — adds perspective foreshortening (perspective)
//   - gl_Position = projection × view × model × localVertex
//
// Build:
//
//	CGO_ENABLED=0 go build -o 06_coordinate_systems.exe .
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
layout(location = 0) in vec3 aPos;
layout(location = 1) in vec3 aColor;
out vec3 vColor;

// Three separate matrices — the MVP pipeline made explicit.
uniform mat4 uModel;
uniform mat4 uView;
uniform mat4 uProjection;

void main() {
    // Order: Projection × View × Model × local vertex position
    gl_Position = uProjection * uView * uModel * vec4(aPos, 1.0);
    vColor = aColor;
}`

const fragSrc = `#version 330 core
in  vec3 vColor;
out vec4 fragColor;
void main() {
    fragColor = vec4(vColor, 1.0);
}`

// ── geometry: unit cube, 36 vertices (6 faces × 2 tris × 3 verts) ────────────
// Layout: [X, Y, Z, R, G, B]

var cubeVerts = []float32{
	// -Z face (front)
	-0.5, -0.5, -0.5, 0.9, 0.2, 0.2,
	0.5, -0.5, -0.5, 0.9, 0.2, 0.2,
	0.5, 0.5, -0.5, 0.9, 0.2, 0.2,
	0.5, 0.5, -0.5, 0.9, 0.2, 0.2,
	-0.5, 0.5, -0.5, 0.9, 0.2, 0.2,
	-0.5, -0.5, -0.5, 0.9, 0.2, 0.2,
	// +Z face (back)
	-0.5, -0.5, 0.5, 0.2, 0.7, 0.3,
	0.5, -0.5, 0.5, 0.2, 0.7, 0.3,
	0.5, 0.5, 0.5, 0.2, 0.7, 0.3,
	0.5, 0.5, 0.5, 0.2, 0.7, 0.3,
	-0.5, 0.5, 0.5, 0.2, 0.7, 0.3,
	-0.5, -0.5, 0.5, 0.2, 0.7, 0.3,
	// -X face (left)
	-0.5, 0.5, 0.5, 0.2, 0.4, 0.9,
	-0.5, 0.5, -0.5, 0.2, 0.4, 0.9,
	-0.5, -0.5, -0.5, 0.2, 0.4, 0.9,
	-0.5, -0.5, -0.5, 0.2, 0.4, 0.9,
	-0.5, -0.5, 0.5, 0.2, 0.4, 0.9,
	-0.5, 0.5, 0.5, 0.2, 0.4, 0.9,
	// +X face (right)
	0.5, 0.5, 0.5, 0.9, 0.7, 0.1,
	0.5, 0.5, -0.5, 0.9, 0.7, 0.1,
	0.5, -0.5, -0.5, 0.9, 0.7, 0.1,
	0.5, -0.5, -0.5, 0.9, 0.7, 0.1,
	0.5, -0.5, 0.5, 0.9, 0.7, 0.1,
	0.5, 0.5, 0.5, 0.9, 0.7, 0.1,
	// -Y face (bottom)
	-0.5, -0.5, -0.5, 0.6, 0.2, 0.8,
	0.5, -0.5, -0.5, 0.6, 0.2, 0.8,
	0.5, -0.5, 0.5, 0.6, 0.2, 0.8,
	0.5, -0.5, 0.5, 0.6, 0.2, 0.8,
	-0.5, -0.5, 0.5, 0.6, 0.2, 0.8,
	-0.5, -0.5, -0.5, 0.6, 0.2, 0.8,
	// +Y face (top)
	-0.5, 0.5, -0.5, 0.2, 0.8, 0.9,
	0.5, 0.5, -0.5, 0.2, 0.8, 0.9,
	0.5, 0.5, 0.5, 0.2, 0.8, 0.9,
	0.5, 0.5, 0.5, 0.2, 0.8, 0.9,
	-0.5, 0.5, 0.5, 0.2, 0.8, 0.9,
	-0.5, 0.5, -0.5, 0.2, 0.8, 0.9,
}

// World-space positions for 10 cubes.
var cubePositions = [][3]float32{
	{0, 0, 0},
	{2, 0, -1},
	{-2, 0, -1},
	{1, 0, -3},
	{-1, 0, -3},
	{3, 0, -4},
	{-3, 0, -4},
	{0, 0, -5},
	{2, 0, -6},
	{-2, 0, -6},
}

var winW, winH = 800, 600

func main() {
	if err := glfw.Init(); err != nil {
		log.Fatalf("glfw.Init: %v", err)
	}
	defer glfw.Terminate()

	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 3)
	glfw.WindowHint(glfw.OpenGLProfileHint, int(glfw.CoreProfile))
	glfw.WindowHint(glfw.OpenGLForwardCompatible, 1)

	win, err := glfw.CreateWindow(winW, winH, "06 — Coordinate Systems: Model · View · Projection", nil, nil)
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
		if action == glfw.Press && key == glfw.KeyEscape {
			w.SetShouldClose(true)
		}
	})

	gl.Enable(gl.DEPTH_TEST)

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
	gl.BufferData(gl.ARRAY_BUFFER, len(cubeVerts)*4, gl.Ptr(cubeVerts), gl.STATIC_DRAW)
	const stride = int32(6 * 4)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, stride, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(1, 3, gl.FLOAT, false, stride, gl.PtrOffset(12))
	gl.EnableVertexAttribArray(1)
	gl.BindVertexArray(0)

	uModel      := gl.GetUniformLocation(prog, gl.Str("uModel\x00"))
	uView       := gl.GetUniformLocation(prog, gl.Str("uView\x00"))
	uProjection := gl.GetUniformLocation(prog, gl.Str("uProjection\x00"))

	// ── VIEW matrix — fixed camera, computed once ─────────────────────────
	// Camera sits above and behind, looks toward the origin.
	view := glutil.LookAt(
		[3]float32{0, 3.5, 5},  // eye
		[3]float32{0, 0, -2},   // centre
		[3]float32{0, 1, 0},    // up
	)

	fmt.Println("Ten cubes in world space — M·V·P pipeline. ESC to quit.")

	for !win.ShouldClose() {
		gl.ClearColor(0.07, 0.07, 0.1, 1)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

		t := float32(glfw.GetTime())

		// Projection changes if the window is resized.
		proj := glutil.Perspective(glutil.ToRad(45), float32(winW)/float32(winH), 0.1, 100)

		gl.UseProgram(prog)
		gl.UniformMatrix4fv(uView, 1, false, &view[0])
		gl.UniformMatrix4fv(uProjection, 1, false, &proj[0])

		gl.BindVertexArray(vao)
		for i, pos := range cubePositions {
			// Each cube rotates at a slightly different speed so they don't
			// all look identical — this shows the model matrix clearly.
			angle := t*0.5 + float32(i)*0.314
			model := glutil.MatMul(
				glutil.Translate3(pos[0], pos[1], pos[2]),
				glutil.RotY(angle),
			)
			gl.UniformMatrix4fv(uModel, 1, false, &model[0])
			gl.DrawArrays(gl.TRIANGLES, 0, 36)
		}
		gl.BindVertexArray(0)

		win.SwapBuffers()
		glfw.PollEvents()
	}
}

