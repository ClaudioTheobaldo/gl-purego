//go:build windows

// Package glutil provides shared helpers for the gl-purego examples:
// shader compilation, 4×4 matrix math, vec3 math, and a free-fly camera.
package glutil

import (
	"fmt"
	"math"

	gl "github.com/ClaudioTheobaldo/gl-purego/v2.1/gl"
)

// ── Shader helpers ────────────────────────────────────────────────────────────

// BuildProgram compiles vs and fs then links them into a program.
func BuildProgram(vs, fs string) (uint32, error) {
	v, err := CompileShader(vs, gl.VERTEX_SHADER)
	if err != nil {
		return 0, fmt.Errorf("vertex: %w", err)
	}
	f, err := CompileShader(fs, gl.FRAGMENT_SHADER)
	if err != nil {
		gl.DeleteShader(v)
		return 0, fmt.Errorf("fragment: %w", err)
	}
	p := gl.CreateProgram()
	gl.AttachShader(p, v)
	gl.AttachShader(p, f)
	gl.LinkProgram(p)
	gl.DeleteShader(v)
	gl.DeleteShader(f)
	var status int32
	gl.GetProgramiv(p, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var n int32
		gl.GetProgramiv(p, gl.INFO_LOG_LENGTH, &n)
		buf := make([]uint8, n+1)
		gl.GetProgramInfoLog(p, n, nil, &buf[0])
		gl.DeleteProgram(p)
		return 0, fmt.Errorf("link: %s", buf)
	}
	return p, nil
}

// CompileShader compiles a single shader of the given kind (gl.VERTEX_SHADER etc.).
func CompileShader(src string, kind uint32) (uint32, error) {
	sh := gl.CreateShader(kind)
	cstr, free := gl.Strs(src)
	gl.ShaderSource(sh, 1, cstr, nil)
	free()
	gl.CompileShader(sh)
	var status int32
	gl.GetShaderiv(sh, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var n int32
		gl.GetShaderiv(sh, gl.INFO_LOG_LENGTH, &n)
		buf := make([]uint8, n+1)
		gl.GetShaderInfoLog(sh, n, nil, &buf[0])
		gl.DeleteShader(sh)
		return 0, fmt.Errorf("%s", buf)
	}
	return sh, nil
}

// ── Vec3 helpers ──────────────────────────────────────────────────────────────

func Add3(a, b [3]float32) [3]float32 {
	return [3]float32{a[0] + b[0], a[1] + b[1], a[2] + b[2]}
}
func Sub3(a, b [3]float32) [3]float32 {
	return [3]float32{a[0] - b[0], a[1] - b[1], a[2] - b[2]}
}

// Scale3 multiplies vec3 a by scalar s.
func Scale3(a [3]float32, s float32) [3]float32 {
	return [3]float32{a[0] * s, a[1] * s, a[2] * s}
}
func Dot3(a, b [3]float32) float32 {
	return a[0]*b[0] + a[1]*b[1] + a[2]*b[2]
}
func Cross3(a, b [3]float32) [3]float32 {
	return [3]float32{
		a[1]*b[2] - a[2]*b[1],
		a[2]*b[0] - a[0]*b[2],
		a[0]*b[1] - a[1]*b[0],
	}
}
func Norm3(a [3]float32) [3]float32 {
	l := float32(math.Sqrt(float64(Dot3(a, a))))
	if l == 0 {
		return [3]float32{0, 0, -1}
	}
	return [3]float32{a[0] / l, a[1] / l, a[2] / l}
}

// ── Matrix math (column-major 4×4) ───────────────────────────────────────────

func Identity() [16]float32 {
	return [16]float32{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}
}

func MatMul(a, b [16]float32) [16]float32 {
	var m [16]float32
	for col := 0; col < 4; col++ {
		for row := 0; row < 4; row++ {
			var v float32
			for k := 0; k < 4; k++ {
				v += a[k*4+row] * b[col*4+k]
			}
			m[col*4+row] = v
		}
	}
	return m
}

func Translate3(tx, ty, tz float32) [16]float32 {
	m := Identity()
	m[12], m[13], m[14] = tx, ty, tz
	return m
}

// ScaleU returns a uniform-scale matrix (all three axes scaled by s).
func ScaleU(s float32) [16]float32 {
	m := Identity()
	m[0], m[5], m[10] = s, s, s
	return m
}

// RotX returns a rotation matrix around the X axis (angle in radians).
func RotX(a float32) [16]float32 {
	c := float32(math.Cos(float64(a)))
	s := float32(math.Sin(float64(a)))
	m := Identity()
	m[5] = c; m[6] = s; m[9] = -s; m[10] = c
	return m
}

// RotY returns a rotation matrix around the Y axis (angle in radians).
func RotY(a float32) [16]float32 {
	c := float32(math.Cos(float64(a)))
	s := float32(math.Sin(float64(a)))
	m := Identity()
	m[0] = c; m[2] = -s; m[8] = s; m[10] = c
	return m
}

