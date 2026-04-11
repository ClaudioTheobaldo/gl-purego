package main

import (
	"encoding/xml"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// Registry mirrors the top-level structure of gl.xml.
type Registry struct {
	XMLName  xml.Name    `xml:"registry"`
	Enums    []EnumGroup `xml:"enums"`
	Commands struct {
		Commands []Command `xml:"command"`
	} `xml:"commands"`
	Features []Feature `xml:"feature"`
}

// EnumGroup is a <enums> block.
type EnumGroup struct {
	Enums []Enum `xml:"enum"`
}

// Enum is a <enum> element.
type Enum struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
	API   string `xml:"api,attr"` // "gl", "gles2", etc.; empty = both
	Alias string `xml:"alias,attr"`
}

// Command is a <command> element.
type Command struct {
	Proto  MixedContent   `xml:"proto"`
	Params []MixedContent `xml:"param"`
}

// MixedContent captures raw innerXML for mixed-content elements (<proto>/<param>).
type MixedContent struct {
	Inner string `xml:",innerxml"`
}

// Feature is a <feature> element (e.g. <feature api="gl" number="2.1">).
type Feature struct {
	API      string    `xml:"api,attr"`
	Number   string    `xml:"number,attr"`
	Requires []Require `xml:"require"`
	Removes  []Require `xml:"remove"`
}

// Require / Remove sections list commands and enums by name.
type Require struct {
	Profile  string      `xml:"profile,attr"`
	Commands []NamedItem `xml:"command"`
	Enums    []NamedItem `xml:"enum"`
}

// NamedItem is a <command name="..."/> or <enum name="..."/> reference.
type NamedItem struct {
	Name string `xml:"name,attr"`
}

// ─── helpers ─────────────────────────────────────────────────────────────────

var (
	reName = regexp.MustCompile(`<name>([^<]+)</name>`)
	reTag  = regexp.MustCompile(`<[^>]+>`)
)

// extractNameAndCType pulls the identifier name and bare C type string out of
// the raw innerXML from a <proto> or <param> element.
//
// Example:
//
//	`const <ptype>GLubyte</ptype> *<name>glGetString</name>`
//	→ name="glGetString", ctype="const GLubyte *"
func extractNameAndCType(inner string) (name, ctype string) {
	m := reName.FindStringSubmatch(inner)
	if len(m) < 2 {
		// No <name> tag — return cleaned text as type, empty name
		return "", strings.TrimSpace(reTag.ReplaceAllString(inner, " "))
	}
	name = m[1]
	withoutName := reName.ReplaceAllString(inner, "")
	// Strip remaining XML tags (ptype etc.), normalise whitespace
	ctype = strings.TrimSpace(
		strings.Join(
			strings.Fields(reTag.ReplaceAllString(withoutName, " ")),
			" ",
		),
	)
	return
}

// parseRegistry parses gl.xml at path.
func parseRegistry(path string) (*Registry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var reg Registry
	if err := xml.NewDecoder(f).Decode(&reg); err != nil {
		return nil, err
	}
	return &reg, nil
}

// versionOK returns true when the feature version string (e.g. "2.1") is ≤ maxVersion.
func versionOK(number, maxVersion string) bool {
	a := parseVer(number)
	b := parseVer(maxVersion)
	if a[0] != b[0] {
		return a[0] < b[0]
	}
	return a[1] <= b[1]
}

func parseVer(v string) [2]int {
	parts := strings.SplitN(v, ".", 2)
	major, _ := strconv.Atoi(parts[0])
	minor := 0
	if len(parts) > 1 {
		minor, _ = strconv.Atoi(parts[1])
	}
	return [2]int{major, minor}
}
