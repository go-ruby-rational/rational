// Copyright (c) the go-ruby-rational/rational authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rational

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
)

// sprintf is a thin alias so the oracle file reads without importing fmt
// directly; it exists only in the test binary.
func sprintf(format string, a ...any) string { return fmt.Sprintf(format, a...) }

// itoa renders an int the way Ruby's puts of an Integer does.
func itoa(n int) string { return strconv.Itoa(n) }

// boolStr renders a bool the way Ruby's puts of true/false does.
func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// mustFloat parses a float literal for an oracle case, failing the test on error.
func mustFloat(t *testing.T, s string) float64 {
	t.Helper()
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		t.Fatalf("ParseFloat(%q): %v", s, err)
	}
	return f
}

// formatFloatLikeRuby renders a float64 the way Ruby's Float#to_s does: the
// shortest round-trip form, but with a trailing ".0" for whole values (Ruby
// prints 3.0, not 3, where Go's 'g' would drop the fraction).
func formatFloatLikeRuby(f float64) string {
	s := strconv.FormatFloat(f, 'g', -1, 64)
	if !strings.ContainsAny(s, ".eE") {
		s += ".0"
	}
	return s
}
