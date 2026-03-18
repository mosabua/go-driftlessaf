/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package testevals

import (
	"fmt"
	"strings"
)

// WordWrap breaks s into lines of at most width characters, splitting on spaces.
func WordWrap(s string, width int) []string {
	words := strings.Fields(s)
	if len(words) == 0 {
		return []string{""}
	}
	var lines []string
	line := words[0]
	for _, w := range words[1:] {
		if len(line)+1+len(w) > width {
			lines = append(lines, line)
			line = w
		} else {
			line += " " + w
		}
	}
	return append(lines, line)
}

// YAMLScalar formats a string as a YAML scalar, using >- for long or
// multiline strings. The returned string never ends with a newline.
func YAMLScalar(s, indent string) string {
	if !strings.Contains(s, "\n") && len(s) <= 80 {
		return fmt.Sprintf("%q", s)
	}
	var b strings.Builder
	b.WriteString(">-\n")
	for _, line := range WordWrap(strings.ReplaceAll(s, "\n", " "), 78) {
		b.WriteString(indent + "  " + line + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}
