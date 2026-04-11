//go:build windows

// triangle renders a colour-interpolated triangle that rotates slowly, using
// OpenGL 3.3 core via glfw-purego + gl-purego (zero CGO).
//
// Build (CGO disabled):
//
//	CGO_ENABLED=0 go build -o triangle.exe .
package main

import (
	"fmt"
	"log"
	"unsafe"

	gl     "github.com/ClaudioTheobaldo/gl-purego/v2.1/gl"
	glutil "github.com/ClaudioTheobaldo/gl-purego/examples/glutil"
	glfw   "github.com/ClaudioTheobaldo/glfw-purego/v3.3/glfw"
)

// Vertex layout: [X, Y, R, G, B]  — 5 × float32 per vertex, 3 vertices
var vertices = []float32{
	//   X      Y      R     G     B
	0.00, 0.80, 1.0, 0.25, 0.25, // top    — red
	-0.70, -0.50, 0.25, 1.0, 0.25, // left   — green
	0.70, -0.50, 0.25, 0.25, 1.0, // right  — blue
}

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

func main() {
	if err := glfw.Init(); err != nil {
		log.Fatalf("glfw.Init: %v", err)
	}
	defer glfw.Terminate()

	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 3)
	glfw.WindowHint(glfw.OpenGLProfileHint, int(glfw.CoreProfile))
	glfw.WindowHint(glfw.OpenGLForwardCompatible, 1)

	win, err := glfw.CreateWindow(800, 600, "Triangle — glfw-purego + gl-purego", nil, nil)
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
		gl.Viewport(0, 0, int32(width), int32(height))
	})
	fw, fh := win.GetFramebufferSize()
	gl.Viewport(0, 0, int32(fw), int32(fh))

	prog, err := glutil.BuildProgram(vertSrc, fragSrc)
	if err != nil {
		log.Fatalf("shader build: %v", err)
	}
	defer gl.DeleteProgram(prog)

	var vao, vbo uint32
	gl.GenVertexArrays(1, &vao)
	gl.GenBuffers(1, &vbo)
	defer func() {
		gl.DeleteVertexArrays(1, &vao)
		gl.DeleteBuffers(1, &vbo)
	}()

	gl.BindVertexArray(vao)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(vertices)*4, unsafe.Pointer(&vertices[0]), gl.STATIC_DRAW)

	const stride = int32(5 * 4) // 5 float32 × 4 bytes
	gl.VertexAttribPointer(0, 2, gl.FLOAT, false, stride, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(1, 3, gl.FLOAT, false, stride, gl.PtrOffset(8))
	gl.EnableVertexAttribArray(1)

	gl.BindVertexArray(0)

	uModel := gl.GetUniformLocation(prog, gl.Str("uModel"))

	fmt.Println("Rendering spinning triangle — press ESC to quit.")

	for !win.ShouldClose() {
		gl.ClearColor(0.08, 0.08, 0.12, 1.0)
		gl.Clear(gl.COLOR_BUFFER_BIT)

		angle := float32(glfw.GetTime() * 0.8) // 0.8 rad/s
		model := glutil.RotZ(angle)

		gl.UseProgram(prog)
		gl.UniformMatrix4fv(uModel, 1, false, &model[0])
		gl.BindVertexArray(vao)
		gl.DrawArrays(gl.TRIANGLES, 0, 3)
		gl.BindVertexArray(0)

		win.SwapBuffers()
		glfw.PollEvents()
	}
}
