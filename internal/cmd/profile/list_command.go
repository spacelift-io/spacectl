package profile

import (
	"fmt"
	"sort"

	"github.com/pterm/pterm"
	"github.com/urfave/cli/v2"

	"github.com/spacelift-io/spacectl/client/session"
)

func listCommand() *cli.Command {
	return &cli.Command{
		Name:      "list",
		Usage:     "List all your Spacelift account profiles",
		ArgsUsage: " ",
		Action: func(*cli.Context) error {
			profiles, err := manager.GetAll()
			if err != nil {
				return fmt.Errorf("could not load Spacelift profiles: %w", err)
			}

			currentProfile, err := manager.Current()
			if err != nil {
				return fmt.Errorf("could not get current profile: %w", err)
			}

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
	if profile.Alias == currentProfile.Alias {
		return "*"
	}

	return ""
}
