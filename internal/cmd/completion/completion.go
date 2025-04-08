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
				Action: func(ctx context.Context, cmd *cli.Command) error {
					_, err := io.Copy(os.Stdout, bytes.NewReader(bashAutocomplete))
					return err
				},
			},
			{
				Name:  "zsh",
				Usage: "Print out zsh shell completion script",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					_, err := io.Copy(os.Stdout, bytes.NewReader(zshAutocomplete))
					return err
				},
			},
			{
				Name:  "fish",
				Usage: "Print out fish shell completion script",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					// Create fish completion script manually since API changed in v3
					s := "function __complete_spacectl\n" +
						"    set -lx COMP_LINE (commandline -cp)\n" +
						"    test -z (commandline -ct) && set COMP_LINE \"$COMP_LINE \"\n" +
						"    spacectl\n" +
						"end\n" +
						"complete -f -c spacectl -a \"(__complete_spacectl)\"\n"
					_, err := io.Copy(os.Stdout, strings.NewReader(s))
					return err
				},
			},
		},
	}
}
