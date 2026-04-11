//go:build windows

// platonic cycles through the five Platonic solids:
//   Tetrahedron (4 faces) → Cube (6) → Octahedron (8) →
//   Dodecahedron (12) → Icosahedron (20)
//
// Press LEFT / RIGHT to switch solid. Each face gets a distinct colour.
//
// Build:
//
//	CGO_ENABLED=0 go build -o platonic.exe .
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
	winW, winH  int = 800, 600
	solidIdx        = 0
)

// solidDef holds a normalised solid ready for flat-shading.
// tris is a list of triangles already expanded so that all 3 vertices
// of each triangle belonging to the same original face share a face index.
type solidDef struct {
	name  string
	verts [][3]float32
	// Each element: [v0, v1, v2, faceIdx]
	tris [][4]int
}

// faceColors is the palette cycled over when colouring faces.
var faceColors = [][3]float32{
	{1.00, 0.25, 0.25}, // red
	{0.25, 1.00, 0.25}, // green
	{0.25, 0.25, 1.00}, // blue
	{1.00, 1.00, 0.25}, // yellow
	{0.25, 1.00, 1.00}, // cyan
	{1.00, 0.25, 1.00}, // magenta
	{1.00, 0.60, 0.20}, // orange
	{0.60, 0.20, 1.00}, // purple
	{0.20, 0.80, 0.40}, // lime
	{0.80, 0.50, 0.20}, // amber
	{0.30, 0.70, 1.00}, // sky
	{1.00, 0.80, 0.80}, // pink
}

// φ — golden ratio
const phi = 1.6180339887498948482

// normalise scales a vertex slice so every point lies on the unit sphere.
func normalise(vs [][3]float32) [][3]float32 {
	out := make([][3]float32, len(vs))
	for i, v := range vs {
		l := float32(math.Sqrt(float64(v[0]*v[0] + v[1]*v[1] + v[2]*v[2])))
		out[i] = [3]float32{v[0] / l, v[1] / l, v[2] / l}
	}
	return out
}

// trisFan triangulates a convex face polygon (v0..vN-1) as a fan from v[0].
func trisFan(verts []int, face int) [][4]int {
	var t [][4]int
	for i := 1; i < len(verts)-1; i++ {
		t = append(t, [4]int{verts[0], verts[i], verts[i+1], face})
	}
	return t
}

// ---- Tetrahedron -----------------------------------------------------------

var tetra = func() solidDef {
	v := normalise([][3]float32{
		{1, 1, 1}, {1, -1, -1}, {-1, 1, -1}, {-1, -1, 1},
	})
	faces := [][]int{{0, 1, 2}, {0, 3, 1}, {0, 2, 3}, {1, 3, 2}}
	var tris [][4]int
	for fi, f := range faces {
		tris = append(tris, trisFan(f, fi)...)
	}
	return solidDef{"Tetrahedron (4 faces)", v, tris}
}()

// ---- Cube ------------------------------------------------------------------

var cube = func() solidDef {
	v := normalise([][3]float32{
		{-1, -1, -1}, {1, -1, -1}, {1, 1, -1}, {-1, 1, -1},
		{-1, -1, 1}, {1, -1, 1}, {1, 1, 1}, {-1, 1, 1},
	})
	faces := [][]int{
		{0, 3, 2, 1}, // back
		{4, 5, 6, 7}, // front
		{0, 1, 5, 4}, // bottom
		{3, 7, 6, 2}, // top
		{0, 4, 7, 3}, // left
		{1, 2, 6, 5}, // right
	}
	var tris [][4]int
	for fi, f := range faces {
		tris = append(tris, trisFan(f, fi)...)
	}
	return solidDef{"Cube / Hexahedron (6 faces)", v, tris}
}()

// ---- Octahedron ------------------------------------------------------------

var octa = func() solidDef {
	v := [][3]float32{
		{1, 0, 0}, {-1, 0, 0},
		{0, 1, 0}, {0, -1, 0},
		{0, 0, 1}, {0, 0, -1},
	}
	faces := [][]int{
		{4, 0, 2}, {4, 2, 1}, {4, 1, 3}, {4, 3, 0},
		{5, 2, 0}, {5, 1, 2}, {5, 3, 1}, {5, 0, 3},
	}
	var tris [][4]int
	for fi, f := range faces {
		tris = append(tris, trisFan(f, fi)...)
	}
	return solidDef{"Octahedron (8 faces)", v, tris}
}()

