//go:build windows

// 07_camera — free-fly first-person camera.
//
// Controls:
//   W / S        — move forward / backward
//   A / D        — strafe left / right
//   Q / E        — move up / down
//   Hold RMB     — drag to look around (pitch / yaw)
//   Mouse wheel  — adjust movement speed
//   ESC          — quit
//
// Key concepts:
//   - Camera position + yaw + pitch stored as state; rebuilt into a view
//     matrix every frame.
//   - Delta time (dt) used to make movement speed frame-rate independent.
//   - Pitch clamped to ±89° to prevent gimbal-lock "flip".
//   - The view matrix is simply lookAt(pos, pos+front, up).
//
// Scene: 20 cubes scattered in a field — same as chapter 06 but now you
// can walk among them.
//
// Build:
//
//	CGO_ENABLED=0 go build -o 07_camera.exe .
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
uniform mat4 uMVP;
void main() {
    gl_Position = uMVP * vec4(aPos, 1.0);
    vColor = aColor;
}`

const fragSrc = `#version 330 core
in  vec3 vColor;
out vec4 fragColor;
void main() { fragColor = vec4(vColor, 1.0); }`

// ── geometry: unit cube ───────────────────────────────────────────────────────

var cubeVerts = []float32{
	// -Z
	-0.5, -0.5, -0.5, 0.9, 0.2, 0.2,
	0.5, -0.5, -0.5, 0.9, 0.2, 0.2,
	0.5, 0.5, -0.5, 0.9, 0.2, 0.2,
	0.5, 0.5, -0.5, 0.9, 0.2, 0.2,
	-0.5, 0.5, -0.5, 0.9, 0.2, 0.2,
	-0.5, -0.5, -0.5, 0.9, 0.2, 0.2,
	// +Z
	-0.5, -0.5, 0.5, 0.2, 0.7, 0.3,
	0.5, -0.5, 0.5, 0.2, 0.7, 0.3,
	0.5, 0.5, 0.5, 0.2, 0.7, 0.3,
	0.5, 0.5, 0.5, 0.2, 0.7, 0.3,
	-0.5, 0.5, 0.5, 0.2, 0.7, 0.3,
	-0.5, -0.5, 0.5, 0.2, 0.7, 0.3,
	// -X
	-0.5, 0.5, 0.5, 0.2, 0.4, 0.9,
	-0.5, 0.5, -0.5, 0.2, 0.4, 0.9,
	-0.5, -0.5, -0.5, 0.2, 0.4, 0.9,
	-0.5, -0.5, -0.5, 0.2, 0.4, 0.9,
	-0.5, -0.5, 0.5, 0.2, 0.4, 0.9,
	-0.5, 0.5, 0.5, 0.2, 0.4, 0.9,
	// +X
	0.5, 0.5, 0.5, 0.9, 0.7, 0.1,
	0.5, 0.5, -0.5, 0.9, 0.7, 0.1,
	0.5, -0.5, -0.5, 0.9, 0.7, 0.1,
	0.5, -0.5, -0.5, 0.9, 0.7, 0.1,
	0.5, -0.5, 0.5, 0.9, 0.7, 0.1,
	0.5, 0.5, 0.5, 0.9, 0.7, 0.1,
	// -Y
	-0.5, -0.5, -0.5, 0.6, 0.2, 0.8,
	0.5, -0.5, -0.5, 0.6, 0.2, 0.8,
	0.5, -0.5, 0.5, 0.6, 0.2, 0.8,
	0.5, -0.5, 0.5, 0.6, 0.2, 0.8,
	-0.5, -0.5, 0.5, 0.6, 0.2, 0.8,
	-0.5, -0.5, -0.5, 0.6, 0.2, 0.8,
	// +Y
	-0.5, 0.5, -0.5, 0.2, 0.8, 0.9,
	0.5, 0.5, -0.5, 0.2, 0.8, 0.9,
	0.5, 0.5, 0.5, 0.2, 0.8, 0.9,
	0.5, 0.5, 0.5, 0.2, 0.8, 0.9,
	-0.5, 0.5, 0.5, 0.2, 0.8, 0.9,
	-0.5, 0.5, -0.5, 0.2, 0.8, 0.9,
}

// 20 cube positions scattered across the scene.
var cubePositions = [][3]float32{
	{0, 0, 0}, {3, 0, -2}, {-3, 0, -2}, {1.5, 0, -5}, {-1.5, 0, -5},
	{4, 0, -7}, {-4, 0, -7}, {0, 0, -9}, {2.5, 0, -11}, {-2.5, 0, -11},
	{0, 0, -13}, {3.5, 0, -15}, {-3.5, 0, -15}, {1, 0, -17}, {-1, 0, -17},
	{4.5, 0, -4}, {-4.5, 0, -4}, {0.5, 2, -6}, {-0.5, 2, -6}, {0, 1, -10},
}

var (
	cam        = glutil.NewCamera([3]float32{0, 0, 4})
	winW, winH = 800, 600
	lastTime   = float64(0)
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

	win, err := glfw.CreateWindow(winW, winH,
		"07 — Camera: WASD + RMB drag | wheel = speed | ESC quit", nil, nil)
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
	win.SetMouseButtonCallback(func(_ *glfw.Window, btn glfw.MouseButton, action glfw.Action, _ glfw.ModifierKey) {
		if btn == glfw.MouseButtonRight {
			cam.SetRMB(action == glfw.Press)
		}
	})
	win.SetCursorPosCallback(func(_ *glfw.Window, x, y float64) {
		cam.MousePos(x, y)
	})
	win.SetScrollCallback(func(_ *glfw.Window, _, yoff float64) {
		cam.Scroll(yoff, 0.5, 50)
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

	uMVP := gl.GetUniformLocation(prog, gl.Str("uMVP\x00"))

	lastTime = glfw.GetTime()
	fmt.Println("WASD = move, hold RMB = look, wheel = speed, ESC = quit.")

	for !win.ShouldClose() {
		now := glfw.GetTime()
		dt := float32(now - lastTime)
		lastTime = now

		cam.HandleKeys(
			win.GetKey(glfw.KeyW) == glfw.Press,
			win.GetKey(glfw.KeyS) == glfw.Press,
			win.GetKey(glfw.KeyA) == glfw.Press,
			win.GetKey(glfw.KeyD) == glfw.Press,
			win.GetKey(glfw.KeyE) == glfw.Press,
			win.GetKey(glfw.KeyQ) == glfw.Press,
			dt,
		)

		view := cam.ViewMatrix()
		proj := glutil.Perspective(glutil.ToRad(60), float32(winW)/float32(winH), 0.05, 200)

		gl.ClearColor(0.07, 0.07, 0.1, 1)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

		gl.UseProgram(prog)
		gl.BindVertexArray(vao)

		for i, pos := range cubePositions {
			angle := float32(i) * 0.523
			model := glutil.MatMul(glutil.Translate3(pos[0], pos[1], pos[2]), glutil.RotY(angle))
			mvp := glutil.MatMul(proj, glutil.MatMul(view, model))
			gl.UniformMatrix4fv(uMVP, 1, false, &mvp[0])
			gl.DrawArrays(gl.TRIANGLES, 0, 36)
		}

		gl.BindVertexArray(0)
		win.SwapBuffers()
		glfw.PollEvents()
	}
}
