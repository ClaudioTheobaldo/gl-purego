//go:build windows

// 10_light_casters — directional light, point light, spotlight.
//
// A scene of 9 cubes lit by three different light types.
// Press 1 / 2 / 3 to switch between them while the scene stays the same.
//
//   1 — Directional light (sun): parallel rays, no falloff, controlled by direction
//   2 — Point light: radiates in all spheres, attenuates with distance (1/(kc + kl·d + kq·d²))
//   3 — Spotlight (flashlight): conical beam attached to the camera, soft outer edge via cosine
//
// Controls:  WASD + RMB look,  1/2/3 to switch light type,  ESC quit.
//
// Build:
//
//	CGO_ENABLED=0 go build -o 10_light_casters.exe .
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

const objVert = `#version 330 core
layout(location = 0) in vec3 aPos;
layout(location = 1) in vec3 aNormal;
out vec3 vNormal;
out vec3 vFragPos;
uniform mat4 uModel;
uniform mat4 uView;
uniform mat4 uProjection;
void main() {
    vec4 worldPos   = uModel * vec4(aPos, 1.0);
    vFragPos        = worldPos.xyz;
    vNormal         = mat3(transpose(inverse(uModel))) * aNormal;
    gl_Position     = uProjection * uView * worldPos;
}`

const objFrag = `#version 330 core
in  vec3 vNormal;
in  vec3 vFragPos;
out vec4 fragColor;

uniform int  uMode;      // 1=directional 2=point 3=spotlight
uniform vec3 uViewPos;

// ---- Directional light ----
uniform vec3 uDirDirection;
uniform vec3 uDirColor;

// ---- Point light ----
uniform vec3  uPointPos;
uniform vec3  uPointColor;
uniform float uPointConstant;
uniform float uPointLinear;
uniform float uPointQuadratic;

// ---- Spotlight ----
uniform vec3  uSpotPos;
uniform vec3  uSpotDir;
uniform vec3  uSpotColor;
uniform float uSpotCutoff;       // cos of inner cone angle
uniform float uSpotOuterCutoff;  // cos of outer cone angle
uniform float uSpotConstant;
uniform float uSpotLinear;
uniform float uSpotQuadratic;

// Object colour (fixed for all cubes in this demo).
const vec3 objectColor = vec3(0.8, 0.6, 0.3);
const float shininess  = 32.0;

vec3 phong(vec3 norm, vec3 lightDir, vec3 lightColor, vec3 viewDir) {
    float diff = max(dot(norm, lightDir), 0.0);
    vec3  ref  = reflect(-lightDir, norm);
    float spec = pow(max(dot(viewDir, ref), 0.0), shininess);
    vec3  amb  = 0.1 * lightColor;
    vec3  dif  = diff * lightColor;
    vec3  spc  = 0.5 * spec * lightColor;
    return (amb + dif + spc) * objectColor;
}

void main() {
    vec3 norm    = normalize(vNormal);
    vec3 viewDir = normalize(uViewPos - vFragPos);
    vec3 color   = vec3(0.0);

    if (uMode == 1) {
        // Directional: light direction is global, no falloff.
        vec3 lightDir = normalize(-uDirDirection);
        color = phong(norm, lightDir, uDirColor, viewDir);

    } else if (uMode == 2) {
        // Point: attenuate by distance.
        vec3  toLight = uPointPos - vFragPos;
        float dist    = length(toLight);
        float att     = 1.0 / (uPointConstant + uPointLinear*dist + uPointQuadratic*dist*dist);
        color = phong(norm, normalize(toLight), uPointColor, viewDir) * att;

    } else {
        // Spotlight: conical beam with soft edge.
        vec3  toLight = uSpotPos - vFragPos;
        float dist    = length(toLight);
        vec3  lightDir = normalize(toLight);
        float att     = 1.0 / (uSpotConstant + uSpotLinear*dist + uSpotQuadratic*dist*dist);
        float theta   = dot(lightDir, normalize(-uSpotDir));
        float eps     = uSpotCutoff - uSpotOuterCutoff;
        float intens  = clamp((theta - uSpotOuterCutoff) / eps, 0.0, 1.0);
        color = phong(norm, lightDir, uSpotColor, viewDir) * att * intens;
    }

    fragColor = vec4(color, 1.0);
}`

// ── cube with normals ─────────────────────────────────────────────────────────

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

var cubePositions = [][3]float32{
	{0,0,0}, {2.5,0,-1}, {-2.5,0,-1},
	{1,0,-3.5}, {-1,0,-3.5}, {3,0,-4},
	{-3,0,-4}, {0,0,-6}, {1.5,0,-7},
}

// ── app state ─────────────────────────────────────────────────────────────────

