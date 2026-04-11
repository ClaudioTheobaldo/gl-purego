//go:build windows

// 15_instancing — GL 3.3 core-profile: instanced rendering.
//
// Renders a 10×10 wave of cubes in a single glDrawElementsInstanced call.
// Each cube has a per-instance XZ position and colour passed via vertex
// attribute divisors — the signature GL 3.3 instancing feature.
//
// Key differences from the v2.1 examples:
//
//   - Context created with ContextVersionMajor=3, Minor=3,
//     OpenGLProfile=CoreProfile, OpenGLForwardCompatible=true
//   - A VAO is mandatory in the core profile; omitting it is an error
//   - Per-instance attributes use glVertexAttribDivisor(attrib, 1)
//   - Draw call is glDrawElementsInstanced instead of glDrawElements
//
// Controls:
//
//	WASD / Q / E  — move camera
//	Hold RMB      — look around
//	Mouse wheel   — adjust speed
//	ESC           — quit
//
// Build:
//
//	CGO_ENABLED=0 go build -o 15_instancing.exe .
package main

import (
	"log"
	"math"
	"unsafe"

	gl     "github.com/ClaudioTheobaldo/gl-purego/v3.3/gl"
	glfw   "github.com/ClaudioTheobaldo/glfw-purego/v3.3/glfw"
	glutil "github.com/ClaudioTheobaldo/gl-purego/examples/glutil"
)

// ── shaders ───────────────────────────────────────────────────────────────────

const vertSrc = `#version 330 core
layout(location = 0) in vec3 aPos;      // per-vertex: cube corner
layout(location = 1) in vec2 aOffset;   // per-instance: XZ grid position
layout(location = 2) in vec3 aColor;    // per-instance: cube colour

uniform mat4 uView;
uniform mat4 uProj;
uniform float uTime;

out vec3 vColor;
out vec3 vNormal;

void main() {
    // Wave: each cube bobs up and down with a phase derived from its grid pos.
    float wave = sin(uTime + aOffset.x * 0.8 + aOffset.y * 0.6) * 0.6;

    // Simple Y-axis rotation per cube so they spin as they bob.
    float angle = uTime * 0.7 + aOffset.x * 0.3 + aOffset.y * 0.4;
    float c = cos(angle), s = sin(angle);

    // Rotate the vertex around Y, then translate to grid position.
    vec3 rotated = vec3(
        aPos.x * c - aPos.z * s,
        aPos.y,
        aPos.x * s + aPos.z * c
    );
    vec3 worldPos = rotated + vec3(aOffset.x, wave, aOffset.y);

    gl_Position = uProj * uView * vec4(worldPos, 1.0);

    // Pass un-rotated normal for a cheap face-based tint in the fragment shader.
    vNormal = aPos;
    vColor  = aColor;
}
` + "\x00"

const fragSrc = `#version 330 core
in vec3 vColor;
in vec3 vNormal;

out vec4 FragColor;

void main() {
    // Directional light from above-front; gives each face a different shade.
    vec3 lightDir = normalize(vec3(0.4, 1.0, 0.6));
    float diff = max(dot(normalize(vNormal), lightDir), 0.0);
    float light = 0.35 + 0.65 * diff;
    FragColor = vec4(vColor * light, 1.0);
}
` + "\x00"

// ── cube geometry ─────────────────────────────────────────────────────────────

// Unit cube — 24 vertices (4 per face) with unique normals per face.
// We store only position; the normal is approximated from the position in
// the fragment shader for simplicity.
var cubeVerts = []float32{
	// Front (+Z)
	-0.5, -0.5, +0.5,
	+0.5, -0.5, +0.5,
	+0.5, +0.5, +0.5,
	-0.5, +0.5, +0.5,
	// Back (-Z)
	+0.5, -0.5, -0.5,
	-0.5, -0.5, -0.5,
	-0.5, +0.5, -0.5,
	+0.5, +0.5, -0.5,
	// Left (-X)
	-0.5, -0.5, -0.5,
	-0.5, -0.5, +0.5,
	-0.5, +0.5, +0.5,
	-0.5, +0.5, -0.5,
	// Right (+X)
	+0.5, -0.5, +0.5,
	+0.5, -0.5, -0.5,
	+0.5, +0.5, -0.5,
	+0.5, +0.5, +0.5,
	// Top (+Y)
	-0.5, +0.5, +0.5,
	+0.5, +0.5, +0.5,
	+0.5, +0.5, -0.5,
	-0.5, +0.5, -0.5,
	// Bottom (-Y)
	-0.5, -0.5, -0.5,
	+0.5, -0.5, -0.5,
	+0.5, -0.5, +0.5,
	-0.5, -0.5, +0.5,
}

