package profile

import (
	"context"
	"fmt"
	"sort"

	"github.com/urfave/cli/v3"

	"github.com/spacelift-io/spacectl/client/session"
	internalCmd "github.com/spacelift-io/spacectl/internal/cmd"
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
			internalCmd.FlagOutputFormat,
			internalCmd.FlagNoColor,
		},
		ArgsUsage: internalCmd.EmptyArgsUsage,
		Before:    internalCmd.HandleNoColor,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			profiles := manager.GetAll()

			currentProfile := manager.Current()

			// Make sure we output the profiles in a consistent order
			sort.SliceStable(profiles, func(i int, j int) bool {
				return profiles[i].Alias < profiles[j].Alias
			})

			var outputFormat internalCmd.OutputFormat
			var err error
			if outputFormat, err = internalCmd.GetOutputFormat(cmd); err != nil {
				return err
			}

			switch outputFormat {
			case internalCmd.OutputFormatTable:
				tableData := [][]string{{"Current", "Alias", "Endpoint", "Type"}}
				for _, profile := range profiles {
					tableData = append(tableData, []string{
						getCurrentProfileString(profile, currentProfile),
						profile.Alias,
						profile.Credentials.Endpoint,
						profile.Credentials.Type.String(),
					})
				}

				return internalCmd.OutputTable(tableData, true)

			case internalCmd.OutputFormatJSON:
				var profileList []profileListOutput

				for _, profile := range profiles {
					profileList = append(profileList, profileListOutput{
						Current:  isCurrentProfile(profile, currentProfile),
						Alias:    profile.Alias,
						Endpoint: profile.Credentials.Endpoint,
						Type:     profile.Credentials.Type.String(),
					})
				}

				return internalCmd.OutputJSON(&profileList)
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
