package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/dorkitude/deadlore/internal/deadlockio"
	"github.com/dorkitude/deadlore/internal/wiki"
	"github.com/spf13/cobra"
)

type options struct {
	json          bool
	noColor       bool
	refresh       bool
	cacheDir      string
	wikiURL       string
	deadlockIOURL string
}

func Execute() {
	if err := newRootCommand().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func newRootCommand() *cobra.Command {
	options := &options{}
	root := &cobra.Command{
		Use:           "deadlore [title]",
		Short:         "A source-aware Deadlock Wiki CLI",
		Long:          "Look up one canonical Deadlock Wiki article at a time and keep a small local cache with revision metadata.",
		Example:       "  deadlore Haze\n  deadlore item \"Heroic Aura\"\n  deadlore source Infuser\n  deadlore cache status",
		Args:          cobra.ArbitraryArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRun: func(_ *cobra.Command, _ []string) {
			configureColor(options.noColor || options.json)
		},
		RunE: func(command *cobra.Command, args []string) error {
			if len(args) == 0 {
				return command.Help()
			}
			return lookup(command.Context(), command, options, strings.Join(args, " "))
		},
	}

	root.PersistentFlags().BoolVar(&options.json, "json", false, "print structured JSON")
	root.PersistentFlags().BoolVar(&options.noColor, "no-color", false, "disable ANSI color output")
	root.PersistentFlags().BoolVar(&options.refresh, "refresh", false, "bypass the local cache")
	root.PersistentFlags().StringVar(&options.cacheDir, "cache-dir", "", "cache directory (default: system user cache)")
	root.PersistentFlags().StringVar(&options.wikiURL, "wiki-url", wiki.DefaultBaseURL, "Deadlock Wiki base URL")
	root.PersistentFlags().StringVar(&options.deadlockIOURL, "deadlockio-url", deadlockio.DefaultBaseURL, "Deadlock.io API base URL")

	root.AddCommand(newHeroCommand(options))
	root.AddCommand(newLookupCommand("item", "Look up an item", options))
	root.AddCommand(newLookupCommand("mechanic", "Look up a mechanic", options))
	root.AddCommand(newAbilityCommand(options))
	root.AddCommand(newSourceCommand(options))
	root.AddCommand(newCacheCommand(options))
	root.AddCommand(newTimersCommand(options))
	root.AddCommand(newCheatCommand(options))
	return root
}

func newAbilityCommand(options *options) *cobra.Command {
	return &cobra.Command{
		Use:           "ability <name>",
		Short:         "Look up one ability",
		Args:          cobra.MinimumNArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(command *cobra.Command, args []string) error {
			if len(args) == 1 && strings.EqualFold(args[0], "list") {
				return listAbilities(command.Context(), command, options)
			}
			return lookupAbility(command.Context(), command, options, strings.Join(args, " "))
		},
	}
}

func newLookupCommand(name, short string, options *options) *cobra.Command {
	return &cobra.Command{
		Use:           name + " <title>",
		Short:         short,
		Args:          cobra.MinimumNArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(command *cobra.Command, args []string) error {
			if len(args) == 1 && strings.EqualFold(args[0], "list") {
				return listEntities(command.Context(), command, options, name)
			}
			return lookup(command.Context(), command, options, strings.Join(args, " "))
		},
	}
}

func newSourceCommand(options *options) *cobra.Command {
	return &cobra.Command{
		Use:           "source <title>",
		Short:         "Show provenance for a wiki article",
		Args:          cobra.MinimumNArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(command *cobra.Command, args []string) error {
			client, err := clientFor(options)
			if err != nil {
				return err
			}
			page, cached, err := client.Get(command.Context(), strings.Join(args, " "), options.refresh)
			ioClient, clientErr := deadlockIOClientFor(options)
			if clientErr != nil {
				return clientErr
			}
			ioPage, ioCached, ioErr := ioClient.Get(command.Context(), strings.Join(args, " "), "", options.refresh)
			if page == nil && ioPage == nil {
				return err
			}
			if options.json {
				result := map[string]any{"source": sourceFor(page, cached), "deadlock_io": sourceFor(ioPage, ioCached)}
				if writeErr := writeJSON(command, result); writeErr != nil {
					return writeErr
				}
			} else {
				if page != nil {
					writeSource(command, page, cached)
				}
				if ioPage != nil {
					writeSource(command, ioPage, ioCached)
				}
				writeSourceAvailability(command.OutOrStdout(), page != nil, ioPage != nil, ioErr)
			}
			if err != nil && page != nil {
				warning(command, err)
			}
			if ioErr != nil && ioPage != nil {
				warning(command, ioErr)
			}
			return nil
		},
	}
}

func newCacheCommand(options *options) *cobra.Command {
	cache := &cobra.Command{Use: "cache", Short: "Inspect or clear the local wiki cache"}
	statusCommand := &cobra.Command{
		Use:   "status",
		Short: "Show local cache status",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, args []string) error {
			client, err := clientFor(options)
			if err != nil {
				return err
			}
			count, newest, err := client.Cache().Status()
			if err != nil {
				return err
			}
			ioClient, err := deadlockIOClientFor(options)
			if err != nil {
				return err
			}
			ioCount, ioNewest, err := ioClient.CacheStatus()
			if err != nil {
				return err
			}
			if options.json {
				return writeJSON(command, map[string]any{"entries": count, "newest": newest, "deadlock_io": map[string]any{"entries": ioCount, "newest": ioNewest}})
			}
			fmt.Fprintf(command.OutOrStdout(), "Deadlock Wiki entries: %d\n", count)
			if !newest.IsZero() {
				fmt.Fprintf(command.OutOrStdout(), "Deadlock Wiki newest:  %s\n", newest.Format(time.RFC3339))
			}
			fmt.Fprintf(command.OutOrStdout(), "Deadlock.io entries: %d\n", ioCount)
			if !ioNewest.IsZero() {
				fmt.Fprintf(command.OutOrStdout(), "Deadlock.io newest:  %s\n", ioNewest.Format(time.RFC3339))
			}
			return nil
		},
	}
	clearCommand := &cobra.Command{
		Use:           "clear [title]",
		Short:         "Clear one cached title, or all cached entries with --all",
		Args:          cobra.MaximumNArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(command *cobra.Command, args []string) error {
			all, err := command.Flags().GetBool("all")
			if err != nil {
				return err
			}
			if !all && len(args) == 0 {
				return errors.New("provide a title or use --all")
			}
			client, err := clientFor(options)
			if err != nil {
				return err
			}
			title := ""
			if len(args) == 1 {
				title = args[0]
			}
			if err := client.Cache().Clear(title); err != nil {
				return err
			}
			ioClient, err := deadlockIOClientFor(options)
			if err != nil {
				return err
			}
			if err := ioClient.ClearCache(title); err != nil {
				return err
			}
			if all {
				fmt.Fprintln(command.OutOrStdout(), "Cleared all cached pages.")
			} else {
				fmt.Fprintf(command.OutOrStdout(), "Cleared %q from the cache.\n", title)
			}
			return nil
		},
	}
	clearCommand.Flags().Bool("all", false, "clear all cached pages")
	cache.AddCommand(statusCommand)
	cache.AddCommand(clearCommand)
	return cache
}