var cubeIdx = []uint32{
	0, 1, 2, 2, 3, 0, // front
	4, 5, 6, 6, 7, 4, // back
	8, 9, 10, 10, 11, 8, // left
	12, 13, 14, 14, 15, 12, // right
	16, 17, 18, 18, 19, 16, // top
	20, 21, 22, 22, 23, 20, // bottom
}

// ── instance data ─────────────────────────────────────────────────────────────

const (
	gridW   = 10
	gridH   = 10
	spacing = 2.4 // world-space distance between cube centres
)

// buildInstances returns a flat []float32 of [xOffset, zOffset, r, g, b] × N.
func buildInstances() []float32 {
	data := make([]float32, 0, gridW*gridH*5)
	for row := 0; row < gridH; row++ {
		for col := 0; col < gridW; col++ {
			x := (float32(col) - float32(gridW-1)*0.5) * spacing
			z := (float32(row) - float32(gridH-1)*0.5) * spacing

			// HSV-based colour: hue cycles across the grid.
			hue := float64(row*gridW+col) / float64(gridW*gridH)
			r, g, b := hsvToRGB(hue, 0.75, 0.95)

			data = append(data, x, z, r, g, b)
		}
	}
	return data
}

func hsvToRGB(h, s, v float64) (float32, float32, float32) {
	h6 := h * 6.0
	i := int(h6)
	f := h6 - float64(i)
	p := v * (1 - s)
	q := v * (1 - s*f)
	t := v * (1 - s*(1-f))
	switch i % 6 {
	case 0:
		return float32(v), float32(t), float32(p)
	case 1:
		return float32(q), float32(v), float32(p)
	case 2:
		return float32(p), float32(v), float32(t)
	case 3:
		return float32(p), float32(q), float32(v)
	case 4:
		return float32(t), float32(p), float32(v)
	default:
		return float32(v), float32(p), float32(q)
	}
}

// ── main ──────────────────────────────────────────────────────────────────────

