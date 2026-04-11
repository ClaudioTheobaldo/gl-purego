//go:build windows

// 14_framebuffers — render to texture + post-processing effects.
//
// The entire scene is rendered into an off-screen Framebuffer Object (FBO).
// The FBO's colour attachment is then drawn as a fullscreen quad in the
// default framebuffer, passing through a post-process fragment shader.
//
// Press 1-4 to cycle between effects:
//
//	1 — Normal (no effect)
//	2 — Inversion (1.0 - colour)
//	3 — Grayscale  (luminance = dot(colour, vec3(0.2126, 0.7152, 0.0722)))
//	4 — Edge detection kernel (Laplacian 3×3 convolution on the texture)
//
// Key concepts:
//   - glGenFramebuffers / glFramebufferTexture2D
//   - Render-to-texture: bind FBO → render scene → unbind FBO
//   - Post-process with a fullscreen quad sampling the colour texture
//   - Kernel convolution (sampling 9 neighbours) in the fragment shader
//
// Controls:  WASD + RMB look,  1/2/3/4 effects,  ESC quit.
//
// Build:
//
//	CGO_ENABLED=0 go build -o 14_framebuffers.exe .
package main

import (
	"fmt"
	"log"

	gl     "github.com/ClaudioTheobaldo/gl-purego/v2.1/gl"
	glfw   "github.com/ClaudioTheobaldo/glfw-purego/v3.3/glfw"
	glutil "github.com/ClaudioTheobaldo/gl-purego/examples/glutil"
)

// ── scene shaders ─────────────────────────────────────────────────────────────

const sceneVert = `#version 330 core
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

const sceneFrag = `#version 330 core
in vec3 vNormal; in vec3 vFragPos;
out vec4 fragColor;
uniform vec3 uColor; uniform vec3 uLightPos; uniform vec3 uViewPos;
void main() {
    vec3 n    = normalize(vNormal);
    vec3 l    = normalize(uLightPos - vFragPos);
    float d   = max(dot(n, l), 0.0);
    vec3 v    = normalize(uViewPos - vFragPos);
    float s   = pow(max(dot(v, reflect(-l, n)), 0.0), 32.0);
    fragColor = vec4((0.15 + 0.7*d + 0.4*s) * uColor, 1.0);
}`

// ── post-process shaders ──────────────────────────────────────────────────────

const ppVert = `#version 330 core
layout(location = 0) in vec2 aPos;
layout(location = 1) in vec2 aUV;
out vec2 vUV;
void main() { gl_Position = vec4(aPos, 0.0, 1.0); vUV = aUV; }`

const ppFrag = `#version 330 core
in  vec2 vUV;
out vec4 fragColor;
uniform sampler2D uScreen;
uniform int       uEffect;   // 1=normal 2=invert 3=grayscale 4=edge
uniform vec2      uTexelSize;

