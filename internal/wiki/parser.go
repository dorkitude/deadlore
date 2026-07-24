package wiki

import (
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"
)

var revisionPattern = regexp.MustCompile(`"wgRevisionId"\s*:\s*([0-9]+)`)

func ParsePage(source io.Reader, pageURL string, fetchedAt time.Time) (*Page, error) {
	document, err := html.Parse(source)
	if err != nil {
		return nil, err
	}

	titleNode := findByID(document, "firstHeading")
	contentNode := findByID(document, "mw-content-text")
	if titleNode == nil || contentNode == nil {
		return nil, fmt.Errorf("expected a MediaWiki article page")
	}

	page := &Page{
		Title:     strings.TrimSpace(strings.TrimSuffix(cleanText(text(titleNode)), "Give feedback")),
		URL:       pageURL,
		FetchedAt: fetchedAt,
	}
	page.Summary = firstParagraph(contentNode)
	page.Facts = append(infoboxFacts(contentNode), itemStats(contentNode)...)
	page.Abilities = abilities(contentNode)
	page.Catalog = catalogEntries(contentNode)
	page.Sections = sections(contentNode)
	page.LastModified = lastModified(document)
	page.RevisionID = revisionID(document)
	return page, nil
}

func catalogEntries(content *html.Node) []string {
	var entries []string
	for _, hero := range descendants(content, func(node *html.Node) bool {
		return node.Type == html.ElementNode && strings.Contains(attribute(node, "class"), "HeroName")
	}) {
		entries = append(entries, cleanText(text(hero)))
	}
	if len(entries) > 0 {
		return uniqueStrings(entries)
	}

	categoryPages := findByID(content, "mw-pages")
	if categoryPages == nil {
		return nil
	}
	for _, link := range descendants(categoryPages, func(node *html.Node) bool {
		return node.Type == html.ElementNode && node.Data == "a"
	}) {
		title := strings.TrimSpace(attribute(link, "title"))
		href := attribute(link, "href")
		if title == "" || !strings.HasPrefix(href, "/") || strings.Contains(title, ":") || strings.Contains(title, "/") {
			continue
		}
		entries = append(entries, title)
	}
	return uniqueStrings(entries)
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func itemStats(content *html.Node) []Fact {
	itemInfobox := findFirst(content, func(node *html.Node) bool {
		return node.Type == html.ElementNode && strings.Contains(attribute(node, "class"), "item-infobox")
	})
	if itemInfobox == nil {
		return nil
	}

	var facts []Fact
	for _, stat := range descendants(itemInfobox, func(node *html.Node) bool {
		return node.Type == html.ElementNode && strings.Contains(attribute(node, "class"), "infobox-stat")
	}) {
		value := cleanText(text(stat))
		if value == "" || containsFactValue(facts, value) {
			continue
		}
		facts = append(facts, Fact{Value: clamp(value, 240)})
	}
	return facts
}

func containsFactValue(facts []Fact, value string) bool {
	for _, fact := range facts {
		if fact.Value == value {
			return true
		}
	}
	return false
}

func abilities(content *html.Node) []Ability {
	var result []Ability
	for _, wrapper := range descendants(content, func(node *html.Node) bool {
		return node.Type == html.ElementNode && strings.Contains(attribute(node, "class"), "ability-section-wrapper")
	}) {
		heading := findFirst(wrapper, func(node *html.Node) bool {
			return node.Type == html.ElementNode && node.Data == "h3" && strings.Contains(attribute(node, "class"), "ability-name")
		})
		description := findFirst(wrapper, func(node *html.Node) bool {
			return node.Type == html.ElementNode && strings.Contains(attribute(node, "class"), "ac-info-desc")
		})
		if heading == nil || description == nil {
			continue
		}

		name := cleanText(text(heading))
		body := cleanText(text(description))
		if name == "" || body == "" {
			continue
		}
		result = append(result, Ability{Name: name, Description: clamp(body, 700)})
	}
	return result
}

func findByID(node *html.Node, id string) *html.Node {
	if node.Type == html.ElementNode && attribute(node, "id") == id {
		return node
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if found := findByID(child, id); found != nil {
			return found
		}
	}
	return nil
}

func findFirst(node *html.Node, predicate func(*html.Node) bool) *html.Node {
	if predicate(node) {
		return node
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if found := findFirst(child, predicate); found != nil {
			return found
		}
	}
	return nil
}

func descendants(node *html.Node, predicate func(*html.Node) bool) []*html.Node {
	var found []*html.Node
	var visit func(*html.Node)
	visit = func(current *html.Node) {
		if predicate(current) {
			found = append(found, current)
		}
		for child := current.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	visit(node)
	return found
}

func firstParagraph(content *html.Node) string {
	paragraph := findFirst(content, func(node *html.Node) bool {
		return node.Type == html.ElementNode && node.Data == "p" && len(cleanText(text(node))) > 40
	})
	if paragraph == nil {
		return ""
	}
	return clamp(cleanText(text(paragraph)), 900)
}

func infoboxFacts(content *html.Node) []Fact {
	infobox := findFirst(content, func(node *html.Node) bool {
		if node.Type != html.ElementNode {
			return false
		}
		class := attribute(node, "class")
		return (node.Data == "table" && strings.Contains(class, "infobox")) || strings.Contains(class, "infobox-h")
	})
	if infobox == nil {
		return nil
	}

	var facts []Fact
	for _, row := range descendants(infobox, func(node *html.Node) bool {
		return node.Type == html.ElementNode && node.Data == "tr"
	}) {
		var headers, values []string
		for child := row.FirstChild; child != nil; child = child.NextSibling {
			if child.Type != html.ElementNode {
				continue
			}
			switch child.Data {
			case "th":
				if value := cleanText(text(child)); value != "" {
					headers = append(headers, value)
				}
			case "td":
				if value := cleanText(text(child)); value != "" {
					values = append(values, value)
				}
			}
		}
		if len(headers) == 0 && len(values) >= 2 {
			headers = append(headers, values[0])
			values = values[1:]
		}
		if len(headers) == 0 || len(values) == 0 {
			continue
		}
		facts = append(facts, Fact{Label: strings.Join(headers, " "), Value: clamp(strings.Join(values, " · "), 240)})
		if len(facts) == 12 {
			break
		}
	}
	return facts
}

func sections(content *html.Node) []Section {
	var result []Section
	var current *Section
	for _, node := range descendants(content, func(node *html.Node) bool {
		if node.Type != html.ElementNode {
			return false
		}
		if node.Data == "p" || node.Data == "h2" {
			return true
		}
		return node.Data == "h3" && !strings.Contains(attribute(node, "class"), "ability-name")
	}) {
		switch node.Data {
		case "h2", "h3":
			heading := cleanText(text(node))
			if heading == "" || strings.EqualFold(heading, "Contents") {
				continue
			}
			result = append(result, Section{Title: heading})
			current = &result[len(result)-1]
		case "p":
			if current == nil || len(current.Text) >= 2 {
				continue
			}
			paragraph := cleanText(text(node))
			if len(paragraph) >= 40 {
				current.Text = append(current.Text, clamp(paragraph, 600))
			}
		}
		if len(result) == 6 && current != nil && len(current.Text) >= 1 {
			break
		}
	}
	return result
}

func lastModified(document *html.Node) string {
	node := findByID(document, "footer-info-lastmod")
	if node == nil {
		return ""
	}
	return cleanText(text(node))
}

func revisionID(document *html.Node) string {
	for _, script := range descendants(document, func(node *html.Node) bool {
		return node.Type == html.ElementNode && node.Data == "script"
	}) {
		matches := revisionPattern.FindStringSubmatch(rawText(script))
		if len(matches) == 2 {
			return matches[1]
		}
	}
	return ""
}

func attribute(node *html.Node, name string) string {
	for _, attribute := range node.Attr {
		if attribute.Key == name {
			return attribute.Val
		}
	}
	return ""
}

func text(node *html.Node) string {
	var builder strings.Builder
	var walk func(*html.Node)
	walk = func(current *html.Node) {
		if current.Type == html.ElementNode && (current.Data == "script" || current.Data == "style" || current.Data == "noscript") {
			return
		}
		if current.Type == html.TextNode {
			builder.WriteString(current.Data)
			builder.WriteByte(' ')
		}
		for child := current.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(node)
	return builder.String()
}

func rawText(node *html.Node) string {
	var builder strings.Builder
	var walk func(*html.Node)
	walk = func(current *html.Node) {
		if current.Type == html.TextNode {
			builder.WriteString(current.Data)
		}
		for child := current.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(node)
	return builder.String()
}

func cleanText(value string) string {
	return strings.TrimSpace(strings.Join(strings.Fields(value), " "))
}

func clamp(value string, maximum int) string {
	if len(value) <= maximum {
		return value
	}
	return strings.TrimSpace(value[:maximum-1]) + "…"
}
