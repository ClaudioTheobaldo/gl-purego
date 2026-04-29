//go:build windows

// 19_gles_compute demonstrates GLES 3.1 compute shaders via ANGLE.
// A compute shader writes a UV-gradient pattern (red=U, green=V, blue=0.5)
// into a 512×512 RGBA8 texture, which is then displayed on a fullscreen
// triangle pair using a display program.
//
// Requires GLES 3.1. ANGLE's D3D11 backend is hardcapped at GLES 3.0; this
// example sets ANGLE_DEFAULT_PLATFORM=vulkan automatically so it works with a
// Vulkan-enabled ANGLE build or Linux + Mesa. If only GLES 3.0 is available
// the program exits with a clear message.
//
// ANGLE must be available on PATH or in the same directory as the executable.
//
// Build (CGO disabled):
//
//	CGO_ENABLED=0 go build -o gles_compute.exe .
package main

import (
	"log"
	"os"
	"strings"
	"unsafe"

	gl   "github.com/ClaudioTheobaldo/gl-purego/gles2/v3.1/gl"
	glfw "github.com/ClaudioTheobaldo/glfw-purego/v3.3/glfw"
)

// Compute shader: writes a UV-gradient into a 512×512 image.
const computeSrc = `#version 310 es
layout(local_size_x = 16, local_size_y = 16) in;
layout(rgba8, binding = 0) writeonly uniform highp image2D uImg;
void main() {
    ivec2 coord = ivec2(gl_GlobalInvocationID.xy);
    ivec2 size  = imageSize(uImg);
    vec2 uv = vec2(coord) / vec2(size);
    imageStore(uImg, coord, vec4(uv, 0.5, 1.0));
}`

// Display vertex shader: emits a fullscreen triangle using gl_VertexID.
const displayVertSrc = `#version 310 es
out vec2 vUV;
void main() {
    vec2 pos = vec2((gl_VertexID == 1) ? 3.0 : -1.0,
                    (gl_VertexID == 2) ? 3.0 : -1.0);
    vUV = pos * 0.5 + 0.5;
    gl_Position = vec4(pos, 0.0, 1.0);
}`

// Display fragment shader: samples the texture produced by the compute shader.
const displayFragSrc = `#version 310 es
precision mediump float;
uniform sampler2D uTex;
in  vec2 vUV;
out vec4 fragColor;
void main() { fragColor = texture(uTex, vUV); }`

const texSize = 512

