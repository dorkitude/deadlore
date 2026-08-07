package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/dorkitude/deadlore/internal/deadlockapi"
	"github.com/spf13/cobra"
)

func newBuildCommand(options *options) *cobra.Command {
	build := &cobra.Command{
		Use:           "build <hero>",
		Aliases:       []string{"builds"},
		Short:         "Find popular public in-game builds for a hero",
		Long:          "Lists compact metadata for public in-game builds. Open a viewer link for the complete item path and author annotations.",
		Example:       "  deadlore build Haze\n  deadlore build Haze --sort all-time --limit 10\n  deadlore build Haze --language English\n  deadlore --json build Haze",
		Args:          cobra.MinimumNArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(command *cobra.Command, args []string) error {
			limit, err := command.Flags().GetInt("limit")
			if err != nil {
				return err
			}
			sortName, err := command.Flags().GetString("sort")
			if err != nil {
				return err
			}
			language, err := command.Flags().GetString("language")
			if err != nil {
				return err
			}
			sort, err := buildSort(sortName)
			if err != nil {
				return err
			}
			client, err := deadlockAPIClientFor(options)
			if err != nil {
				return err
			}
			hero, builds, cached, fetchedAt, err := client.ListBuilds(command.Context(), strings.Join(args, " "), deadlockapi.ListOptions{Limit: limit, Sort: sort, Language: language}, options.refresh)
			if err != nil {
				return err
			}
			if options.json {
				return writeJSON(command, map[string]any{
					"type":     "builds",
					"hero":     hero,
					"sort":     sortName,
					"language": language,
					"builds":   builds,
					"source":   map[string]any{"name": "Deadlock API", "url": deadlockapi.DefaultBaseURL, "fetched_at": fetchedAt, "cached": cached},
					"viewer":   map[string]any{"name": "Deadlock Labs", "build_url_format": deadlockapi.DefaultBuildViewerURL + "/{build_id}/"},
				})
			}
			return writeBuilds(command, hero, builds, sortName, language, cached, fetchedAt)
		},
	}
	build.Flags().Int("limit", 5, "maximum builds to show (1-100)")
	build.Flags().String("sort", "weekly", "popularity: weekly, all-time, or recent")
	build.Flags().String("language", "", "filter by build language, e.g. English")
	return build
}

func buildSort(name string) (deadlockapi.Sort, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "weekly":
		return deadlockapi.SortWeekly, nil
	case "all-time", "alltime":
		return deadlockapi.SortAllTime, nil
	case "recent":
		return deadlockapi.SortRecent, nil
	default:
		return "", fmt.Errorf("unknown build sort %q (try weekly, all-time, or recent)", name)
	}
}

func writeBuilds(command *cobra.Command, hero string, builds []deadlockapi.Build, sort, language string, cached bool, fetchedAt time.Time) error {
	title := fmt.Sprintf("%s builds · %s", hero, sort)
	lines := []string{"Public in-game builds; each link opens the full build on Deadlock Labs."}
	if language != "" {
		lines = append(lines, "Language: "+language)
	}
	if len(builds) == 0 {
		lines = append(lines, "No public builds matched those filters.")
	} else {
		for index, build := range builds {
			lines = append(lines, fmt.Sprintf("%d. %s", index+1, build.Name))
			lines = append(lines, "   "+buildPopularity(build, sort)+" · updated "+build.UpdatedAt.Format("2006-01-02"))
			lines = append(lines, fmt.Sprintf("   Build ID %d · view: %s", build.ID, build.ViewerURL))
		}
	}
	fetched := "Deadlock API fetched: " + fetchedAt.Format(time.RFC3339)
	if cached {
		fetched += " (cached)"
	}
	lines = append(lines, fetched)
	writeBox(command.OutOrStdout(), title, lines)
	return nil
}

func buildPopularity(build deadlockapi.Build, sort string) string {
	if sort == "weekly" {
		return fmt.Sprintf("%s weekly favorites", compactCount(build.WeeklyFavorites))
	}
	if sort == "all-time" || sort == "alltime" {
		return fmt.Sprintf("%s all-time favorites", compactCount(build.Favorites))
	}
	return "most recently updated"
}

func compactCount(value int) string {
	if value >= 1000 {
		return fmt.Sprintf("%.1fK", float64(value)/1000)
	}
	return fmt.Sprintf("%d", value)
}
