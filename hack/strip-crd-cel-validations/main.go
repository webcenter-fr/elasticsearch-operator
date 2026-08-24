// Command strip-crd-cel-validations removes `x-kubernetes-validations` CEL
// blocks from generated CRD YAML files.
//
// The OpenShift Route types embedded in the operator's CRDs (via shared.Route /
// routev1.RouteSpec) carry `+kubebuilder:validation:XValidation` markers that
// controller-gen emits as `x-kubernetes-validations` blocks. The
// apiextensions-apiserver enforces a static CEL cost budget that these rules
// exceed (~100x), so the CRDs cannot be installed on vanilla Kubernetes (and
// therefore in envtest). This tool strips only those Route-schema blocks.
//
// It is line-based (surgical): it removes the key and its block list without
// touching any other byte. Any block that is NOT nested under a `route`/
// `routes` key is reported on stderr and makes the tool exit non-zero, so a
// future operator-owned CEL validation is never silently removed. It is
// idempotent.
package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// keyRe matches a YAML mapping key line: "<indent><key>:" (with optional "- "
// list-item prefix), with an optional inline value.
var keyRe = regexp.MustCompile(`^(\s*)(?:- )?([A-Za-z0-9_.@/-]+):(?:\s|$)`)

func main() {
	patterns := os.Args[1:]
	if len(patterns) == 0 {
		patterns = []string{"config/crd/bases/*.yaml", "config/crd/externals/*.yaml"}
	}

	exitCode := 0
	for _, pattern := range patterns {
		files, err := filepath.Glob(pattern)
		if err != nil {
			fmt.Fprintf(os.Stderr, "glob %s: %v\n", pattern, err)
			exitCode = 1
			continue
		}
		for _, file := range files {
			changed, warnings, err := stripFile(file)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error processing %s: %v\n", file, err)
				exitCode = 1
				continue
			}
			if changed {
				fmt.Printf("stripped x-kubernetes-validations from %s\n", file)
			}
			for _, w := range warnings {
				fmt.Fprintf(os.Stderr, "WARNING %s: %s\n", file, w)
				exitCode = 1
			}
		}
	}
	os.Exit(exitCode)
}

func stripFile(path string) (changed bool, warnings []string, err error) {
	f, err := os.Open(path)
	if err != nil {
		return false, nil, err
	}
	defer f.Close()

	var lines []string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1024*1024), 16*1024*1024)
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	if err := sc.Err(); err != nil {
		return false, nil, err
	}

	out := make([]string, 0, len(lines))
	i := 0
	for i < len(lines) {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		if trimmed == "x-kubernetes-validations:" {
			indent := len(line) - len(strings.TrimLeft(line, " "))
			if !underRouteKey(lines, i, indent) {
				warnings = append(warnings, fmt.Sprintf(
					"line %d: x-kubernetes-validations is not nested under a route/routes key; refusing to silently drop an operator-owned CEL validation", i+1))
			}
			changed = true
			i++
			// Consume the block list: "- item" lines at the same indent and
			// continuation lines indented deeper.
			for i < len(lines) {
				l2 := lines[i]
				t2 := strings.TrimSpace(l2)
				if t2 == "" {
					break
				}
				i2 := len(l2) - len(strings.TrimLeft(l2, " "))
				if i2 > indent {
					i++
					continue
				}
				if i2 == indent && strings.HasPrefix(t2, "- ") {
					i++
					continue
				}
				break
			}
			continue
		}

		out = append(out, line)
		i++
	}

	if !changed {
		return false, warnings, nil
	}
	if err := os.WriteFile(path, []byte(strings.Join(out, "\n")+"\n"), 0o644); err != nil {
		return false, nil, err
	}
	return true, warnings, nil
}

// underRouteKey reports whether the line at index i (a mapping key at the given
// indent) is nested under a "route" or "routes" mapping key. It walks the actual
// ancestor chain upward: at each step it finds the nearest enclosing key with a
// strictly shallower indent, checks whether that key is route/routes, then
// continues from that key's indent. Tracking the current indent level (rather
// than comparing every shallower line against the original indent) is what
// distinguishes a true ancestor from a sibling "routes" key that merely appears
// earlier in the file at the same level as the block's real parent.
func underRouteKey(lines []string, i, indent int) bool {
	current := indent
	for j := i - 1; j >= 0; j-- {
		line := lines[j]
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		jIndent := len(line) - len(strings.TrimLeft(line, " "))
		if jIndent >= current {
			continue // sibling or deeper, not an ancestor
		}
		m := keyRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		key := m[2]
		if key == "route" || key == "routes" {
			return true
		}
		current = jIndent
	}
	return false
}
