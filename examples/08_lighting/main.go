//go:build windows

// 08_lighting — Phong shading: ambient + diffuse + specular.
//
// A white point light orbits a coloured cube.  A second tiny cube marks the
// light's position so you can see where it is.  The free-fly camera from
// chapter 07 is included so you can inspect the shading from any angle.
//
// The Phong model computed in the fragment shader:
//
//	ambient  = ambientStrength × lightColor
//	diffuse  = max(dot(normal, lightDir), 0) × lightColor
//	specular = pow(max(dot(reflectDir, viewDir), 0), shininess) × lightColor
//	result   = (ambient + diffuse + specular) × objectColor
//
// Controls:
//
//	WASD / Q / E   — move camera
//	Hold RMB       — look around
//	Mouse wheel    — movement speed
//	ESC            — quit
//
// Build:
//
//	CGO_ENABLED=0 go build -o 08_lighting.exe .
package main

import (
	"fmt"
	"log"
	"math"

	gl     "github.com/ClaudioTheobaldo/gl-purego/v2.1/gl"
	glfw   "github.com/ClaudioTheobaldo/glfw-purego/v3.3/glfw"
	glutil "github.com/ClaudioTheobaldo/gl-purego/examples/glutil"
)

// ── shaders ──────────────────────────────────────────────────────────────────

// Object shader — computes Phong lighting in world space.
const objVert = `#version 330 core
layout(location = 0) in vec3 aPos;
layout(location = 1) in vec3 aNormal;
out vec3 vNormal;
out vec3 vFragPos;
uniform mat4 uModel;
uniform mat4 uView;
uniform mat4 uProjection;
void main() {
    vec4 worldPos = uModel * vec4(aPos, 1.0);
    vFragPos  = worldPos.xyz;
    // Normal matrix: transpose of inverse of the upper-left 3x3 of uModel.
    // For uniform scaling this equals the model matrix — good enough here.
    vNormal   = mat3(transpose(inverse(uModel))) * aNormal;
    gl_Position = uProjection * uView * worldPos;
}`

const objFrag = `#version 330 core
in vec3 vNormal;
in vec3 vFragPos;
out vec4 fragColor;

uniform vec3 uObjectColor;
uniform vec3 uLightColor;
uniform vec3 uLightPos;
uniform vec3 uViewPos;

void main() {
    // Ambient
    float ambientStrength = 0.15;
    vec3 ambient = ambientStrength * uLightColor;

    // Diffuse
    vec3 norm     = normalize(vNormal);
    vec3 lightDir = normalize(uLightPos - vFragPos);
    float diff    = max(dot(norm, lightDir), 0.0);
    vec3 diffuse  = diff * uLightColor;

    // Specular
    float specularStrength = 0.6;
    vec3 viewDir    = normalize(uViewPos - vFragPos);
    vec3 reflectDir = reflect(-lightDir, norm);
    float spec      = pow(max(dot(viewDir, reflectDir), 0.0), 32.0);
    vec3 specular   = specularStrength * spec * uLightColor;

    vec3 result = (ambient + diffuse + specular) * uObjectColor;
    fragColor = vec4(result, 1.0);
}`

// Light-source shader — plain white, unaffected by lighting.
const lightVert = `#version 330 core
layout(location = 0) in vec3 aPos;
uniform mat4 uMVP;
void main() { gl_Position = uMVP * vec4(aPos, 1.0); }`

const lightFrag = `#version 330 core
out vec4 fragColor;
void main() { fragColor = vec4(1.0); }`

// ── geometry: cube with per-face normals ──────────────────────────────────────
// Layout: [X, Y, Z,  NX, NY, NZ]

