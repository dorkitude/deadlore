package cli

import (
	"context"
	"fmt"
	"io"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/dorkitude/deadlore/internal/wiki"
	"github.com/spf13/cobra"
)

type heroMetric struct {
	Name       string
	Fact       string
	Aliases    []string
	Ascending  bool
	WeaponStat bool
}

var heroMetrics = []heroMetric{
	{Name: "health", Fact: "Health", Aliases: []string{"hp"}},
	{Name: "health-regen", Fact: "Health Regen", Aliases: []string{"regen", "health regen"}},
	{Name: "move-speed", Fact: "Move Speed", Aliases: []string{"move speed", "movespeed", "ms"}},
	{Name: "sprint-speed", Fact: "Sprint Speed", Aliases: []string{"sprint speed", "sprint", "sprintspeed"}},
	{Name: "dps", Fact: "Damage Per Second", Aliases: []string{"damage-per-second", "damage per second"}, WeaponStat: true},
	{Name: "fire-rate", Fact: "Bullets per sec", Aliases: []string{"bullets-per-sec", "bullets per sec", "bullets/sec", "bps"}, WeaponStat: true},
	{Name: "bullet-damage", Fact: "Bullet Damage", Aliases: []string{"bullet damage", "damage"}, WeaponStat: true},
	{Name: "ammo", Fact: "Ammo", WeaponStat: true},
	{Name: "reload-time", Fact: "Reload Time", Aliases: []string{"reload", "reload time"}, Ascending: true, WeaponStat: true},
	{Name: "bullet-velocity", Fact: "Bullet Velocity", Aliases: []string{"bullet velocity", "velocity"}, WeaponStat: true},
}

type heroRecord struct {
	Page   *wiki.Page
	Cached bool
}

type heroRankEntry struct {
	Hero         string         `json:"hero"`
	Value        float64        `json:"value"`
	DisplayValue string         `json:"display_value"`
	Source       map[string]any `json:"source"`
}

var statValuePattern = regexp.MustCompile(`^\s*([+-]?[0-9][0-9,]*(?:\.[0-9]+)?)(?:\s*\+\s*([+-]?[0-9][0-9,]*(?:\.[0-9]+)?))?`)

func newHeroCommand(options *options) *cobra.Command {
	hero := &cobra.Command{
		Use:           "hero <name>",
		Short:         "Look up, compare, and rank heroes",
		Args:          cobra.MinimumNArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(command *cobra.Command, args []string) error {
			if len(args) == 1 && strings.EqualFold(args[0], "list") {
				return listEntities(command.Context(), command, options, "hero")
			}
			return lookup(command.Context(), command, options, strings.Join(args, " "))
		},
	}
	hero.AddCommand(newHeroRankCommand(options))
	hero.AddCommand(newHeroCompareCommand(options))
	hero.AddCommand(newHeroFindCommand(options))
	hero.AddCommand(newHeroWeaponCommand(options))
	return hero
}

func newHeroRankCommand(options *options) *cobra.Command {
	command := &cobra.Command{
		Use:           "rank <metric>",
		Short:         "Rank heroes by a base stat",
		Long:          "Rank heroes by health, health-regen, move-speed, or sprint-speed. Use hero weapon rank for weapon stats.",
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(command *cobra.Command, args []string) error {
			metric, found := findHeroMetric(args[0], false)
			if !found || metric.WeaponStat {
				return fmt.Errorf("unknown hero metric %q (try health, health-regen, move-speed, or sprint-speed)", args[0])
			}
			boons, err := command.Flags().GetInt("boons")
			if err != nil {
				return err
			}
			return rankHeroes(command.Context(), command, options, metric, boons)
		},
	}
	command.Flags().Int("boons", 0, "evaluate stats after this many boons")
	return command
}

func newHeroWeaponCommand(options *options) *cobra.Command {
	weapon := &cobra.Command{
		Use:           "weapon <hero>",
		Aliases:       []string{"weapons", "gun", "guns"},
		Short:         "Show a hero's weapon stats",
		Args:          cobra.MinimumNArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(command *cobra.Command, args []string) error {
			boons, err := command.Flags().GetInt("boons")
			if err != nil {
				return err
			}
			if boons < 0 {
				return fmt.Errorf("--boons cannot be negative")
			}
			return showHeroWeapon(command.Context(), command, options, strings.Join(args, " "), boons)
		},
	}
	weapon.Flags().Int("boons", 0, "evaluate stats after this many boons")
	weapon.AddCommand(newHeroWeaponRankCommand(options))
	weapon.AddCommand(newHeroWeaponCompareCommand(options))
	return weapon
}

