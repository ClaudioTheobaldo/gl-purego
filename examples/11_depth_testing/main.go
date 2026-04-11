//go:build windows

// 11_depth_testing — visualising the depth buffer.
//
// A scene of cubes sits on a flat floor plane.  Press D to toggle between:
//
//   Normal render  — Phong-lit cubes, depth testing enabled (closest wins)
//   Depth view     — raw gl_FragCoord.z linearised to [0,1] shown as grey
//
// The depth view makes it easy to see how fragments far from the camera are
// compressed into a narrow range (non-linear depth), and why the near plane
// value matters so much for precision.
//
// Key concepts:
//   - gl.Enable(gl.DEPTH_TEST), gl.DepthFunc
//   - gl_FragCoord.z in the fragment shader
//   - Linearising the non-linear depth value with near/far
//
// Controls:  WASD + RMB look,  D toggle depth view,  ESC quit.
//
// Build:
//
//	CGO_ENABLED=0 go build -o 11_depth_testing.exe .
package main

import (
	"fmt"
	"log"

	gl     "github.com/ClaudioTheobaldo/gl-purego/v2.1/gl"
	glfw   "github.com/ClaudioTheobaldo/glfw-purego/v3.3/glfw"
	glutil "github.com/ClaudioTheobaldo/gl-purego/examples/glutil"
)

const near = float32(0.1)
const far  = float32(50.0)

// ── shaders ──────────────────────────────────────────────────────────────────

const objVert = `#version 330 core
layout(location = 0) in vec3 aPos;
layout(location = 1) in vec3 aNormal;
out vec3 vNormal;
out vec3 vFragPos;
uniform mat4 uMVP;
uniform mat4 uModel;
void main() {
    vec4 worldPos = uModel * vec4(aPos, 1.0);
    vFragPos      = worldPos.xyz;
    vNormal       = mat3(transpose(inverse(uModel))) * aNormal;
    gl_Position   = uMVP * vec4(aPos, 1.0);
}`

const objFrag = `#version 330 core
in  vec3 vNormal;
in  vec3 vFragPos;
out vec4 fragColor;

uniform bool  uDepthMode;
uniform vec3  uLightPos;
uniform vec3  uViewPos;
uniform vec3  uColor;
uniform float uNear;
uniform float uFar;

float lineariseDepth(float depth) {
    float z = depth * 2.0 - 1.0;   // back to NDC
    return (2.0 * uNear * uFar) / (uFar + uNear - z * (uFar - uNear));
}

void main() {
    if (uDepthMode) {
        float d = lineariseDepth(gl_FragCoord.z) / uFar;
        fragColor = vec4(vec3(d), 1.0);
        return;
    }
    // Simple diffuse + ambient.
    vec3 norm     = normalize(vNormal);
    vec3 lightDir = normalize(uLightPos - vFragPos);
    float diff    = max(dot(norm, lightDir), 0.0);
    vec3 result   = (0.2 + 0.8*diff) * uColor;
    fragColor = vec4(result, 1.0);
}`

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

// Large floor quad (XZ plane, Y=-0.5), normal pointing up.
var floorVerts = []float32{
	-8, -0.5, -8, 0, 1, 0,
	 8, -0.5, -8, 0, 1, 0,
	 8, -0.5,  8, 0, 1, 0,
	-8, -0.5, -8, 0, 1, 0,
	 8, -0.5,  8, 0, 1, 0,
	-8, -0.5,  8, 0, 1, 0,
}

var cubePositions = [][3]float32{
	{0, 0, 0}, {2.5, 0, -2}, {-2, 0, -3},
	{1, 0, -5}, {-3, 0, -5}, {0, 1, -7},
}
var cubeColors = [][3]float32{
	{0.6, 0.3, 0.9}, {0.3, 0.8, 0.5}, {0.9, 0.5, 0.2},
	{0.2, 0.6, 0.9}, {0.9, 0.2, 0.4}, {0.7, 0.9, 0.2},
}

// ── camera & state ────────────────────────────────────────────────────────────

