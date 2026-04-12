//go:build ignore

// setup_workspace.go recreates the go.work file at the repository root.
//
// Run from the repository root:
//
//	go run ./scripts/setup_workspace.go
//
// This is necessary because go.work is gitignored (Go tooling convention —
// workspace files are considered local developer configuration).
package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	// Must be run from the repo root (where go.mod lives).
	if _, err := os.Stat("go.mod"); err != nil {
		log.Fatal("run this script from the gl-purego repository root")
	}

	// Remove stale go.work if present.
	_ = os.Remove("go.work")
	_ = os.Remove("go.work.sum")

	// Collect all module directories: root + all examples/*/
	modules := []string{"."}

	entries, err := os.ReadDir("examples")
	if err != nil {
		log.Fatalf("reading examples/: %v", err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		modPath := filepath.Join("examples", e.Name(), "go.mod")
		if _, err := os.Stat(modPath); err == nil {
			modules = append(modules, filepath.Join("examples", e.Name()))
		}
	}

	// go work init <modules...>
	args := append([]string{"work", "init"}, modules...)
	cmd := exec.Command("go", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatalf("go work init: %v", err)
	}

	// Add glfw-purego sibling repo if it exists.
	glfwPath := filepath.Join("..", "glfw-purego")
	if _, err := os.Stat(filepath.Join(glfwPath, "go.mod")); err == nil {
		cmd = exec.Command("go", "work", "use", glfwPath)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			log.Fatalf("go work use glfw-purego: %v", err)
		}
		fmt.Println("added ../glfw-purego")
	} else {
		fmt.Println("note: ../glfw-purego not found — examples that import it won't build")
		fmt.Println("      clone it alongside this repo: git clone https://github.com/ClaudioTheobaldo/glfw-purego ../glfw-purego")
	}

	fmt.Printf("\ngo.work created with %d modules. You're ready to go.\n", len(modules)+1)
	fmt.Println("  cd examples/08_lighting && CGO_ENABLED=0 go run .")
}
