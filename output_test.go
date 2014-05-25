package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestTagStream(t *testing.T) {
	for _, tc := range []struct {
		input  string
		output string
	}{
		{"foo\n", "PREFIXfoo\n"},
		{"foo", "PREFIXfoo\n"},
		{"foo\n\nbar", "PREFIXfoo\nPREFIX\nPREFIXbar\n"},
	} {
		var buf bytes.Buffer
		if err := TagStream("PREFIX", &buf, strings.NewReader(tc.input)); err != nil {
			t.Errorf("TagStream failed: %v", err)
			continue
		}
		if got := buf.String(); got != tc.output {
			t.Errorf("TagStream returned invalid output: got=%q, want=%q", got, tc.output)
		}
	}
}
