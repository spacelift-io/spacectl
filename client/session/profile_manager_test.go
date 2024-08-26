package session_test

import (
	"fmt"
	"os"
	"path"
	"sort"
	"testing"

	"github.com/franela/goblin"
	"github.com/onsi/gomega"
	"github.com/spacelift-io/spacectl/client/session"
)

func TestProfileManager(t *testing.T) {
	g := goblin.Goblin(t)
	gomega.RegisterFailHandler(func(m string, _ ...int) { g.Fail(m) })

	g.Describe("ProfileManager", func() {
		var testDirectory string
		var profilesDirectory string
		var manager *session.ProfileManager

		g.BeforeEach(func() {
			var err error
			if testDirectory, err = os.MkdirTemp("", "spacectlProfiles"); err != nil {
				g.Fail(fmt.Errorf("Could not create a temp profiles directory: %w", err))
			}

			profilesDirectory = path.Join(testDirectory, "profiles")

			if manager, err = session.NewProfileManager(profilesDirectory); err != nil {
				g.Fail(fmt.Errorf("could not create profile manager: %w", err))
			}
		})

		g.AfterEach(func() {
			if err := os.RemoveAll(testDirectory); err != nil {
				g.Fail(fmt.Errorf("failed to cleanup temp profiles directory: %w", err))
			}
		})

		g.Describe("NewProfileManager", func() {
			g.Describe("profiles directory doesn't exist", func() {
				g.It("creates directory", func() {
					gomega.Expect(profilesDirectory).Should(gomega.BeADirectory())
				})
			})

			g.Describe("profiles map is null", func() {
				g.It("initializes profiles map", func() {
					configFilename := path.Join(profilesDirectory, session.ConfigFileName)
					err := os.WriteFile(configFilename, []byte("{}"), 0600)
					gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

					manager, err = session.NewProfileManager(profilesDirectory)
					gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

					gomega.Expect(manager.Configuration.Profiles).ToNot(gomega.BeNil())
				})
			})
		})

		g.Describe("Current", func() {
			g.Describe("no profiles exist", func() {
				g.It("returns nil", func() {
					profile := manager.Current()

					gomega.Expect(profile).To(gomega.BeNil())
				})
			})
		})

		g.Describe("Create", func() {
			g.It("sets current profile", func() {
				testProfile := &session.Profile{
					Alias: "test-profile",
					Credentials: &session.StoredCredentials{
						Type:        session.CredentialsTypeGitHubToken,
						Endpoint:    "https://spacectl.app.spacelift.io",
						AccessToken: "abc123",
					},
				}

				err := manager.Create(testProfile)
				gomega.Expect(err).To(gomega.BeNil())

				currentProfile := manager.Current()

				gomega.Expect(currentProfile).NotTo(gomega.BeNil())
				gomega.Expect(currentProfile.Alias).To(gomega.Equal(testProfile.Alias))
			})

			g.It("rejects invalid credential types", func() {
				credentialType := session.CredentialsTypeInvalid
				testProfile := &session.Profile{
					Alias: "invalid-credentials",
					Credentials: &session.StoredCredentials{
						Endpoint: "https://spacectl.app.spacelift.io",
						Type:     credentialType,
					},
				}

				err := manager.Create(testProfile)

				gomega.Expect(err).Should(gomega.MatchError(fmt.Sprintf("'%d' is an invalid credential type", credentialType)))
			})

			g.Describe("All credential types", func() {
				for _, testProfile := range createAllValidProfileTypes() {
					g.It(fmt.Sprintf("fails if Endpoint is not specified for %s", testProfile.Alias), func() {
						testProfile.Credentials.Endpoint = ""

						err := manager.Create(testProfile)

						gomega.Expect(err).Should(gomega.MatchError("'Endpoint' must be provided"))
					})
				}
			})

			g.Describe("GitHub credentials", func() {
				profileAlias := "github-test-profile"

				g.It("creates a new profile", func() {
					testProfile := &session.Profile{
						Alias:       profileAlias,
						Credentials: createValidGitHubCredentials(),
					}

					err := manager.Create(testProfile)
					gomega.Expect(err).To(gomega.BeNil())

					savedProfile, err := manager.Get(testProfile.Alias)

					gomega.Expect(err).To(gomega.BeNil())
					gomega.Expect(savedProfile).ToNot(gomega.BeNil())
					gomega.Expect(savedProfile.Credentials.Type).To(gomega.Equal(testProfile.Credentials.Type))
					gomega.Expect(savedProfile.Credentials.Endpoint).To(gomega.Equal(testProfile.Credentials.Endpoint))
					gomega.Expect(savedProfile.Credentials.AccessToken).To(gomega.Equal(testProfile.Credentials.AccessToken))
				})

				g.It("rejects GitHub credentials if no access token is specified", func() {
					testProfile := &session.Profile{
						Alias: profileAlias,
						Credentials: &session.StoredCredentials{
							Type:     session.CredentialsTypeGitHubToken,
							Endpoint: "https://spacectl.app.spacelift.io",
						},
					}

					err := manager.Create(testProfile)

					gomega.Expect(err).Should(gomega.MatchError("'AccessToken' must be provided for GitHub token credentials"))
				})
			})

			g.Describe("Spacelift API Key credentials", func() {
				profileAlias := "api-key-profile"

				g.It("creates a new profile", func() {
					testProfile := &session.Profile{
						Alias:       profileAlias,
						Credentials: createValidAPIKeyCredentials(),
					}

					err := manager.Create(testProfile)
					gomega.Expect(err).To(gomega.BeNil())

					savedProfile, err := manager.Get(testProfile.Alias)

					gomega.Expect(err).To(gomega.BeNil())
					gomega.Expect(savedProfile).ToNot(gomega.BeNil())
					gomega.Expect(savedProfile.Credentials.Type).To(gomega.Equal(testProfile.Credentials.Type))
					gomega.Expect(savedProfile.Credentials.Endpoint).To(gomega.Equal(testProfile.Credentials.Endpoint))
					gomega.Expect(savedProfile.Credentials.KeyID).To(gomega.Equal(testProfile.Credentials.KeyID))
					gomega.Expect(savedProfile.Credentials.KeySecret).To(gomega.Equal(testProfile.Credentials.KeySecret))
				})

				g.It("rejects credentials if no KeyID is specified", func() {
					testProfile := &session.Profile{
						Alias: profileAlias,
						Credentials: &session.StoredCredentials{
							Type:      session.CredentialsTypeAPIKey,
							Endpoint:  "https://spacectl.app.spacelift.io",
							KeySecret: "SuperSecret",
						},
					}

					err := manager.Create(testProfile)

					gomega.Expect(err).Should(gomega.MatchError("'KeyID' must be provided for API Key credentials"))
				})

				g.It("rejects credentials if no KeySecret is specified", func() {
					testProfile := &session.Profile{
						Alias: profileAlias,
						Credentials: &session.StoredCredentials{
							Type:     session.CredentialsTypeAPIKey,
							Endpoint: "https://spacectl.app.spacelift.io",
							KeyID:    "ABC123",
						},
					}

					err := manager.Create(testProfile)

					gomega.Expect(err).Should(gomega.MatchError("'KeySecret' must be provided for API Key credentials"))
				})
			})

			g.It("fails if profile alias is not specified", func() {
				err := manager.Create(&session.Profile{Alias: ""})

				gomega.Expect(err).Should(gomega.MatchError("a profile alias must be specified"))
			})

			g.It("fails if profile is nil", func() {
				err := manager.Create(nil)

				gomega.Expect(err).Should(gomega.MatchError("profile must not be nil"))
			})

			g.It("rejects invalid profile aliases", func() {
				invalidAliases := []string{
					"my/profile",
					"my\\profile",
					"current",
					".",
					"..",
				}

				for _, profileAlias := range invalidAliases {
					testProfile := createValidProfile(profileAlias)
					err := manager.Create(testProfile)

					gomega.Expect(err).Should(gomega.MatchError(fmt.Sprintf("'%s' is not a valid profile alias", profileAlias)))
				}
			})
		})

		g.Describe("Get", func() {
			g.It("can get a profile", func() {
				profileAlias := "test-profile"
				gomega.Expect(manager.Create(&session.Profile{
					Alias:       profileAlias,
					Credentials: createValidGitHubCredentials(),
				})).To(gomega.Succeed())

				testProfile, err := manager.Get(profileAlias)

				gomega.Expect(err).To(gomega.BeNil())
				gomega.Expect(testProfile.Alias).To(gomega.Equal(profileAlias))
			})

			g.It("returns nil if profile does not exist", func() {
				profileAlias := "non-existent"
				profile, err := manager.Get(profileAlias)

				gomega.Expect(err).Should(gomega.BeNil())
				gomega.Expect(profile).Should(gomega.BeNil())
			})

			g.It("returns error if profile alias is empty", func() {
				_, err := manager.Get("")

				gomega.Expect(err).Should(gomega.MatchError("a profile alias must be specified"))
			})
		})

		g.Describe("Select", func() {
			g.It("can set the current profile", func() {
				gomega.Expect(manager.Create(createValidProfile("profile1"))).To(gomega.Succeed())
				gomega.Expect(manager.Create(createValidProfile("profile2"))).To(gomega.Succeed())

				gomega.Expect(manager.Select("profile1")).To(gomega.Succeed())

				currentProfile := manager.Current()
				gomega.Expect(currentProfile).NotTo(gomega.BeNil())
				gomega.Expect(currentProfile.Alias).To(gomega.Equal("profile1"))
			})

			g.It("returns error if profile to select does not exist", func() {
				profileAlias := "non-existent"

				err := manager.Select(profileAlias)

				gomega.Expect(err).Should(gomega.MatchError(fmt.Sprintf("could not find a profile named '%s'", "non-existent")))
			})
		})

		g.Describe("Delete", func() {
			g.It("can delete a profile", func() {
				gomega.Expect(manager.Create(createValidProfile("profile1"))).To(gomega.Succeed())

				gomega.Expect(manager.Delete("profile1")).To(gomega.Succeed())

				profile, err := manager.Get("profile1")
				gomega.Expect(err).To(gomega.BeNil())
				gomega.Expect(profile).To(gomega.BeNil())
			})

			g.It("returns error if profile does not exist", func() {
				err := manager.Delete("non-existent")

				gomega.Expect(err).Should(gomega.MatchError(fmt.Sprintf("no profile named '%s' exists", "non-existent")))
			})

			g.It("returns error if profile alias is empty", func() {
				err := manager.Delete("")

				gomega.Expect(err).Should(gomega.MatchError("a profile alias must be specified"))
			})

			g.It("unsets profile if it is the current profile", func() {
				gomega.Expect(manager.Create(createValidProfile("profile1"))).To(gomega.Succeed())

				gomega.Expect(manager.Delete("profile1")).To(gomega.Succeed())

				current := manager.Current()
				gomega.Expect(current).To(gomega.BeNil())
			})

			g.It("does not unset profile if it is not the current profile", func() {
				gomega.Expect(manager.Create(createValidProfile("profile1"))).To(gomega.Succeed())
				gomega.Expect(manager.Create(createValidProfile("profile2"))).To(gomega.Succeed())

				gomega.Expect(manager.Delete("profile1")).To(gomega.Succeed())

				current := manager.Current()
				gomega.Expect(current).NotTo(gomega.BeNil())
				gomega.Expect(current.Alias).To(gomega.Equal("profile2"))
			})
		})

		g.Describe("GetAll", func() {
			g.It("returns empty when no profiles exist", func() {
				profiles := manager.GetAll()

				gomega.Expect(profiles).To(gomega.BeEmpty())
			})

			g.It("returns all profiles", func() {
				gomega.Expect(manager.Create(createValidProfile("profile-1"))).To(gomega.Succeed())
				gomega.Expect(manager.Create(createValidProfile("profile-2"))).To(gomega.Succeed())
				gomega.Expect(manager.Create(createValidProfile("profile-3"))).To(gomega.Succeed())

				profiles := manager.GetAll()

				// Sort the slice to guarantee the order when comparing the results
				sort.SliceStable(profiles, func(i int, j int) bool {
					return profiles[i].Alias < profiles[j].Alias
				})

				gomega.Expect(len(profiles)).To(gomega.Equal(3))
				gomega.Expect(profiles[0].Alias).To(gomega.Equal("profile-1"))
				gomega.Expect(profiles[1].Alias).To(gomega.Equal("profile-2"))
				gomega.Expect(profiles[2].Alias).To(gomega.Equal("profile-3"))
			})
		})
	})
}

