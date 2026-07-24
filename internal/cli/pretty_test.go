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

func TestWriteStatTilesFitTheCard(t *testing.T) {
	var output bytes.Buffer
	writeStatTiles(&output, "Key stats", []statTile{
		{Label: "Damage per second", Value: "50.1"},
		{Label: "Health", Value: "730 +33"},
		{Label: "Move speed", Value: "8.2m/s"},
	})

	for _, line := range strings.Split(strings.TrimSpace(output.String()), "\n") {
		if len([]rune(line)) != boxWidth {
			t.Fatalf("line has width %d, expected %d: %q", len([]rune(line)), boxWidth, line)
		}
	}
}

func TestPaintUsesColorOnlyWhenEnabled(t *testing.T) {
	original := colorEnabled
	t.Cleanup(func() { colorEnabled = original })

	colorEnabled = false
	if got := paint("Deadlore", ansiCyan); got != "Deadlore" {
		t.Fatalf("disabled color = %q", got)
	}
	colorEnabled = true
	if got := paint("Deadlore", ansiCyan); got != ansiCyan+"Deadlore"+ansiReset {
		t.Fatalf("enabled color = %q", got)
	}
}

func TestWriteAbilityListGroupsAbilitiesByHero(t *testing.T) {
	var output bytes.Buffer
	writeAbilityList(&output, []string{
		"Vindicta · Stake", "Vindicta · Flight", "Vindicta · Crow Familiar",
		"Haze · Sleep Dagger", "Haze · Smoke Bomb",
	})

	result := output.String()
	if !strings.Contains(result, "• Vindicta — Stake · Flight · Crow Familiar") {
		t.Fatalf("expected grouped Vindicta abilities, got:\n%s", result)
	}
	if !strings.Contains(result, "• Haze — Sleep Dagger · Smoke Bomb") {
		t.Fatalf("expected grouped Haze abilities, got:\n%s", result)
	}
}