// ---- Icosahedron -----------------------------------------------------------

var icosa = func() solidDef {
	p := float32(phi)
	v := normalise([][3]float32{
		{0, 1, p}, {0, -1, p}, {0, 1, -p}, {0, -1, -p},
		{1, p, 0}, {-1, p, 0}, {1, -p, 0}, {-1, -p, 0},
		{p, 0, 1}, {-p, 0, 1}, {p, 0, -1}, {-p, 0, -1},
	})
	faces := [][]int{
		{0, 1, 8}, {0, 8, 4}, {0, 4, 5}, {0, 5, 9}, {0, 9, 1},
		{1, 6, 8}, {8, 6, 10}, {8, 10, 4}, {4, 10, 2}, {4, 2, 5},
		{5, 2, 11}, {5, 11, 9}, {9, 11, 7}, {9, 7, 1}, {1, 7, 6},
		{3, 10, 6}, {3, 6, 7}, {3, 7, 11}, {3, 11, 2}, {3, 2, 10},
	}
	var tris [][4]int
	for fi, f := range faces {
		tris = append(tris, trisFan(f, fi)...)
	}
	return solidDef{"Icosahedron (20 faces)", v, tris}
}()

// ---- Dodecahedron ----------------------------------------------------------

var dodeca = func() solidDef {
	ip := float32(1 / phi) // 1/φ
	p := float32(phi)
	v := normalise([][3]float32{
		// (±1, ±1, ±1)
		{1, 1, 1}, {1, 1, -1}, {1, -1, 1}, {1, -1, -1},
		{-1, 1, 1}, {-1, 1, -1}, {-1, -1, 1}, {-1, -1, -1},
		// (0, ±1/φ, ±φ)
		{0, ip, p}, {0, ip, -p}, {0, -ip, p}, {0, -ip, -p},
		// (±1/φ, ±φ, 0)
		{ip, p, 0}, {ip, -p, 0}, {-ip, p, 0}, {-ip, -p, 0},
		// (±φ, 0, ±1/φ)
		{p, 0, ip}, {p, 0, -ip}, {-p, 0, ip}, {-p, 0, -ip},
	})
	// 12 pentagonal faces (CCW from outside)
	faces := [][]int{
		{0, 8, 10, 2, 16},
		{0, 16, 17, 1, 12},
		{0, 12, 14, 4, 8},
		{4, 18, 6, 10, 8},
		{2, 10, 6, 15, 13},
		{2, 13, 3, 17, 16},
		{1, 17, 3, 11, 9},
		{1, 9, 5, 14, 12},
		{5, 19, 18, 4, 14},
		{7, 15, 6, 18, 19},
		{7, 11, 3, 13, 15},
		{7, 19, 5, 9, 11},
	}
	var tris [][4]int
	for fi, f := range faces {
		tris = append(tris, trisFan(f, fi)...)
	}
	return solidDef{"Dodecahedron (12 faces)", v, tris}
}()

var solids = []solidDef{tetra, cube, octa, dodeca, icosa}

// ----------------------------------------------------------------------------

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

