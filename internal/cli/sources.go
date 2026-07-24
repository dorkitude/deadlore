package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/dorkitude/deadlore/internal/wiki"
	"github.com/spf13/cobra"
)

func writeDeadlockIOPage(command *cobra.Command, page *wiki.Page, cached bool) {
	output := command.OutOrStdout()
	lines := make([]string, 0, len(page.Facts)+2)
	if page.Summary != "" {
		lines = append(lines, page.Summary)
	}
	if len(page.Tags) > 0 {
		lines = append(lines, "Tags: "+strings.Join(page.Tags, " · "))
	}
	if len(lines) > 0 {
		writeBox(output, page.Title+" · Deadlock.io", lines)
	}
	if len(page.Facts) > 0 {
		fmt.Fprintln(output)
		facts := make([]string, 0, len(page.Facts))
		for _, fact := range page.Facts {
			facts = append(facts, "• "+fact.Label+": "+fact.Value)
		}
		writeBox(output, "Deadlock.io stats", facts)
	}
	if len(page.Abilities) > 0 {
		fmt.Fprintln(output)
		abilities := make([]string, 0, len(page.Abilities))
		for _, ability := range page.Abilities {
			abilities = append(abilities, "• "+ability.Name)
		}
		writeBox(output, "Deadlock.io abilities", abilities)
	}
	for _, section := range page.Sections {
		if len(section.Text) == 0 {
			continue
		}
		fmt.Fprintln(output)
		writeBox(output, "Deadlock.io · "+section.Title, section.Text)
	}
	fmt.Fprintln(output)
	writeSource(command, page, cached)
}

func writeSourceComparison(output io.Writer, wikiPage, ioPage *wiki.Page) {
	if wikiPage == nil || ioPage == nil {
		writeSourceAvailability(output, wikiPage != nil, ioPage != nil, nil)
		return
	}
	wikiFacts := factsByLabel(wikiPage)
	ioFacts := factsByLabel(ioPage)
	shared, matching, different := 0, 0, make([]string, 0)
	for label, wikiValue := range wikiFacts {
		ioValue, found := ioFacts[label]
		if !found {
			continue
		}
		shared++
		if equivalentFactValues(wikiValue, ioValue) {
			matching++
			continue
		}
		different = append(different, label+": Wiki "+wikiValue+" · Deadlock.io "+ioValue)
	}
	lines := []string{fmt.Sprintf("Both sources returned %s.", wikiPage.Title)}
	if shared == 0 {
		lines = append(lines, "No directly comparable structured stats were exposed by both sources.")
	} else {
		lines = append(lines, fmt.Sprintf("Shared stats: %d · Matching: %d · Different: %d", shared, matching, len(different)))
	}
	lines = append(lines, different...)
	writeBox(output, "Source comparison", lines)
}

func equivalentFactValues(left, right string) bool {
	left = strings.ReplaceAll(normalizeName(left), ",", "")
	right = strings.ReplaceAll(normalizeName(right), ",", "")
	return left == right
}

func writeSourceAvailability(output io.Writer, wikiAvailable, ioAvailable bool, ioErr error) {
	lines := []string{fmt.Sprintf("Deadlock Wiki: %s", availability(wikiAvailable)), fmt.Sprintf("Deadlock.io: %s", availability(ioAvailable))}
	if ioErr != nil && !ioAvailable {
		lines = append(lines, "Deadlock.io did not have a matching page.")
	}
	writeBox(output, "Source comparison", lines)
}

func availability(available bool) string {
	if available {
		return "available"
	}
	return "not available"
}

func factsByLabel(page *wiki.Page) map[string]string {
	result := make(map[string]string, len(page.Facts))
	for _, fact := range page.Facts {
		result[normalizeFactLabel(fact.Label)] = fact.Value
	}
	return result
}

func writeCatalogComparison(output io.Writer, label string, wikiEntries, ioEntries []string) {
	wikiSet := make(map[string]struct{}, len(wikiEntries))
	for _, entry := range wikiEntries {
		wikiSet[normalizeName(entry)] = struct{}{}
	}
	shared := 0
	for _, entry := range ioEntries {
		if _, found := wikiSet[normalizeName(entry)]; found {
			shared++
		}
	}
	writeBox(output, "Source comparison", []string{
		fmt.Sprintf("%s — Wiki: %d · Deadlock.io: %d · Shared names: %d", label, len(wikiEntries), len(ioEntries), shared),
	})
}

func writeAbilityComparison(output io.Writer, wikiAbility wiki.Ability, wikiFound bool, ioAbility wiki.Ability, ioFound bool) {
	if !wikiFound || !ioFound {
		writeSourceAvailability(output, wikiFound, ioFound, nil)
		return
	}
	lines := []string{"Both sources returned " + wikiAbility.Name + "."}
	if normalizeName(wikiAbility.Description) == normalizeName(ioAbility.Description) {
		lines = append(lines, "Descriptions match.")
	} else {
		lines = append(lines, "Descriptions differ or one source omits detail.")
	}
	writeBox(output, "Source comparison", lines)
}

func writeRankComparison(output io.Writer, metric heroMetric, wikiEntries, ioEntries []heroRankEntry) {
	if len(ioEntries) == 0 {
		writeSourceAvailability(output, len(wikiEntries) > 0, false, nil)
		return
	}
	lines := []string{fmt.Sprintf("%s — Wiki entries: %d · Deadlock.io entries: %d", metric.Fact, len(wikiEntries), len(ioEntries))}
	if len(wikiEntries) > 0 && len(ioEntries) > 0 {
		lines = append(lines, "Leaders: Wiki "+wikiEntries[0].Hero+" ("+wikiEntries[0].DisplayValue+") · Deadlock.io "+ioEntries[0].Hero+" ("+ioEntries[0].DisplayValue+")")
	}
	writeBox(output, "Source comparison", lines)
}

func writeComparisonSourceSummary(output io.Writer, wikiRecords, ioRecords []heroRecord) {
	writeBox(output, "Source comparison", []string{fmt.Sprintf("Compared heroes — Wiki: %d · Deadlock.io: %d", len(wikiRecords), len(ioRecords))})
}

func writeTagSourceComparison(output io.Writer, tag string, wikiRecords, ioRecords []heroRecord) {
	writeBox(output, "Source comparison", []string{fmt.Sprintf("Heroes tagged %s — Wiki: %d · Deadlock.io: %d", tag, len(wikiRecords), len(ioRecords))})
}
