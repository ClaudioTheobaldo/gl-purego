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
	"math"
	"unsafe"

	gl   "github.com/ClaudioTheobaldo/gl-purego/v2.1/gl"
	glfw "github.com/ClaudioTheobaldo/glfw-purego/v3.3/glfw"
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
	prog, err := buildProgram(vertSrc, fragSrc)
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
		model := matMul(rotX(t*0.5), rotY(t*0.7))
		view  := lookAt(0, 1.5, 3, 0, 0, 0, 0, 1, 0)
		proj  := perspective(toRad(45), float32(winW)/float32(winH), 0.1, 100)
		mvp   := matMul(proj, matMul(view, model))

		gl.UseProgram(prog)
		gl.UniformMatrix4fv(uMVP, 1, false, &mvp[0])
		gl.BindVertexArray(vao)
		gl.DrawArrays(gl.TRIANGLES, 0, 36)
		gl.BindVertexArray(0)

		win.SwapBuffers()
		glfw.PollEvents()
	}
}

// ----------------------------------------------------------------------------
// Matrix math (column-major, OpenGL convention)
// ----------------------------------------------------------------------------

func matMul(a, b [16]float32) [16]float32 {
	var m [16]float32
	for col := 0; col < 4; col++ {
		for row := 0; row < 4; row++ {
			var v float32
			for k := 0; k < 4; k++ {
				v += a[k*4+row] * b[col*4+k]
			}
			m[col*4+row] = v
		}
	}
	return m
}

func perspective(fovY, aspect, near, far float32) [16]float32 {
	f := float32(1.0 / math.Tan(float64(fovY)/2.0))
	return [16]float32{
		f / aspect, 0, 0, 0,
		0, f, 0, 0,
		0, 0, -(far + near) / (far - near), -1,
		0, 0, -2 * far * near / (far - near), 0,
	}
}

func lookAt(eyeX, eyeY, eyeZ, cX, cY, cZ, upX, upY, upZ float32) [16]float32 {
	fx := cX - eyeX; fy := cY - eyeY; fz := cZ - eyeZ
	fl := float32(math.Sqrt(float64(fx*fx + fy*fy + fz*fz)))
	fx /= fl; fy /= fl; fz /= fl

	rx := fy*upZ - fz*upY
	ry := fz*upX - fx*upZ
	rz := fx*upY - fy*upX
	rl := float32(math.Sqrt(float64(rx*rx + ry*ry + rz*rz)))
	rx /= rl; ry /= rl; rz /= rl

	ux := ry*fz - rz*fy
	uy := rz*fx - rx*fz
	uz := rx*fy - ry*fx

	return [16]float32{
		rx, ux, -fx, 0,
		ry, uy, -fy, 0,
		rz, uz, -fz, 0,
		-(rx*eyeX + ry*eyeY + rz*eyeZ),
		-(ux*eyeX + uy*eyeY + uz*eyeZ),
		fx*eyeX + fy*eyeY + fz*eyeZ,
		1,
	}
}

func rotX(a float32) [16]float32 {
	c := float32(math.Cos(float64(a)))
	s := float32(math.Sin(float64(a)))
	return [16]float32{1, 0, 0, 0, 0, c, s, 0, 0, -s, c, 0, 0, 0, 0, 1}
}

func rotY(a float32) [16]float32 {
	c := float32(math.Cos(float64(a)))
	s := float32(math.Sin(float64(a)))
	return [16]float32{c, 0, -s, 0, 0, 1, 0, 0, s, 0, c, 0, 0, 0, 0, 1}
}

func toRad(deg float32) float32 { return deg * math.Pi / 180 }

// ----------------------------------------------------------------------------
// Shader helpers (shared with other examples)
// ----------------------------------------------------------------------------

func buildProgram(vertSrc, fragSrc string) (uint32, error) {
	vs, err := compileShader(vertSrc, gl.VERTEX_SHADER)
	if err != nil {
		return 0, fmt.Errorf("vertex: %w", err)
	}
	fs, err := compileShader(fragSrc, gl.FRAGMENT_SHADER)
	if err != nil {
		gl.DeleteShader(vs)
		return 0, fmt.Errorf("fragment: %w", err)
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
		var n int32
		gl.GetProgramiv(prog, gl.INFO_LOG_LENGTH, &n)
		buf := make([]uint8, n+1)
		gl.GetProgramInfoLog(prog, n, nil, &buf[0])
		gl.DeleteProgram(prog)
		return 0, fmt.Errorf("link: %s", string(buf))
	}
	return prog, nil
}

func compileShader(src string, kind uint32) (uint32, error) {
	sh := gl.CreateShader(kind)
	cstr, free := gl.Strs(src)
	gl.ShaderSource(sh, 1, cstr, nil)
	free()
	gl.CompileShader(sh)
	var status int32
	gl.GetShaderiv(sh, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var n int32
		gl.GetShaderiv(sh, gl.INFO_LOG_LENGTH, &n)
		buf := make([]uint8, n+1)
		gl.GetShaderInfoLog(sh, n, nil, &buf[0])
		gl.DeleteShader(sh)
		return 0, fmt.Errorf("%s", string(buf))
	}
	return sh, nil
}
