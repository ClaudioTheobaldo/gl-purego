package main

import "testing"

// ── mapCType ──────────────────────────────────────────────────────────────────

func TestMapCType_Void(t *testing.T) {
	got, nu := mapCType("void")
	if got != "" || nu {
		t.Fatalf("void: want (\"\", false), got (%q, %v)", got, nu)
	}
}

func TestMapCType_Scalars(t *testing.T) {
	cases := []struct {
		c    string
		want string
	}{
		{"GLenum", "uint32"},
		{"GLbitfield", "uint32"},
		{"GLuint", "uint32"},
		{"GLint", "int32"},
		{"GLsizei", "int32"},
		{"GLfloat", "float32"},
		{"GLclampf", "float32"},
		{"GLdouble", "float64"},
		{"GLboolean", "bool"},
		{"GLubyte", "uint8"},
		{"GLchar", "uint8"},
		{"GLsizeiptr", "int"},
		{"GLintptr", "int"},
		{"GLuint64", "uint64"},
		{"GLint64", "int64"},
		{"GLsync", "uintptr"},
	}
	for _, tc := range cases {
		got, nu := mapCType(tc.c)
		if got != tc.want {
			t.Errorf("mapCType(%q) = %q, want %q", tc.c, got, tc.want)
		}
		if nu {
			t.Errorf("mapCType(%q): unexpected needsUnsafe=true", tc.c)
		}
	}
}

func TestMapCType_Pointers(t *testing.T) {
	cases := []struct {
		c         string
		want      string
		wantUnsafe bool
	}{
		{"const GLvoid *", "unsafe.Pointer", true},
		{"GLvoid *", "unsafe.Pointer", true},
		{"const void *", "unsafe.Pointer", true},
		{"void *", "unsafe.Pointer", true},
		{"const GLchar *", "*uint8", false},
		{"GLchar *", "*uint8", false},
		{"const GLchar *const*", "**uint8", false},
		{"const GLfloat *", "*float32", false},
		{"GLfloat *", "*float32", false},
		{"const GLint *", "*int32", false},
		{"GLint *", "*int32", false},
		{"GLuint *", "*uint32", false},
		// GLboolean pointer → *uint8, not *bool
		{"GLboolean *", "*uint8", false},
		{"const GLboolean *", "*uint8", false},
		// GLubyte pointer (e.g. glGetString return type)
		{"const GLubyte *", "*uint8", false},
	}
	for _, tc := range cases {
		got, nu := mapCType(tc.c)
		if got != tc.want {
			t.Errorf("mapCType(%q) = %q, want %q", tc.c, got, tc.want)
		}
		if nu != tc.wantUnsafe {
			t.Errorf("mapCType(%q) needsUnsafe = %v, want %v", tc.c, nu, tc.wantUnsafe)
		}
	}
}

// ── name helpers ──────────────────────────────────────────────────────────────

func TestGoFuncName(t *testing.T) {
	cases := [][2]string{
		{"glActiveTexture", "ActiveTexture"},
		{"glBindVertexArray", "BindVertexArray"},
		{"glCreateShader", "CreateShader"},
	}
	for _, tc := range cases {
		if got := goFuncName(tc[0]); got != tc[1] {
			t.Errorf("goFuncName(%q) = %q, want %q", tc[0], got, tc[1])
		}
	}
}

func TestVarName(t *testing.T) {
	cases := [][2]string{
		{"glActiveTexture", "gpActiveTexture"},
		{"glBindVertexArray", "gpBindVertexArray"},
		{"glCreateShader", "gpCreateShader"},
	}
	for _, tc := range cases {
		if got := varName(tc[0]); got != tc[1] {
			t.Errorf("varName(%q) = %q, want %q", tc[0], got, tc[1])
		}
	}
}

func TestConstName(t *testing.T) {
	cases := [][2]string{
		{"GL_POINTS", "POINTS"},
		{"GL_TRIANGLES", "TRIANGLES"},
		{"GL_TRUE", "TRUE"},
		{"GL_TEXTURE_2D", "TEXTURE_2D"},
	}
	for _, tc := range cases {
		if got := constName(tc[0]); got != tc[1] {
			t.Errorf("constName(%q) = %q, want %q", tc[0], got, tc[1])
		}
	}
}

func TestSafeParamName(t *testing.T) {
	cases := [][2]string{
		{"type", "xtype"},
		{"func", "xfunc"},
		{"range", "xrange"},
		{"string", "xstring"},
		{"near", "zNear"},
		{"far", "zFar"},
		{"index", "index"},   // no change
		{"shader", "shader"}, // no change
	}
	for _, tc := range cases {
		if got := safeParamName(tc[0]); got != tc[1] {
			t.Errorf("safeParamName(%q) = %q, want %q", tc[0], got, tc[1])
		}
	}
}

// ── extractNameAndCType ───────────────────────────────────────────────────────

func TestExtractNameAndCType(t *testing.T) {
	cases := []struct {
		inner     string
		wantName  string
		wantCType string
	}{
		{
			`<ptype>GLuint</ptype> <name>glCreateProgram</name>`,
			"glCreateProgram", "GLuint",
		},
		{
			`void <name>glActiveTexture</name>`,
			"glActiveTexture", "void",
		},
		{
			`const <ptype>GLubyte</ptype> *<name>glGetString</name>`,
			"glGetString", "const GLubyte *",
		},
		{
			`<ptype>GLenum</ptype> <name>target</name>`,
			"target", "GLenum",
		},
		{
			`const <ptype>GLchar</ptype> *const* <name>string</name>`,
			"string", "const GLchar *const*",
		},
		{
			`const <ptype>GLvoid</ptype> * <name>pointer</name>`,
			"pointer", "const GLvoid *",
		},
	}
	for _, tc := range cases {
		gotName, gotCType := extractNameAndCType(tc.inner)
		if gotName != tc.wantName {
			t.Errorf("extractNameAndCType(%q): name = %q, want %q", tc.inner, gotName, tc.wantName)
		}
		if gotCType != tc.wantCType {
			t.Errorf("extractNameAndCType(%q): ctype = %q, want %q", tc.inner, gotCType, tc.wantCType)
		}
	}
}