var (
	lightMode  = 1
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

	win, err := glfw.CreateWindow(winW, winH, lightTitle(), nil, nil)
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
		case glfw.Key1: lightMode = 1; w.SetTitle(lightTitle())
		case glfw.Key2: lightMode = 2; w.SetTitle(lightTitle())
		case glfw.Key3: lightMode = 3; w.SetTitle(lightTitle())
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

	prog, err := glutil.BuildProgram(objVert, objFrag)
	if err != nil { log.Fatalf("shader: %v", err) }
	defer gl.DeleteProgram(prog)

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

	gl.UseProgram(prog)
	// Set attenuation constants once (realistic for ~7 unit range).
	gl.Uniform1f(gl.GetUniformLocation(prog, gl.Str("uPointConstant\x00")),  1.0)
	gl.Uniform1f(gl.GetUniformLocation(prog, gl.Str("uPointLinear\x00")),    0.09)
	gl.Uniform1f(gl.GetUniformLocation(prog, gl.Str("uPointQuadratic\x00")), 0.032)
	gl.Uniform1f(gl.GetUniformLocation(prog, gl.Str("uSpotConstant\x00")),   1.0)
	gl.Uniform1f(gl.GetUniformLocation(prog, gl.Str("uSpotLinear\x00")),     0.09)
	gl.Uniform1f(gl.GetUniformLocation(prog, gl.Str("uSpotQuadratic\x00")),  0.032)
	// Spotlight cone angles.
	gl.Uniform1f(gl.GetUniformLocation(prog, gl.Str("uSpotCutoff\x00")),      float32(math.Cos(12.5*math.Pi/180)))
	gl.Uniform1f(gl.GetUniformLocation(prog, gl.Str("uSpotOuterCutoff\x00")), float32(math.Cos(17.5*math.Pi/180)))

	uModel   := gl.GetUniformLocation(prog, gl.Str("uModel\x00"))
	uView    := gl.GetUniformLocation(prog, gl.Str("uView\x00"))
	uProj    := gl.GetUniformLocation(prog, gl.Str("uProjection\x00"))
	uMode    := gl.GetUniformLocation(prog, gl.Str("uMode\x00"))
	uViewPos := gl.GetUniformLocation(prog, gl.Str("uViewPos\x00"))
	uDirDir  := gl.GetUniformLocation(prog, gl.Str("uDirDirection\x00"))
	uDirCol  := gl.GetUniformLocation(prog, gl.Str("uDirColor\x00"))
	uPtPos   := gl.GetUniformLocation(prog, gl.Str("uPointPos\x00"))
	uPtCol   := gl.GetUniformLocation(prog, gl.Str("uPointColor\x00"))
	uSpPos   := gl.GetUniformLocation(prog, gl.Str("uSpotPos\x00"))
	uSpDir   := gl.GetUniformLocation(prog, gl.Str("uSpotDir\x00"))
	uSpCol   := gl.GetUniformLocation(prog, gl.Str("uSpotColor\x00"))

	lastTime = glfw.GetTime()
	fmt.Println("1=Directional  2=Point  3=Spotlight   WASD+RMB to fly  ESC quit")

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

		gl.ClearColor(0.05, 0.05, 0.08, 1)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

		front := cam.Front()

		gl.UseProgram(prog)
		gl.UniformMatrix4fv(uView, 1, false, &view[0])
		gl.UniformMatrix4fv(uProj, 1, false, &proj[0])
		gl.Uniform3f(uViewPos, cam.Pos[0], cam.Pos[1], cam.Pos[2])
		gl.Uniform1i(uMode, int32(lightMode))

		// Per-mode uniforms.
		gl.Uniform3f(uDirDir, -0.3, -1, -0.5)
		gl.Uniform3f(uDirCol, 1, 0.95, 0.8)
		gl.Uniform3f(uPtPos, 0, 2, 0)
		gl.Uniform3f(uPtCol, 0.4, 0.8, 1)
		gl.Uniform3f(uSpPos, cam.Pos[0], cam.Pos[1], cam.Pos[2])
		gl.Uniform3f(uSpDir, front[0], front[1], front[2])
		gl.Uniform3f(uSpCol, 1, 1, 0.9)

		gl.BindVertexArray(vao)
		for _, pos := range cubePositions {
			model := glutil.Translate3(pos[0], pos[1], pos[2])
			gl.UniformMatrix4fv(uModel, 1, false, &model[0])
			gl.DrawArrays(gl.TRIANGLES, 0, 36)
		}
		gl.BindVertexArray(0)
		win.SwapBuffers()
		glfw.PollEvents()
	}
}

func lightTitle() string {
	names := [4]string{"", "Directional", "Point", "Spotlight"}
	return fmt.Sprintf("10 — Light Casters: [%s]  press 1/2/3 to switch", names[lightMode])
}