// buildVBO creates a flat vertex buffer [X Y Z R G B] for a solid.
func buildVBO(s solidDef) []float32 {
	buf := make([]float32, 0, len(s.tris)*3*6)
	for _, t := range s.tris {
		col := faceColors[t[3]%len(faceColors)]
		for vi := 0; vi < 3; vi++ {
			p := s.verts[t[vi]]
			buf = append(buf, p[0], p[1], p[2], col[0], col[1], col[2])
		}
	}
	return buf
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

	win, err := glfw.CreateWindow(winW, winH, solidTitle(), nil, nil)
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
			solidIdx = (solidIdx + 1) % len(solids)
			w.SetTitle(solidTitle())
		case glfw.KeyLeft, glfw.KeyDown:
			solidIdx = (solidIdx - 1 + len(solids)) % len(solids)
			w.SetTitle(solidTitle())
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
	// Max: dodecahedron = 36 tris × 3 verts × 6 floats = 648
	gl.BufferData(gl.ARRAY_BUFFER, 648*4, nil, gl.DYNAMIC_DRAW)
	const stride = int32(6 * 4)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, stride, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(1, 3, gl.FLOAT, false, stride, gl.PtrOffset(12))
	gl.EnableVertexAttribArray(1)
	gl.BindVertexArray(0)

	uMVP := gl.GetUniformLocation(prog, gl.Str("uMVP"))

	gl.Enable(gl.DEPTH_TEST)

	fmt.Println("LEFT/RIGHT to cycle solids. ESC to quit.")

	prevIdx := -1
	var vertCount int32

	for !win.ShouldClose() {
		gl.ClearColor(0.08, 0.08, 0.12, 1.0)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

		if solidIdx != prevIdx {
			data := buildVBO(solids[solidIdx])
			gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
			gl.BufferSubData(gl.ARRAY_BUFFER, 0, len(data)*4, unsafe.Pointer(&data[0]))
			vertCount = int32(len(data) / 6)
			prevIdx = solidIdx
		}

		t := float32(glfw.GetTime())
		model := matMul(rotX(t*0.4), rotY(t*0.6))
		view  := lookAt(0, 1.2, 2.8, 0, 0, 0, 0, 1, 0)
		proj  := perspective(toRad(45), float32(winW)/float32(winH), 0.1, 100)
		mvp   := matMul(proj, matMul(view, model))

		gl.UseProgram(prog)
		gl.UniformMatrix4fv(uMVP, 1, false, &mvp[0])
		gl.BindVertexArray(vao)
		gl.DrawArrays(gl.TRIANGLES, 0, vertCount)
		gl.BindVertexArray(0)

		win.SwapBuffers()
		glfw.PollEvents()
	}
}

func solidTitle() string {
	return fmt.Sprintf("%s — press LEFT/RIGHT", solids[solidIdx].name)
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

func perspective(fovY, aspect, near, far float32) [16]float32 {
	f := float32(1.0 / math.Tan(float64(fovY)/2.0))
	return [16]float32{
		f / aspect, 0, 0, 0,
		0, f, 0, 0,
		0, 0, -(far + near) / (far - near), -1,
		0, 0, -2 * far * near / (far - near), 0,
	}
}

func lookAt(ex, ey, ez, cx, cy, cz, ux, uy, uz float32) [16]float32 {
	fx := cx - ex; fy := cy - ey; fz := cz - ez
	fl := float32(math.Sqrt(float64(fx*fx + fy*fy + fz*fz)))
	fx /= fl; fy /= fl; fz /= fl
	rx := fy*uz - fz*uy; ry := fz*ux - fx*uz; rz := fx*uy - fy*ux
	rl := float32(math.Sqrt(float64(rx*rx + ry*ry + rz*rz)))
	rx /= rl; ry /= rl; rz /= rl
	upx := ry*fz - rz*fy; upy := rz*fx - rx*fz; upz := rx*fy - ry*fx
	return [16]float32{
		rx, upx, -fx, 0, ry, upy, -fy, 0, rz, upz, -fz, 0,
		-(rx*ex + ry*ey + rz*ez), -(upx*ex + upy*ey + upz*ez), fx*ex + fy*ey + fz*ez, 1,
	}
}

func rotX(a float32) [16]float32 {
	c, s := float32(math.Cos(float64(a))), float32(math.Sin(float64(a)))
	return [16]float32{1, 0, 0, 0, 0, c, s, 0, 0, -s, c, 0, 0, 0, 0, 1}
}

func rotY(a float32) [16]float32 {
	c, s := float32(math.Cos(float64(a))), float32(math.Sin(float64(a)))
	return [16]float32{c, 0, -s, 0, 0, 1, 0, 0, s, 0, c, 0, 0, 0, 0, 1}
}

func toRad(d float32) float32 { return d * math.Pi / 180 }

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
	gl.AttachShader(p, v); gl.AttachShader(p, f)
	gl.LinkProgram(p)
	gl.DeleteShader(v); gl.DeleteShader(f)
	var st int32
	gl.GetProgramiv(p, gl.LINK_STATUS, &st)
	if st == gl.FALSE {
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
	var st int32
	gl.GetShaderiv(sh, gl.COMPILE_STATUS, &st)
	if st == gl.FALSE {
		var n int32
		gl.GetShaderiv(sh, gl.INFO_LOG_LENGTH, &n)
		buf := make([]uint8, n+1)
		gl.GetShaderInfoLog(sh, n, nil, &buf[0])
		gl.DeleteShader(sh)
		return 0, fmt.Errorf("%s", buf)
	}
	return sh, nil
}
