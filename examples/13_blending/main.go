//go:build windows

// 13_blending — alpha blending and transparency.
//
// Scene layout:
//   - Several opaque cubes on a floor
//   - Five "glass" quads (semi-transparent, alpha = 0.45) floating in the scene
//
// Correct transparency requires drawing opaque objects first, then sorting
// transparent objects back-to-front (farthest from camera first) before
// drawing them with blending enabled.
//
// Key concepts:
//   - gl.Enable(gl.BLEND) + gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
//   - Opaque pass first, no blend needed
//   - Sort transparent objects by distance from camera each frame
//   - Do NOT write to depth buffer from transparent geometry (gl.DepthMask(false))
//
// Controls:  WASD + RMB look,  ESC quit.
//
// Build:
//
//	CGO_ENABLED=0 go build -o 13_blending.exe .
package main

import (
	"fmt"
	"log"
	"sort"

	gl     "github.com/ClaudioTheobaldo/gl-purego/v2.1/gl"
	glfw   "github.com/ClaudioTheobaldo/glfw-purego/v3.3/glfw"
	glutil "github.com/ClaudioTheobaldo/gl-purego/examples/glutil"
)

// ── shaders ──────────────────────────────────────────────────────────────────

const vert = `#version 330 core
layout(location = 0) in vec3 aPos;
layout(location = 1) in vec3 aNormal;
out vec3 vNormal; out vec3 vFragPos;
uniform mat4 uMVP; uniform mat4 uModel;
void main() {
    vec4 w = uModel * vec4(aPos, 1.0);
    vFragPos = w.xyz;
    vNormal  = mat3(transpose(inverse(uModel))) * aNormal;
    gl_Position = uMVP * vec4(aPos, 1.0);
}`

const frag = `#version 330 core
in vec3 vNormal; in vec3 vFragPos;
out vec4 fragColor;
uniform vec4 uColor; uniform vec3 uLightPos; uniform vec3 uViewPos;
void main() {
    vec3 norm  = normalize(vNormal);
    vec3 ldir  = normalize(uLightPos - vFragPos);
    float diff = max(dot(norm, ldir), 0.0);
    vec3 col3  = (0.2 + 0.75*diff) * uColor.rgb;
    fragColor  = vec4(col3, uColor.a);
}`

// ── geometry ──────────────────────────────────────────────────────────────────

// Unit cube with normals.
var cubeVerts = []float32{
	-0.5,-0.5,-0.5, 0,0,-1,  0.5,-0.5,-0.5, 0,0,-1,  0.5,0.5,-0.5, 0,0,-1,
	 0.5,0.5,-0.5, 0,0,-1,  -0.5,0.5,-0.5, 0,0,-1,  -0.5,-0.5,-0.5, 0,0,-1,
	-0.5,-0.5,0.5, 0,0,1,   0.5,-0.5,0.5, 0,0,1,   0.5,0.5,0.5, 0,0,1,
	 0.5,0.5,0.5, 0,0,1,   -0.5,0.5,0.5, 0,0,1,   -0.5,-0.5,0.5, 0,0,1,
	-0.5,0.5,0.5, -1,0,0,  -0.5,0.5,-0.5, -1,0,0, -0.5,-0.5,-0.5, -1,0,0,
	-0.5,-0.5,-0.5, -1,0,0, -0.5,-0.5,0.5, -1,0,0, -0.5,0.5,0.5, -1,0,0,
	 0.5,0.5,0.5, 1,0,0,    0.5,0.5,-0.5, 1,0,0,   0.5,-0.5,-0.5, 1,0,0,
	 0.5,-0.5,-0.5, 1,0,0,  0.5,-0.5,0.5, 1,0,0,   0.5,0.5,0.5, 1,0,0,
	-0.5,-0.5,-0.5, 0,-1,0,  0.5,-0.5,-0.5, 0,-1,0,  0.5,-0.5,0.5, 0,-1,0,
	 0.5,-0.5,0.5, 0,-1,0,  -0.5,-0.5,0.5, 0,-1,0, -0.5,-0.5,-0.5, 0,-1,0,
	-0.5,0.5,-0.5, 0,1,0,   0.5,0.5,-0.5, 0,1,0,   0.5,0.5,0.5, 0,1,0,
	 0.5,0.5,0.5, 0,1,0,   -0.5,0.5,0.5, 0,1,0,   -0.5,0.5,-0.5, 0,1,0,
}

