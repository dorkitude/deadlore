package deadlockio

import (
	"context"
	"html"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/dorkitude/deadlore/internal/wiki"
)

type localized struct {
	English string `json:"english"`
}

type source struct {
	BuildID     string `json:"buildId"`
	GeneratedAt string `json:"generatedAt"`
}

type catalogEntry struct {
	Slug        string    `json:"slug"`
	DisplayName localized `json:"displayName"`
}

type catalogResponse struct {
	Heroes    []catalogEntry `json:"heroes"`
	Items     []catalogEntry `json:"items"`
	Mechanics []catalogEntry `json:"mechanics"`
	Source    source         `json:"source"`
}

func (c catalogResponse) entries(kind string) []string {
	var raw []catalogEntry
	switch kind {
	case "hero":
		raw = c.Heroes
	case "item":
		raw = c.Items
	case "mechanic":
		raw = c.Mechanics
	}
	entries := make([]string, 0, len(raw))
	for _, entry := range raw {
		if entry.DisplayName.English != "" && entry.Slug != "" {
			entries = append(entries, entry.DisplayName.English+"\t"+entry.Slug)
		}
	}
	sort.Strings(entries)
	return entries
}

type heroPayload struct {
	Slug        string             `json:"slug"`
	DisplayName localized          `json:"displayName"`
	Playstyle   localized          `json:"playstyle"`
	UpdatedAt   string             `json:"updatedAt"`
	SourcePath  string             `json:"sourcePath"`
	Stats       map[string]float64 `json:"stats"`
	Weapon      struct {
		Name            localized `json:"name"`
		DPS             float64   `json:"dps"`
		SustainedDPS    float64   `json:"sustainedDps"`
		BulletDamage    float64   `json:"bulletDamage"`
		RoundsPerSecond float64   `json:"roundsPerSecond"`
		ClipSize        float64   `json:"clipSize"`
		ReloadTime      float64   `json:"reloadTime"`
		BulletSpeed     float64   `json:"bulletSpeed"`
	} `json:"weapon"`
	Tags      []localized `json:"tags"`
	Abilities []struct {
		DisplayName localized `json:"displayName"`
		Description localized `json:"description"`
	} `json:"abilities"`
}

func (c *Client) fetchHero(ctx context.Context, slug string) (*wiki.Page, error) {
	var payload heroPayload
	if err := c.fetchJSON(ctx, "heroes/"+slug+".json", &payload); err != nil {
		return nil, err
	}
	page := &wiki.Page{Title: payload.DisplayName.English, URL: "https://deadlock.io/en/heroes/" + payload.Slug, Summary: clean(payload.Playstyle.English), RevisionID: payload.SourcePath, LastModified: "Deadlock.io updated: " + payload.UpdatedAt, FetchedAt: time.Now().UTC()}
	page.Facts = []wiki.Fact{
		{Label: "Damage Per Second", Value: number(payload.Weapon.DPS)}, {Label: "Sustained DPS", Value: number(payload.Weapon.SustainedDPS)}, {Label: "Bullet Damage", Value: number(payload.Weapon.BulletDamage)}, {Label: "Bullets per sec", Value: number(payload.Weapon.RoundsPerSecond)}, {Label: "Ammo", Value: number(payload.Weapon.ClipSize)}, {Label: "Reload Time", Value: number(payload.Weapon.ReloadTime) + "s"}, {Label: "Bullet Speed", Value: number(payload.Weapon.BulletSpeed)},
		{Label: "Health", Value: number(payload.Stats["EMaxHealth"])}, {Label: "Health Regen", Value: number(payload.Stats["EBaseHealthRegen"])}, {Label: "Move Speed", Value: number(payload.Stats["EMaxMoveSpeed"]) + "m/s"}, {Label: "Sprint Speed", Value: number(payload.Stats["ESprintSpeed"]) + "m/s"},
	}
	for _, tag := range payload.Tags {
		if tag.English != "" {
			page.Tags = append(page.Tags, tag.English)
		}
	}
	for _, ability := range payload.Abilities {
		if ability.DisplayName.English != "" {
			page.Abilities = append(page.Abilities, wiki.Ability{Name: ability.DisplayName.English, Description: clean(ability.Description.English)})
		}
	}
	return page, nil
}

type itemPayload struct {
	Slug        string    `json:"slug"`
	DisplayName localized `json:"displayName"`
	Description localized `json:"description"`
	UpdatedAt   string    `json:"updatedAt"`
	SourcePath  string    `json:"sourcePath"`
	Shop        struct {
		Cost float64 `json:"cost"`
		Tier float64 `json:"tier"`
	} `json:"shop"`
}

func (c *Client) fetchItem(ctx context.Context, slug string) (*wiki.Page, error) {
	var payload itemPayload
	if err := c.fetchJSON(ctx, "items/"+slug+".json", &payload); err != nil {
		return nil, err
	}
	page := &wiki.Page{Title: payload.DisplayName.English, URL: "https://deadlock.io/en/items/" + payload.Slug, Summary: clean(payload.Description.English), RevisionID: payload.SourcePath, LastModified: "Deadlock.io updated: " + payload.UpdatedAt, FetchedAt: time.Now().UTC()}
	if payload.Shop.Cost != 0 {
		page.Facts = append(page.Facts, wiki.Fact{Label: "Cost", Value: number(payload.Shop.Cost)})
	}
	if payload.Shop.Tier != 0 {
		page.Facts = append(page.Facts, wiki.Fact{Label: "Tier", Value: number(payload.Shop.Tier)})
	}
	return page, nil
}

type mechanicPayload struct {
	Slug        string    `json:"slug"`
	DisplayName localized `json:"displayName"`
	Summary     localized `json:"summary"`
	Takeaway    localized `json:"takeaway"`
	UpdatedAt   string    `json:"updatedAt"`
	SourcePath  string    `json:"sourcePath"`
	Sections    []struct {
		Title localized `json:"title"`
		Body  localized `json:"body"`
	} `json:"sections"`
	Aliases []localized `json:"aliases"`
}

func (c *Client) fetchMechanic(ctx context.Context, slug string) (*wiki.Page, error) {
	var payload mechanicPayload
	if err := c.fetchJSON(ctx, "mechanics/"+slug+".json", &payload); err != nil {
		return nil, err
	}
	page := &wiki.Page{Title: payload.DisplayName.English, URL: "https://deadlock.io/en/articles/mechanics/" + payload.Slug, Summary: clean(payload.Summary.English), RevisionID: payload.SourcePath, LastModified: "Deadlock.io updated: " + payload.UpdatedAt, FetchedAt: time.Now().UTC()}
	if text := clean(payload.Takeaway.English); text != "" {
		page.Sections = append(page.Sections, wiki.Section{Title: "Takeaway", Text: []string{text}})
	}
	for _, section := range payload.Sections {
		if text := clean(section.Body.English); text != "" {
			page.Sections = append(page.Sections, wiki.Section{Title: section.Title.English, Text: []string{text}})
		}
	}
	return page, nil
}

var tags = regexp.MustCompile(`<[^>]*>`)
var spaceBeforePunctuation = regexp.MustCompile(`\s+([.,!?;:])`)

func clean(value string) string {
	value = strings.TrimSpace(strings.Join(strings.Fields(html.UnescapeString(tags.ReplaceAllString(value, " "))), " "))
	return spaceBeforePunctuation.ReplaceAllString(value, "$1")
}
func number(value float64) string { return strconv.FormatFloat(value, 'f', -1, 64) }
