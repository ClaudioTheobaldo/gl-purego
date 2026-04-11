//go:build windows

// 09_materials — GLSL material and light structs.
//
// Nine cubes arranged in a 3×3 grid, each with a different material from the
// classic LearnOpenGL table (emerald, gold, obsidian, …).  The light colour
// gently animates so you can watch how each material reacts differently.
//
// Key concepts:
//   - Material struct in GLSL: ambient, diffuse, specular, shininess
//   - Light struct in GLSL: ambient, diffuse, specular components
//   - Separating the light's own colour from the material's response
//
// Controls:  WASD + RMB look,  ESC quit.
//
// Build:
//
//	CGO_ENABLED=0 go build -o 09_materials.exe .
package main

import (
	"fmt"
	"log"
	"math"

	gl     "github.com/ClaudioTheobaldo/gl-purego/v2.1/gl"
	glutil "github.com/ClaudioTheobaldo/gl-purego/examples/glutil"
	glfw   "github.com/ClaudioTheobaldo/glfw-purego/v3.3/glfw"
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
    vec4 worldPos = uModel * vec4(aPos, 1.0);
    vFragPos  = worldPos.xyz;
    vNormal   = mat3(transpose(inverse(uModel))) * aNormal;
    gl_Position = uProjection * uView * worldPos;
}`

const objFrag = `#version 330 core
in vec3 vNormal;
in vec3 vFragPos;
out vec4 fragColor;

struct Material {
    vec3  ambient;
    vec3  diffuse;
    vec3  specular;
    float shininess;
};

struct Light {
    vec3 position;
    vec3 ambient;
    vec3 diffuse;
    vec3 specular;
};

uniform Material uMaterial;
uniform Light    uLight;
uniform vec3     uViewPos;