func main() {
	if err := glfw.Init(); err != nil {
		log.Fatal(err)
	}
	defer glfw.Terminate()

	// ── Core-profile context hints ────────────────────────────────────────────
	// These three hints together request an OpenGL 3.3 core-profile context.
	// The core profile removes all deprecated fixed-function API (glBegin/glEnd,
	// glColor*, immediate-mode, etc.) and makes VAOs mandatory.
	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 3)
	glfw.WindowHint(glfw.OpenGLProfileHint, int(glfw.CoreProfile))
	glfw.WindowHint(glfw.OpenGLForwardCompatible, 1) // required on macOS; harmless elsewhere

	win, err := glfw.CreateWindow(1024, 768, "15 — GL 3.3 Instancing (100 cubes)", nil, nil)
	if err != nil {
		log.Fatal(err)
	}
	win.MakeContextCurrent()
	glfw.SwapInterval(1)

	if err := gl.InitWithProcAddrFunc(func(name string) unsafe.Pointer {
		return glfw.GetProcAddress(name)
	}); err != nil {
		log.Fatal(err)
	}

	gl.Enable(gl.DEPTH_TEST)

	prog, err := glutil.BuildProgram(vertSrc, fragSrc)
	if err != nil {
		log.Fatal(err)
	}

	// ── VAO — mandatory in the GL 3.3 core profile ───────────────────────────
	// Without a bound VAO every vertex attribute call below would be an error.
	var vao uint32
	gl.GenVertexArrays(1, &vao)
	gl.BindVertexArray(vao)

	// ── Cube geometry VBO ─────────────────────────────────────────────────────
	var vbo uint32
	gl.GenBuffers(1, &vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(cubeVerts)*4, gl.Ptr(cubeVerts), gl.STATIC_DRAW)

	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 12, gl.PtrOffset(0))

	// ── Index buffer ─────────────────────────────────────────────────────────
	var ebo uint32
	gl.GenBuffers(1, &ebo)
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, ebo)
	gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, len(cubeIdx)*4, gl.Ptr(cubeIdx), gl.STATIC_DRAW)

	// ── Per-instance VBO ──────────────────────────────────────────────────────
	instances := buildInstances()
	var instanceVBO uint32
	gl.GenBuffers(1, &instanceVBO)
	gl.BindBuffer(gl.ARRAY_BUFFER, instanceVBO)
	gl.BufferData(gl.ARRAY_BUFFER, len(instances)*4, gl.Ptr(instances), gl.STATIC_DRAW)

	const stride = int32(5 * 4) // 5 floats × 4 bytes

	// aOffset (location 1): XZ position, 2 floats, one per instance.
	gl.EnableVertexAttribArray(1)
	gl.VertexAttribPointer(1, 2, gl.FLOAT, false, stride, gl.PtrOffset(0))
	gl.VertexAttribDivisor(1, 1) // advance once per instance, not per vertex

	// aColor (location 2): RGB colour, 3 floats, one per instance.
	gl.EnableVertexAttribArray(2)
	gl.VertexAttribPointer(2, 3, gl.FLOAT, false, stride, gl.PtrOffset(2*4))
	gl.VertexAttribDivisor(2, 1) // advance once per instance, not per vertex

	// Uniform locations
	uView := gl.GetUniformLocation(prog, gl.Str("uView\x00"))
	uProj := gl.GetUniformLocation(prog, gl.Str("uProj\x00"))
	uTime := gl.GetUniformLocation(prog, gl.Str("uTime\x00"))

	// Camera
	cam := glutil.NewCamera([3]float32{0, 6, 18})
	cam.Pitch = -18
	cam.Speed = 8

	win.SetMouseButtonCallback(func(_ *glfw.Window, btn glfw.MouseButton, action glfw.Action, _ glfw.ModifierKey) {
		if btn == glfw.MouseButtonRight {
			cam.SetRMB(action == glfw.Press)
		}
	})
	win.SetCursorPosCallback(func(_ *glfw.Window, x, y float64) { cam.MousePos(x, y) })
	win.SetScrollCallback(func(_ *glfw.Window, _, yoff float64) { cam.Scroll(yoff, 1, 40) })

	winW, winH := 1024, 768
	win.SetFramebufferSizeCallback(func(_ *glfw.Window, w, h int) {
		winW, winH = w, h
		gl.Viewport(0, 0, int32(w), int32(h))
	})

	lastTime := glfw.GetTime()

	for !win.ShouldClose() {
		now := glfw.GetTime()
		dt := float32(now - lastTime)
		lastTime = now

		if win.GetKey(glfw.KeyEscape) == glfw.Press {
			win.SetShouldClose(true)
		}
		cam.HandleKeys(
			win.GetKey(glfw.KeyW) == glfw.Press,
			win.GetKey(glfw.KeyS) == glfw.Press,
			win.GetKey(glfw.KeyA) == glfw.Press,
			win.GetKey(glfw.KeyD) == glfw.Press,
			win.GetKey(glfw.KeyE) == glfw.Press,
			win.GetKey(glfw.KeyQ) == glfw.Press,
			dt,
		)

		gl.ClearColor(0.08, 0.08, 0.12, 1.0)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

		gl.UseProgram(prog)

		view := cam.ViewMatrix()
		proj := glutil.Perspective(glutil.ToRad(50), float32(winW)/float32(winH), 0.1, 200)

		gl.UniformMatrix4fv(uView, 1, false, &view[0])
		gl.UniformMatrix4fv(uProj, 1, false, &proj[0])
		gl.Uniform1f(uTime, float32(math.Mod(now, math.Pi*200)))

		gl.BindVertexArray(vao)

		// Single draw call renders all 100 cubes (36 indices × 100 instances).
		gl.DrawElementsInstanced(gl.TRIANGLES, 36, gl.UNSIGNED_INT, gl.PtrOffset(0), 100)

		win.SwapBuffers()
		glfw.PollEvents()
	}
}
