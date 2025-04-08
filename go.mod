module github.com/spacelift-io/spacectl

go 1.23.0

toolchain go1.23.6

require (
	github.com/ProtonMail/gopenpgp/v2 v2.8.3
	github.com/charmbracelet/bubbles v0.20.0
	github.com/charmbracelet/bubbletea v1.3.4
	github.com/charmbracelet/lipgloss v1.1.0
	github.com/cheggaaa/pb/v3 v3.1.7
	github.com/franela/goblin v0.0.0-20211003143422-0a4f594942bf
	github.com/golang-jwt/jwt/v4 v4.5.2
	github.com/manifoldco/promptui v0.9.0
	github.com/mattn/go-isatty v0.0.20
	github.com/mholt/archiver/v3 v3.5.1
	github.com/onsi/gomega v1.37.0
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c
	github.com/pkg/errors v0.9.1
	github.com/pterm/pterm v0.12.80
	github.com/sabhiram/go-gitignore v0.0.0-20210923224102-525f6e181f06
	github.com/shurcooL/graphql v0.0.0-20230722043721-ed46e5a46466
	github.com/urfave/cli/v3 v3.1.1
	golang.org/x/oauth2 v0.29.0
	golang.org/x/sync v0.13.0
	golang.org/x/term v0.31.0
)

require (
	atomicgo.dev/cursor v0.2.0 // indirect
	atomicgo.dev/keyboard v0.2.9 // indirect
	atomicgo.dev/schedule v0.1.0 // indirect
	github.com/ProtonMail/go-crypto v1.1.6 // indirect
	github.com/ProtonMail/go-mime v0.0.0-20230322103455-7d82a3887f2f // indirect
	github.com/VividCortex/ewma v1.2.0 // indirect
	github.com/andybalholm/brotli v0.0.0-20190621154722-5f990b63d2d6 // indirect
	github.com/aymanbagabas/go-osc52/v2 v2.0.1 // indirect
	github.com/charmbracelet/colorprofile v0.2.3-0.20250311203215-f60798e515dc // indirect
	github.com/charmbracelet/x/ansi v0.8.0 // indirect
	github.com/charmbracelet/x/cellbuf v0.0.13-0.20250311204145-2c3ea96c31dd // indirect
	github.com/charmbracelet/x/term v0.2.1 // indirect
	github.com/chzyer/readline v0.0.0-20180603132655-2972be24d48e // indirect
	github.com/cloudflare/circl v1.3.7 // indirect
	github.com/containerd/console v1.0.4-0.20230313162750-1ae8d489ac81 // indirect
	github.com/dsnet/compress v0.0.1 // indirect
	github.com/erikgeiser/coninput v0.0.0-20211004153227-1c3628e74d0f // indirect
	github.com/fatih/color v1.18.0 // indirect
	github.com/golang/gddo v0.0.0-20190419222130-af0f2af80721 // indirect
	github.com/golang/snappy v0.0.1 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/gookit/color v1.5.4 // indirect
	github.com/klauspost/compress v1.9.2 // indirect
	github.com/klauspost/pgzip v1.2.1 // indirect
	github.com/lithammer/fuzzysearch v1.1.8 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-localereader v0.0.1 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/muesli/ansi v0.0.0-20230316100256-276c6243b2f6 // indirect
	github.com/muesli/cancelreader v0.2.2 // indirect
	github.com/muesli/termenv v0.16.0 // indirect
	github.com/nwaples/rardecode v1.0.0 // indirect
	github.com/pierrec/lz4 v2.0.5+incompatible // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/ulikunitz/xz v0.5.6 // indirect
	github.com/xi2/xz v0.0.0-20171230120015-48954b6210f8 // indirect
	github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e // indirect
	golang.org/x/crypto v0.36.0 // indirect
	golang.org/x/exp v0.0.0-20240719175910-8a7402abbf56 // indirect
	golang.org/x/net v0.37.0 // indirect
	golang.org/x/sys v0.32.0 // indirect
	golang.org/x/text v0.23.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/mholt/archiver/v3 => github.com/spacelift-io/archiver/v3 v3.3.1-0.20221117135619-d7d90ab08987

replace github.com/shurcooL/graphql => github.com/spacelift-io/graphql v1.2.0
