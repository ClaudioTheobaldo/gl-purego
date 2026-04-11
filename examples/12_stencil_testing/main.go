//go:build windows

// 12_stencil_testing — object outlining with the stencil buffer.
//
// Three cubes are drawn in the scene.  Selected cubes (toggle with SPACE)
// get a coloured outline drawn via the classic stencil trick:
//
//   Pass 1 — draw object normally, write 1 to stencil everywhere it covers
//   Pass 2 — draw the same object scaled up slightly, but only where stencil == 0
//             (i.e. the border ring outside the original footprint)
//
// This is the same technique used by 3-D DCC tools to highlight selection.
//
// Controls:
//   WASD + RMB   — fly
//   SPACE        — toggle outlines on all cubes
//   ESC          — quit
//
// Build:
//
//	CGO_ENABLED=0 go build -o 12_stencil_testing.exe .
package main

import (
	"fmt"
	"log"

	gl     "github.com/ClaudioTheobaldo/gl-purego/v2.1/gl"
	glfw   "github.com/ClaudioTheobaldo/glfw-purego/v3.3/glfw"
	glutil "github.com/ClaudioTheobaldo/gl-purego/examples/glutil"
)

// ── shaders ──────────────────────────────────────────────────────────────────

const objVert = `#version 330 core
layout(location = 0) in vec3 aPos;
layout(location = 1) in vec3 aNormal;
out vec3 vNormal; out vec3 vFragPos;
uniform mat4 uMVP; uniform mat4 uModel;
void main() {
    vec4 world = uModel * vec4(aPos, 1.0);
    vFragPos = world.xyz;
    vNormal  = mat3(transpose(inverse(uModel))) * aNormal;
    gl_Position = uMVP * vec4(aPos, 1.0);
}`

const objFrag = `#version 330 core
in vec3 vNormal; in vec3 vFragPos;
out vec4 fragColor;
uniform vec3 uColor; uniform vec3 uLightPos; uniform vec3 uViewPos;
void main() {
    vec3 norm  = normalize(vNormal);
    vec3 ldir  = normalize(uLightPos - vFragPos);
    float diff = max(dot(norm, ldir), 0.0);
    vec3 vdir  = normalize(uViewPos - vFragPos);
    vec3 rdir  = reflect(-ldir, norm);
    float spec = pow(max(dot(vdir, rdir), 0.0), 32.0);
    fragColor = vec4((0.15 + 0.7*diff + 0.4*spec) * uColor, 1.0);
}`

// Outline shader — flat colour, no lighting.
const outlineVert = `#version 330 core
layout(location = 0) in vec3 aPos;
uniform mat4 uMVP;
void main() { gl_Position = uMVP * vec4(aPos, 1.0); }`

const outlineFrag = `#version 330 core
out vec4 fragColor;
uniform vec3 uOutlineColor;
void main() { fragColor = vec4(uOutlineColor, 1.0); }`

// ── geometry ──────────────────────────────────────────────────────────────────

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

type cube struct {
	pos     [3]float32
	color   [3]float32
	outline [3]float32
}

var cubes = []cube{
	{[3]float32{-2, 0, 0}, [3]float32{0.5, 0.8, 1.0}, [3]float32{1.0, 0.8, 0.0}},
	{[3]float32{0, 0, 0},  [3]float32{0.8, 0.5, 1.0}, [3]float32{0.0, 1.0, 0.5}},
	{[3]float32{2, 0, 0},  [3]float32{1.0, 0.7, 0.4}, [3]float32{1.0, 0.2, 0.6}},
}

// ── state ─────────────────────────────────────────────────────────────────────