func createValidProfile(alias string) *session.Profile {
	return &session.Profile{
		Alias:       alias,
		Credentials: createValidGitHubCredentials(),
	}
}

func createAllValidProfileTypes() []*session.Profile {
	return []*session.Profile{
		{
			Alias:       "github",
			Credentials: createValidGitHubCredentials(),
		},
		{
			Alias:       "spacelift-api-key",
			Credentials: createValidAPIKeyCredentials(),
		},
		{
			Alias:       "spacelift-api-token",
			Credentials: createValidAPITokenCredentials(),
		},
	}
}

func createValidGitHubCredentials() *session.StoredCredentials {
	return &session.StoredCredentials{
		Type:        session.CredentialsTypeGitHubToken,
		Endpoint:    "https://spacectl.app.spacelift.io",
		AccessToken: "abc123",
	}
}

func createValidAPITokenCredentials() *session.StoredCredentials {
	return &session.StoredCredentials{
		Type:        session.CredentialsTypeAPIToken,
		Endpoint:    "https://spacectl.app.spacelift.io",
		AccessToken: "abc123",
	}
}

func createValidAPIKeyCredentials() *session.StoredCredentials {
	return &session.StoredCredentials{
		Type:      session.CredentialsTypeAPIKey,
		Endpoint:  "https://spacectl.app.spacelift.io",
		KeyID:     "ABC123",
		KeySecret: "supersecret",
	}
}