func main() {
	// Force ANGLE to use its Vulkan backend (D3D11 backend caps at GLES 3.0;
	// GLES 3.1 compute shaders require Vulkan).
	os.Setenv("ANGLE_DEFAULT_PLATFORM", "vulkan")

	if err := glfw.Init(); err != nil {
		log.Fatalf("glfw.Init: %v", err)
	}
	defer glfw.Terminate()

	glfw.WindowHint(glfw.ClientAPIs, int(glfw.OpenGLESAPI))
	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)

	win, err := glfw.CreateWindow(800, 600, "GLES 3.1 Compute Shader — EGL/ANGLE", nil, nil)
	if err != nil {
		log.Fatalf("CreateWindow: %v\n\nMake sure libEGL.dll and libGLESv2.dll (ANGLE) are on PATH.", err)
	}
	defer win.Destroy()

	win.MakeContextCurrent()
	glfw.SwapInterval(1)

	if err := gl.InitWithProcAddrFunc(func(name string) unsafe.Pointer {
		return glfw.GetProcAddress(name)
	}); err != nil {
		log.Fatalf("gl.Init: %v", err)
	}

	renderer := gl.GoStr(gl.GetString(gl.RENDERER))
	version  := gl.GoStr(gl.GetString(gl.VERSION))
	log.Printf("GLES Renderer : %s", renderer)
	log.Printf("GLES Version  : %s", version)

	// Compute shaders require GLES 3.1. Browser-shipped ANGLE (Chrome, Brave)
	// caps at GLES 3.0. Check before attempting shader compilation.
	if !strings.Contains(version, "3.1") && !strings.Contains(version, "3.2") {
		log.Fatalf("GLES 3.1 required for compute shaders; got: %s\n"+
			"Use a standalone ANGLE build (not browser-shipped) or run on Linux with Mesa.", version)
	}

	// -------------------------------------------------------------------------
	// Create the output texture (immutable storage via TexStorage2D).
	// -------------------------------------------------------------------------
	var tex uint32
	gl.GenTextures(1, &tex)
	gl.BindTexture(gl.TEXTURE_2D, tex)
	gl.TexStorage2D(gl.TEXTURE_2D, 1, gl.RGBA8, texSize, texSize)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.BindTexture(gl.TEXTURE_2D, 0)

	// -------------------------------------------------------------------------
	// Build the compute program and run it once.
	// -------------------------------------------------------------------------
	computeProg := buildComputeProgram(computeSrc)

	// Bind the texture to image unit 0 for write-only access.
	gl.BindImageTexture(0, tex, 0, false, 0, gl.WRITE_ONLY, gl.RGBA8)

	gl.UseProgram(computeProg)
	// Dispatch: 512/16 = 32 groups in each dimension.
	gl.DispatchCompute(texSize/16, texSize/16, 1)
	// Ensure image writes are visible to subsequent texture reads.
	gl.MemoryBarrier(gl.SHADER_IMAGE_ACCESS_BARRIER_BIT)
	gl.UseProgram(0)

	// -------------------------------------------------------------------------
	// Build the display (fullscreen quad) program.
	// -------------------------------------------------------------------------
	displayProg := buildProgram(displayVertSrc, displayFragSrc)

	gl.UseProgram(displayProg)
	uTex := gl.GetUniformLocation(displayProg, &[]byte("uTex\x00")[0])
	gl.Uniform1i(uTex, 0)
	gl.UseProgram(0)

	// A VAO is required even though the display vertex shader uses no attributes.
	var vao uint32
	gl.GenVertexArrays(1, &vao)

	// -------------------------------------------------------------------------
	// Render loop
	// -------------------------------------------------------------------------
	win.SetKeyCallback(func(w *glfw.Window, key glfw.Key, _ int, action glfw.Action, _ glfw.ModifierKey) {
		if key == glfw.KeyEscape && action == glfw.Press {
			w.SetShouldClose(true)
		}
	})

	win.SetFramebufferSizeCallback(func(_ *glfw.Window, width, height int) {
		gl.Viewport(0, 0, int32(width), int32(height))
	})

	gl.ClearColor(0.0, 0.0, 0.0, 1.0)

	for !win.ShouldClose() {
		glfw.PollEvents()

		gl.Clear(gl.COLOR_BUFFER_BIT)

		gl.UseProgram(displayProg)
		gl.ActiveTexture(gl.TEXTURE0)
		gl.BindTexture(gl.TEXTURE_2D, tex)
		gl.BindVertexArray(vao)
		// 3 vertices → one large triangle that covers the entire screen.
		gl.DrawArrays(gl.TRIANGLES, 0, 3)
		gl.BindVertexArray(0)
		gl.BindTexture(gl.TEXTURE_2D, 0)

		win.SwapBuffers()
	}
}

// ----------------------------------------------------------------------------
// Local shader helpers (use gles2/v3.1/gl — not glutil which imports v2.1/gl)
// ----------------------------------------------------------------------------

func buildProgram(vertSrc, fragSrc string) uint32 {
	vs := compileShader(gl.VERTEX_SHADER, vertSrc)
	fs := compileShader(gl.FRAGMENT_SHADER, fragSrc)

	prog := gl.CreateProgram()
	gl.AttachShader(prog, vs)
	gl.AttachShader(prog, fs)
	gl.LinkProgram(prog)

	var status int32
	gl.GetProgramiv(prog, gl.LINK_STATUS, &status)
	if status == 0 {
		var logLen int32
		gl.GetProgramiv(prog, gl.INFO_LOG_LENGTH, &logLen)
		logBuf := make([]byte, logLen)
		gl.GetProgramInfoLog(prog, logLen, nil, &logBuf[0])
		log.Fatalf("link: %s", logBuf)
	}

	gl.DeleteShader(vs)
	gl.DeleteShader(fs)
	return prog
}

func buildComputeProgram(src string) uint32 {
	cs := compileShader(gl.COMPUTE_SHADER, src)

	prog := gl.CreateProgram()
	gl.AttachShader(prog, cs)
	gl.LinkProgram(prog)

	var status int32
	gl.GetProgramiv(prog, gl.LINK_STATUS, &status)
	if status == 0 {
		var logLen int32
		gl.GetProgramiv(prog, gl.INFO_LOG_LENGTH, &logLen)
		logBuf := make([]byte, logLen)
		gl.GetProgramInfoLog(prog, logLen, nil, &logBuf[0])
		log.Fatalf("link compute: %s", logBuf)
	}

	gl.DeleteShader(cs)
	return prog
}

func compileShader(kind uint32, src string) uint32 {
	shader := gl.CreateShader(kind)
	cstr, free := gl.Strs(src + "\x00")
	gl.ShaderSource(shader, 1, cstr, nil)
	free()
	gl.CompileShader(shader)

	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == 0 {
		var logLen int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLen)
		logBuf := make([]byte, logLen)
		gl.GetShaderInfoLog(shader, logLen, nil, &logBuf[0])
		log.Fatalf("compile shader (kind=%d): %s", kind, logBuf)
	}
	return shader
}