var cubeVerts = []float32{
	// -Z  normal (0, 0, -1)
	-0.5, -0.5, -0.5, 0, 0, -1,
	0.5, -0.5, -0.5, 0, 0, -1,
	0.5, 0.5, -0.5, 0, 0, -1,
	0.5, 0.5, -0.5, 0, 0, -1,
	-0.5, 0.5, -0.5, 0, 0, -1,
	-0.5, -0.5, -0.5, 0, 0, -1,
	// +Z  normal (0, 0, 1)
	-0.5, -0.5, 0.5, 0, 0, 1,
	0.5, -0.5, 0.5, 0, 0, 1,
	0.5, 0.5, 0.5, 0, 0, 1,
	0.5, 0.5, 0.5, 0, 0, 1,
	-0.5, 0.5, 0.5, 0, 0, 1,
	-0.5, -0.5, 0.5, 0, 0, 1,
	// -X  normal (-1, 0, 0)
	-0.5, 0.5, 0.5, -1, 0, 0,
	-0.5, 0.5, -0.5, -1, 0, 0,
	-0.5, -0.5, -0.5, -1, 0, 0,
	-0.5, -0.5, -0.5, -1, 0, 0,
	-0.5, -0.5, 0.5, -1, 0, 0,
	-0.5, 0.5, 0.5, -1, 0, 0,
	// +X  normal (1, 0, 0)
	0.5, 0.5, 0.5, 1, 0, 0,
	0.5, 0.5, -0.5, 1, 0, 0,
	0.5, -0.5, -0.5, 1, 0, 0,
	0.5, -0.5, -0.5, 1, 0, 0,
	0.5, -0.5, 0.5, 1, 0, 0,
	0.5, 0.5, 0.5, 1, 0, 0,
	// -Y  normal (0, -1, 0)
	-0.5, -0.5, -0.5, 0, -1, 0,
	0.5, -0.5, -0.5, 0, -1, 0,
	0.5, -0.5, 0.5, 0, -1, 0,
	0.5, -0.5, 0.5, 0, -1, 0,
	-0.5, -0.5, 0.5, 0, -1, 0,
	-0.5, -0.5, -0.5, 0, -1, 0,
	// +Y  normal (0, 1, 0)
	-0.5, 0.5, -0.5, 0, 1, 0,
	0.5, 0.5, -0.5, 0, 1, 0,
	0.5, 0.5, 0.5, 0, 1, 0,
	0.5, 0.5, 0.5, 0, 1, 0,
	-0.5, 0.5, 0.5, 0, 1, 0,
	-0.5, 0.5, -0.5, 0, 1, 0,
}

// ── camera state ──────────────────────────────────────────────────────────────

var (
	cam    = glutil.NewCamera([3]float32{0, 1, 4})
	winW, winH = 800, 600
	lastTime   = float64(0)
)

