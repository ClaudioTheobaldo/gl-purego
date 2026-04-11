//go:build windows

// hello_window opens an 800×600 window, prints GL/driver info to stdout, and
// cycles the background colour through a slow rainbow gradient until the user
// presses ESC or clicks the close button.
//
// Build (CGO disabled):
//
//	CGO_ENABLED=0 go build -o hello_window.exe .
package main

import (
	"fmt"
	"log"
	"math"

	gl   "github.com/ClaudioTheobaldo/gl-purego/v2.1/gl"
	glfw "github.com/ClaudioTheobaldo/glfw-purego/v3.3/glfw"
)

func main() {
	if err := glfw.Init(); err != nil {
		log.Fatalf("glfw.Init: %v", err)
	}
	defer glfw.Terminate()

	// Request OpenGL 3.3 core
	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 3)
	glfw.WindowHint(glfw.OpenGLProfileHint, int(glfw.CoreProfile))
	glfw.WindowHint(glfw.OpenGLForwardCompatible, 1)

	win, err := glfw.CreateWindow(800, 600, "Hello Window — glfw-purego", nil, nil)
	if err != nil {
		log.Fatalf("CreateWindow: %v", err)
	}
	defer win.Destroy()

	win.MakeContextCurrent()
	glfw.SwapInterval(1) // vsync

	// Boot gl-purego through glfw's proc-address resolver (no CGO, no opengl32
	// import — everything comes from the live WGL context).
	if err := gl.InitWithProcAddrFunc(glfw.GetProcAddress); err != nil {
		log.Fatalf("gl.Init: %v", err)
	}

	// Keep the viewport in sync with the framebuffer size.
	win.SetFramebufferSizeCallback(func(w *glfw.Window, width, height int) {
		gl.Viewport(0, 0, int32(width), int32(height))
	})
	fw, fh := win.GetFramebufferSize()
	gl.Viewport(0, 0, int32(fw), int32(fh))

	// ESC → close
	win.SetKeyCallback(func(w *glfw.Window, key glfw.Key, _ int, action glfw.Action, _ glfw.ModifierKey) {
		if key == glfw.KeyEscape && action == glfw.Press {
			w.SetShouldClose(true)
		}
	})

	// Print driver info
	fmt.Printf("Vendor:   %s\n", gl.GoStr(gl.GetString(gl.VENDOR)))
	fmt.Printf("Renderer: %s\n", gl.GoStr(gl.GetString(gl.RENDERER)))
	fmt.Printf("Version:  %s\n", gl.GoStr(gl.GetString(gl.VERSION)))
	fmt.Printf("GLSL:     %s\n", gl.GoStr(gl.GetString(gl.SHADING_LANGUAGE_VERSION)))
	fmt.Println("Press ESC to quit.")

	for !win.ShouldClose() {
		t := glfw.GetTime()
		// Three sine waves, 120° apart → smooth pastel rainbow cycle
		r := float32(0.5 + 0.5*math.Sin(t*0.4))
		g := float32(0.5 + 0.5*math.Sin(t*0.4+2.094))
		b := float32(0.5 + 0.5*math.Sin(t*0.4+4.189))

		gl.ClearColor(r*0.25, g*0.25, b*0.25, 1.0)
		gl.Clear(gl.COLOR_BUFFER_BIT)

		win.SwapBuffers()
		glfw.PollEvents()
	}
}
