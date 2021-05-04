module github.com/spacelift-io/spacectl

go 1.16

require (
	github.com/cheggaaa/pb/v3 v3.0.8
	github.com/dgrijalva/jwt-go/v4 v4.0.0-preview1
	github.com/franela/goblin v0.0.0-20210113153425-413781f5e6c8
	github.com/gookit/color v1.4.2 // indirect
	github.com/mholt/archiver/v3 v3.5.0
	github.com/onsi/gomega v1.11.0
	github.com/pterm/pterm v0.12.13
	github.com/sabhiram/go-gitignore v0.0.0-20201211210132-54b8a0bf510f
	github.com/shurcooL/graphql v0.0.0-20200928012149-18c5c3165e3a
	github.com/urfave/cli/v2 v2.3.0
	golang.org/x/net v0.0.0-20210405180319-a5a99cb37ef4 // indirect
	golang.org/x/oauth2 v0.0.0-20210402161424-2e8d93401602
	golang.org/x/sys v0.0.0-20210412220455-f1c623a9e750 // indirect
	golang.org/x/term v0.0.0-20210406210042-72f3dc4e9b72
)

replace github.com/mholt/archiver/v3 => github.com/spacelift-io/archiver/v3 v3.3.1-0.20210427125142-c305b5a627ba
