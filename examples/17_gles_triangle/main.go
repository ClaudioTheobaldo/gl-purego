//go:build windows

// 17_gles_triangle renders a colour-interpolated triangle using OpenGL ES 3.0
// via the new EGL backend in glfw-purego + ANGLE (libGLESv2.dll / libEGL.dll).
//
// ANGLE must be available on PATH or in the same directory as the executable.
// Download from https://github.com/nicowillis/angle-windows or extract from
// a Chromium/Chrome installation.
//
// Build (CGO disabled):
//
//	CGO_ENABLED=0 go build -o gles_triangle.exe .
package main

import (
	"log"
	"unsafe"

	gl   "github.com/ClaudioTheobaldo/gl-purego/gles2/v3.0/gl"
	glfw "github.com/ClaudioTheobaldo/glfw-purego/v3.3/glfw"
)

// Vertex layout: [X, Y, R, G, B] — 5 × float32 per vertex, 3 vertices.
var vertices = []float32{
	//   X       Y      R     G     B
	0.00, 0.80, 1.0, 0.25, 0.25, // top    — red
	-0.70, -0.50, 0.25, 1.0, 0.25, // left   — green
	0.70, -0.50, 0.25, 0.25, 1.0, // right  — blue
}

// GLES 3.0 shaders use "#version 300 es" and require a precision qualifier.
const vertSrc = `#version 300 es
layout(location = 0) in vec2 aPos;
layout(location = 1) in vec3 aColor;
out vec3 vColor;
void main() {
    gl_Position = vec4(aPos, 0.0, 1.0);
    vColor = aColor;
}`

const fragSrc = `#version 300 es
precision mediump float;
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

	// Request an OpenGL ES 3.0 context — this triggers the EGL backend.
	glfw.WindowHint(glfw.ClientAPIs, int(glfw.OpenGLESAPI))
	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 0)

	win, err := glfw.CreateWindow(800, 600, "GLES 3.0 Triangle — EGL/ANGLE", nil, nil)
	if err != nil {
		log.Fatalf("CreateWindow: %v\n\nMake sure libEGL.dll and libGLESv2.dll (ANGLE) are on PATH.", err)
	}
	defer win.Destroy()

	win.MakeContextCurrent()
	glfw.SwapInterval(1)

	// Initialise GLES bindings using glfw's proc-address resolver.
	if err := gl.InitWithProcAddrFunc(func(name string) unsafe.Pointer {
		return glfw.GetProcAddress(name)
	}); err != nil {
		log.Fatalf("gl.Init: %v", err)
	}

	// Print renderer info to confirm we have a real GLES context.
	log.Printf("GLES Renderer : %s", gl.GoStr(gl.GetString(gl.RENDERER)))
	log.Printf("GLES Version  : %s", gl.GoStr(gl.GetString(gl.VERSION)))

	// -------------------------------------------------------------------------
	// GPU resources
	// -------------------------------------------------------------------------

	prog := buildProgram(vertSrc, fragSrc)

	var vao uint32
	gl.GenVertexArrays(1, &vao)
	gl.BindVertexArray(vao)

	var vbo uint32
	gl.GenBuffers(1, &vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(vertices)*4, unsafe.Pointer(&vertices[0]), gl.STATIC_DRAW)

	stride := int32(5 * 4) // 5 float32s × 4 bytes

	// aPos  — location 0, 2 floats
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 2, gl.FLOAT, false, stride, gl.PtrOffset(0))

	// aColor — location 1, 3 floats, offset 8 bytes
	gl.EnableVertexAttribArray(1)
	gl.VertexAttribPointer(1, 3, gl.FLOAT, false, stride, gl.PtrOffset(2*4))

	gl.BindVertexArray(0)

	// -------------------------------------------------------------------------
	// Render loop
	// -------------------------------------------------------------------------

	win.SetKeyCallback(func(w *glfw.Window, key glfw.Key, _ int, action glfw.Action, _ glfw.ModifierKey) {
		if key == glfw.KeyEscape && action == glfw.Press {
			w.SetShouldClose(true)
		}
	})

	win.SetFramebufferSizeCallback(func(_ *glfw.Window, width, height int) {
		gl.Viewport(0, 0, int32(width), int32(height))
	})

	gl.ClearColor(0.1, 0.1, 0.15, 1.0)

	for !win.ShouldClose() {
		glfw.PollEvents()

		gl.Clear(gl.COLOR_BUFFER_BIT)

		gl.UseProgram(prog)
		gl.BindVertexArray(vao)
		gl.DrawArrays(gl.TRIANGLES, 0, 3)
		gl.BindVertexArray(0)

		win.SwapBuffers()
	}
}

// ----------------------------------------------------------------------------
// Local shader helpers (use gles2/v3.0/gl — not glutil which imports v2.1/gl)
// ----------------------------------------------------------------------------

func buildProgram(vertSrc, fragSrc string) uint32 {
	vs := compileShader(gl.VERTEX_SHADER, vertSrc)
	fs := compileShader(gl.FRAGMENT_SHADER, fragSrc)

	prog := gl.CreateProgram()
	gl.AttachShader(prog, vs)
	gl.AttachShader(prog, fs)
	gl.LinkProgram(prog)

	var status int32
	gl.GetProgramiv(prog, gl.LINK_STATUS, &status)
	if status == 0 {
		var logLen int32
		gl.GetProgramiv(prog, gl.INFO_LOG_LENGTH, &logLen)
		logBuf := make([]byte, logLen)
		gl.GetProgramInfoLog(prog, logLen, nil, &logBuf[0])
		log.Fatalf("link: %s", logBuf)
	}

	gl.DeleteShader(vs)
	gl.DeleteShader(fs)
	return prog
}

func compileShader(kind uint32, src string) uint32 {
	shader := gl.CreateShader(kind)
	cstr, free := gl.Strs(src + "\x00")
	gl.ShaderSource(shader, 1, cstr, nil)
	free()
	gl.CompileShader(shader)

	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == 0 {
		var logLen int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLen)
		logBuf := make([]byte, logLen)
		gl.GetShaderInfoLog(shader, logLen, nil, &logBuf[0])
		log.Fatalf("compile shader (kind=%d): %s", kind, logBuf)
	}
	return shader
}

