package profile_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/franela/goblin"
	. "github.com/onsi/gomega"
	"github.com/spacelift-io/spacelift-cli/client/session"
	"github.com/spacelift-io/spacelift-cli/cmd/internal/profile"
)

func TestProfileManager(t *testing.T) {
	g := goblin.Goblin(t)
	RegisterFailHandler(func(m string, _ ...int) { g.Fail(m) })

	g.Describe("ProfileManager", func() {
		var testDirectory string
		var profilesDirectory string
		var manager *profile.ProfileManager

		g.BeforeEach(func() {
			var err error
			if testDirectory, err = ioutil.TempDir("", "spacectlProfiles"); err != nil {
				t.Errorf("Could not create a temp profiles directory: %w", err)
			}

			profilesDirectory = path.Join(testDirectory, "profiles")

			manager = profile.NewManager(profilesDirectory)
		})

		g.AfterEach(func() {
			if err := os.RemoveAll(testDirectory); err != nil {
				g.Fail(fmt.Errorf("Failed to cleanup temp profiles directory: %w", err))
			}
		})

		g.Describe("Init", func() {
			g.Describe("profiles directory doesn't exist", func() {
				g.It("creates directory", func() {
					manager.Init()

					if _, err := os.Stat(profilesDirectory); err != nil {
						g.Fail(fmt.Errorf("Profiles directory has not been created: %w", err))
					}
				})
			})
		})

		g.Describe("an initialised manager", func() {
			g.BeforeEach(func() {
				if err := manager.Init(); err != nil {
					g.Fail(fmt.Errorf("Failed to initialise the profile manager: %w", err))
				}
			})

			g.Describe("Current", func() {
				g.Describe("no profiles exist", func() {
					g.It("returns nil", func() {
						profile, err := manager.Current()

						Expect(profile).To(BeNil())
						Expect(err).To(BeNil())
					})
				})
			})

			g.Describe("Create", func() {
				g.It("sets current profile", func() {
					testProfile := &profile.Profile{
						Name: "test-profile",
						Credentials: &session.StoredCredentials{
							Type:        session.CredentialsTypeGitHubToken,
							Endpoint:    "https://spacectl.app.spacelift.io",
							AccessToken: "abc123",
						},
					}

					manager.Create(testProfile)

					currentProfile, err := manager.Current()

					Expect(err).To(BeNil())
					Expect(currentProfile.Name).To(Equal(testProfile.Name))
				})

				g.It("rejects invalid credential types", func() {
					credentialType := session.CredentialsTypeInvalid
					testProfile := &profile.Profile{
						Name: "invalid-credentials",
						Credentials: &session.StoredCredentials{
							Type: credentialType,
						},
					}

					err := manager.Create(testProfile)

					Expect(err).Should(MatchError(fmt.Sprintf("'%d' is an invalid credential type", credentialType)))
				})

				g.Describe("GitHub credentials", func() {
					profileName := "github-test-profile"

					g.It("creates a new profile", func() {
						testProfile := &profile.Profile{
							Name:        profileName,
							Credentials: createValidGitHubCredentials(),
						}

						err := manager.Create(testProfile)
						Expect(err).To(BeNil())

						savedProfile, err := manager.Get(testProfile.Name)
						if err != nil {
							g.Fail(fmt.Errorf("Failed to get the profile from the manager: %w", err))
						}

						Expect(savedProfile).ToNot(BeNil())
						Expect(savedProfile.Credentials.Type).To(Equal(testProfile.Credentials.Type))
						Expect(savedProfile.Credentials.Endpoint).To(Equal(testProfile.Credentials.Endpoint))
						Expect(savedProfile.Credentials.AccessToken).To(Equal(testProfile.Credentials.AccessToken))
					})

					g.It("rejects GitHub credentials if no access token is specified", func() {
						testProfile := &profile.Profile{
							Name: profileName,
							Credentials: &session.StoredCredentials{
								Type:     session.CredentialsTypeGitHubToken,
								Endpoint: "https://spacectl.app.spacelift.io",
							},
						}

						err := manager.Create(testProfile)

						Expect(err).Should(MatchError("'AccessToken' must be provided for GitHub token credentials"))
					})
				})

				g.Describe("Spacelift API Key credentials", func() {
					profileName := "api-key-profile"

					g.It("creates a new profile", func() {
						testProfile := &profile.Profile{
							Name: profileName,
							Credentials: &session.StoredCredentials{
								Type:      session.CredentialsTypeAPIKey,
								Endpoint:  "https://spacectl.app.spacelift.io",
								KeyID:     "ABC123",
								KeySecret: "SuperSecret",
							},
						}

						err := manager.Create(testProfile)
						Expect(err).To(BeNil())

						savedProfile, err := manager.Get(testProfile.Name)
						if err != nil {
							g.Fail(fmt.Errorf("Failed to get the profile from the manager: %w", err))
						}

						Expect(savedProfile).ToNot(BeNil())
						Expect(savedProfile.Credentials.Type).To(Equal(testProfile.Credentials.Type))
						Expect(savedProfile.Credentials.Endpoint).To(Equal(testProfile.Credentials.Endpoint))
						Expect(savedProfile.Credentials.KeyID).To(Equal(testProfile.Credentials.KeyID))
						Expect(savedProfile.Credentials.KeySecret).To(Equal(testProfile.Credentials.KeySecret))
					})

					g.It("rejects credentials if no KeyID is specified", func() {
						testProfile := &profile.Profile{
							Name: profileName,
							Credentials: &session.StoredCredentials{
								Type:      session.CredentialsTypeAPIKey,
								Endpoint:  "https://spacectl.app.spacelift.io",
								KeySecret: "SuperSecret",
							},
						}

						err := manager.Create(testProfile)

						Expect(err).Should(MatchError("'KeyID' must be provided for API Key credentials"))
					})

					g.It("rejects credentials if no KeySecret is specified", func() {
						testProfile := &profile.Profile{
							Name: profileName,
							Credentials: &session.StoredCredentials{
								Type:     session.CredentialsTypeAPIKey,
								Endpoint: "https://spacectl.app.spacelift.io",
								KeyID:    "ABC123",
							},
						}

						err := manager.Create(testProfile)

						Expect(err).Should(MatchError("'KeySecret' must be provided for API Key credentials"))
					})
				})

				g.It("fails if profile name is not specified", func() {
					err := manager.Create(&profile.Profile{Name: ""})

					Expect(err).Should(MatchError("a profile name must be specified"))
				})

				g.It("fails if profile is nil", func() {
					err := manager.Create(nil)

					Expect(err).Should(MatchError("profile must not be nil"))
				})

				g.It("rejects invalid profile names", func() {
					invalidNames := []string{
						"my/profile",
						"my\\profile",
						"current",
					}

					for _, profileName := range invalidNames {
						err := manager.Create(&profile.Profile{Name: profileName})

						Expect(err).Should(MatchError(fmt.Sprintf("'%s' is not a valid profile name", profileName)))
					}
				})
			})

			g.Describe("Get", func() {
				g.It("can get a profile", func() {
					profileName := "test-profile"
					manager.Create(&profile.Profile{
						Name:        profileName,
						Credentials: createValidGitHubCredentials(),
					})

					testProfile, err := manager.Get(profileName)

					if err != nil {
						g.Fail(fmt.Errorf("failed to retrieve profile: %w", err))
					}

					Expect(testProfile.Name).To(Equal(profileName))
				})

				g.It("returns error if profile file does not exist", func() {
					profileName := "non-existent"
					_, err := manager.Get(profileName)

					Expect(err).Should(MatchError(fmt.Sprintf("a profile named '%s' could not be found", profileName)))
				})

				g.It("returns error if profile name is empty", func() {
					_, err := manager.Get("")

					Expect(err).Should(MatchError("a profile name must be specified"))
				})
			})

			g.Describe("Select", func() {
				g.It("can set the current profile", func() {
					manager.Create(createValidProfile("profile1"))
					manager.Create(createValidProfile("profile2"))

					manager.Select("profile1")

					currentProfile, _ := manager.Current()
					Expect(currentProfile.Name).To(Equal("profile1"))
				})

				g.It("returns error if profile to select does not exist", func() {
					profileName := "non-existent"

					err := manager.Select(profileName)

					Expect(err).Should(MatchError(fmt.Sprintf("could not find a profile named '%s'", "non-existent")))
				})
			})

			g.Describe("Delete", func() {
				g.It("can delete a profile", func() {
					manager.Create(createValidProfile("profile1"))

					manager.Delete("profile1")

					_, err := os.Stat(manager.ProfilePath("profile1"))
					Expect(os.IsNotExist(err)).To(BeTrue())
				})

				g.It("returns error if profile does not exist", func() {
					err := manager.Delete("non-existent")

					Expect(err).Should(MatchError(fmt.Sprintf("no profile named '%s' exists", "non-existent")))
				})

				g.It("returns error if profile name is empty", func() {
					err := manager.Delete("")

					Expect(err).Should(MatchError("a profile name must be specified"))
				})

				g.It("unsets profile if it is the current profile", func() {
					manager.Create(createValidProfile("profile1"))

					manager.Delete("profile1")

					_, err := os.Lstat(manager.CurrentPath)
					Expect(os.IsNotExist(err)).To(BeTrue())
				})

				g.It("does not unset profile if it is not the current profile", func() {
					manager.Create(createValidProfile("profile1"))
					manager.Create(createValidProfile("profile2"))

					manager.Delete("profile1")

					_, err := os.Lstat(manager.CurrentPath)
					Expect(err).To(BeNil())
				})
			})
		})
	})
}

func createValidProfile(name string) *profile.Profile {
	return &profile.Profile{
		Name:        name,
		Credentials: createValidGitHubCredentials(),
	}
}

func createValidGitHubCredentials() *session.StoredCredentials {
	return &session.StoredCredentials{
		Type:        session.CredentialsTypeGitHubToken,
		Endpoint:    "https://spacectl.app.spacelift.io",
		AccessToken: "abc123",
	}
}
