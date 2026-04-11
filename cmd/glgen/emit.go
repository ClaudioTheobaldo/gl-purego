package main

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

// GLParam is a resolved function parameter.
type GLParam struct {
	Name   string
	GoType string
}

// GLFunc is a resolved GL function ready for code generation.
type GLFunc struct {
	GLName      string    // original: "glActiveTexture"
	GoName      string    // exported: "ActiveTexture"
	VarName     string    // slot:     "gpActiveTexture"
	RetType     string    // Go return type; "" for void
	Params      []GLParam
	Required    bool // false → load with required=false
	NeedsUnsafe bool
}

// GLConst is a resolved GL constant.
type GLConst struct {
	GoName string
	Value  string
}

// optionalFuncs lists entry points not in GL 2.1 core but commonly available
// and used by our examples. They are included with required=false.
var optionalFuncs = map[string]bool{
	// VAO — GL 3.0 core / ARB_vertex_array_object
	"glGenVertexArrays":    true,
	"glDeleteVertexArrays": true,
	"glBindVertexArray":    true,
	"glIsVertexArray":      true,
	// FBO — GL 3.0 core / ARB_framebuffer_object
	"glGenFramebuffers":         true,
	"glDeleteFramebuffers":      true,
	"glBindFramebuffer":         true,
	"glCheckFramebufferStatus":  true,
	"glFramebufferTexture2D":    true,
	"glFramebufferRenderbuffer": true,
	"glGenRenderbuffers":        true,
	"glDeleteRenderbuffers":     true,
	"glBindRenderbuffer":        true,
	"glRenderbufferStorage":     true,
	// Other 3.0+ commonly available
	"glGenerateMipmap": true,
	"glGetStringi":     true,
	// GL 4.1+
	"glClearDepthf": true,
}

// collect resolves all GL functions and constants for API "gl" up to maxVer,
// plus the optional extension functions in optionalFuncs.
func collect(reg *Registry, maxVer string) (funcs []GLFunc, consts []GLConst) {
	// ── index all commands by GL name ────────────────────────────────────────
	cmdMap := make(map[string]Command, len(reg.Commands.Commands))
	for _, cmd := range reg.Commands.Commands {
		name, _ := extractNameAndCType(cmd.Proto.Inner)
		if name != "" {
			cmdMap[name] = cmd
		}
	}

	// ── index all enums (GL only; skip GLES-only) ────────────────────────────
	enumMap := make(map[string]string)
	for _, grp := range reg.Enums {
		for _, e := range grp.Enums {
			if e.Name == "" || e.Value == "" {
				continue
			}
			if e.API == "gles2" || e.API == "gles1" {
				continue
			}
			if _, exists := enumMap[e.Name]; !exists {
				enumMap[e.Name] = e.Value
			}
		}
	}

	// ── walk features to build required sets for GL ≤ maxVer ─────────────────
	requiredCmds := make(map[string]bool)
	requiredEnums := make(map[string]bool)
	removedCmds := make(map[string]bool)
	removedEnums := make(map[string]bool)
	// featureCmds tracks commands that appear in the spec feature walk.
	// A function present here is always load(required=true), even if it also
	// appears in optionalFuncs (which only applies when the function is NOT in
	// the core spec for the target version — e.g. VAO is core in GL 3.0 but an
	// ARB extension in GL 2.1).
	featureCmds := make(map[string]bool)

	for _, feat := range reg.Features {
		if feat.API != "gl" {
			continue
		}
		if !versionOK(feat.Number, maxVer) {
			continue
		}
		for _, req := range feat.Requires {
			if req.Profile == "compatibility" {
				continue // skip compatibility-only additions
			}
			for _, c := range req.Commands {
				requiredCmds[c.Name] = true
				featureCmds[c.Name] = true
			}
			for _, e := range req.Enums {
				requiredEnums[e.Name] = true
			}
		}
		for _, rem := range feat.Removes {
			for _, c := range rem.Commands {
				removedCmds[c.Name] = true
			}
			for _, e := range rem.Enums {
				removedEnums[e.Name] = true
			}
		}
	}

	// Add optional extension functions that are available even in GL 2.1
	// contexts and used by our examples.
	for name := range optionalFuncs {
		if _, ok := cmdMap[name]; ok {
			requiredCmds[name] = true
		}
	}

	// ── resolve functions ─────────────────────────────────────────────────────
	seen := make(map[string]bool)
	for glName := range requiredCmds {
		if removedCmds[glName] || seen[glName] {
			continue
		}
		seen[glName] = true

		cmd, ok := cmdMap[glName]
		if !ok {
			continue
		}

		_, retCType := extractNameAndCType(cmd.Proto.Inner)
		retGoType, retNU := mapCType(retCType)

		needsUnsafe := retNU
		var params []GLParam
		for _, p := range cmd.Params {
			pName, pCType := extractNameAndCType(p.Inner)
			pGoType, pNU := mapCType(pCType)
			if pNU {
				needsUnsafe = true
			}
			params = append(params, GLParam{
				Name:   safeParamName(pName),
				GoType: pGoType,
			})
		}

		funcs = append(funcs, GLFunc{
			GLName:  glName,
			GoName:  goFuncName(glName),
			VarName: varName(glName),
			RetType: retGoType,
			Params:  params,
			// Required=true when the function is in the GL spec for this
			// version (featureCmds). When a function is only in optionalFuncs
			// (not found via the feature walk), it is an extra we include
			// but mark as required=false so Init doesn't fail if the driver
			// doesn't expose it.
			Required:    featureCmds[glName],
			NeedsUnsafe: needsUnsafe,
		})
	}
	sort.Slice(funcs, func(i, j int) bool { return funcs[i].GoName < funcs[j].GoName })

	// ── resolve constants ─────────────────────────────────────────────────────
	for enumName := range requiredEnums {
		if removedEnums[enumName] {
			continue
		}
		val, ok := enumMap[enumName]
		if !ok {
			continue
		}
		goName := constName(enumName)
		// Skip constants whose Go name starts with a digit (e.g. GL_2D → "2D")
		// — they are invalid Go identifiers and belong to the deprecated
		// fixed-function pipeline (evaluators, etc.).
		if goName == "" || goName[0] >= '0' && goName[0] <= '9' {
			continue
		}
		consts = append(consts, GLConst{
			GoName: goName,
			Value:  val,
		})
	}
	sort.Slice(consts, func(i, j int) bool { return consts[i].GoName < consts[j].GoName })

	return
}

