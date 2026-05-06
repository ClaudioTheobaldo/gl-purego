package main_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestGenerateStable runs the generator for each GL version and verifies that
// the output is byte-for-byte identical to the files already committed.
// A failure here means either gl.xml changed upstream (expected occasionally)
// or a code change accidentally altered the generator's output.
//
// The test is skipped when network access is unavailable and no cached gl.xml
// exists, to avoid CI failures in offline environments.
func TestGenerateStable(t *testing.T) {
	// Locate repo root: two levels up from cmd/glgen.
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}

	// Check cached gl.xml exists; if not, skip rather than hit the network.
	cachedXML := filepath.Join(repoRoot, "cmd", "glgen", "gl.xml")
	if _, err := os.Stat(cachedXML); err != nil {
		t.Skip("gl.xml not cached; skipping stability test (run the generator once to cache it)")
	}

	versions := []struct {
		ver    string
		out    string
		compat bool
	}{
		{"2.1", "v2.1/gl", false},
		{"3.3", "v3.3-core/gl", false},
		{"4.1", "v4.1-core/gl", false},
		{"4.6", "v4.6-core/gl", false},
		{"3.3", "v3.3-compatibility/gl", true},
		{"4.1", "v4.1-compatibility/gl", true},
		{"4.6", "v4.6-compatibility/gl", true},
	}

	for _, v := range versions {
		v := v
		name := "v" + v.ver
		if v.compat {
			name += "-compat"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Run generator into a temp directory.
			tmp := t.TempDir()
			args := []string{"run", "./cmd/glgen/",
				"-ver", v.ver,
				"-xml", cachedXML,
				"-out", tmp,
				"-ext",
			}
			if v.compat {
				args = append(args, "-compat")
			}
			cmd := exec.Command("go", args...)
			cmd.Dir = repoRoot
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("generator failed: %v\n%s", err, out)
			}

			// Compare generated files against committed files.
			for _, fname := range []string{"package.go", "init.go"} {
				committed := filepath.Join(repoRoot, v.out, fname)
				generated := filepath.Join(tmp, fname)

				want, err := os.ReadFile(committed)
				if err != nil {
					t.Fatalf("reading committed %s: %v", fname, err)
				}
				got, err := os.ReadFile(generated)
				if err != nil {
					t.Fatalf("reading generated %s: %v", fname, err)
				}

				if !bytes.Equal(want, got) {
					t.Errorf("%s/%s: generated output differs from committed file\n"+
						"  run: go run ./cmd/glgen/ -ver %s -out %s -ext%s\n"+
						"  and commit the result",
						v.out, fname, v.ver, v.out,
						func() string {
							if v.compat {
								return " -compat"
							}
							return ""
						}(),
					)
				}
			}
		})
	}
}
