//go:build windows

// cube renders a rotating 3-D cube with six differently-coloured faces using
// OpenGL 3.3 core via glfw-purego + gl-purego (zero CGO).
//
// Build:
//
//	CGO_ENABLED=0 go build -o cube.exe .
package main

import (
	"fmt"
	"log"
	"unsafe"

	gl     "github.com/ClaudioTheobaldo/gl-purego/v2.1/gl"
	glutil "github.com/ClaudioTheobaldo/gl-purego/examples/glutil"
	glfw   "github.com/ClaudioTheobaldo/glfw-purego/v3.3/glfw"
)

// 36 vertices (6 faces × 2 triangles × 3 verts), layout: [X Y Z R G B]
var cubeVertices = []float32{
	// Front (+Z) — red
	-0.5, -0.5, 0.5, 1.0, 0.2, 0.2,
	0.5, -0.5, 0.5, 1.0, 0.2, 0.2,
	0.5, 0.5, 0.5, 1.0, 0.2, 0.2,
	-0.5, -0.5, 0.5, 1.0, 0.2, 0.2,
	0.5, 0.5, 0.5, 1.0, 0.2, 0.2,
	-0.5, 0.5, 0.5, 1.0, 0.2, 0.2,
	// Back (-Z) — green
	0.5, -0.5, -0.5, 0.2, 1.0, 0.2,
	-0.5, -0.5, -0.5, 0.2, 1.0, 0.2,
	-0.5, 0.5, -0.5, 0.2, 1.0, 0.2,
	0.5, -0.5, -0.5, 0.2, 1.0, 0.2,
	-0.5, 0.5, -0.5, 0.2, 1.0, 0.2,
	0.5, 0.5, -0.5, 0.2, 1.0, 0.2,
	// Left (-X) — blue
	-0.5, -0.5, -0.5, 0.2, 0.2, 1.0,
	-0.5, -0.5, 0.5, 0.2, 0.2, 1.0,
	-0.5, 0.5, 0.5, 0.2, 0.2, 1.0,
	-0.5, -0.5, -0.5, 0.2, 0.2, 1.0,
	-0.5, 0.5, 0.5, 0.2, 0.2, 1.0,
	-0.5, 0.5, -0.5, 0.2, 0.2, 1.0,
	// Right (+X) — yellow
	0.5, -0.5, 0.5, 1.0, 1.0, 0.2,
	0.5, -0.5, -0.5, 1.0, 1.0, 0.2,
	0.5, 0.5, -0.5, 1.0, 1.0, 0.2,
	0.5, -0.5, 0.5, 1.0, 1.0, 0.2,
	0.5, 0.5, -0.5, 1.0, 1.0, 0.2,
	0.5, 0.5, 0.5, 1.0, 1.0, 0.2,
	// Top (+Y) — cyan
	-0.5, 0.5, 0.5, 0.2, 1.0, 1.0,
	0.5, 0.5, 0.5, 0.2, 1.0, 1.0,
	0.5, 0.5, -0.5, 0.2, 1.0, 1.0,
	-0.5, 0.5, 0.5, 0.2, 1.0, 1.0,
	0.5, 0.5, -0.5, 0.2, 1.0, 1.0,
	-0.5, 0.5, -0.5, 0.2, 1.0, 1.0,
	// Bottom (-Y) — magenta
	-0.5, -0.5, -0.5, 1.0, 0.2, 1.0,
	0.5, -0.5, -0.5, 1.0, 0.2, 1.0,
	0.5, -0.5, 0.5, 1.0, 0.2, 1.0,
	-0.5, -0.5, -0.5, 1.0, 0.2, 1.0,
	0.5, -0.5, 0.5, 1.0, 0.2, 1.0,
	-0.5, -0.5, 0.5, 1.0, 0.2, 1.0,
}

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
void main() {
    fragColor = vec4(vColor, 1.0);
}`

var (
	winW, winH int = 800, 600
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

	win, err := glfw.CreateWindow(winW, winH, "Cube — glfw-purego + gl-purego", nil, nil)
	if err != nil {
		log.Fatalf("CreateWindow: %v", err)
	}
	defer win.Destroy()

	win.MakeContextCurrent()
	glfw.SwapInterval(1)

	if err := gl.InitWithProcAddrFunc(glfw.GetProcAddress); err != nil {
		log.Fatalf("gl.Init: %v", err)
	}

	win.SetKeyCallback(func(w *glfw.Window, key glfw.Key, _ int, action glfw.Action, _ glfw.ModifierKey) {
		if key == glfw.KeyEscape && action == glfw.Press {
			w.SetShouldClose(true)
		}
	})
	win.SetFramebufferSizeCallback(func(w *glfw.Window, width, height int) {
		winW, winH = width, height
		gl.Viewport(0, 0, int32(width), int32(height))
	})
	winW, winH = win.GetFramebufferSize()
	gl.Viewport(0, 0, int32(winW), int32(winH))

	// Compile shaders
	prog, err := glutil.BuildProgram(vertSrc, fragSrc)
	if err != nil {
		log.Fatalf("shader: %v", err)
	}
	defer gl.DeleteProgram(prog)

	// Upload geometry
	var vao, vbo uint32
	gl.GenVertexArrays(1, &vao)
	gl.GenBuffers(1, &vbo)
	defer func() { gl.DeleteVertexArrays(1, &vao); gl.DeleteBuffers(1, &vbo) }()

	gl.BindVertexArray(vao)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(cubeVertices)*4, unsafe.Pointer(&cubeVertices[0]), gl.STATIC_DRAW)

	const stride = int32(6 * 4) // 6 float32 × 4 bytes
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, stride, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(1, 3, gl.FLOAT, false, stride, gl.PtrOffset(12))
	gl.EnableVertexAttribArray(1)
	gl.BindVertexArray(0)

	uMVP := gl.GetUniformLocation(prog, gl.Str("uMVP"))

	gl.Enable(gl.DEPTH_TEST)

	fmt.Println("Rendering cube — press ESC to quit.")

	for !win.ShouldClose() {
		gl.ClearColor(0.08, 0.08, 0.12, 1.0)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

		t := float32(glfw.GetTime())
		model := glutil.MatMul(glutil.RotX(t*0.5), glutil.RotY(t*0.7))
		view  := glutil.LookAt([3]float32{0, 1.5, 3}, [3]float32{0, 0, 0}, [3]float32{0, 1, 0})
		proj  := glutil.Perspective(glutil.ToRad(45), float32(winW)/float32(winH), 0.1, 100)
		mvp   := glutil.MatMul(proj, glutil.MatMul(view, model))

		gl.UseProgram(prog)
		gl.UniformMatrix4fv(uMVP, 1, false, &mvp[0])
		gl.BindVertexArray(vao)
		gl.DrawArrays(gl.TRIANGLES, 0, 36)
		gl.BindVertexArray(0)

		win.SwapBuffers()
		glfw.PollEvents()
	}
}
