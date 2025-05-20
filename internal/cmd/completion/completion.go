package completion

import (
	"bytes"
	"context"
	_ "embed"
	"io"
	"os"
	"strings"

	"github.com/urfave/cli/v3"
)

var (
	//go:embed bash_autocomplete
	bashAutocomplete []byte

	//go:embed zsh_autocomplete
	zshAutocomplete []byte
)

func Command() *cli.Command {
	return &cli.Command{
		Name:  "completion",
		Usage: "Print out shell completion script",
		Commands: []*cli.Command{
			{
				Name:  "bash",
				Usage: "Print out bash shell completion script",
				Action: func(_ context.Context, _ *cli.Command) error {
					_, err := io.Copy(os.Stdout, bytes.NewReader(bashAutocomplete))
					return err
				},
			},
			{
				Name:  "zsh",
				Usage: "Print out zsh shell completion script",
				Action: func(_ context.Context, _ *cli.Command) error {
					_, err := io.Copy(os.Stdout, bytes.NewReader(zshAutocomplete))
					return err
				},
			},
			{
				Name:  "fish",
				Usage: "Print out fish shell completion script",
				Action: func(_ context.Context, cliCmd *cli.Command) error {
					s, err := cliCmd.Root().ToFishCompletion()
					if err != nil {
						return err
					}
					_, err = io.Copy(os.Stdout, strings.NewReader(s))
					return err
				},
			},
		},
	}
}