void main() {
    // Ambient
    vec3 ambient = uLight.ambient * uMaterial.ambient;

    // Diffuse
    vec3 norm     = normalize(vNormal);
    vec3 lightDir = normalize(uLight.position - vFragPos);
    float diff    = max(dot(norm, lightDir), 0.0);
    vec3 diffuse  = uLight.diffuse * (diff * uMaterial.diffuse);

    // Specular
    vec3 viewDir    = normalize(uViewPos - vFragPos);
    vec3 reflectDir = reflect(-lightDir, norm);
    float spec      = pow(max(dot(viewDir, reflectDir), 0.0), uMaterial.shininess);
    vec3 specular   = uLight.specular * (spec * uMaterial.specular);

    fragColor = vec4(ambient + diffuse + specular, 1.0);
}`

const lightVert = `#version 330 core
layout(location = 0) in vec3 aPos;
uniform mat4 uMVP;
void main() { gl_Position = uMVP * vec4(aPos, 1.0); }`

const lightFrag = `#version 330 core
uniform vec3 uColor;
out vec4 fragColor;
void main() { fragColor = vec4(uColor, 1.0); }`

// ── cube geometry with normals ────────────────────────────────────────────────

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

// ── material table ─────────────────────────────────────────────────────────────
// From learnopengl.com/Lighting/Materials

type material struct {
	name      string
	ambient   [3]float32
	diffuse   [3]float32
	specular  [3]float32
	shininess float32
}

var materials = []material{
	{"Emerald",   [3]float32{0.0215, 0.1745, 0.0215},    [3]float32{0.07568, 0.61424, 0.07568}, [3]float32{0.633, 0.727811, 0.633},          76.8},
	{"Gold",      [3]float32{0.24725, 0.1995, 0.0745},   [3]float32{0.75164, 0.60648, 0.22648}, [3]float32{0.628281, 0.555802, 0.366065},     51.2},
	{"Obsidian",  [3]float32{0.05375, 0.05, 0.06625},    [3]float32{0.18275, 0.17, 0.22525},    [3]float32{0.332741, 0.328634, 0.346435},     38.4},
	{"Pearl",     [3]float32{0.25, 0.20725, 0.20725},    [3]float32{1.0, 0.829, 0.829},         [3]float32{0.296648, 0.296648, 0.296648},     11.264},
	{"Ruby",      [3]float32{0.1745, 0.01175, 0.01175},  [3]float32{0.61424, 0.04136, 0.04136}, [3]float32{0.727811, 0.626959, 0.626959},     76.8},
	{"Turquoise", [3]float32{0.1, 0.18725, 0.1745},      [3]float32{0.396, 0.74151, 0.69102},   [3]float32{0.297254, 0.30829, 0.306678},      12.8},
	{"Brass",     [3]float32{0.329412, 0.223529, 0.027451},[3]float32{0.780392, 0.568627, 0.113725},[3]float32{0.992157, 0.941176, 0.807843}, 27.897},
	{"Chrome",    [3]float32{0.25, 0.25, 0.25},          [3]float32{0.4, 0.4, 0.4},             [3]float32{0.774597, 0.774597, 0.774597},     76.8},
	{"Copper",    [3]float32{0.19125, 0.0735, 0.0225},   [3]float32{0.7038, 0.27048, 0.0828},   [3]float32{0.256777, 0.137622, 0.086014},     12.8},
}

// ── camera state ──────────────────────────────────────────────────────────────

var (
	cam            = glutil.NewCamera([3]float32{0, 2, 9})
	winW, winH     = 900, 600
	lastTime       = float64(0)
)

func main() {
	// camPitch was -12 (non-zero), so set it explicitly.
	cam.Pitch = -12

	if err := glfw.Init(); err != nil { log.Fatalf("glfw.Init: %v", err) }
	defer glfw.Terminate()

	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 3)
	glfw.WindowHint(glfw.OpenGLProfileHint, int(glfw.CoreProfile))
	glfw.WindowHint(glfw.OpenGLForwardCompatible, 1)

	win, err := glfw.CreateWindow(winW, winH, "09 — Materials", nil, nil)
	if err != nil { log.Fatalf("CreateWindow: %v", err) }
	defer win.Destroy()
	win.MakeContextCurrent()
	glfw.SwapInterval(1)

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

	objProg, err := glutil.BuildProgram(objVert, objFrag)
	if err != nil { log.Fatalf("obj shader: %v", err) }
	defer gl.DeleteProgram(objProg)

	lightProg, err := glutil.BuildProgram(lightVert, lightFrag)
	if err != nil { log.Fatalf("light shader: %v", err) }
	defer gl.DeleteProgram(lightProg)

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

	// Uniform locations.
	uModel  := gl.GetUniformLocation(objProg, gl.Str("uModel\x00"))
	uView   := gl.GetUniformLocation(objProg, gl.Str("uView\x00"))
	uProj   := gl.GetUniformLocation(objProg, gl.Str("uProjection\x00"))
	uViewP  := gl.GetUniformLocation(objProg, gl.Str("uViewPos\x00"))
	uMatAmb := gl.GetUniformLocation(objProg, gl.Str("uMaterial.ambient\x00"))
	uMatDif := gl.GetUniformLocation(objProg, gl.Str("uMaterial.diffuse\x00"))
	uMatSpc := gl.GetUniformLocation(objProg, gl.Str("uMaterial.specular\x00"))
	uMatShn := gl.GetUniformLocation(objProg, gl.Str("uMaterial.shininess\x00"))
	uLgtAmb := gl.GetUniformLocation(objProg, gl.Str("uLight.ambient\x00"))
	uLgtDif := gl.GetUniformLocation(objProg, gl.Str("uLight.diffuse\x00"))
	uLgtSpc := gl.GetUniformLocation(objProg, gl.Str("uLight.specular\x00"))
	uLgtPos := gl.GetUniformLocation(objProg, gl.Str("uLight.position\x00"))
	uLightMVP   := gl.GetUniformLocation(lightProg, gl.Str("uMVP\x00"))
	uLightColor := gl.GetUniformLocation(lightProg, gl.Str("uColor\x00"))

	lastTime = glfw.GetTime()
	fmt.Println("9 materials: Emerald Gold Obsidian Pearl Ruby Turquoise Brass Chrome Copper")
	fmt.Println("WASD + RMB to fly. ESC to quit.")

	// 3×3 grid positions
	var positions [9][3]float32
	for i := 0; i < 9; i++ {
		positions[i] = [3]float32{float32(i%3)*2.5 - 2.5, 0, 0}
	}

	for !win.ShouldClose() {
		now := glfw.GetTime()
		dt := float32(now - lastTime)
		lastTime = now
		t := float32(now)

		cam.HandleKeys(
			win.GetKey(glfw.KeyW) == glfw.Press,
			win.GetKey(glfw.KeyS) == glfw.Press,
			win.GetKey(glfw.KeyA) == glfw.Press,
			win.GetKey(glfw.KeyD) == glfw.Press,
			win.GetKey(glfw.KeyE) == glfw.Press,
			win.GetKey(glfw.KeyQ) == glfw.Press,
			dt,
		)

		// Animate light colour with a slow cycle.
		lc := [3]float32{
			float32(math.Sin(float64(t)*0.7)*0.5 + 0.5),
			float32(math.Sin(float64(t)*0.3)*0.5 + 0.5),
			float32(math.Sin(float64(t)*0.5)*0.5 + 0.5),
		}
		lightPos := [3]float32{2, 3, 4}

		view := cam.ViewMatrix()
		proj := glutil.Perspective(glutil.ToRad(60), float32(winW)/float32(winH), 0.05, 100)

		gl.ClearColor(0.05, 0.05, 0.08, 1)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

		// ── draw 9 cubes ───────────────────────────────────────────────────
		gl.UseProgram(objProg)
		gl.UniformMatrix4fv(uView, 1, false, &view[0])
		gl.UniformMatrix4fv(uProj, 1, false, &proj[0])
		gl.Uniform3f(uViewP, cam.Pos[0], cam.Pos[1], cam.Pos[2])
		gl.Uniform3f(uLgtPos, lightPos[0], lightPos[1], lightPos[2])
		gl.Uniform3f(uLgtAmb, lc[0]*0.3, lc[1]*0.3, lc[2]*0.3)
		gl.Uniform3f(uLgtDif, lc[0], lc[1], lc[2])
		gl.Uniform3f(uLgtSpc, 1, 1, 1)

		gl.BindVertexArray(vao)
		for i, mat := range materials {
			model := glutil.Translate3(positions[i][0], positions[i][1], positions[i][2])
			gl.UniformMatrix4fv(uModel, 1, false, &model[0])
			gl.Uniform3f(uMatAmb, mat.ambient[0], mat.ambient[1], mat.ambient[2])
			gl.Uniform3f(uMatDif, mat.diffuse[0], mat.diffuse[1], mat.diffuse[2])
			gl.Uniform3f(uMatSpc, mat.specular[0], mat.specular[1], mat.specular[2])
			gl.Uniform1f(uMatShn, mat.shininess)
			gl.DrawArrays(gl.TRIANGLES, 0, 36)
		}

		// ── draw light cube ────────────────────────────────────────────────
		lightModel := glutil.MatMul(glutil.Translate3(lightPos[0], lightPos[1], lightPos[2]), glutil.ScaleU(0.2))
		lightMVP := glutil.MatMul(proj, glutil.MatMul(view, lightModel))
		gl.UseProgram(lightProg)
		gl.UniformMatrix4fv(uLightMVP, 1, false, &lightMVP[0])
		gl.Uniform3f(uLightColor, lc[0], lc[1], lc[2])
		gl.BindVertexArray(lightVAO)
		gl.DrawArrays(gl.TRIANGLES, 0, 36)

		gl.BindVertexArray(0)
		win.SwapBuffers()
		glfw.PollEvents()
	}
}