func lookup(ctx context.Context, command *cobra.Command, options *options, title string) error {
	client, err := clientFor(options)
	if err != nil {
		return err
	}
	page, cached, lookupError := client.Get(ctx, title, options.refresh)
	ioClient, err := deadlockIOClientFor(options)
	if err != nil {
		return err
	}
	ioPage, ioCached, ioError := ioClient.Get(ctx, title, deadlockIOKind(command.Name()), options.refresh)
	if page == nil && ioPage == nil {
		return lookupError
	}
	if options.json {
		if err := writeJSON(command, map[string]any{"page": page, "cached": cached, "deadlock_io": map[string]any{"page": ioPage, "cached": ioCached}}); err != nil {
			return err
		}
	} else {
		if page != nil {
			writePage(command, page, cached)
		}
		if ioPage != nil {
			fmt.Fprintln(command.OutOrStdout())
			writeDeadlockIOPage(command, ioPage, ioCached)
		}
		writeSourceComparison(command.OutOrStdout(), page, ioPage)
	}
	if lookupError != nil && page != nil {
		warning(command, lookupError)
	}
	if ioError != nil && ioPage != nil {
		warning(command, ioError)
	}
	return nil
}

func lookupAbility(ctx context.Context, command *cobra.Command, options *options, name string) error {
	client, err := clientFor(options)
	if err != nil {
		return err
	}
	page, cached, lookupError := client.Get(ctx, name, options.refresh)
	ioClient, err := deadlockIOClientFor(options)
	if err != nil {
		return err
	}
	ioPage, ioAbility, ioFound, ioCached, ioError := ioClient.GetAbility(ctx, name, options.refresh)
	if page == nil && !ioFound {
		return lookupError
	}
	ability, found := wiki.Ability{}, false
	if page != nil {
		ability, found = findAbility(page.Abilities, name)
	}
	if options.json {
		result := map[string]any{"page": page, "cached": cached, "deadlock_io": map[string]any{"ability": ioAbility, "page": ioPage, "cached": ioCached, "found": ioFound}}
		if found {
			result["ability"] = ability
		} else if page != nil {
			result["notice"] = fmt.Sprintf("%q is not a hero ability; showing the resolved %s page instead.", name, page.Title)
		}
		if err := writeJSON(command, result); err != nil {
			return err
		}
	} else {
		if page != nil && found {
			if tiles := abilityStatTiles(ability); len(tiles) > 0 {
				writeStatTiles(command.OutOrStdout(), ability.Name+" · Key stats", tiles)
				fmt.Fprintln(command.OutOrStdout())
			}
			writeBox(command.OutOrStdout(), ability.Name+" · Ability", abilityLines(ability, false))
			fmt.Fprintln(command.OutOrStdout())
			writeSource(command, page, cached)
		} else if page != nil {
			fmt.Fprintf(command.OutOrStdout(), "Note: %q is not a hero ability; showing the resolved %s page instead.\n\n", name, page.Title)
			writePage(command, page, cached)
		}
		if ioFound {
			fmt.Fprintln(command.OutOrStdout())
			writeBox(command.OutOrStdout(), ioAbility.Name+" · Deadlock.io ability", abilityLines(ioAbility, false))
			fmt.Fprintln(command.OutOrStdout())
			writeSource(command, ioPage, ioCached)
		}
		writeAbilityComparison(command.OutOrStdout(), ability, found, ioAbility, ioFound)
	}
	if lookupError != nil && page != nil {
		warning(command, lookupError)
	}
	if ioError != nil && ioFound {
		warning(command, ioError)
	}
	return nil
}

