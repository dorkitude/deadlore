package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestWriteBoxWrapsLongText(t *testing.T) {
	var output bytes.Buffer
	writeBox(&output, "A title", []string{strings.Repeat("long ", 30)})

	result := output.String()
	if !strings.Contains(result, "╭─ A title") || !strings.Contains(result, "╰") {
		t.Fatalf("expected box borders, got:\n%s", result)
	}
	for _, line := range strings.Split(strings.TrimSpace(result), "\n") {
		if len([]rune(line)) != boxWidth {
			t.Fatalf("line has width %d, expected %d: %q", len([]rune(line)), boxWidth, line)
		}
	}
}