var (
	outlineOn  = false
	cam        = glutil.NewCamera([3]float32{0, 1.5, 6})
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

	win, err := glfw.CreateWindow(winW, winH, stencilTitle(), nil, nil)
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
		if action != glfw.Press { return }
		switch key {
		case glfw.KeyEscape: w.SetShouldClose(true)
		case glfw.KeySpace:
			outlineOn = !outlineOn
			w.SetTitle(stencilTitle())
		}
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
	gl.Enable(gl.STENCIL_TEST)

	objProg, err := glutil.BuildProgram(objVert, objFrag)
	if err != nil { log.Fatalf("obj shader: %v", err) }
	defer gl.DeleteProgram(objProg)

	outlineProg, err := glutil.BuildProgram(outlineVert, outlineFrag)
	if err != nil { log.Fatalf("outline shader: %v", err) }
	defer gl.DeleteProgram(outlineProg)

	var vao, vbo uint32
	gl.GenVertexArrays(1, &vao); gl.GenBuffers(1, &vbo)
	gl.BindVertexArray(vao)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(cubeVerts)*4, gl.Ptr(cubeVerts), gl.STATIC_DRAW)
	const stride = int32(6 * 4)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, stride, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(1, 3, gl.FLOAT, false, stride, gl.PtrOffset(12))
	gl.EnableVertexAttribArray(1)
	gl.BindVertexArray(0)
	defer func() { gl.DeleteVertexArrays(1, &vao); gl.DeleteBuffers(1, &vbo) }()

	uMVP      := gl.GetUniformLocation(objProg, gl.Str("uMVP\x00"))
	uModel    := gl.GetUniformLocation(objProg, gl.Str("uModel\x00"))
	uColor    := gl.GetUniformLocation(objProg, gl.Str("uColor\x00"))
	uLightPos := gl.GetUniformLocation(objProg, gl.Str("uLightPos\x00"))
	uViewPos  := gl.GetUniformLocation(objProg, gl.Str("uViewPos\x00"))
	uOutMVP   := gl.GetUniformLocation(outlineProg, gl.Str("uMVP\x00"))
	uOutColor := gl.GetUniformLocation(outlineProg, gl.Str("uOutlineColor\x00"))

	lastTime = glfw.GetTime()
	fmt.Println("SPACE = toggle outlines.  WASD + RMB to fly.  ESC quit.")

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

		// Always clear colour + depth.  Clear stencil to 0.
		gl.ClearColor(0.06, 0.06, 0.1, 1)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT | gl.STENCIL_BUFFER_BIT)

		gl.BindVertexArray(vao)

		// ── Pass 1: draw cubes normally, stamp stencil = 1 ─────────────────
		// StencilOp: always replace stencil value with ref (1).
		gl.StencilFunc(gl.ALWAYS, 1, 0xFF)
		gl.StencilOp(gl.KEEP, gl.KEEP, gl.REPLACE)
		gl.StencilMask(0xFF)

		gl.UseProgram(objProg)
		gl.Uniform3f(uLightPos, 3, 5, 3)
		gl.Uniform3f(uViewPos, cam.Pos[0], cam.Pos[1], cam.Pos[2])

		for _, c := range cubes {
			model := glutil.Translate3(c.pos[0], c.pos[1], c.pos[2])
			mvp := glutil.MatMul(vp, model)
			gl.UniformMatrix4fv(uMVP, 1, false, &mvp[0])
			gl.UniformMatrix4fv(uModel, 1, false, &model[0])
			gl.Uniform3f(uColor, c.color[0], c.color[1], c.color[2])
			gl.DrawArrays(gl.TRIANGLES, 0, 36)
		}

		// ── Pass 2: draw scaled-up cubes only where stencil != 1 ──────────
		if outlineOn {
			gl.StencilFunc(gl.NOTEQUAL, 1, 0xFF)
			gl.StencilMask(0x00)       // don't write stencil in this pass
			gl.Disable(gl.DEPTH_TEST)  // always draw outline on top

			const outlineScale = float32(1.06)

			gl.UseProgram(outlineProg)
			for _, c := range cubes {
				scm := glutil.ScaleU(outlineScale)
				model := glutil.MatMul(glutil.Translate3(c.pos[0], c.pos[1], c.pos[2]), scm)
				mvp := glutil.MatMul(vp, model)
				gl.UniformMatrix4fv(uOutMVP, 1, false, &mvp[0])
				gl.Uniform3f(uOutColor, c.outline[0], c.outline[1], c.outline[2])
				gl.DrawArrays(gl.TRIANGLES, 0, 36)
			}

			// Restore state.
			gl.StencilMask(0xFF)
			gl.StencilFunc(gl.ALWAYS, 0, 0xFF)
			gl.Enable(gl.DEPTH_TEST)
		}

		gl.BindVertexArray(0)
		win.SwapBuffers()
		glfw.PollEvents()
	}
}

func stencilTitle() string {
	if outlineOn {
		return "12 — Stencil Testing  [outlines ON]  SPACE to toggle"
	}
	return "12 — Stencil Testing  [outlines off]  SPACE to toggle"
}