func listEntities(ctx context.Context, command *cobra.Command, options *options, kind string) error {
	title := map[string]string{"hero": "Heroes", "item": "Category:Items"}[kind]
	client, err := clientFor(options)
	if err != nil {
		return err
	}
	page, cached, lookupError := client.Get(ctx, title, options.refresh)
	ioClient, err := deadlockIOClientFor(options)
	if err != nil {
		return err
	}
	ioCatalog, ioCached, ioError := ioClient.Catalog(ctx, kind, options.refresh)
	if page == nil && ioCatalog == nil {
		return lookupError
	}
	if page != nil && len(page.Catalog) == 0 {
		return fmt.Errorf("the wiki did not provide a %s catalog", kind)
	}
	ioEntries := deadlockIOCatalogEntries(ioCatalog)
	if options.json {
		if err := writeJSON(command, map[string]any{"type": kind, "entries": catalogEntries(page), "page": page, "cached": cached, "deadlock_io": map[string]any{"entries": ioEntries, "page": ioCatalog, "cached": ioCached}}); err != nil {
			return err
		}
	} else {
		displayName := map[string]string{"hero": "Heroes", "item": "Items"}[kind]
		if page != nil {
			writeList(command.OutOrStdout(), displayName, page.Catalog)
		}
		fmt.Fprintln(command.OutOrStdout())
		if page != nil {
			writeSource(command, page, cached)
		}
		writeCatalogComparison(command.OutOrStdout(), displayName, catalogEntries(page), ioEntries)
		if ioCatalog != nil {
			writeSource(command, ioCatalog, ioCached)
		}
	}
	if lookupError != nil && page != nil {
		warning(command, lookupError)
	}
	if ioError != nil && ioCatalog != nil {
		warning(command, ioError)
	}
	return nil
}