// writePackageGo writes the generated package.go (constants + wrapper functions).
func writePackageGo(w io.Writer, funcs []GLFunc, consts []GLConst) {
	fmt.Fprintln(w, `// Code generated by cmd/glgen; DO NOT EDIT.`)
	fmt.Fprintln(w, `// Source: Khronos OpenGL XML registry (gl.xml), API: gl, Version: 2.1`)
	fmt.Fprintln(w)
	fmt.Fprintln(w, `package gl`)
	fmt.Fprintln(w)
	fmt.Fprintln(w, `import "unsafe"`)
	fmt.Fprintln(w)

	// Constants block
	fmt.Fprintln(w, `// -----------------------------------------------------------------------------`)
	fmt.Fprintln(w, `// Constants`)
	fmt.Fprintln(w, `// -----------------------------------------------------------------------------`)
	fmt.Fprintln(w)
	fmt.Fprintln(w, `const (`)
	for _, c := range consts {
		fmt.Fprintf(w, "\t%s = %s\n", c.GoName, c.Value)
	}
	fmt.Fprintln(w, `)`)
	fmt.Fprintln(w)

	// Wrapper functions
	fmt.Fprintln(w, `// -----------------------------------------------------------------------------`)
	fmt.Fprintln(w, `// Functions`)
	fmt.Fprintln(w, `// -----------------------------------------------------------------------------`)
	fmt.Fprintln(w)
	for _, f := range funcs {
		writeFunc(w, f)
	}
}

func writeFunc(w io.Writer, f GLFunc) {
	var paramParts []string
	for _, p := range f.Params {
		paramParts = append(paramParts, p.Name+" "+p.GoType)
	}

	sig := "func " + f.GoName + "(" + strings.Join(paramParts, ", ") + ")"
	if f.RetType != "" {
		sig += " " + f.RetType
	}

	var argParts []string
	for _, p := range f.Params {
		argParts = append(argParts, p.Name)
	}
	call := f.VarName + "(" + strings.Join(argParts, ", ") + ")"

	if f.RetType != "" {
		fmt.Fprintf(w, "%s { return %s }\n", sig, call)
	} else {
		fmt.Fprintf(w, "%s { %s }\n", sig, call)
	}
}

