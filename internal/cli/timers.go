package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

// TimerReferenceDate is the game patch date through which the built-in timer
// reference has been checked. It is deliberately static: timer commands never
// fetch a web page or depend on the local wiki cache.
const TimerReferenceDate = "2026-06-30"

type timer struct {
	Name       string `json:"name"`
	FirstSpawn string `json:"first_spawn"`
	Respawn    string `json:"respawn"`
	Notes      string `json:"notes,omitempty"`
}

type timerGroup struct {
	Name    string   `json:"name"`
	Aliases []string `json:"-"`
	Timers  []timer  `json:"timers"`
}

var timerGroups = []timerGroup{
	{
		Name:    "Jungle & farm",
		Aliases: []string{"camps", "camp", "jungle", "farm"},
		Timers: []timer{
			{Name: "Small camps", FirstSpawn: "2:00", Respawn: "~1:25", Notes: "Per-camp timer after the camp is cleared."},
			{Name: "Medium camps", FirstSpawn: "5:00", Respawn: "~5:00", Notes: "Per-camp timer after the camp is cleared."},
			{Name: "Large camps", FirstSpawn: "8:00", Respawn: "~5–6:00", Notes: "Per-camp timer after the camp is cleared."},
			{Name: "Sinner's Sacrifice", FirstSpawn: "8:00", Respawn: "5:00", Notes: "Also called a vault."},
		},
	},
	{
		Name:    "Map pickups",
		Aliases: []string{"pickups", "pickup", "map", "breakables", "powerups", "power-ups"},
		Timers: []timer{
			{Name: "Breakables", FirstSpawn: "3:00", Respawn: "3:00", Notes: "Crates, jars, statues, and similar props."},
			{Name: "Power-ups", FirstSpawn: "5:00", Respawn: "5:00", Notes: "Map power-up spawners."},
		},
	},
	{
		Name:    "Objectives",
		Aliases: []string{"objectives", "objective", "obj", "boss", "urn", "rift"},
		Timers: []timer{
			{Name: "Mid-Boss", FirstSpawn: "Match start", Respawn: "~7:00", Notes: "Respawn starts after it is defeated."},
			{Name: "Soul Urn", FirstSpawn: "10:00", Respawn: "5:00", Notes: "Scheduled at 10:00, 15:00, 20:00, and so on."},
			{Name: "Unstable Rift", FirstSpawn: "Variable", Respawn: "~7:00", Notes: "A lane is marked before a 25-second global countdown."},
		},
	},
}

func newTimersCommand(options *options) *cobra.Command {
	timers := &cobra.Command{
		Use:           "timers [group]",
		Aliases:       []string{"timer"},
		Short:         "Show built-in Deadlock spawn and respawn timers",
		Long:          "Show the offline Deadlore timer reference. Use cheatsheet, camps, pickups, or objectives to focus the output.",
		Args:          cobra.MaximumNArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(command *cobra.Command, args []string) error {
			groups := timerGroups
			if len(args) == 1 {
				var found bool
				groups, found = findTimerGroup(args[0])
				if !found {
					return fmt.Errorf("unknown timer group %q (try camps, pickups, or objectives)", args[0])
				}
			}
			return writeTimers(command, options, groups)
		},
	}

	timers.AddCommand(newTimerCheatsheetCommand(options))
	timers.AddCommand(newTimerGroupCommand("camps", []string{"camp", "jungle", "farm"}, "Show jungle-camp and vault timers", options))
	timers.AddCommand(newTimerGroupCommand("pickups", []string{"pickup", "map", "breakables", "powerups", "power-ups"}, "Show breakable and power-up timers", options))
	timers.AddCommand(newTimerGroupCommand("objectives", []string{"objective", "obj", "boss", "urn", "rift"}, "Show Mid-Boss, Urn, and Rift timers", options))
	return timers
}

func newTimerCheatsheetCommand(options *options) *cobra.Command {
	return &cobra.Command{
		Use:           "cheatsheet",
		Aliases:       []string{"cheat", "sheet"},
		Short:         "Show the complete timer cheat sheet",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(command *cobra.Command, _ []string) error {
			return writeTimers(command, options, timerGroups)
		},
	}
}

func newTimerGroupCommand(name string, aliases []string, short string, options *options) *cobra.Command {
	return &cobra.Command{
		Use:           name,
		Aliases:       aliases,
		Short:         short,
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(command *cobra.Command, _ []string) error {
			groups, _ := findTimerGroup(name)
			return writeTimers(command, options, groups)
		},
	}
}

func newCheatCommand(options *options) *cobra.Command {
	return &cobra.Command{
		Use:           "cheat",
		Aliases:       []string{"cheatsheet"},
		Short:         "Show the built-in Deadlock timer cheat sheet",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(command *cobra.Command, _ []string) error {
			return writeTimers(command, options, timerGroups)
		},
	}
}

func findTimerGroup(name string) ([]timerGroup, bool) {
	name = normalizeName(name)
	for _, group := range timerGroups {
		if normalizeName(group.Name) == name {
			return []timerGroup{group}, true
		}
		for _, alias := range group.Aliases {
			if normalizeName(alias) == name {
				return []timerGroup{group}, true
			}
		}
	}
	return nil, false
}

func writeTimers(command *cobra.Command, options *options, groups []timerGroup) error {
	if options.json {
		return writeJSON(command, map[string]any{
			"reference": map[string]string{
				"source":          "Deadlore built-in timer reference",
				"checked_through": TimerReferenceDate,
			},
			"groups": groups,
		})
	}

	output := command.OutOrStdout()
	for index, group := range groups {
		if index > 0 {
			fmt.Fprintln(output)
		}
		writeTimerGroup(output, group)
	}
	fmt.Fprintln(output)
	writeBox(output, "Deadlore timer reference", []string{
		"Built in to deadlore · checked through " + TimerReferenceDate,
		"~ means the game timer is approximate or variable. Fixed schedules have no ~.",
	})
	return nil
}

func writeTimerGroup(output io.Writer, group timerGroup) {
	lines := make([]string, 0, len(group.Timers)*2)
	for _, item := range group.Timers {
		lines = append(lines, fmt.Sprintf("• %s — First: %s · Respawn: %s", item.Name, item.FirstSpawn, item.Respawn))
		if item.Notes != "" {
			lines = append(lines, "  "+item.Notes)
		}
	}
	writeBox(output, group.Name, lines)
}
