package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestTimerCheatsheetHasAllGroups(t *testing.T) {
	var output bytes.Buffer
	command := newRootCommand()
	command.SetOut(&output)
	command.SetErr(&output)
	command.SetArgs([]string{"--no-color", "cheat"})

	if err := command.Execute(); err != nil {
		t.Fatalf("execute cheat: %v", err)
	}
	result := output.String()
	for _, want := range []string{"Jungle & farm", "Map pickups", "Objectives", "Small camps", "Soul Urn", TimerReferenceDate} {
		if !strings.Contains(result, want) {
			t.Fatalf("cheatsheet missing %q:\n%s", want, result)
		}
	}
}

func TestTimerSubcommandsAndAliasesSelectAGroup(t *testing.T) {
	for _, args := range [][]string{
		{"--no-color", "timers", "camps"},
		{"--no-color", "timers", "jungle"},
		{"--no-color", "timers", "objectives"},
	} {
		var output bytes.Buffer
		command := newRootCommand()
		command.SetOut(&output)
		command.SetErr(&output)
		command.SetArgs(args)

		if err := command.Execute(); err != nil {
			t.Fatalf("execute %q: %v", args, err)
		}
		if strings.Contains(output.String(), "Map pickups") {
			t.Fatalf("%q should not include unrelated groups:\n%s", args, output.String())
		}
	}
}

func TestTimersJSONIsStructuredAndHasNoTerminalFormatting(t *testing.T) {
	var output bytes.Buffer
	command := newRootCommand()
	command.SetOut(&output)
	command.SetErr(&output)
	command.SetArgs([]string{"--json", "timers", "pickups"})

	if err := command.Execute(); err != nil {
		t.Fatalf("execute JSON timers: %v", err)
	}
	if strings.Contains(output.String(), "\x1b[") {
		t.Fatalf("JSON contains ANSI formatting: %q", output.String())
	}
	var result struct {
		Reference struct {
			Source         string `json:"source"`
			CheckedThrough string `json:"checked_through"`
		} `json:"reference"`
		Groups []timerGroup `json:"groups"`
	}
	if err := json.Unmarshal(output.Bytes(), &result); err != nil {
		t.Fatalf("decode JSON: %v\n%s", err, output.String())
	}
	if result.Reference.Source != "Deadlore built-in timer reference" || result.Reference.CheckedThrough != TimerReferenceDate {
		t.Fatalf("unexpected reference: %#v", result.Reference)
	}
	if len(result.Groups) != 1 || result.Groups[0].Name != "Map pickups" || len(result.Groups[0].Timers) != 2 {
		t.Fatalf("unexpected groups: %#v", result.Groups)
	}
}