func main() {
	// Non-default initial camera values from the original code.
	cam.Pitch = -10
	cam.Speed = 4

	if err := glfw.Init(); err != nil {
		log.Fatalf("glfw.Init: %v", err)
	}
	defer glfw.Terminate()

	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 3)
	glfw.WindowHint(glfw.OpenGLProfileHint, int(glfw.CoreProfile))
	glfw.WindowHint(glfw.OpenGLForwardCompatible, 1)

	win, err := glfw.CreateWindow(winW, winH, "08 — Phong Lighting: ambient · diffuse · specular", nil, nil)
	if err != nil {
		log.Fatalf("CreateWindow: %v", err)
	}
	defer win.Destroy()
	win.MakeContextCurrent()
	glfw.SwapInterval(1)

	if err := gl.InitWithProcAddrFunc(glfw.GetProcAddress); err != nil {
		log.Fatalf("gl.Init: %v", err)
	}

	win.SetFramebufferSizeCallback(func(_ *glfw.Window, w, h int) {
		winW, winH = w, h; gl.Viewport(0, 0, int32(w), int32(h))
	})
	winW, winH = win.GetFramebufferSize()
	gl.Viewport(0, 0, int32(winW), int32(winH))

	win.SetKeyCallback(func(w *glfw.Window, key glfw.Key, _ int, action glfw.Action, _ glfw.ModifierKey) {
		if action == glfw.Press && key == glfw.KeyEscape {
			w.SetShouldClose(true)
		}
	})
	win.SetMouseButtonCallback(func(_ *glfw.Window, btn glfw.MouseButton, action glfw.Action, _ glfw.ModifierKey) {
		if btn == glfw.MouseButtonRight {
			cam.SetRMB(action == glfw.Press)
		}
	})
	win.SetCursorPosCallback(func(_ *glfw.Window, x, y float64) {
		cam.MousePos(x, y)
	})
	win.SetScrollCallback(func(_ *glfw.Window, _, yoff float64) {
		cam.Scroll(yoff, 0.5, 30)
	})

	gl.Enable(gl.DEPTH_TEST)

	// ── shader programs ────────────────────────────────────────────────────
	objProg, err := glutil.BuildProgram(objVert, objFrag)
	if err != nil { log.Fatalf("obj shader: %v", err) }
	defer gl.DeleteProgram(objProg)

	lightProg, err := glutil.BuildProgram(lightVert, lightFrag)
	if err != nil { log.Fatalf("light shader: %v", err) }
	defer gl.DeleteProgram(lightProg)

	// ── VAO/VBO (shared geometry: unit cube) ───────────────────────────────
	var vao, vbo uint32
	gl.GenVertexArrays(1, &vao)
	gl.GenBuffers(1, &vbo)
	gl.BindVertexArray(vao)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(cubeVerts)*4, gl.Ptr(cubeVerts), gl.STATIC_DRAW)
	const stride = int32(6 * 4)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, stride, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(1, 3, gl.FLOAT, false, stride, gl.PtrOffset(12))
	gl.EnableVertexAttribArray(1)
	gl.BindVertexArray(0)

	// Light cube VAO reuses the same VBO but only binds position (attr 0).
	var lightVAO uint32
	gl.GenVertexArrays(1, &lightVAO)
	gl.BindVertexArray(lightVAO)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, stride, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(0)
	gl.BindVertexArray(0)

	defer func() {
		gl.DeleteVertexArrays(1, &vao)
		gl.DeleteVertexArrays(1, &lightVAO)
		gl.DeleteBuffers(1, &vbo)
	}()

	// Object shader uniform locations.
	uObjModel   := gl.GetUniformLocation(objProg, gl.Str("uModel\x00"))
	uObjView    := gl.GetUniformLocation(objProg, gl.Str("uView\x00"))
	uObjProj    := gl.GetUniformLocation(objProg, gl.Str("uProjection\x00"))
	uObjColor   := gl.GetUniformLocation(objProg, gl.Str("uObjectColor\x00"))
	uLightColor := gl.GetUniformLocation(objProg, gl.Str("uLightColor\x00"))
	uLightPos   := gl.GetUniformLocation(objProg, gl.Str("uLightPos\x00"))
	uViewPos    := gl.GetUniformLocation(objProg, gl.Str("uViewPos\x00"))

	// Light shader uniform location.
	uLightMVP := gl.GetUniformLocation(lightProg, gl.Str("uMVP\x00"))

	lastTime = glfw.GetTime()
	fmt.Println("WASD + RMB to fly. Light orbits the cube. ESC to quit.")

	for !win.ShouldClose() {
		now := glfw.GetTime()
		dt := float32(now - lastTime)
		lastTime = now

		// ── camera ─────────────────────────────────────────────────────────
		cam.HandleKeys(
			win.GetKey(glfw.KeyW) == glfw.Press,
			win.GetKey(glfw.KeyS) == glfw.Press,
			win.GetKey(glfw.KeyA) == glfw.Press,
			win.GetKey(glfw.KeyD) == glfw.Press,
			win.GetKey(glfw.KeyE) == glfw.Press,
			win.GetKey(glfw.KeyQ) == glfw.Press,
			dt,
		)

		// ── orbiting light position ─────────────────────────────────────────
		t := float32(now)
		lightPos := [3]float32{
			float32(math.Cos(float64(t))) * 2.0,
			1.2,
			float32(math.Sin(float64(t))) * 2.0,
		}

		view := cam.ViewMatrix()
		proj := glutil.Perspective(glutil.ToRad(60), float32(winW)/float32(winH), 0.05, 100)

		gl.ClearColor(0.05, 0.05, 0.08, 1)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

		// ── draw main cube ─────────────────────────────────────────────────
		model := glutil.Identity()
		gl.UseProgram(objProg)
		gl.UniformMatrix4fv(uObjModel, 1, false, &model[0])
		gl.UniformMatrix4fv(uObjView, 1, false, &view[0])
		gl.UniformMatrix4fv(uObjProj, 1, false, &proj[0])
		gl.Uniform3f(uObjColor, 0.4, 0.7, 1.0)
		gl.Uniform3f(uLightColor, 1, 1, 1)
		gl.Uniform3f(uLightPos, lightPos[0], lightPos[1], lightPos[2])
		gl.Uniform3f(uViewPos, cam.Pos[0], cam.Pos[1], cam.Pos[2])
		gl.BindVertexArray(vao)
		gl.DrawArrays(gl.TRIANGLES, 0, 36)

		// ── draw light-source cube (small, at light position) ──────────────
		lightModel := glutil.MatMul(glutil.Translate3(lightPos[0], lightPos[1], lightPos[2]), glutil.ScaleU(0.15))
		lightMVP := glutil.MatMul(proj, glutil.MatMul(view, lightModel))
		gl.UseProgram(lightProg)
		gl.UniformMatrix4fv(uLightMVP, 1, false, &lightMVP[0])
		gl.BindVertexArray(lightVAO)
		gl.DrawArrays(gl.TRIANGLES, 0, 36)

		gl.BindVertexArray(0)
		win.SwapBuffers()
		glfw.PollEvents()
	}
}