// RotZ returns a rotation matrix around the Z axis (angle in radians).
func RotZ(a float32) [16]float32 {
	c := float32(math.Cos(float64(a)))
	s := float32(math.Sin(float64(a)))
	m := Identity()
	m[0] = c; m[1] = s; m[4] = -s; m[5] = c
	return m
}

func Perspective(fovY, aspect, near, far float32) [16]float32 {
	f := float32(1.0 / math.Tan(float64(fovY)*0.5))
	m := [16]float32{}
	m[0] = f / aspect
	m[5] = f
	m[10] = (far + near) / (near - far)
	m[11] = -1
	m[14] = (2 * far * near) / (near - far)
	return m
}

func LookAt(eye, centre, up [3]float32) [16]float32 {
	f := Norm3(Sub3(centre, eye))
	r := Norm3(Cross3(f, up))
	u := Cross3(r, f)
	m := Identity()
	m[0] = r[0]; m[4] = r[1]; m[8] = r[2]
	m[1] = u[0]; m[5] = u[1]; m[9] = u[2]
	m[2] = -f[0]; m[6] = -f[1]; m[10] = -f[2]
	m[12] = -Dot3(r, eye)
	m[13] = -Dot3(u, eye)
	m[14] = Dot3(f, eye)
	return m
}

func ToRad(deg float32) float32 { return deg * math.Pi / 180 }

// ── Free-fly Camera ───────────────────────────────────────────────────────────

// Camera is a free-fly first-person camera driven by yaw + pitch.
// All exported fields may be set before or during use.
type Camera struct {
	Pos   [3]float32
	Yaw   float32 // degrees; -90 faces -Z
	Pitch float32 // degrees; clamped to ±89

	Speed float32 // movement speed in units/second (default 5.0)
	Sens  float32 // mouse sensitivity in degrees/pixel (default 0.12)

	rmbDown    bool
	lastX      float64
	lastY      float64
	firstMouse bool
}

// NewCamera returns a Camera centred at pos, facing -Z, with sensible defaults.
func NewCamera(pos [3]float32) *Camera {
	return &Camera{
		Pos:        pos,
		Yaw:        -90,
		Speed:      5.0,
		Sens:       0.12,
		firstMouse: true,
	}
}

// Front returns the unit forward vector derived from Yaw and Pitch.
func (c *Camera) Front() [3]float32 {
	yRad := float64(c.Yaw) * math.Pi / 180
	pRad := float64(c.Pitch) * math.Pi / 180
	return Norm3([3]float32{
		float32(math.Cos(pRad) * math.Cos(yRad)),
		float32(math.Sin(pRad)),
		float32(math.Cos(pRad) * math.Sin(yRad)),
	})
}

// ViewMatrix returns lookAt(Pos, Pos+Front, worldUp).
func (c *Camera) ViewMatrix() [16]float32 {
	return LookAt(c.Pos, Add3(c.Pos, c.Front()), [3]float32{0, 1, 0})
}

// HandleKeys moves the camera. Pass win.GetKey(key)==glfw.Press for each
// direction: forward (W), back (S), left (A), right (D), up (E), down (Q).
func (c *Camera) HandleKeys(fwd, back, left, right, up, down bool, dt float32) {
	front := c.Front()
	r := Norm3(Cross3(front, [3]float32{0, 1, 0}))
	vel := c.Speed * dt
	if fwd   { c.Pos = Add3(c.Pos, Scale3(front, vel)) }
	if back  { c.Pos = Sub3(c.Pos, Scale3(front, vel)) }
	if left  { c.Pos = Sub3(c.Pos, Scale3(r, vel)) }
	if right { c.Pos = Add3(c.Pos, Scale3(r, vel)) }
	if up    { c.Pos[1] += vel }
	if down  { c.Pos[1] -= vel }
}

// SetRMB should be called from SetMouseButtonCallback.
// Pass (action == glfw.Press) when btn == glfw.MouseButtonRight.
func (c *Camera) SetRMB(pressed bool) {
	c.rmbDown = pressed
	c.firstMouse = true
}

// MousePos should be called from SetCursorPosCallback with the raw x, y.
// Does nothing if RMB is not held.
func (c *Camera) MousePos(x, y float64) {
	if !c.rmbDown {
		return
	}
	if c.firstMouse {
		c.lastX, c.lastY = x, y
		c.firstMouse = false
		return
	}
	c.Yaw += float32(x-c.lastX) * c.Sens
	c.Pitch += float32(c.lastY-y) * c.Sens // inverted: window Y is down
	c.lastX, c.lastY = x, y
	if c.Pitch > 89 {
		c.Pitch = 89
	}
	if c.Pitch < -89 {
		c.Pitch = -89
	}
}

// Scroll should be called from SetScrollCallback with the yoff value.
// It adjusts Speed (clamped to [0.5, 50]).
func (c *Camera) Scroll(yoff float64, minSpeed, maxSpeed float32) {
	c.Speed += float32(yoff) * 0.5
	if c.Speed < minSpeed {
		c.Speed = minSpeed
	}
	if c.Speed > maxSpeed {
		c.Speed = maxSpeed
	}
}