// writeInitGo writes the generated init.go (Init, InitWithProcAddrFunc, gp vars).
func writeInitGo(w io.Writer, funcs []GLFunc) {
	// File header
	fmt.Fprintln(w, `// Code generated by cmd/glgen; DO NOT EDIT.`)
	fmt.Fprintln(w, `// Source: Khronos OpenGL XML registry (gl.xml), API: gl, Version: 2.1`)
	fmt.Fprintln(w)
	fmt.Fprintln(w, `package gl`)
	fmt.Fprintln(w)
	fmt.Fprintln(w, `import (`)
	fmt.Fprintln(w, `	"fmt"`)
	fmt.Fprintln(w, `	"unsafe"`)
	fmt.Fprintln(w)
	fmt.Fprintln(w, `	"github.com/ebitengine/purego"`)
	fmt.Fprintln(w, `)`)
	fmt.Fprintln(w)

	// Static boilerplate: Init
	fmt.Fprintln(w, `// Init loads all OpenGL function symbols using the platform's default`)
	fmt.Fprintln(w, `// proc-address resolver (wglGetProcAddress on Windows, dlsym on macOS/Linux).`)
	fmt.Fprintln(w, `//`)
	fmt.Fprintln(w, `// A current OpenGL context must exist before calling Init.`)
	fmt.Fprintln(w, `func Init() error {`)
	fmt.Fprintln(w, `	if err := initProcAddr(); err != nil {`)
	fmt.Fprintln(w, `		return fmt.Errorf("gl: failed to load OpenGL library: %w", err)`)
	fmt.Fprintln(w, `	}`)
	fmt.Fprintln(w, `	return InitWithProcAddrFunc(getProcAddress)`)
	fmt.Fprintln(w, `}`)
	fmt.Fprintln(w)

	// Static boilerplate: InitWithProcAddrFunc
	fmt.Fprintln(w, `// InitWithProcAddrFunc loads all OpenGL function symbols using the supplied`)
	fmt.Fprintln(w, `// resolver. This is useful when the GL context is managed by a third-party`)
	fmt.Fprintln(w, `// windowing library that exposes its own GetProcAddress (e.g. GLFW).`)
	fmt.Fprintln(w, `//`)
	fmt.Fprintln(w, `//	gl.InitWithProcAddrFunc(func(name string) unsafe.Pointer {`)
	fmt.Fprintln(w, `//	    return glfw.GetCurrentContext().GetProcAddress(name)`)
	fmt.Fprintln(w, `//	})`)
	fmt.Fprintln(w, `func InitWithProcAddrFunc(getProcAddr func(name string) unsafe.Pointer) error {`)
	fmt.Fprintln(w, `	var missing []string`)
	fmt.Fprintln(w)
	fmt.Fprintln(w, `	load := func(fptr any, name string, required bool) {`)
	fmt.Fprintln(w, `		addr := getProcAddr(name)`)
	fmt.Fprintln(w, `		if addr == nil {`)
	fmt.Fprintln(w, `			if required {`)
	fmt.Fprintln(w, `				missing = append(missing, name)`)
	fmt.Fprintln(w, `			}`)
	fmt.Fprintln(w, `			return`)
	fmt.Fprintln(w, `		}`)
	fmt.Fprintln(w, `		purego.RegisterFunc(fptr, uintptr(addr))`)
	fmt.Fprintln(w, `	}`)
	fmt.Fprintln(w)

	// load() call for every function
	for _, f := range funcs {
		req := "true"
		if !f.Required {
			req = "false"
		}
		fmt.Fprintf(w, "\tload(&%s, %q, %s)\n", f.VarName, f.GLName, req)
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, `	if len(missing) > 0 {`)
	fmt.Fprintf(w, "\t\treturn fmt.Errorf(\"gl: %%d required functions not found: %%v\", len(missing), missing)\n")
	fmt.Fprintln(w, `	}`)
	fmt.Fprintln(w, `	return nil`)
	fmt.Fprintln(w, `}`)
	fmt.Fprintln(w)

	// gp var declarations — right-align types for readability
	fmt.Fprintln(w, `// function pointer slots — populated by InitWithProcAddrFunc.`)
	fmt.Fprintln(w, `var (`)

	maxLen := 0
	for _, f := range funcs {
		if len(f.VarName) > maxLen {
			maxLen = len(f.VarName)
		}
	}

	for _, f := range funcs {
		var paramTypes []string
		for _, p := range f.Params {
			paramTypes = append(paramTypes, p.GoType)
		}
		funcType := "func(" + strings.Join(paramTypes, ", ") + ")"
		if f.RetType != "" {
			funcType += " " + f.RetType
		}
		pad := strings.Repeat(" ", maxLen-len(f.VarName)+1)
		fmt.Fprintf(w, "\t%s%s%s\n", f.VarName, pad, funcType)
	}
	fmt.Fprintln(w, `)`)
}
