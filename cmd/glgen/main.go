// Command glgen generates the OpenGL 2.1 binding files (package.go and init.go)
// from the official Khronos OpenGL XML registry.
//
// Usage:
//
//	go run ./cmd/glgen [flags]
//
// Flags:
//
//	-xml  path   Path to gl.xml (downloads and caches if omitted)
//	-out  dir    Output directory (default: v2.1/gl)
//	-ver  x.y    Maximum GL version (default: 2.1)
//
//go:generate go run ./cmd/glgen
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

const glxmlURL = "https://raw.githubusercontent.com/KhronosGroup/OpenGL-Registry/main/xml/gl.xml"

func main() {
	var (
		xmlPath = flag.String("xml", "", "path to gl.xml (downloads if empty)")
		outDir  = flag.String("out", "v2.1/gl", "output directory for generated files")
		maxVer  = flag.String("ver", "2.1", "maximum GL version to include")
		api     = flag.String("api", "gl", "API to generate: gl or gles2")
	)
	flag.Parse()

	xmlFile := *xmlPath
	if xmlFile == "" {
		// Use a local cache next to the generator source.
		cached := filepath.Join("cmd", "glgen", "gl.xml")
		if _, err := os.Stat(cached); err == nil {
			fmt.Fprintf(os.Stderr, "glgen: using cached %s\n", cached)
			xmlFile = cached
		} else {
			fmt.Fprintf(os.Stderr, "glgen: downloading gl.xml from Khronos...\n")
			if err := downloadFile(glxmlURL, cached); err != nil {
				log.Fatalf("glgen: download failed: %v", err)
			}
			fmt.Fprintf(os.Stderr, "glgen: saved to %s\n", cached)
			xmlFile = cached
		}
	}

	if *api != "gl" && *api != "gles2" {
		log.Fatalf("glgen: -api must be 'gl' or 'gles2', got %q", *api)
	}

	reg, err := parseRegistry(xmlFile)
	if err != nil {
		log.Fatalf("glgen: parse error: %v", err)
	}

	funcs, consts := collect(reg, *maxVer, *api)
	fmt.Fprintf(os.Stderr, "glgen: collected %d functions, %d constants (%s ≤ %s)\n",
		len(funcs), len(consts), *api, *maxVer)

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		log.Fatalf("glgen: mkdir %s: %v", *outDir, err)
	}

	// package.go
	pkgPath := filepath.Join(*outDir, "package.go")
	if err := writeFile(pkgPath, func(w io.Writer) { writePackageGo(w, funcs, consts, *api, *maxVer) }); err != nil {
		log.Fatalf("glgen: %v", err)
	}
	fmt.Fprintf(os.Stderr, "glgen: wrote %s\n", pkgPath)

	// init.go
	initPath := filepath.Join(*outDir, "init.go")
	if err := writeFile(initPath, func(w io.Writer) { writeInitGo(w, funcs, *api, *maxVer) }); err != nil {
		log.Fatalf("glgen: %v", err)
	}
	fmt.Fprintf(os.Stderr, "glgen: wrote %s\n", initPath)
}

func writeFile(path string, fn func(io.Writer)) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	fn(f)
	return f.Close()
}

func downloadFile(url, dest string) error {
	resp, err := http.Get(url) //nolint:noctx
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d fetching %s", resp.StatusCode, url)
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}
