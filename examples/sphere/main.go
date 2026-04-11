//go:build windows

// sphere renders a UV sphere with indexed geometry. Vertex colours are derived
// from the surface normal (|nx|, |ny|, |nz|), producing a multicolour shading
// that highlights the sphere's curvature. Rotates slowly around Y.
//
// Build:
//
//	CGO_ENABLED=0 go build -o sphere.exe .
package main

import (
	"fmt"
	"log"
	"math"
	"unsafe"

	gl     "github.com/ClaudioTheobaldo/gl-purego/v2.1/gl"
	glutil "github.com/ClaudioTheobaldo/gl-purego/examples/glutil"
	glfw   "github.com/ClaudioTheobaldo/glfw-purego/v3.3/glfw"
)

const (
	stacks = 48
	slices = 48
)

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

var (
	winW, winH  int     = 800, 600
	camDist     float32 = 2.5 // scroll to zoom
)

func main() {
	if err := glfw.Init(); err != nil {
		log.Fatalf("glfw.Init: %v", err)
	}
	defer glfw.Terminate()

	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 3)
	glfw.WindowHint(glfw.OpenGLProfileHint, int(glfw.CoreProfile))
	glfw.WindowHint(glfw.OpenGLForwardCompatible, 1)

	win, err := glfw.CreateWindow(winW, winH, "Sphere — glfw-purego + gl-purego", nil, nil)
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
		if key == glfw.KeyEscape && action == glfw.Press {
			w.SetShouldClose(true)
		}
	})
	win.SetFramebufferSizeCallback(func(w *glfw.Window, width, height int) {
		winW, winH = width, height
		gl.Viewport(0, 0, int32(width), int32(height))
	})
	win.SetScrollCallback(func(w *glfw.Window, xoff, yoff float64) {
		camDist -= float32(yoff) * 0.15
		if camDist < 1.1 {
			camDist = 1.1 // don't clip inside the sphere
		}
		if camDist > 20 {
			camDist = 20
		}
	})
	winW, winH = win.GetFramebufferSize()
	gl.Viewport(0, 0, int32(winW), int32(winH))

	// Generate sphere geometry
	vertices, indices := generateSphere()
	indexCount := int32(len(indices))
	fmt.Printf("Sphere: %d vertices, %d triangles\n", len(vertices)/6, indexCount/3)

	// Compile shaders
	prog, err := glutil.BuildProgram(vertSrc, fragSrc)
	if err != nil {
		log.Fatalf("shader: %v", err)
	}
	defer gl.DeleteProgram(prog)

	// Upload geometry — VAO + VBO + EBO
	var vao, vbo, ebo uint32
	gl.GenVertexArrays(1, &vao)
	gl.GenBuffers(1, &vbo)
	gl.GenBuffers(1, &ebo)
	defer func() {
		gl.DeleteVertexArrays(1, &vao)
		gl.DeleteBuffers(1, &vbo)
		gl.DeleteBuffers(1, &ebo)
	}()

	gl.BindVertexArray(vao)

	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(vertices)*4, unsafe.Pointer(&vertices[0]), gl.STATIC_DRAW)

	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, ebo)
	gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, len(indices)*4, unsafe.Pointer(&indices[0]), gl.STATIC_DRAW)

	const stride = int32(6 * 4)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, stride, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(1, 3, gl.FLOAT, false, stride, gl.PtrOffset(12))
	gl.EnableVertexAttribArray(1)

	gl.BindVertexArray(0)

	uMVP := gl.GetUniformLocation(prog, gl.Str("uMVP"))

	gl.Enable(gl.DEPTH_TEST)

	fmt.Println("Rendering sphere — press ESC to quit.")

	for !win.ShouldClose() {
		gl.ClearColor(0.08, 0.08, 0.12, 1.0)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

		t := float32(glfw.GetTime())
		model := glutil.RotY(t * 0.5)
		view  := glutil.LookAt([3]float32{0, camDist * 0.32, camDist}, [3]float32{0, 0, 0}, [3]float32{0, 1, 0})
		proj  := glutil.Perspective(glutil.ToRad(45), float32(winW)/float32(winH), 0.1, 100)
		mvp   := glutil.MatMul(proj, glutil.MatMul(view, model))

		gl.UseProgram(prog)
		gl.UniformMatrix4fv(uMVP, 1, false, &mvp[0])
		gl.BindVertexArray(vao)
		gl.DrawElements(gl.TRIANGLES, indexCount, gl.UNSIGNED_INT, nil)
		gl.BindVertexArray(0)

		win.SwapBuffers()
		glfw.PollEvents()
	}
}

// generateSphere builds an interleaved vertex buffer [X Y Z R G B] and a
// uint32 index buffer for a UV sphere. Colours are abs(normal), so each
// axis contributes a distinct hue and the surface shows smooth gradients.
func generateSphere() (vertices []float32, indices []uint32) {
	for i := 0; i <= stacks; i++ {
		theta := math.Pi * float64(i) / float64(stacks) // 0 → π
		sinT := float32(math.Sin(theta))
		cosT := float32(math.Cos(theta))
		for j := 0; j <= slices; j++ {
			phi := 2 * math.Pi * float64(j) / float64(slices) // 0 → 2π
			x := sinT * float32(math.Cos(phi))
			y := cosT
			z := sinT * float32(math.Sin(phi))
			// Colour = |normal| — pure red at ±X, green at ±Y, blue at ±Z
			vertices = append(vertices, x, y, z, abs32(x), abs32(y), abs32(z))
		}
	}
	for i := 0; i < stacks; i++ {
		for j := 0; j < slices; j++ {
			p0 := uint32(i*(slices+1) + j)
			p1 := uint32((i+1)*(slices+1) + j)
			p2 := uint32((i+1)*(slices+1) + j + 1)
			p3 := uint32(i*(slices+1) + j + 1)
			indices = append(indices, p0, p1, p2, p0, p2, p3)
		}
	}
	return
}

func abs32(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}