// Flat quad standing upright in XY plane, normal = +Z.
var quadVerts = []float32{
	-0.5,-0.5,0, 0,0,1,
	 0.5,-0.5,0, 0,0,1,
	 0.5, 0.5,0, 0,0,1,
	-0.5,-0.5,0, 0,0,1,
	 0.5, 0.5,0, 0,0,1,
	-0.5, 0.5,0, 0,0,1,
}

// Floor quad (XZ plane).
var floorVerts = []float32{
	-8,-0.5,-8, 0,1,0,   8,-0.5,-8, 0,1,0,   8,-0.5,8, 0,1,0,
	-8,-0.5,-8, 0,1,0,   8,-0.5,8,  0,1,0,  -8,-0.5,8, 0,1,0,
}

// Opaque cube positions.
var opaqueCubes = [][3]float32{
	{-2, 0, -1}, {1.5, 0, -2}, {-0.5, 0, -4}, {2.5, 0, -5},
}

// Transparent glass quad positions + colour (RGBA with alpha < 1).
type glassPane struct {
	pos   [3]float32
	color [4]float32
}

var glassPanes = []glassPane{
	{[3]float32{0, 0.5, -2},   [4]float32{0.2, 0.6, 1.0, 0.45}},
	{[3]float32{-1.5, 0.5, -3.5}, [4]float32{1.0, 0.3, 0.4, 0.45}},
	{[3]float32{1, 0.5, -5},   [4]float32{0.3, 1.0, 0.5, 0.45}},
	{[3]float32{-2, 0.5, -6},  [4]float32{1.0, 0.8, 0.2, 0.45}},
	{[3]float32{2, 0.5, -7},   [4]float32{0.7, 0.3, 1.0, 0.45}},
}

// ── camera ────────────────────────────────────────────────────────────────────

var (
	cam        = glutil.NewCamera([3]float32{0, 1.5, 8})
	winW, winH = 800, 600
	lastTime   = float64(0)
)

