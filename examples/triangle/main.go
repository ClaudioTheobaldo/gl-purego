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
	"math"
	"unsafe"

	gl   "github.com/ClaudioTheobaldo/gl-purego/v2.1/gl"
	glfw "github.com/ClaudioTheobaldo/glfw-purego/v3.3/glfw"
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

	// Compile shaders --------------------------------------------------------
	prog, err := buildProgram(vertSrc, fragSrc)
	if err != nil {
		log.Fatalf("shader build: %v", err)
	}
	defer gl.DeleteProgram(prog)

	// Upload geometry --------------------------------------------------------
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
	// aPos  — location 0, 2 floats, byte offset 0
	gl.VertexAttribPointer(0, 2, gl.FLOAT, false, stride, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(0)
	// aColor — location 1, 3 floats, byte offset 8 (2 floats × 4 bytes)
	gl.VertexAttribPointer(1, 3, gl.FLOAT, false, stride, gl.PtrOffset(8))
	gl.EnableVertexAttribArray(1)

	gl.BindVertexArray(0)

	uModel := gl.GetUniformLocation(prog, gl.Str("uModel"))

	fmt.Println("Rendering spinning triangle — press ESC to quit.")

	// Render loop ------------------------------------------------------------
	for !win.ShouldClose() {
		gl.ClearColor(0.08, 0.08, 0.12, 1.0)
		gl.Clear(gl.COLOR_BUFFER_BIT)

		angle := float32(glfw.GetTime() * 0.8) // 0.8 rad/s
		model := rotZ(angle)

		gl.UseProgram(prog)
		gl.UniformMatrix4fv(uModel, 1, false, &model[0])
		gl.BindVertexArray(vao)
		gl.DrawArrays(gl.TRIANGLES, 0, 3)
		gl.BindVertexArray(0)

		win.SwapBuffers()
		glfw.PollEvents()
	}
}

// rotZ returns a column-major 4×4 rotation matrix around the Z axis.
// (OpenGL / GLSL column-major convention, transpose=false)
func rotZ(angle float32) [16]float32 {
	c := float32(math.Cos(float64(angle)))
	s := float32(math.Sin(float64(angle)))
	// Column 0: ( c,  s, 0, 0)
	// Column 1: (-s,  c, 0, 0)
	// Column 2: ( 0,  0, 1, 0)
	// Column 3: ( 0,  0, 0, 1)
	return [16]float32{
		c, s, 0, 0,
		-s, c, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
	}
}

// buildProgram compiles vertex + fragment sources and links an OpenGL program.
func buildProgram(vertSrc, fragSrc string) (uint32, error) {
	vs, err := compileShader(vertSrc, gl.VERTEX_SHADER)
	if err != nil {
		return 0, fmt.Errorf("vertex shader: %w", err)
	}
	fs, err := compileShader(fragSrc, gl.FRAGMENT_SHADER)
	if err != nil {
		gl.DeleteShader(vs)
		return 0, fmt.Errorf("fragment shader: %w", err)
	}

	prog := gl.CreateProgram()
	gl.AttachShader(prog, vs)
	gl.AttachShader(prog, fs)
	gl.LinkProgram(prog)
	gl.DeleteShader(vs)
	gl.DeleteShader(fs)

	var status int32
	gl.GetProgramiv(prog, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var logLen int32
		gl.GetProgramiv(prog, gl.INFO_LOG_LENGTH, &logLen)
		buf := make([]uint8, logLen+1)
		gl.GetProgramInfoLog(prog, logLen, nil, &buf[0])
		gl.DeleteProgram(prog)
		return 0, fmt.Errorf("link: %s", string(buf))
	}
	return prog, nil
}

// compileShader compiles a single GLSL shader of the given type.
func compileShader(src string, kind uint32) (uint32, error) {
	shader := gl.CreateShader(kind)

	cstr, free := gl.Strs(src)
	gl.ShaderSource(shader, 1, cstr, nil)
	free()

	gl.CompileShader(shader)

	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLen int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLen)
		buf := make([]uint8, logLen+1)
		gl.GetShaderInfoLog(shader, logLen, nil, &buf[0])
		gl.DeleteShader(shader)
		return 0, fmt.Errorf("%s", string(buf))
	}
	return shader, nil
}