void main() {
    if (uEffect == 1) {
        fragColor = texture(uScreen, vUV);

    } else if (uEffect == 2) {
        vec4 c = texture(uScreen, vUV);
        fragColor = vec4(1.0 - c.rgb, 1.0);

    } else if (uEffect == 3) {
        vec3 c   = texture(uScreen, vUV).rgb;
        float lum = dot(c, vec3(0.2126, 0.7152, 0.0722));
        fragColor = vec4(vec3(lum), 1.0);

    } else {
        // Laplacian edge-detection kernel.
        //  -1 -1 -1
        //  -1  8 -1
        //  -1 -1 -1
        vec2 o = uTexelSize;
        vec3 sum =
            -texture(uScreen, vUV + vec2(-o.x,-o.y)).rgb
            -texture(uScreen, vUV + vec2( 0.0,-o.y)).rgb
            -texture(uScreen, vUV + vec2( o.x,-o.y)).rgb
            -texture(uScreen, vUV + vec2(-o.x, 0.0)).rgb
            +texture(uScreen, vUV) .rgb * 8.0
            -texture(uScreen, vUV + vec2( o.x, 0.0)).rgb
            -texture(uScreen, vUV + vec2(-o.x, o.y)).rgb
            -texture(uScreen, vUV + vec2( 0.0, o.y)).rgb
            -texture(uScreen, vUV + vec2( o.x, o.y)).rgb;
        fragColor = vec4(abs(sum), 1.0);
    }
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

// Fullscreen quad: [X, Y, U, V]
var screenQuad = []float32{
	-1,-1, 0,0,   1,-1, 1,0,   1,1, 1,1,
	-1,-1, 0,0,   1,1,  1,1,  -1,1, 0,1,
}

var cubePosColors = [][2][3]float32{
	{{-2,0,0}, {0.9,0.3,0.3}},
	{{ 0,0,0}, {0.3,0.9,0.4}},
	{{ 2,0,0}, {0.3,0.5,0.9}},
	{{-1,0,-3},{0.9,0.7,0.2}},
	{{ 1,0,-3},{0.7,0.3,0.9}},
}

// ── state ─────────────────────────────────────────────────────────────────────

var (
	effect     = 1
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

	win, err := glfw.CreateWindow(winW, winH, fxTitle(), nil, nil)
	if err != nil { log.Fatalf("CreateWindow: %v", err) }
	defer win.Destroy()
	win.MakeContextCurrent(); glfw.SwapInterval(1)

	if err := gl.InitWithProcAddrFunc(glfw.GetProcAddress); err != nil { log.Fatalf("gl.Init: %v", err) }

	win.SetFramebufferSizeCallback(func(_ *glfw.Window, w, h int) {
		winW, winH = w, h
		gl.Viewport(0, 0, int32(w), int32(h))
		// Resize FBO attachments on window resize would be needed in production;
		// for simplicity we just note it here and tolerate the slight mismatch.
	})
	winW, winH = win.GetFramebufferSize()
	gl.Viewport(0, 0, int32(winW), int32(winH))

	win.SetKeyCallback(func(w *glfw.Window, key glfw.Key, _ int, action glfw.Action, _ glfw.ModifierKey) {
		if action != glfw.Press { return }
		switch key {
		case glfw.KeyEscape: w.SetShouldClose(true)
		case glfw.Key1: effect = 1; w.SetTitle(fxTitle())
		case glfw.Key2: effect = 2; w.SetTitle(fxTitle())
		case glfw.Key3: effect = 3; w.SetTitle(fxTitle())
		case glfw.Key4: effect = 4; w.SetTitle(fxTitle())
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

	// ── FBO setup ─────────────────────────────────────────────────────────
	var fbo uint32
	gl.GenFramebuffers(1, &fbo)
	gl.BindFramebuffer(gl.FRAMEBUFFER, fbo)

	// Colour attachment (the texture we post-process).
	var colTex uint32
	gl.GenTextures(1, &colTex)
	gl.BindTexture(gl.TEXTURE_2D, colTex)
	gl.TexImage2D(gl.TEXTURE_2D, 0, int32(gl.RGB), int32(winW), int32(winH), 0,
		gl.RGB, gl.UNSIGNED_BYTE, nil)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, int32(gl.LINEAR))
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, int32(gl.LINEAR))
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, colTex, 0)

	// Depth+stencil renderbuffer.
	var rbo uint32
	gl.GenRenderbuffers(1, &rbo)
	gl.BindRenderbuffer(gl.RENDERBUFFER, rbo)
	gl.RenderbufferStorage(gl.RENDERBUFFER, gl.DEPTH24_STENCIL8, int32(winW), int32(winH))
	gl.FramebufferRenderbuffer(gl.FRAMEBUFFER, gl.DEPTH_STENCIL_ATTACHMENT, gl.RENDERBUFFER, rbo)

	if gl.CheckFramebufferStatus(gl.FRAMEBUFFER) != gl.FRAMEBUFFER_COMPLETE {
		log.Fatal("framebuffer not complete")
	}
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
	defer func() {
		gl.DeleteFramebuffers(1, &fbo)
		gl.DeleteTextures(1, &colTex)
		gl.DeleteRenderbuffers(1, &rbo)
	}()

	// ── scene VAO ─────────────────────────────────────────────────────────
	var sceneVAO, sceneVBO uint32
	gl.GenVertexArrays(1, &sceneVAO); gl.GenBuffers(1, &sceneVBO)
	gl.BindVertexArray(sceneVAO)
	gl.BindBuffer(gl.ARRAY_BUFFER, sceneVBO)
	gl.BufferData(gl.ARRAY_BUFFER, len(cubeVerts)*4, gl.Ptr(cubeVerts), gl.STATIC_DRAW)
	const stride = int32(6 * 4)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, stride, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(1, 3, gl.FLOAT, false, stride, gl.PtrOffset(12))
	gl.EnableVertexAttribArray(1)
	gl.BindVertexArray(0)

	// ── screen quad VAO ───────────────────────────────────────────────────
	var quadVAO, quadVBO uint32
	gl.GenVertexArrays(1, &quadVAO); gl.GenBuffers(1, &quadVBO)
	gl.BindVertexArray(quadVAO)
	gl.BindBuffer(gl.ARRAY_BUFFER, quadVBO)
	gl.BufferData(gl.ARRAY_BUFFER, len(screenQuad)*4, gl.Ptr(screenQuad), gl.STATIC_DRAW)
	const qstride = int32(4 * 4)
	gl.VertexAttribPointer(0, 2, gl.FLOAT, false, qstride, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(1, 2, gl.FLOAT, false, qstride, gl.PtrOffset(8))
	gl.EnableVertexAttribArray(1)
	gl.BindVertexArray(0)

	defer func() {
		gl.DeleteVertexArrays(1, &sceneVAO); gl.DeleteBuffers(1, &sceneVBO)
		gl.DeleteVertexArrays(1, &quadVAO); gl.DeleteBuffers(1, &quadVBO)
	}()

	// ── shader programs ───────────────────────────────────────────────────
	sceneProg, err := glutil.BuildProgram(sceneVert, sceneFrag)
	if err != nil { log.Fatalf("scene shader: %v", err) }
	defer gl.DeleteProgram(sceneProg)

	ppProg, err := glutil.BuildProgram(ppVert, ppFrag)
	if err != nil { log.Fatalf("pp shader: %v", err) }
	defer gl.DeleteProgram(ppProg)

	// Scene uniforms.
	uMVP      := gl.GetUniformLocation(sceneProg, gl.Str("uMVP\x00"))
	uModel    := gl.GetUniformLocation(sceneProg, gl.Str("uModel\x00"))
	uColor    := gl.GetUniformLocation(sceneProg, gl.Str("uColor\x00"))
	uLightPos := gl.GetUniformLocation(sceneProg, gl.Str("uLightPos\x00"))
	uViewPos  := gl.GetUniformLocation(sceneProg, gl.Str("uViewPos\x00"))

	// Post-process uniforms.
	gl.UseProgram(ppProg)
	gl.Uniform1i(gl.GetUniformLocation(ppProg, gl.Str("uScreen\x00")), 0)
	uEffect    := gl.GetUniformLocation(ppProg, gl.Str("uEffect\x00"))
	uTexelSize := gl.GetUniformLocation(ppProg, gl.Str("uTexelSize\x00"))

	lastTime = glfw.GetTime()
	fmt.Println("1=Normal  2=Invert  3=Grayscale  4=Edge detect   WASD+RMB  ESC quit")

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

		// ── Pass 1: render scene into FBO ──────────────────────────────────
		gl.BindFramebuffer(gl.FRAMEBUFFER, fbo)
		gl.Enable(gl.DEPTH_TEST)
		gl.ClearColor(0.08, 0.08, 0.12, 1)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

		gl.UseProgram(sceneProg)
		gl.Uniform3f(uLightPos, 3, 5, 3)
		gl.Uniform3f(uViewPos, cam.Pos[0], cam.Pos[1], cam.Pos[2])

		gl.BindVertexArray(sceneVAO)
		for _, pc := range cubePosColors {
			pos, col := pc[0], pc[1]
			model := glutil.Translate3(pos[0], pos[1], pos[2])
			mvp := glutil.MatMul(vp, model)
			gl.UniformMatrix4fv(uMVP, 1, false, &mvp[0])
			gl.UniformMatrix4fv(uModel, 1, false, &model[0])
			gl.Uniform3f(uColor, col[0], col[1], col[2])
			gl.DrawArrays(gl.TRIANGLES, 0, 36)
		}
		gl.BindVertexArray(0)

		// ── Pass 2: fullscreen post-process quad ───────────────────────────
		gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
		gl.Disable(gl.DEPTH_TEST)
		gl.ClearColor(0, 0, 0, 1)
		gl.Clear(gl.COLOR_BUFFER_BIT)

		gl.UseProgram(ppProg)
		gl.ActiveTexture(gl.TEXTURE0)
		gl.BindTexture(gl.TEXTURE_2D, colTex)
		gl.Uniform1i(uEffect, int32(effect))
		gl.Uniform2f(uTexelSize, 1.0/float32(winW), 1.0/float32(winH))

		gl.BindVertexArray(quadVAO)
		gl.DrawArrays(gl.TRIANGLES, 0, 6)
		gl.BindVertexArray(0)

		win.SwapBuffers()
		glfw.PollEvents()
	}
}

func fxTitle() string {
	names := [5]string{"", "Normal", "Inversion", "Grayscale", "Edge Detection"}
	return fmt.Sprintf("14 — Framebuffers  [%s]  press 1/2/3/4", names[effect])
}