func catalogEntries(page *wiki.Page) []string {
	if page == nil {
		return nil
	}
	return page.Catalog
}

func deadlockIOCatalogEntries(page *wiki.Page) []string {
	if page == nil {
		return nil
	}
	entries := make([]string, 0, len(page.Catalog))
	for _, entry := range page.Catalog {
		name, _, found := strings.Cut(entry, "\t")
		if found {
			entries = append(entries, name)
		}
	}
	return entries
}

func listAbilities(ctx context.Context, command *cobra.Command, options *options) error {
	client, err := clientFor(options)
	if err != nil {
		return err
	}
	heroes, _, err := client.Get(ctx, "Heroes", options.refresh)
	if heroes == nil {
		return err
	}
	if len(heroes.Catalog) == 0 {
		return errors.New("the wiki did not provide a hero catalog")
	}

	var entries []string
	var failed []string
	for _, hero := range heroes.Catalog {
		page, _, fetchError := client.Get(ctx, hero, options.refresh)
		if page == nil || fetchError != nil {
			failed = append(failed, hero)
			continue
		}
		for _, ability := range page.Abilities {
			entries = append(entries, page.Title+" · "+ability.Name)
		}
	}
	entries = unique(entries)
	if len(entries) == 0 {
		return errors.New("no abilities could be read from the hero pages")
	}
	ioRecords, ioFailed, ioError := loadDeadlockIOHeroRecords(ctx, options)
	ioEntries := make([]string, 0)
	for _, record := range ioRecords {
		for _, ability := range record.Page.Abilities {
			ioEntries = append(ioEntries, record.Page.Title+" · "+ability.Name)
		}
	}
	ioEntries = unique(ioEntries)
	if options.json {
		if err := writeJSON(command, map[string]any{"type": "abilities", "entries": entries, "failed_heroes": failed, "deadlock_io": map[string]any{"entries": ioEntries, "failed_heroes": ioFailed}}); err != nil {
			return err
		}
	} else {
		writeAbilityList(command.OutOrStdout(), entries)
		writeCatalogComparison(command.OutOrStdout(), "Abilities", entries, ioEntries)
		if len(failed) > 0 {
			fmt.Fprintf(command.ErrOrStderr(), "warning: skipped %d hero pages while building the ability list\n", len(failed))
		}
		if ioError != nil {
			fmt.Fprintln(command.ErrOrStderr(), "warning: Deadlock.io ability list unavailable:", ioError)
		}
	}
	return nil
}

func writeAbilityList(output io.Writer, entries []string) {
	type heroAbilities struct {
		hero      string
		abilities []string
	}

	groups := make([]heroAbilities, 0)
	for _, entry := range entries {
		hero, ability, found := strings.Cut(entry, " · ")
		if !found {
			groups = append(groups, heroAbilities{hero: entry})
			continue
		}
		if len(groups) > 0 && groups[len(groups)-1].hero == hero {
			groups[len(groups)-1].abilities = append(groups[len(groups)-1].abilities, ability)
			continue
		}
		groups = append(groups, heroAbilities{hero: hero, abilities: []string{ability}})
	}

	lines := make([]string, 0, len(groups))
	for _, group := range groups {
		if len(group.abilities) == 0 {
			lines = append(lines, "• "+group.hero)
			continue
		}
		lines = append(lines, "• "+group.hero+" — "+strings.Join(group.abilities, " · "))
	}
	writeBox(output, fmt.Sprintf("Abilities · %d", len(entries)), lines)
}

