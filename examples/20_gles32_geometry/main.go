//go:build windows

// 20_gles32_geometry demonstrates GLES 3.2 geometry shaders via ANGLE.
// It renders a colour-interpolated triangle (base pass), then uses a geometry
// shader to draw short yellow lines outward from each edge midpoint, visually
// indicating the face normal direction.
//
// ANGLE must be available on PATH or in the same directory as the executable.
//
// Build (CGO disabled):
//
//	CGO_ENABLED=0 go build -o gles32_geometry.exe .
package main

import (
	"log"
	"os"
	"strings"
	"unsafe"

	gl   "github.com/ClaudioTheobaldo/gl-purego/v3.2/gles2"
	glfw "github.com/ClaudioTheobaldo/glfw-purego/v3.3/glfw"
)

// Triangle: interleaved position (vec2) + colour (vec3).
var vertices = []float32{
	//  X      Y     R     G     B
	0.00, 0.75, 1.0, 0.25, 0.25, // top    — red
	-0.65, -0.50, 0.25, 1.0, 0.25, // left   — green
	0.65, -0.50, 0.25, 0.25, 1.0, // right  — blue
}

// posOnly extracts just the XY position for the geometry shader pass.
var posOnly = []float32{
	0.00, 0.75,
	-0.65, -0.50,
	0.65, -0.50,
}

// ---- Base pass shaders (#version 320 es) ------------------------------------

const baseVertSrc = `#version 320 es
layout(location = 0) in vec2 aPos;
layout(location = 1) in vec3 aColor;
out vec3 vColor;
void main() {
    gl_Position = vec4(aPos, 0.0, 1.0);
    vColor = aColor;
}`

const baseFragSrc = `#version 320 es
precision mediump float;
in  vec3 vColor;
out vec4 fragColor;
void main() {
    fragColor = vec4(vColor, 1.0);
}`

// ---- Geometry-shader pass (#version 320 es) ---------------------------------

// Geometry vertex shader: passes position straight through.
const geomVertSrc = `#version 320 es
layout(location = 0) in vec2 aPos;
void main() {
    gl_Position = vec4(aPos, 0.0, 1.0);
}`

// Geometry shader: emits a short yellow line at each edge midpoint.
const geomSrc = `#version 320 es
layout(triangles) in;
layout(line_strip, max_vertices = 6) out;
out vec4 gColor;
void main() {
    vec4 yellow = vec4(1.0, 1.0, 0.0, 1.0);
    // Compute face normal from first two edges (2-D cross product).
    vec2 e1 = gl_in[1].gl_Position.xy - gl_in[0].gl_Position.xy;
    vec2 e2 = gl_in[2].gl_Position.xy - gl_in[0].gl_Position.xy;
    vec2 n  = normalize(vec2(-e1.y + e2.y, e1.x - e2.x) * 0.5);
    for (int i = 0; i < 3; i++) {
        int j = (i + 1) % 3;
        vec4 mid = (gl_in[i].gl_Position + gl_in[j].gl_Position) * 0.5;
        gColor = yellow;
        gl_Position = mid;
        EmitVertex();
        gl_Position = mid + vec4(n * 0.15, 0.0, 0.0);
        EmitVertex();
        EndPrimitive();
    }
}`

const geomFragSrc = `#version 320 es
precision mediump float;
in  vec4 gColor;
out vec4 fragColor;
void main() {
    fragColor = gColor;
}`

