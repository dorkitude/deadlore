package cli

import (
	"bytes"
	"math"
	"strings"
	"testing"

	"github.com/dorkitude/deadlore/internal/wiki"
)

func TestStatAtBoons(t *testing.T) {
	for _, test := range []struct {
		raw   string
		boons int
		want  float64
		ok    bool
	}{
		{raw: "730 +33", boons: 0, want: 730, ok: true},
		{raw: "730 +33", boons: 10, want: 1060, ok: true},
		{raw: "5.26 +0.143", boons: 10, want: 6.69, ok: true},
		{raw: "1,100", boons: 0, want: 1100, ok: true},
		{raw: "not a stat", boons: 0, ok: false},
	} {
		got, ok := statAtBoons(test.raw, test.boons)
		if ok != test.ok || (ok && math.Abs(got-test.want) > 0.000001) {
			t.Fatalf("statAtBoons(%q, %d) = (%v, %v), want (%v, %v)", test.raw, test.boons, got, ok, test.want, test.ok)
		}
	}
}

func TestDisplayStatAtBoonsRetainsUnitsAndExtraAmmoDetail(t *testing.T) {
	for _, test := range []struct {
		raw   string
		boons int
		want  string
	}{
		{raw: "2.35s", boons: 0, want: "2.35s"},
		{raw: "762m/s", boons: 0, want: "762m/s"},
		{raw: "25 x 0.5", boons: 0, want: "25 x 0.5"},
		{raw: "5.26 +0.143", boons: 10, want: "6.69"},
	} {
		if got := displayStatAtBoons(test.raw, test.boons); got != test.want {
			t.Fatalf("displayStatAtBoons(%q, %d) = %q, want %q", test.raw, test.boons, got, test.want)
		}
	}
}

func TestFindHeroMetricAcceptsUsefulAliases(t *testing.T) {
	for _, test := range []struct {
		input       string
		allowWeapon bool
		want        string
	}{
		{input: "hp", want: "health"},
		{input: "move speed", want: "move-speed"},
		{input: "bullets-per-sec", allowWeapon: true, want: "fire-rate"},
		{input: "reload", allowWeapon: true, want: "reload-time"},
	} {
		metric, found := findHeroMetric(test.input, test.allowWeapon)
		if !found || metric.Name != test.want {
			t.Fatalf("findHeroMetric(%q) = (%#v, %v), want %q", test.input, metric, found, test.want)
		}
	}
	if _, found := findHeroMetric("dps", false); found {
		t.Fatal("hero rank should not accept weapon metrics")
	}
}

func TestWeaponFactLinesIncludesOnlyWeaponStats(t *testing.T) {
	page := &wiki.Page{Facts: []wiki.Fact{
		{Label: "Health :", Value: "730 +33"},
		{Label: "Bullet Damage :", Value: "5.26 +0.143"},
		{Label: "Bullets per sec :", Value: "9.52"},
	}}
	lines := weaponFactLines(page, 10)
	joined := strings.Join(lines, "\n")
	if strings.Contains(joined, "Health") || !strings.Contains(joined, "6.69") || !strings.Contains(joined, "9.52") {
		t.Fatalf("unexpected weapon lines: %q", joined)
	}
}

func TestWeaponFactsAtBoonsUseScaledDisplayValues(t *testing.T) {
	page := &wiki.Page{Facts: []wiki.Fact{{Label: "Bullet Damage :", Value: "5.26 +0.143"}}}
	facts := weaponFactsAtBoons(page, 10)
	if facts["bullet-damage"] != "6.69" {
		t.Fatalf("bullet damage at 10 boons = %q, want 6.69", facts["bullet-damage"])
	}
}

func TestHeroCommandRoutesLegacyLookupAndAnalysisCommands(t *testing.T) {
	root := newRootCommand()
	hero, _, err := root.Find([]string{"hero", "weapon", "rank", "fire-rate"})
	if err != nil {
		t.Fatalf("find weapon rank command: %v", err)
	}
	if hero.Name() != "rank" {
		t.Fatalf("resolved command = %q, want rank", hero.Name())
	}

	var output bytes.Buffer
	command := newRootCommand()
	command.SetOut(&output)
	command.SetErr(&output)
	command.SetArgs([]string{"--no-color", "hero", "rank", "dps"})
	if err := command.Execute(); err == nil || !strings.Contains(err.Error(), "unknown hero metric") {
		t.Fatalf("hero rank dps error = %v, want hero-metric guidance", err)
	}
}