func newHeroWeaponRankCommand(options *options) *cobra.Command {
	command := &cobra.Command{
		Use:           "rank <metric>",
		Short:         "Rank heroes by weapon stats",
		Long:          "Rank by dps, fire-rate, bullet-damage, ammo, reload-time, or bullet-velocity.",
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(command *cobra.Command, args []string) error {
			metric, found := findHeroMetric(args[0], true)
			if !found || !metric.WeaponStat {
				return fmt.Errorf("unknown weapon metric %q (try dps, fire-rate, bullet-damage, ammo, reload-time, or bullet-velocity)", args[0])
			}
			boons, err := command.Flags().GetInt("boons")
			if err != nil {
				return err
			}
			return rankHeroes(command.Context(), command, options, metric, boons)
		},
	}
	command.Flags().Int("boons", 0, "evaluate stats after this many boons")
	return command
}

func newHeroCompareCommand(options *options) *cobra.Command {
	command := &cobra.Command{
		Use:           "compare <hero> <hero> [hero...]",
		Short:         "Compare base and weapon stats across heroes",
		Args:          cobra.MinimumNArgs(2),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(command *cobra.Command, args []string) error {
			boons, err := command.Flags().GetInt("boons")
			if err != nil {
				return err
			}
			return compareHeroes(command.Context(), command, options, args, boons, false)
		},
	}
	command.Flags().Int("boons", 0, "evaluate stats after this many boons")
	return command
}

func newHeroWeaponCompareCommand(options *options) *cobra.Command {
	command := &cobra.Command{
		Use:           "compare <hero> <hero> [hero...]",
		Short:         "Compare weapon stats across heroes",
		Args:          cobra.MinimumNArgs(2),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(command *cobra.Command, args []string) error {
			boons, err := command.Flags().GetInt("boons")
			if err != nil {
				return err
			}
			return compareHeroes(command.Context(), command, options, args, boons, true)
		},
	}
	command.Flags().Int("boons", 0, "evaluate stats after this many boons")
	return command
}

func newHeroFindCommand(options *options) *cobra.Command {
	command := &cobra.Command{
		Use:           "find --tag <tag>",
		Short:         "Find heroes with a matching tag",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(command *cobra.Command, _ []string) error {
			tag, err := command.Flags().GetString("tag")
			if err != nil {
				return err
			}
			if strings.TrimSpace(tag) == "" {
				return fmt.Errorf("--tag is required")
			}
			return findHeroes(command.Context(), command, options, tag)
		},
	}
	command.Flags().String("tag", "", "hero tag to match")
	return command
}

func findHeroMetric(name string, allowWeapon bool) (heroMetric, bool) {
	name = normalizeName(name)
	for _, metric := range heroMetrics {
		if metric.WeaponStat && !allowWeapon {
			continue
		}
		if normalizeName(metric.Name) == name {
			return metric, true
		}
		for _, alias := range metric.Aliases {
			if normalizeName(alias) == name {
				return metric, true
			}
		}
	}
	return heroMetric{}, false
}

func showHeroWeapon(ctx context.Context, command *cobra.Command, options *options, title string, boons int) error {
	client, err := clientFor(options)
	if err != nil {
		return err
	}
	page, cached, lookupError := client.Get(ctx, title, options.refresh)
	if page == nil {
		return lookupError
	}
	lines := weaponFactLines(page, boons)
	if len(lines) == 0 {
		return fmt.Errorf("the wiki did not provide weapon stats for %s", page.Title)
	}
	if options.json {
		if err := writeJSON(command, map[string]any{"hero": page.Title, "boons": boons, "weapon": weaponFactsAtBoons(page, boons), "source": sourceFor(page, cached)}); err != nil {
			return err
		}
		return warning(command, lookupError)
	}
	writeBox(command.OutOrStdout(), fmt.Sprintf("%s · Weapon · %d boons", page.Title, boons), lines)
	fmt.Fprintln(command.OutOrStdout())
	writeSource(command, page, cached)
	return warning(command, lookupError)
}

