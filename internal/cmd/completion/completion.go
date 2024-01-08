package completion

import (
	"bytes"
	_ "embed"
	"io"
	"os"
	"strings"

	"github.com/urfave/cli/v2"
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
		Subcommands: []*cli.Command{
			{
				Name:  "bash",
				Usage: "Print out bash shell completion script",
				Action: func(cliCtx *cli.Context) error {
					_, err := io.Copy(os.Stdout, bytes.NewReader(bashAutocomplete))
					return err
				},
			},
			{
				Name:  "zsh",
				Usage: "Print out zsh shell completion script",
				Action: func(cliCtx *cli.Context) error {
					_, err := io.Copy(os.Stdout, bytes.NewReader(zshAutocomplete))
					return err
				},
			},
			{
				Name:  "fish",
				Usage: "Print out fish shell completion script",
				Action: func(cliCtx *cli.Context) error {
					s, err := cliCtx.App.ToFishCompletion()
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
