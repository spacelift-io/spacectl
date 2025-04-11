package profile

import (
	"fmt"
	"sort"

	"github.com/urfave/cli/v2"

	"github.com/spacelift-io/spacectl/client/session"
	"github.com/spacelift-io/spacectl/internal/cmd"
)

type profileListOutput struct {
	Current  bool   `json:"current"`
	Alias    string `json:"alias"`
	Endpoint string `json:"endpoint"`
	Type     string `json:"type"`
}

func listCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List all your Spacelift account profiles",
		Flags: []cli.Flag{
			cmd.FlagOutputFormat,
			cmd.FlagNoColor,
		},
		ArgsUsage: cmd.EmptyArgsUsage,
		Before:    cmd.HandleNoColor,
		Action: func(ctx *cli.Context) error {
			profiles := manager.GetAll()

			currentProfile := manager.Current()

			// Make sure we output the profiles in a consistent order
			sort.SliceStable(profiles, func(i int, j int) bool {
				return profiles[i].Alias < profiles[j].Alias
			})

			var outputFormat cmd.OutputFormat
			var err error
			if outputFormat, err = cmd.GetOutputFormat(ctx); err != nil {
				return err
			}

			switch outputFormat {
			case cmd.OutputFormatTable:
				tableData := [][]string{{"Current", "Alias", "Endpoint", "Type"}}
				for _, profile := range profiles {
					tableData = append(tableData, []string{
						getCurrentProfileString(profile, currentProfile),
						profile.Alias,
						profile.Credentials.Endpoint,
						profile.Credentials.Type.String(),
					})
				}

				return cmd.OutputTable(tableData, true)

			case cmd.OutputFormatJSON:
				var profileList []profileListOutput

				for _, profile := range profiles {
					profileList = append(profileList, profileListOutput{
						Current:  isCurrentProfile(profile, currentProfile),
						Alias:    profile.Alias,
						Endpoint: profile.Credentials.Endpoint,
						Type:     profile.Credentials.Type.String(),
					})
				}

				return cmd.OutputJSON(&profileList)
			}

			return fmt.Errorf("unknown output format: %v", outputFormat)
		},
	}
}

func isCurrentProfile(profile *session.Profile, currentProfile *session.Profile) bool {
	return currentProfile != nil && profile.Alias == currentProfile.Alias
}

func getCurrentProfileString(profile *session.Profile, currentProfile *session.Profile) string {
	if isCurrentProfile(profile, currentProfile) {
		return "*"
	}

	return ""
}