func unique(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func findAbility(abilities []wiki.Ability, name string) (wiki.Ability, bool) {
	target := normalizeName(name)
	for _, ability := range abilities {
		if normalizeName(ability.Name) == target {
			return ability, true
		}
	}
	return wiki.Ability{}, false
}

func normalizeName(value string) string {
	value = strings.ReplaceAll(value, "_", " ")
	return strings.ToLower(strings.Join(strings.Fields(value), " "))
}

func clientFor(options *options) (*wiki.Client, error) {
	return wiki.NewClient(options.wikiURL, options.cacheDir)
}

func deadlockIOClientFor(options *options) (*deadlockio.Client, error) {
	return deadlockio.NewClient(options.deadlockIOURL, options.cacheDir)
}

func deadlockIOKind(command string) string {
	return map[string]string{"hero": "hero", "item": "item", "mechanic": "mechanic"}[command]
}

func writeJSON(command *cobra.Command, value any) error {
	encoder := json.NewEncoder(command.OutOrStdout())
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func writePage(command *cobra.Command, page *wiki.Page, cached bool) {
	output := command.OutOrStdout()
	var overview []string
	if page.Summary != "" {
		overview = append(overview, page.Summary)
	}
	if len(page.Tags) > 0 {
		overview = append(overview, "Tags: "+strings.Join(page.Tags, " · "))
	}
	if len(overview) > 0 {
		writeBox(output, page.Title+" · Deadlock Wiki", overview)
	}
	if tiles := pageStatTiles(page); len(tiles) > 0 {
		fmt.Fprintln(output)
		writeStatTiles(output, "Key stats", tiles)
	}
	if len(page.Facts) > 0 {
		fmt.Fprintln(output)
		facts := make([]string, 0, len(page.Facts))
		for _, fact := range page.Facts {
			if strings.TrimSpace(fact.Label) == "" {
				facts = append(facts, "• "+fact.Value)
				continue
			}
			label := strings.TrimSpace(fact.Label)
			label = strings.TrimSpace(strings.TrimSuffix(label, ":"))
			facts = append(facts, "• "+label+": "+fact.Value)
		}
		writeBox(output, "Stats", facts)
	}
	if len(page.Abilities) > 0 {
		fmt.Fprintln(output)
		abilities := make([]string, 0, len(page.Abilities)*5)
		for index, ability := range page.Abilities {
			abilities = append(abilities, abilityLines(ability, true)...)
			if index < len(page.Abilities)-1 {
				abilities = append(abilities, "")
			}
		}
		writeBox(output, "Abilities", abilities)
	}
	for _, effect := range page.Effects {
		fmt.Fprintln(output)
		lines := make([]string, 0, len(effect.Stats)+1)
		if effect.Description != "" {
			lines = append(lines, effect.Description)
		}
		if len(effect.Stats) > 0 {
			lines = append(lines, "Effect stats: "+strings.Join(effect.Stats, " · "))
		}
		writeBox(output, effect.Kind, lines)
	}
	for _, section := range page.Sections {
		if strings.EqualFold(section.Title, "Update history") || (len(page.Abilities) > 0 && strings.EqualFold(section.Title, "Abilities")) {
			continue
		}
		if len(section.Text) == 0 {
			continue
		}
		fmt.Fprintln(output)
		writeBox(output, section.Title, section.Text)
	}
	fmt.Fprintln(output)
	writeSource(command, page, cached)
}

func pageStatTiles(page *wiki.Page) []statTile {
	if len(page.Facts) == 0 {
		return nil
	}

	preferred := []string{
		"Damage Per Second", "Bullet Damage", "Health", "Ammo", "Move Speed", "Reload Time",
	}
	byLabel := make(map[string]string, len(page.Facts))
	for _, fact := range page.Facts {
		label := strings.TrimSpace(strings.TrimSuffix(fact.Label, ":"))
		if label != "" {
			byLabel[strings.ToLower(label)] = strings.TrimSpace(fact.Value)
		}
	}

	tiles := make([]statTile, 0, 6)
	for _, label := range preferred {
		if value := byLabel[strings.ToLower(label)]; value != "" {
			tiles = append(tiles, statTile{Label: label, Value: value})
		}
	}
	if len(tiles) > 0 {
		return tiles
	}

	for _, fact := range page.Facts {
		if len(tiles) == 6 {
			break
		}
		if label := strings.TrimSpace(strings.TrimSuffix(fact.Label, ":")); label != "" {
			tiles = append(tiles, statTile{Label: label, Value: strings.TrimSpace(fact.Value)})
			continue
		}
		if tile, ok := tileFromStat(strings.TrimSpace(fact.Value)); ok {
			tiles = append(tiles, tile)
		}
	}
	return tiles
}

func abilityStatTiles(ability wiki.Ability) []statTile {
	tiles := make([]statTile, 0, 6)
	for _, stat := range ability.Stats {
		if len(tiles) == 6 {
			break
		}
		if tile, ok := tileFromStat(stat); ok {
			tiles = append(tiles, tile)
		}
	}
	return tiles
}

func tileFromStat(stat string) (statTile, bool) {
	parts := strings.Fields(stat)
	if len(parts) < 2 {
		return statTile{}, false
	}
	valueLength := 1
	if len(parts) > 2 && isStatUnit(parts[1]) {
		valueLength = 2
	}
	label := strings.Join(parts[valueLength:], " ")
	if label == "" {
		return statTile{}, false
	}
	return statTile{Label: label, Value: strings.Join(parts[:valueLength], " ")}, true
}

func isStatUnit(value string) bool {
	return value == "%" || value == "s" || value == "m" || value == "m/s"
}

func abilityLines(ability wiki.Ability, includeName bool) []string {
	lines := make([]string, 0, len(ability.Upgrades)+3)
	if includeName {
		lines = append(lines, ability.Name)
	}
	if ability.Description != "" {
		lines = append(lines, ability.Description)
	}
	if len(ability.Stats) > 0 {
		lines = append(lines, "Stats: "+strings.Join(ability.Stats, " · "))
	}
	if len(ability.Upgrades) > 0 {
		lines = append(lines, "Upgrades:")
		for _, upgrade := range ability.Upgrades {
			lines = append(lines, "• "+upgrade)
		}
	}
	return lines
}

func writeSource(command *cobra.Command, page *wiki.Page, cached bool) {
	output := command.OutOrStdout()
	lines := []string{"Source: " + page.URL}
	if page.RevisionID != "" {
		lines = append(lines, "Revision: "+page.RevisionID)
	}
	if page.LastModified != "" {
		lines = append(lines, page.LastModified)
	}
	fetched := "Fetched: " + page.FetchedAt.Format(time.RFC3339)
	if cached {
		fetched += " (cached)"
	}
	lines = append(lines, fetched)
	writeBox(output, "Provenance", lines)
}

func writeList(output io.Writer, title string, entries []string) {
	lines := make([]string, 0, (len(entries)+2)/3)
	line := ""
	for _, entry := range entries {
		candidate := "• " + entry
		if line != "" {
			candidate = line + "    " + candidate
		}
		if line != "" && len([]rune(candidate)) > boxWidth-4 {
			lines = append(lines, line)
			line = "• " + entry
		} else {
			line = candidate
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	writeBox(output, fmt.Sprintf("%s · %d", title, len(entries)), lines)
}

func warning(command *cobra.Command, err error) error {
	if err != nil {
		fmt.Fprintln(command.ErrOrStderr(), "warning:", err)
	}
	return nil
}

func sourceFor(page *wiki.Page, cached bool) map[string]any {
	if page == nil {
		return nil
	}
	return map[string]any{
		"url":           page.URL,
		"revision_id":   page.RevisionID,
		"last_modified": page.LastModified,
		"fetched_at":    page.FetchedAt,
		"cached":        cached,
	}
}