func main() {
	// Force ANGLE to use its Vulkan backend (D3D11 backend caps at GLES 3.0;
	// GLES 3.2 geometry shaders require Vulkan).
	os.Setenv("ANGLE_DEFAULT_PLATFORM", "vulkan")

	if err := glfw.Init(); err != nil {
		log.Fatalf("glfw.Init: %v", err)
	}
	defer glfw.Terminate()

	glfw.WindowHint(glfw.ClientAPIs, int(glfw.OpenGLESAPI))
	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 2)

	win, err := glfw.CreateWindow(800, 600, "GLES 3.2 Geometry Shader — EGL/ANGLE", nil, nil)
	if err != nil {
		log.Fatalf("CreateWindow: %v\n\nMake sure libEGL.dll and libGLESv2.dll (ANGLE) are on PATH.", err)
	}
	defer win.Destroy()

	win.MakeContextCurrent()
	glfw.SwapInterval(1)

	if err := gl.InitWithProcAddrFunc(func(name string) unsafe.Pointer {
		return glfw.GetProcAddress(name)
	}); err != nil {
		log.Fatalf("gl.Init: %v", err)
	}

	renderer := gl.GoStr(gl.GetString(gl.RENDERER))
	version  := gl.GoStr(gl.GetString(gl.VERSION))
	log.Printf("GLES Renderer : %s", renderer)
	log.Printf("GLES Version  : %s", version)

	// Geometry shaders require GLES 3.2. Browser-shipped ANGLE caps at GLES 3.0.
	if !strings.Contains(version, "3.2") {
		log.Fatalf("GLES 3.2 required for geometry shaders; got: %s\n"+
			"Use a standalone ANGLE build (not browser-shipped) or run on Linux with Mesa.", version)
	}

	// -------------------------------------------------------------------------
	// Base pass: coloured triangle (interleaved pos+color).
	// -------------------------------------------------------------------------
	baseProg := buildProgram(baseVertSrc, "", baseFragSrc)

	var baseVAO, baseVBO uint32
	gl.GenVertexArrays(1, &baseVAO)
	gl.BindVertexArray(baseVAO)

	gl.GenBuffers(1, &baseVBO)
	gl.BindBuffer(gl.ARRAY_BUFFER, baseVBO)
	gl.BufferData(gl.ARRAY_BUFFER, len(vertices)*4, unsafe.Pointer(&vertices[0]), gl.STATIC_DRAW)

	stride := int32(5 * 4) // 5 float32s × 4 bytes

	// aPos   — location 0, 2 floats, offset 0
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 2, gl.FLOAT, false, stride, gl.PtrOffset(0))

	// aColor — location 1, 3 floats, offset 8 bytes
	gl.EnableVertexAttribArray(1)
	gl.VertexAttribPointer(1, 3, gl.FLOAT, false, stride, gl.PtrOffset(2*4))

	gl.BindVertexArray(0)

	// -------------------------------------------------------------------------
	// Geometry-shader pass: position-only VBO fed into vert+geom+frag program.
	// -------------------------------------------------------------------------
	geomProg := buildProgram(geomVertSrc, geomSrc, geomFragSrc)

	var geomVAO, geomVBO uint32
	gl.GenVertexArrays(1, &geomVAO)
	gl.BindVertexArray(geomVAO)

	gl.GenBuffers(1, &geomVBO)
	gl.BindBuffer(gl.ARRAY_BUFFER, geomVBO)
	gl.BufferData(gl.ARRAY_BUFFER, len(posOnly)*4, unsafe.Pointer(&posOnly[0]), gl.STATIC_DRAW)

	// aPos — location 0, 2 floats
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 2, gl.FLOAT, false, int32(2*4), gl.PtrOffset(0))

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

		// Pass 1: draw the coloured triangle.
		gl.UseProgram(baseProg)
		gl.BindVertexArray(baseVAO)
		gl.DrawArrays(gl.TRIANGLES, 0, 3)
		gl.BindVertexArray(0)

		// Pass 2: draw the normal-visualisation lines via geometry shader.
		gl.UseProgram(geomProg)
		gl.BindVertexArray(geomVAO)
		gl.DrawArrays(gl.TRIANGLES, 0, 3)
		gl.BindVertexArray(0)

		win.SwapBuffers()
	}
}

// ----------------------------------------------------------------------------
// Local shader helpers (use v3.2/gles2 — not glutil which imports v2.1/gl)
// buildProgram accepts an optional geometry shader source (pass "" to skip).
// ----------------------------------------------------------------------------

func buildProgram(vertSrc, geomSrc, fragSrc string) uint32 {
	vs := compileShader(gl.VERTEX_SHADER, vertSrc)
	fs := compileShader(gl.FRAGMENT_SHADER, fragSrc)

	prog := gl.CreateProgram()
	gl.AttachShader(prog, vs)

	var gs uint32
	if geomSrc != "" {
		gs = compileShader(gl.GEOMETRY_SHADER, geomSrc)
		gl.AttachShader(prog, gs)
	}

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
	if gs != 0 {
		gl.DeleteShader(gs)
	}
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
