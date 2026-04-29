//go:build windows

// 18_gles_textures renders a textured quad using OpenGL ES 3.0 via ANGLE.
// The texture is a programmatically-generated 64×64 checkerboard (8×8 cells,
// each cell 8 pixels) alternating between orange and dark-grey.
//
// ANGLE must be available on PATH or in the same directory as the executable.
//
// Build (CGO disabled):
//
//	CGO_ENABLED=0 go build -o gles_textures.exe .
package main

import (
	"log"
	"unsafe"

	gl   "github.com/ClaudioTheobaldo/gl-purego/gles2/v3.0/gl"
	glfw "github.com/ClaudioTheobaldo/glfw-purego/v3.3/glfw"
)

// Interleaved position (vec2) + texcoord (vec2), two triangles forming a quad.
var vertices = []float32{
	// X      Y     U    V
	-0.7, -0.7, 0.0, 0.0,
	0.7, -0.7, 1.0, 0.0,
	0.7, 0.7, 1.0, 1.0,
	-0.7, -0.7, 0.0, 0.0,
	0.7, 0.7, 1.0, 1.0,
	-0.7, 0.7, 0.0, 1.0,
}

const vertSrc = `#version 300 es
layout(location = 0) in vec2 aPos;
layout(location = 1) in vec2 aTexCoord;
out vec2 vTexCoord;
void main() {
    gl_Position = vec4(aPos, 0.0, 1.0);
    vTexCoord = aTexCoord;
}`

const fragSrc = `#version 300 es
precision mediump float;
uniform sampler2D uTex;
in  vec2 vTexCoord;
out vec4 fragColor;
void main() {
    fragColor = texture(uTex, vTexCoord);
}`

func main() {
	if err := glfw.Init(); err != nil {
		log.Fatalf("glfw.Init: %v", err)
	}
	defer glfw.Terminate()

	glfw.WindowHint(glfw.ClientAPIs, int(glfw.OpenGLESAPI))
	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 0)

	win, err := glfw.CreateWindow(800, 600, "GLES 3.0 Textures — EGL/ANGLE", nil, nil)
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

	log.Printf("GLES Renderer : %s", gl.GoStr(gl.GetString(gl.RENDERER)))
	log.Printf("GLES Version  : %s", gl.GoStr(gl.GetString(gl.VERSION)))

	// -------------------------------------------------------------------------
	// Checkerboard texture: 64×64 RGBA8, 8×8 grid of 8-pixel cells.
	// -------------------------------------------------------------------------
	const texSize = 64
	const cellSize = 8
	texData := make([]uint8, texSize*texSize*4)
	orange := [4]uint8{255, 140, 0, 255}
	darkGrey := [4]uint8{40, 40, 40, 255}
	for row := 0; row < texSize; row++ {
		for col := 0; col < texSize; col++ {
			cellRow := row / cellSize
			cellCol := col / cellSize
			var color [4]uint8
			if (cellRow+cellCol)%2 == 0 {
				color = orange
			} else {
				color = darkGrey
			}
			idx := (row*texSize + col) * 4
			texData[idx+0] = color[0]
			texData[idx+1] = color[1]
			texData[idx+2] = color[2]
			texData[idx+3] = color[3]
		}
	}

	var tex uint32
	gl.GenTextures(1, &tex)
	gl.BindTexture(gl.TEXTURE_2D, tex)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA8, texSize, texSize, 0,
		gl.RGBA, gl.UNSIGNED_BYTE, unsafe.Pointer(&texData[0]))
	gl.BindTexture(gl.TEXTURE_2D, 0)

	// -------------------------------------------------------------------------
	// Shader program
	// -------------------------------------------------------------------------
	prog := buildProgram(vertSrc, fragSrc)

	// Bind the sampler uniform to texture unit 0.
	gl.UseProgram(prog)
	uTex := gl.GetUniformLocation(prog, &[]byte("uTex\x00")[0])
	gl.Uniform1i(uTex, 0)
	gl.UseProgram(0)

	// -------------------------------------------------------------------------
	// VAO + VBO
	// -------------------------------------------------------------------------
	var vao, vbo uint32
	gl.GenVertexArrays(1, &vao)
	gl.BindVertexArray(vao)

	gl.GenBuffers(1, &vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(vertices)*4, unsafe.Pointer(&vertices[0]), gl.STATIC_DRAW)

	stride := int32(4 * 4) // 4 float32s × 4 bytes

	// aPos — location 0, 2 floats, offset 0
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 2, gl.FLOAT, false, stride, gl.PtrOffset(0))

	// aTexCoord — location 1, 2 floats, offset 8 bytes
	gl.EnableVertexAttribArray(1)
	gl.VertexAttribPointer(1, 2, gl.FLOAT, false, stride, gl.PtrOffset(2*4))

	gl.BindVertexArray(0)

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

	gl.ClearColor(0.1, 0.1, 0.15, 1.0)

	for !win.ShouldClose() {
		glfw.PollEvents()

		gl.Clear(gl.COLOR_BUFFER_BIT)

		gl.UseProgram(prog)
		gl.ActiveTexture(gl.TEXTURE0)
		gl.BindTexture(gl.TEXTURE_2D, tex)
		gl.BindVertexArray(vao)
		gl.DrawArrays(gl.TRIANGLES, 0, 6)
		gl.BindVertexArray(0)
		gl.BindTexture(gl.TEXTURE_2D, 0)

		win.SwapBuffers()
	}
}

// ----------------------------------------------------------------------------
// Local shader helpers (use gles2/v3.0/gl — not glutil which imports v2.1/gl)
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
