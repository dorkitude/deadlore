package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"
)

const boxWidth = 78

var colorEnabled bool

const (
	ansiReset  = "\033[0m"
	ansiDim    = "\033[2m"
	ansiBold   = "\033[1m"
	ansiCyan   = "\033[96m"
	ansiGreen  = "\033[92m"
	ansiYellow = "\033[93m"
)

type statTile struct {
	Label string
	Value string
}

func configureColor(disabled bool) {
	colorEnabled = !disabled && terminalSupportsColor()
}

func terminalSupportsColor() bool {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return false
	}
	info, err := os.Stdout.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func writeBox(output io.Writer, title string, lines []string) {
	title = truncate(strings.TrimSpace(title), boxWidth-5)
	dashes := boxWidth - utf8.RuneCountInString(title) - 5
	fmt.Fprintf(output, "%s %s %s%s\n", paint("╭─", ansiCyan), paint(title, ansiBold+ansiCyan), paint(strings.Repeat("─", dashes), ansiDim+ansiCyan), paint("╮", ansiCyan))

	for _, line := range lines {
		for _, wrapped := range wrap(line, boxWidth-4) {
			fmt.Fprintf(output, "%s %-*s %s\n", paint("│", ansiCyan), boxWidth-4, wrapped, paint("│", ansiCyan))
		}
	}

	fmt.Fprintf(output, "%s%s%s\n", paint("╰", ansiCyan), paint(strings.Repeat("─", boxWidth-2), ansiDim+ansiCyan), paint("╯", ansiCyan))
}

func writeStatTiles(output io.Writer, title string, tiles []statTile) {
	if len(tiles) == 0 {
		return
	}
	title = truncate(strings.TrimSpace(title), boxWidth-5)
	dashes := boxWidth - utf8.RuneCountInString(title) - 5
	fmt.Fprintf(output, "%s %s %s%s\n", paint("╭─", ansiCyan), paint(title, ansiBold+ansiCyan), paint(strings.Repeat("─", dashes), ansiDim+ansiCyan), paint("╮", ansiCyan))

	const tileWidth = 36
	for index := 0; index < len(tiles); index += 2 {
		left := tiles[index]
		right := statTile{}
		if index+1 < len(tiles) {
			right = tiles[index+1]
		}
		writeTileRow(output, left, right, tileWidth)
	}

	fmt.Fprintf(output, "%s%s%s\n", paint("╰", ansiCyan), paint(strings.Repeat("─", boxWidth-2), ansiDim+ansiCyan), paint("╯", ansiCyan))
}

func writeTileRow(output io.Writer, left, right statTile, width int) {
	leftTile := tileLines(left, width)
	rightTile := tileLines(right, width)
	for index := range leftTile {
		if right.Label == "" {
			fmt.Fprintf(output, "%s %s%-*s %s\n", paint("│", ansiCyan), leftTile[index], boxWidth-4-width, "", paint("│", ansiCyan))
			continue
		}
		fmt.Fprintf(output, "%s %s  %s %s\n", paint("│", ansiCyan), leftTile[index], rightTile[index], paint("│", ansiCyan))
	}
}

func tileLines(tile statTile, width int) []string {
	if tile.Label == "" {
		return []string{strings.Repeat(" ", width), strings.Repeat(" ", width), strings.Repeat(" ", width)}
	}
	label := truncate(strings.ToUpper(tile.Label), width-5)
	top := paint("╭─", ansiGreen) + " " + paint(label, ansiBold+ansiGreen) + paint(" "+strings.Repeat("─", width-utf8.RuneCountInString(label)-5)+"╮", ansiGreen)
	middle := paint("│", ansiGreen) + center(truncate(tile.Value, width-4), width-2, ansiBold+ansiYellow) + paint("│", ansiGreen)
	bottom := paint("╰"+strings.Repeat("─", width-2)+"╯", ansiGreen)
	return []string{top, middle, bottom}
}

func center(value string, width int, style string) string {
	padding := width - utf8.RuneCountInString(value)
	left := padding / 2
	right := padding - left
	return strings.Repeat(" ", left) + paint(value, style) + strings.Repeat(" ", right)
}

func paint(value, style string) string {
	if !colorEnabled {
		return value
	}
	return style + value + ansiReset
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