func rankHeroes(ctx context.Context, command *cobra.Command, options *options, metric heroMetric, boons int) error {
	if boons < 0 {
		return fmt.Errorf("--boons cannot be negative")
	}
	records, failed, err := loadHeroRecords(ctx, options)
	if err != nil {
		return err
	}
	entries := make([]heroRankEntry, 0, len(records))
	for _, record := range records {
		raw, found := factValue(record.Page, metric.Fact)
		if !found {
			continue
		}
		value, found := statAtBoons(raw, boons)
		if !found {
			continue
		}
		entries = append(entries, heroRankEntry{Hero: record.Page.Title, Value: value, DisplayValue: displayStatAtBoons(raw, boons), Source: sourceFor(record.Page, record.Cached)})
	}
	if len(entries) == 0 {
		return fmt.Errorf("no heroes exposed the %s stat", metric.Fact)
	}
	sort.SliceStable(entries, func(left, right int) bool {
		if entries[left].Value == entries[right].Value {
			return entries[left].Hero < entries[right].Hero
		}
		if metric.Ascending {
			return entries[left].Value < entries[right].Value
		}
		return entries[left].Value > entries[right].Value
	})

	if options.json {
		return writeJSON(command, map[string]any{"metric": metric.Name, "fact": metric.Fact, "boons": boons, "entries": entries, "failed_heroes": failed})
	}
	lines := make([]string, 0, len(entries)+1)
	for index, entry := range entries {
		lines = append(lines, fmt.Sprintf("%2d. %s — %s", index+1, entry.Hero, entry.DisplayValue))
	}
	if len(failed) > 0 {
		lines = append(lines, fmt.Sprintf("Skipped %d hero pages that could not be read.", len(failed)))
	}
	writeBox(command.OutOrStdout(), "Heroes · "+metric.Name+fmt.Sprintf(" · %d boons", boons), lines)
	writeAggregateProvenance(command.OutOrStdout(), records, "Ranked from canonical hero pages")
	return nil
}

func compareHeroes(ctx context.Context, command *cobra.Command, options *options, titles []string, boons int, weaponOnly bool) error {
	if boons < 0 {
		return fmt.Errorf("--boons cannot be negative")
	}
	client, err := clientFor(options)
	if err != nil {
		return err
	}
	records := make([]heroRecord, 0, len(titles))
	for _, title := range unique(titles) {
		page, cached, lookupError := client.Get(ctx, title, options.refresh)
		if page == nil {
			return lookupError
		}
		if lookupError != nil {
			fmt.Fprintln(command.ErrOrStderr(), "warning:", lookupError)
		}
		records = append(records, heroRecord{Page: page, Cached: cached})
	}
	metrics := comparisonMetrics(weaponOnly)
	if options.json {
		rows := make(map[string]map[string]string, len(metrics))
		for _, metric := range metrics {
			values := make(map[string]string, len(records))
			for _, record := range records {
				values[record.Page.Title] = heroMetricDisplay(record.Page, metric, boons)
			}
			rows[metric.Name] = values
		}
		sources := make([]map[string]any, 0, len(records))
		for _, record := range records {
			sources = append(sources, sourceFor(record.Page, record.Cached))
		}
		return writeJSON(command, map[string]any{"boons": boons, "weapon_only": weaponOnly, "heroes": heroTitles(records), "metrics": rows, "sources": sources})
	}
	lines := make([]string, 0, len(metrics))
	for _, metric := range metrics {
		values := make([]string, 0, len(records))
		for _, record := range records {
			values = append(values, record.Page.Title+": "+heroMetricDisplay(record.Page, metric, boons))
		}
		lines = append(lines, metric.Fact+" — "+strings.Join(values, " · "))
	}
	title := fmt.Sprintf("Hero comparison · %d boons", boons)
	if weaponOnly {
		title = fmt.Sprintf("Weapon comparison · %d boons", boons)
	}
	writeBox(command.OutOrStdout(), title, lines)
	writeAggregateProvenance(command.OutOrStdout(), records, "Compared canonical hero pages")
	return nil
}

func findHeroes(ctx context.Context, command *cobra.Command, options *options, tag string) error {
	records, failed, err := loadHeroRecords(ctx, options)
	if err != nil {
		return err
	}
	matched := make([]heroRecord, 0)
	for _, record := range records {
		for _, candidate := range record.Page.Tags {
			if normalizeName(candidate) == normalizeName(tag) {
				matched = append(matched, record)
				break
			}
		}
	}
	if options.json {
		entries := make([]map[string]any, 0, len(matched))
		for _, record := range matched {
			entries = append(entries, map[string]any{"hero": record.Page.Title, "tags": record.Page.Tags, "source": sourceFor(record.Page, record.Cached)})
		}
		return writeJSON(command, map[string]any{"tag": tag, "entries": entries, "failed_heroes": failed})
	}
	lines := make([]string, 0, len(matched)+1)
	for _, record := range matched {
		lines = append(lines, "• "+record.Page.Title+" — "+strings.Join(record.Page.Tags, " · "))
	}
	if len(lines) == 0 {
		lines = append(lines, "No hero has the "+tag+" tag.")
	}
	if len(failed) > 0 {
		lines = append(lines, fmt.Sprintf("Skipped %d hero pages that could not be read.", len(failed)))
	}
	writeBox(command.OutOrStdout(), "Heroes tagged "+tag, lines)
	writeAggregateProvenance(command.OutOrStdout(), matched, "Matched from canonical hero pages")
	return nil
}

