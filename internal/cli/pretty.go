package cli

import (
	"fmt"
	"io"
	"strings"
	"unicode/utf8"
)

const boxWidth = 78

func writeBox(output io.Writer, title string, lines []string) {
	title = truncate(strings.TrimSpace(title), boxWidth-5)
	dashes := boxWidth - utf8.RuneCountInString(title) - 5
	fmt.Fprintf(output, "╭─ %s %s╮\n", title, strings.Repeat("─", dashes))

	for _, line := range lines {
		for _, wrapped := range wrap(line, boxWidth-4) {
			fmt.Fprintf(output, "│ %-*s │\n", boxWidth-4, wrapped)
		}
	}

	fmt.Fprintf(output, "╰%s╯\n", strings.Repeat("─", boxWidth-2))
}

func wrap(line string, width int) []string {
	if strings.TrimSpace(line) == "" {
		return []string{""}
	}

	words := strings.Fields(line)
	result := make([]string, 0, 1)
	current := ""
	for _, word := range words {
		if utf8.RuneCountInString(word) > width {
			if current != "" {
				result = append(result, current)
				current = ""
			}
			for utf8.RuneCountInString(word) > width {
				chunk, remaining := splitRunes(word, width)
				result = append(result, chunk)
				word = remaining
			}
		}

		if current == "" {
			current = word
			continue
		}
		if utf8.RuneCountInString(current)+1+utf8.RuneCountInString(word) <= width {
			current += " " + word
			continue
		}
		result = append(result, current)
		current = word
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

func splitRunes(value string, count int) (string, string) {
	runes := []rune(value)
	return string(runes[:count]), string(runes[count:])
}

func truncate(value string, width int) string {
	if utf8.RuneCountInString(value) <= width {
		return value
	}
	runes := []rune(value)
	return string(runes[:width-1]) + "…"
}
