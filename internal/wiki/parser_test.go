package wiki

import (
	"strings"
	"testing"
	"time"
)

func TestParsePage(t *testing.T) {
	html := `<!doctype html><html><body>
<h1 id="firstHeading">Heroic Aura<a class="feedback-button">Give feedback</a></h1>
<div id="mw-content-text"><div class="mw-parser-output">
<table class="infobox"><tr><th>Tier</th><td>3</td></tr><tr><th>Cost</th><td>3,200</td></tr></table>
<div class="item-infobox"><div class="infobox-stat">+25% Spirit Lifesteal</div><div class="infobox-stat">+180 Bonus Health</div></div>
<p>Heroic Aura is a Tier 3 Weapon Item that grants nearby allies useful combat stats.</p>
<div class="ability-section-wrapper"><h3 class="ability-name">Aura Burst</h3><div class="ac-info-desc">Burst nearby enemies with a wave of spirit damage.</div></div>
<h2>Overview</h2><p>It improves a team fight when nearby allies can benefit from its aura.</p>
<h2>Update history</h2><p>The item has changed several times across patches.</p>
</div></div>
<footer><li id="footer-info-lastmod">This page was last edited on 18 July 2026, at 12:00.</li></footer>
<script>var mwConfig = {"wgRevisionId":95734};</script>
</body></html>`

	page, err := ParsePage(strings.NewReader(html), "https://deadlock.wiki/Heroic_Aura", time.Date(2026, 7, 23, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if page.Title != "Heroic Aura" {
		t.Fatalf("title = %q", page.Title)
	}
	if page.RevisionID != "95734" {
		t.Fatalf("revision = %q", page.RevisionID)
	}
	if page.LastModified == "" {
		t.Fatal("expected last modified text")
	}
	if len(page.Facts) != 4 || page.Facts[0].Value != "3" || page.Facts[2].Value != "+25% Spirit Lifesteal" {
		t.Fatalf("facts = %#v", page.Facts)
	}
	if len(page.Sections) != 2 || page.Sections[0].Title != "Overview" {
		t.Fatalf("sections = %#v", page.Sections)
	}
	if len(page.Abilities) != 1 || page.Abilities[0].Name != "Aura Burst" {
		t.Fatalf("abilities = %#v", page.Abilities)
	}
}