func loadHeroRecords(ctx context.Context, options *options) ([]heroRecord, []string, error) {
	client, err := clientFor(options)
	if err != nil {
		return nil, nil, err
	}
	catalog, _, lookupError := client.Get(ctx, "Heroes", options.refresh)
	if catalog == nil {
		return nil, nil, lookupError
	}
	if len(catalog.Catalog) == 0 {
		return nil, nil, fmt.Errorf("the wiki did not provide a hero catalog")
	}
	records := make([]heroRecord, 0, len(catalog.Catalog))
	failed := make([]string, 0)
	for _, title := range catalog.Catalog {
		page, cached, _ := client.Get(ctx, title, options.refresh)
		if page == nil {
			failed = append(failed, title)
			continue
		}
		records = append(records, heroRecord{Page: page, Cached: cached})
	}
	return records, failed, nil
}

func factValue(page *wiki.Page, label string) (string, bool) {
	for _, fact := range page.Facts {
		if normalizeFactLabel(fact.Label) == normalizeFactLabel(label) {
			return strings.TrimSpace(fact.Value), true
		}
	}
	return "", false
}

func normalizeFactLabel(value string) string {
	return normalizeName(strings.TrimSpace(strings.TrimSuffix(value, ":")))
}

func statAtBoons(raw string, boons int) (float64, bool) {
	match := statValuePattern.FindStringSubmatch(raw)
	if match == nil {
		return 0, false
	}
	base, err := strconv.ParseFloat(strings.ReplaceAll(match[1], ",", ""), 64)
	if err != nil {
		return 0, false
	}
	if match[2] == "" {
		return base, true
	}
	perBoon, err := strconv.ParseFloat(strings.ReplaceAll(match[2], ",", ""), 64)
	if err != nil {
		return 0, false
	}
	return base + perBoon*float64(boons), true
}

func displayStatAtBoons(raw string, boons int) string {
	indices := statValuePattern.FindStringSubmatchIndex(raw)
	if indices == nil {
		return raw
	}
	value, found := statAtBoons(raw, boons)
	if !found {
		return raw
	}
	suffix := strings.TrimSpace(raw[indices[1]:])
	formatted := strconv.FormatFloat(math.Round(value*1000)/1000, 'f', -1, 64)
	if suffix == "" {
		return formatted
	}
	if strings.HasPrefix(suffix, "x") {
		return formatted + " " + suffix
	}
	return formatted + suffix
}

func weaponFactsAtBoons(page *wiki.Page, boons int) map[string]string {
	result := make(map[string]string)
	for _, metric := range heroMetrics {
		if !metric.WeaponStat {
			continue
		}
		if value, found := factValue(page, metric.Fact); found {
			result[metric.Name] = displayStatAtBoons(value, boons)
		}
	}
	return result
}

func weaponFactLines(page *wiki.Page, boons int) []string {
	lines := make([]string, 0)
	for _, metric := range heroMetrics {
		if !metric.WeaponStat {
			continue
		}
		if value, found := factValue(page, metric.Fact); found {
			lines = append(lines, "• "+metric.Fact+": "+displayStatAtBoons(value, boons))
		}
	}
	return lines
}

func heroMetricDisplay(page *wiki.Page, metric heroMetric, boons int) string {
	value, found := factValue(page, metric.Fact)
	if !found {
		return "—"
	}
	return displayStatAtBoons(value, boons)
}

func comparisonMetrics(weaponOnly bool) []heroMetric {
	metrics := make([]heroMetric, 0, len(heroMetrics))
	for _, metric := range heroMetrics {
		if weaponOnly == metric.WeaponStat {
			metrics = append(metrics, metric)
		}
	}
	return metrics
}

func heroTitles(records []heroRecord) []string {
	titles := make([]string, 0, len(records))
	for _, record := range records {
		titles = append(titles, record.Page.Title)
	}
	return titles
}

func writeAggregateProvenance(output io.Writer, records []heroRecord, description string) {
	if len(records) == 0 {
		return
	}
	cached := 0
	for _, record := range records {
		if record.Cached {
			cached++
		}
	}
	writeBox(output, "Provenance", []string{
		description + fmt.Sprintf(" · %d pages", len(records)),
		fmt.Sprintf("Cached: %d · Fresh: %d", cached, len(records)-cached),
		"Use --json for per-hero canonical URLs, revisions, and fetch times.",
	})
}