var (
	depthMode  = false
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

	win, err := glfw.CreateWindow(winW, winH, depthTitle(), nil, nil)
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
		case glfw.KeyD:
			depthMode = !depthMode
			w.SetTitle(depthTitle())
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
	gl.DepthFunc(gl.LESS)

	prog, err := glutil.BuildProgram(objVert, objFrag)
	if err != nil { log.Fatalf("shader: %v", err) }
	defer gl.DeleteProgram(prog)

	// Upload cube VBO.
	var cubeVAO, cubeVBO uint32
	gl.GenVertexArrays(1, &cubeVAO); gl.GenBuffers(1, &cubeVBO)
	gl.BindVertexArray(cubeVAO)
	gl.BindBuffer(gl.ARRAY_BUFFER, cubeVBO)
	gl.BufferData(gl.ARRAY_BUFFER, len(cubeVerts)*4, gl.Ptr(cubeVerts), gl.STATIC_DRAW)
	const stride = int32(6 * 4)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, stride, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(1, 3, gl.FLOAT, false, stride, gl.PtrOffset(12))
	gl.EnableVertexAttribArray(1)
	gl.BindVertexArray(0)

	// Upload floor VBO.
	var floorVAO, floorVBO uint32
	gl.GenVertexArrays(1, &floorVAO); gl.GenBuffers(1, &floorVBO)
	gl.BindVertexArray(floorVAO)
	gl.BindBuffer(gl.ARRAY_BUFFER, floorVBO)
	gl.BufferData(gl.ARRAY_BUFFER, len(floorVerts)*4, gl.Ptr(floorVerts), gl.STATIC_DRAW)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, stride, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(1, 3, gl.FLOAT, false, stride, gl.PtrOffset(12))
	gl.EnableVertexAttribArray(1)
	gl.BindVertexArray(0)

	defer func() {
		gl.DeleteVertexArrays(1, &cubeVAO); gl.DeleteBuffers(1, &cubeVBO)
		gl.DeleteVertexArrays(1, &floorVAO); gl.DeleteBuffers(1, &floorVBO)
	}()

	uMVP       := gl.GetUniformLocation(prog, gl.Str("uMVP\x00"))
	uModel     := gl.GetUniformLocation(prog, gl.Str("uModel\x00"))
	uDepthMode := gl.GetUniformLocation(prog, gl.Str("uDepthMode\x00"))
	uLightPos  := gl.GetUniformLocation(prog, gl.Str("uLightPos\x00"))
	uViewPos   := gl.GetUniformLocation(prog, gl.Str("uViewPos\x00"))
	uColor     := gl.GetUniformLocation(prog, gl.Str("uColor\x00"))
	uNear      := gl.GetUniformLocation(prog, gl.Str("uNear\x00"))
	uFarU      := gl.GetUniformLocation(prog, gl.Str("uFar\x00"))

	gl.UseProgram(prog)
	gl.Uniform1f(uNear, near)
	gl.Uniform1f(uFarU, far)

	lastTime = glfw.GetTime()
	fmt.Println("D = toggle depth buffer view.  WASD + RMB to fly.  ESC quit.")

	for !win.ShouldClose() {
		now := glfw.GetTime()
		dt := float32(now - lastTime)
		lastTime = now

		cam.HandleKeys(
			win.GetKey(glfw.KeyW) == glfw.Press,
			win.GetKey(glfw.KeyS) == glfw.Press,
			win.GetKey(glfw.KeyA) == glfw.Press,
			false, // D is used for depth toggle, not strafe
			win.GetKey(glfw.KeyE) == glfw.Press,
			win.GetKey(glfw.KeyQ) == glfw.Press,
			dt,
		)

		view := cam.ViewMatrix()
		proj := glutil.Perspective(glutil.ToRad(60), float32(winW)/float32(winH), near, far)
		vp := glutil.MatMul(proj, view)

		gl.ClearColor(0.08, 0.08, 0.1, 1)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

		depthInt := int32(0)
		if depthMode { depthInt = 1 }

		gl.UseProgram(prog)
		gl.Uniform1i(uDepthMode, depthInt)
		gl.Uniform3f(uLightPos, 2, 5, 3)
		gl.Uniform3f(uViewPos, cam.Pos[0], cam.Pos[1], cam.Pos[2])

		// Draw cubes.
		gl.BindVertexArray(cubeVAO)
		for i, pos := range cubePositions {
			model := glutil.Translate3(pos[0], pos[1], pos[2])
			mvp := glutil.MatMul(vp, model)
			gl.UniformMatrix4fv(uMVP, 1, false, &mvp[0])
			gl.UniformMatrix4fv(uModel, 1, false, &model[0])
			c := cubeColors[i]
			gl.Uniform3f(uColor, c[0], c[1], c[2])
			gl.DrawArrays(gl.TRIANGLES, 0, 36)
		}

		// Draw floor.
		gl.BindVertexArray(floorVAO)
		floorModel := glutil.Identity()
		floorMVP := glutil.MatMul(vp, floorModel)
		gl.UniformMatrix4fv(uMVP, 1, false, &floorMVP[0])
		gl.UniformMatrix4fv(uModel, 1, false, &floorModel[0])
		gl.Uniform3f(uColor, 0.35, 0.35, 0.35)
		gl.DrawArrays(gl.TRIANGLES, 0, 6)

		gl.BindVertexArray(0)
		win.SwapBuffers()
		glfw.PollEvents()
	}
}

func depthTitle() string {
	if depthMode {
		return "11 — Depth Testing  [DEPTH VIEW ON]  D to toggle"
	}
	return "11 — Depth Testing  [normal]  D to toggle depth view"
}