func main() {
	cam.Pitch = -10

	if err := glfw.Init(); err != nil { log.Fatalf("glfw.Init: %v", err) }
	defer glfw.Terminate()

	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 3)
	glfw.WindowHint(glfw.OpenGLProfileHint, int(glfw.CoreProfile))
	glfw.WindowHint(glfw.OpenGLForwardCompatible, 1)

	win, err := glfw.CreateWindow(winW, winH, "13 — Blending: opaque first, then sorted transparent quads", nil, nil)
	if err != nil { log.Fatalf("CreateWindow: %v", err) }
	defer win.Destroy()
	win.MakeContextCurrent(); glfw.SwapInterval(1)

	if err := gl.InitWithProcAddrFunc(glfw.GetProcAddress); err != nil { log.Fatalf("gl.Init: %v", err) }

	win.SetFramebufferSizeCallback(func(_ *glfw.Window, w, h int) {
		winW, winH = w, h; gl.Viewport(0, 0, int32(w), int32(h))
	})
	winW, winH = win.GetFramebufferSize()
	gl.Viewport(0, 0, int32(winW), int32(winH))

	win.SetKeyCallback(func(w *glfw.Window, key glfw.Key, _ int, action glfw.Action, _ glfw.ModifierKey) {
		if action == glfw.Press && key == glfw.KeyEscape { w.SetShouldClose(true) }
	})
	win.SetMouseButtonCallback(func(_ *glfw.Window, btn glfw.MouseButton, action glfw.Action, _ glfw.ModifierKey) {
		if btn == glfw.MouseButtonRight { cam.SetRMB(action == glfw.Press) }
	})
	win.SetCursorPosCallback(func(_ *glfw.Window, x, y float64) {
		cam.MousePos(x, y)
	})
	win.SetScrollCallback(func(_ *glfw.Window, _, yoff float64) {
		cam.Scroll(yoff, 0.5, 30)
	})

	gl.Enable(gl.DEPTH_TEST)
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)

	prog, err := glutil.BuildProgram(vert, frag)
	if err != nil { log.Fatalf("shader: %v", err) }
	defer gl.DeleteProgram(prog)

	// Upload all three VBOs.
	makeVAO := func(verts []float32) (vao, vbo uint32) {
		gl.GenVertexArrays(1, &vao); gl.GenBuffers(1, &vbo)
		gl.BindVertexArray(vao)
		gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
		gl.BufferData(gl.ARRAY_BUFFER, len(verts)*4, gl.Ptr(verts), gl.STATIC_DRAW)
		const stride = int32(6 * 4)
		gl.VertexAttribPointer(0, 3, gl.FLOAT, false, stride, gl.PtrOffset(0))
		gl.EnableVertexAttribArray(0)
		gl.VertexAttribPointer(1, 3, gl.FLOAT, false, stride, gl.PtrOffset(12))
		gl.EnableVertexAttribArray(1)
		gl.BindVertexArray(0)
		return
	}
	cubeVAO, cubeVBO := makeVAO(cubeVerts)
	quadVAO, quadVBO := makeVAO(quadVerts)
	floorVAO, floorVBO := makeVAO(floorVerts)
	defer func() {
		gl.DeleteVertexArrays(1, &cubeVAO); gl.DeleteBuffers(1, &cubeVBO)
		gl.DeleteVertexArrays(1, &quadVAO); gl.DeleteBuffers(1, &quadVBO)
		gl.DeleteVertexArrays(1, &floorVAO); gl.DeleteBuffers(1, &floorVBO)
	}()

	uMVP      := gl.GetUniformLocation(prog, gl.Str("uMVP\x00"))
	uModel    := gl.GetUniformLocation(prog, gl.Str("uModel\x00"))
	uColor    := gl.GetUniformLocation(prog, gl.Str("uColor\x00"))
	uLightPos := gl.GetUniformLocation(prog, gl.Str("uLightPos\x00"))
	uViewPos  := gl.GetUniformLocation(prog, gl.Str("uViewPos\x00"))

	lastTime = glfw.GetTime()
	fmt.Println("Walk through the transparent glass panes. WASD + RMB. ESC quit.")

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
		proj := glutil.Perspective(glutil.ToRad(60), float32(winW)/float32(winH), 0.05, 100)
		vp := glutil.MatMul(proj, view)

		gl.ClearColor(0.08, 0.1, 0.12, 1)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

		gl.UseProgram(prog)
		gl.Uniform3f(uLightPos, 3, 5, 3)
		gl.Uniform3f(uViewPos, cam.Pos[0], cam.Pos[1], cam.Pos[2])

		// ── 1. Opaque pass ─────────────────────────────────────────────────
		gl.DepthMask(true) // write depth normally

		// Floor.
		gl.BindVertexArray(floorVAO)
		fm := glutil.Identity()
		fmvp := glutil.MatMul(vp, fm)
		gl.UniformMatrix4fv(uMVP, 1, false, &fmvp[0])
		gl.UniformMatrix4fv(uModel, 1, false, &fm[0])
		gl.Uniform4f(uColor, 0.3, 0.3, 0.3, 1)
		gl.DrawArrays(gl.TRIANGLES, 0, 6)

		// Cubes.
		gl.BindVertexArray(cubeVAO)
		opaqueColors := [][3]float32{{0.8,0.3,0.3},{0.3,0.8,0.3},{0.3,0.3,0.8},{0.8,0.7,0.2}}
		for i, pos := range opaqueCubes {
			model := glutil.Translate3(pos[0], pos[1], pos[2])
			cmvp := glutil.MatMul(vp, model)
			gl.UniformMatrix4fv(uMVP, 1, false, &cmvp[0])
			gl.UniformMatrix4fv(uModel, 1, false, &model[0])
			c := opaqueColors[i%len(opaqueColors)]
			gl.Uniform4f(uColor, c[0], c[1], c[2], 1)
			gl.DrawArrays(gl.TRIANGLES, 0, 36)
		}

		// ── 2. Transparent pass — sort back-to-front ────────────────────────
		sorted := make([]glassPane, len(glassPanes))
		copy(sorted, glassPanes)
		sort.Slice(sorted, func(i, j int) bool {
			di := dist3sq(sorted[i].pos, cam.Pos)
			dj := dist3sq(sorted[j].pos, cam.Pos)
			return di > dj // farthest first
		})

		gl.DepthMask(false) // don't write depth for transparent objects
		gl.BindVertexArray(quadVAO)
		for _, pane := range sorted {
			model := glutil.Translate3(pane.pos[0], pane.pos[1], pane.pos[2])
			pmvp := glutil.MatMul(vp, model)
			gl.UniformMatrix4fv(uMVP, 1, false, &pmvp[0])
			gl.UniformMatrix4fv(uModel, 1, false, &model[0])
			gl.Uniform4f(uColor, pane.color[0], pane.color[1], pane.color[2], pane.color[3])
			gl.DrawArrays(gl.TRIANGLES, 0, 6)
		}
		gl.DepthMask(true)

		gl.BindVertexArray(0)
		win.SwapBuffers()
		glfw.PollEvents()
	}
}

func dist3sq(a, b [3]float32) float32 {
	dx,dy,dz := a[0]-b[0], a[1]-b[1], a[2]-b[2]
	return dx*dx+dy*dy+dz*dz
}
