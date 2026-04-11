//go:build windows

// polygon cycles through regular N-gons from triangle (3) to icosagon (20).
// Press LEFT / RIGHT arrows to change the vertex count.
// Each polygon spins slowly and is filled with a rainbow gradient.
//
// Build:
//
//	CGO_ENABLED=0 go build -o polygon.exe .
package main

import (
	"fmt"
	"log"
	"math"
	"unsafe"

	gl   "github.com/ClaudioTheobaldo/gl-purego/v2.1/gl"
	glfw "github.com/ClaudioTheobaldo/glfw-purego/v3.3/glfw"
)

var (
	winW, winH int = 800, 600
	sides          = 6
)

const vertSrc = `#version 330 core
layout(location = 0) in vec2 aPos;
layout(location = 1) in vec3 aColor;
out vec3 vColor;
uniform mat4 uMVP;
void main() {
    gl_Position = uMVP * vec4(aPos, 0.0, 1.0);
    vColor = aColor;
}`

const fragSrc = `#version 330 core
in  vec3 vColor;
out vec4 fragColor;
void main() {
    fragColor = vec4(vColor, 1.0);
}`

var names = []string{
	"", "", "",
	"Triangle", "Quadrilateral", "Pentagon", "Hexagon", "Heptagon",
	"Octagon", "Nonagon", "Decagon", "Hendecagon", "Dodecagon",
	"Tridecagon", "Tetradecagon", "Pentadecagon", "Hexadecagon",
	"Heptadecagon", "Octadecagon", "Enneadecagon", "Icosagon",
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

	win, err := glfw.CreateWindow(winW, winH, title(), nil, nil)
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
		if action != glfw.Press && action != glfw.Repeat {
			return
		}
		switch key {
		case glfw.KeyEscape:
			w.SetShouldClose(true)
		case glfw.KeyRight, glfw.KeyUp:
			if sides < 20 {
				sides++
				w.SetTitle(title())
			}
		case glfw.KeyLeft, glfw.KeyDown:
			if sides > 3 {
				sides--
				w.SetTitle(title())
			}
		}
	})
	win.SetFramebufferSizeCallback(func(w *glfw.Window, width, height int) {
		winW, winH = width, height
		gl.Viewport(0, 0, int32(width), int32(height))
	})
	winW, winH = win.GetFramebufferSize()
	gl.Viewport(0, 0, int32(winW), int32(winH))

	prog, err := buildProgram(vertSrc, fragSrc)
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
	// Pre-allocate for max polygon (20 + 2 verts) × 5 floats
	gl.BufferData(gl.ARRAY_BUFFER, 22*5*4, nil, gl.DYNAMIC_DRAW)
	const stride = int32(5 * 4) // 2 pos + 3 color
	gl.VertexAttribPointer(0, 2, gl.FLOAT, false, stride, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(1, 3, gl.FLOAT, false, stride, gl.PtrOffset(8))
	gl.EnableVertexAttribArray(1)
	gl.BindVertexArray(0)

	uMVP := gl.GetUniformLocation(prog, gl.Str("uMVP"))

	fmt.Println("LEFT/RIGHT arrows change the polygon. ESC to quit.")

	prevSides := 0
	for !win.ShouldClose() {
		gl.ClearColor(0.08, 0.08, 0.12, 1.0)
		gl.Clear(gl.COLOR_BUFFER_BIT)

		// Rebuild VBO only when sides changes
		if sides != prevSides {
			verts := buildPolygon(sides)
			gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
			gl.BufferSubData(gl.ARRAY_BUFFER, 0, len(verts)*4, unsafe.Pointer(&verts[0]))
			prevSides = sides
		}

		t := float32(glfw.GetTime())
		aspect := float32(winW) / float32(winH)
		mvp := ortho(-aspect, aspect, -1, 1, -1, 1)
		// embed 2D rotation into the XY plane of the MVP
		rot := rotZ(t * 0.4)
		mvp = matMul(mvp, rot)

		gl.UseProgram(prog)
		gl.UniformMatrix4fv(uMVP, 1, false, &mvp[0])
		gl.BindVertexArray(vao)
		gl.DrawArrays(gl.TRIANGLE_FAN, 0, int32(sides+2))
		gl.BindVertexArray(0)

		win.SwapBuffers()
		glfw.PollEvents()
	}
}

func title() string {
	return fmt.Sprintf("%s (%d vertices) — press LEFT/RIGHT", names[sides], sides)
}

// buildPolygon returns a TRIANGLE_FAN vertex buffer:
//   [center] [v0] [v1] ... [vN-1] [v0]   (N+2 vertices)
//
// Layout per vertex: [x, y, r, g, b]
func buildPolygon(n int) []float32 {
	buf := make([]float32, 0, (n+2)*5)
	// Center — white
	buf = append(buf, 0, 0, 1, 1, 1)
	// Outer ring — rainbow, one hue per vertex
	for i := 0; i <= n; i++ {
		angle := float32(i) * 2 * math.Pi / float32(n)
		x := float32(math.Cos(float64(angle))) * 0.85
		y := float32(math.Sin(float64(angle))) * 0.85
		r, g, b := hsvToRGB(float32(i) / float32(n))
		buf = append(buf, x, y, r, g, b)
	}
	return buf
}

func hsvToRGB(h float32) (r, g, b float32) {
	h *= 6
	i := int(h) % 6
	f := h - float32(int(h))
	switch i {
	case 0:
		return 1, f, 0
	case 1:
		return 1 - f, 1, 0
	case 2:
		return 0, 1, f
	case 3:
		return 0, 1 - f, 1
	case 4:
		return f, 0, 1
	default:
		return 1, 0, 1 - f
	}
}

// ----------------------------------------------------------------------------
// Matrix math (column-major)
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

func ortho(left, right, bottom, top, near, far float32) [16]float32 {
	return [16]float32{
		2 / (right - left), 0, 0, 0,
		0, 2 / (top - bottom), 0, 0,
		0, 0, -2 / (far - near), 0,
		-(right + left) / (right - left),
		-(top + bottom) / (top - bottom),
		-(far + near) / (far - near),
		1,
	}
}

func rotZ(a float32) [16]float32 {
	c := float32(math.Cos(float64(a)))
	s := float32(math.Sin(float64(a)))
	return [16]float32{c, s, 0, 0, -s, c, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}
}

// ----------------------------------------------------------------------------
// Shader helpers
// ----------------------------------------------------------------------------

func buildProgram(vs, fs string) (uint32, error) {
	v, err := compileShader(vs, gl.VERTEX_SHADER)
	if err != nil {
		return 0, fmt.Errorf("vertex: %w", err)
	}
	f, err := compileShader(fs, gl.FRAGMENT_SHADER)
	if err != nil {
		gl.DeleteShader(v)
		return 0, fmt.Errorf("fragment: %w", err)
	}
	p := gl.CreateProgram()
	gl.AttachShader(p, v)
	gl.AttachShader(p, f)
	gl.LinkProgram(p)
	gl.DeleteShader(v)
	gl.DeleteShader(f)
	var status int32
	gl.GetProgramiv(p, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var n int32
		gl.GetProgramiv(p, gl.INFO_LOG_LENGTH, &n)
		buf := make([]uint8, n+1)
		gl.GetProgramInfoLog(p, n, nil, &buf[0])
		gl.DeleteProgram(p)
		return 0, fmt.Errorf("link: %s", buf)
	}
	return p, nil
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
		return 0, fmt.Errorf("%s", buf)
	}
	return sh, nil
}
