package profile

import (
	"sort"

	"github.com/pterm/pterm"
	"github.com/urfave/cli/v2"

	"github.com/spacelift-io/spacectl/client/session"
	"github.com/spacelift-io/spacectl/internal/cmd"
)

func listCommand() *cli.Command {
	return &cli.Command{
		Name:      "list",
		Usage:     "List all your Spacelift account profiles",
		ArgsUsage: cmd.EmptyArgsUsage,
		Action: func(*cli.Context) error {
			profiles := manager.GetAll()

			currentProfile := manager.Current()

			// Make sure we output the profiles in a consistent order
			sort.SliceStable(profiles, func(i int, j int) bool {
				return profiles[i].Alias < profiles[j].Alias
			})

			tableData := [][]string{{"Current", "Alias", "Endpoint", "Type"}}
			for _, profile := range profiles {
				tableData = append(tableData, []string{
					getCurrentProfileString(profile, currentProfile),
					profile.Alias,
					profile.Credentials.Endpoint,
					profile.Credentials.Type.String(),
				})
			}

			return pterm.
				DefaultTable.
				WithHasHeader().
				WithData(tableData).
				Render()
		},
	}
}

func getCurrentProfileString(profile *session.Profile, currentProfile *session.Profile) string {
	if currentProfile != nil && profile.Alias == currentProfile.Alias {
		return "*"
	}

	return ""
}
