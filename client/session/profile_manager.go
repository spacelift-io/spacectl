package session

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// SpaceliftConfigDirectory is the name of the Spacelift config directory.
	SpaceliftConfigDirectory = ".spacelift"

	// ConfigFileName is the name of the file containing the spacectl config.
	ConfigFileName = "config.json"
)

// invalidProfileAliases contains a list of strings that cannot be used as profile aliases.
var invalidProfileAliases = []string{"/", "\\", "current", ".", ".."}

// configuration is used to serialise the spacectl configuration to file.
type configuration struct {
	// CurrentProfileAlias contains the alias of the currently selected profile.
	CurrentProfileAlias string `json:"currentProfileAlias,omitempty"`

	// Profiles contains all the profiles.
	Profiles map[string]*Profile `json:"profiles,omitempty"`
}

// A Profile represents a spacectl profile which is used to store credential information
// for accessing Spacelift.
type Profile struct {
	// The alias (name) of the profile.
	Alias string `json:"alias,omitempty"`

	// The credentials used to make Spacelift API requests.
	Credentials *StoredCredentials `json:"credentials,omitempty"`
}

// A ProfileManager is used to interact with Spacelift profiles.
type ProfileManager struct {
	// The full path to the spacectl config file.
	ConfigurationFile string

	// The spacectl configuration.
	Configuration *configuration
}

// NewProfileManager creates a new ProfileManager using the specified directory to store the profile data.
func NewProfileManager(profilesDirectory string) (*ProfileManager, error) {
	if err := os.MkdirAll(profilesDirectory, 0700); err != nil {
		return nil, fmt.Errorf("could not create '%s' directory to store Spacelift profiles: %w", profilesDirectory, err)
	}

	manager := &ProfileManager{
		ConfigurationFile: filepath.Join(profilesDirectory, ConfigFileName),
	}

	if err := manager.loadConfiguration(); err != nil {
		return nil, fmt.Errorf("failed to load configuration information: %w", err)
	}

	return manager, nil
}

// Get returns the profile with the specified alias, returning nil if that profile does not exist.
func (m *ProfileManager) Get(profileAlias string) (*Profile, error) {
	if profileAlias == "" {
		return nil, errors.New("a profile alias must be specified")
	}

	return m.Configuration.Profiles[profileAlias], nil
}

// Current gets the user's currently selected profile, and returns nil if no profile is selected.
func (m *ProfileManager) Current() *Profile {
	if m.Configuration.CurrentProfileAlias == "" {
		return nil
	}

	return m.Configuration.Profiles[m.Configuration.CurrentProfileAlias]
}

// Select sets the currently selected profile.
func (m *ProfileManager) Select(profileAlias string) error {
	profile, err := m.Get(profileAlias)
	if err != nil {
		return fmt.Errorf("could not find a profile named '%s': %w", profileAlias, err)
	}

	if profile == nil {
		return fmt.Errorf("could not find a profile named '%s'", profileAlias)
	}

	m.Configuration.CurrentProfileAlias = profileAlias

	return m.writeConfigurationToFile()
}

// Create adds a new Spacelift profile.
func (m *ProfileManager) Create(profile *Profile) error {
	if err := validateProfile(profile); err != nil {
		return err
	}

	m.Configuration.Profiles[profile.Alias] = profile
	m.Configuration.CurrentProfileAlias = profile.Alias
	m.writeConfigurationToFile()

	return nil
}

// Delete removes the profile with the specified alias, and un-selects it as the current profile
// if it was selected.
func (m *ProfileManager) Delete(profileAlias string) error {
	if profileAlias == "" {
		return errors.New("a profile alias must be specified")
	}

	profile := m.Configuration.Profiles[profileAlias]

	if profile == nil {
		return fmt.Errorf("no profile named '%s' exists", profileAlias)
	}

	delete(m.Configuration.Profiles, profileAlias)
	return m.writeConfigurationToFile()
}

// GetAll returns all the currently stored profiles, returning an empty slice if no profiles exist.
func (m *ProfileManager) GetAll() []*Profile {
	var profiles []*Profile

	for _, profile := range m.Configuration.Profiles {
		profiles = append(profiles, profile)
	}

	return profiles
}

func validateProfile(profile *Profile) error {
	if profile == nil {
		return errors.New("profile must not be nil")
	}

	if profile.Alias == "" {
		return errors.New("a profile alias must be specified")
	}

	for _, invalidAlias := range invalidProfileAliases {
		if strings.Contains(profile.Alias, invalidAlias) {
			return fmt.Errorf("'%s' is not a valid profile alias", profile.Alias)
		}
	}

	if profile.Credentials.Endpoint == "" {
		return errors.New("'Endpoint' must be provided")
	}

	switch credentialType := profile.Credentials.Type; credentialType {
	case CredentialsTypeGitHubToken:
		if err := validateGitHubCredentials(profile); err != nil {
			return err
		}

	case CredentialsTypeAPIKey:
		if err := validateAPIKeyCredentials(profile); err != nil {
			return err
		}

	default:
		return fmt.Errorf("'%d' is an invalid credential type", credentialType)
	}

	return nil
}

func validateGitHubCredentials(profile *Profile) error {
	if profile.Credentials.AccessToken == "" {
		return errors.New("'AccessToken' must be provided for GitHub token credentials")
	}

	return nil
}

func validateAPIKeyCredentials(profile *Profile) error {
	if profile.Credentials.KeyID == "" {
		return errors.New("'KeyID' must be provided for API Key credentials")
	}

	if profile.Credentials.KeySecret == "" {
		return errors.New("'KeySecret' must be provided for API Key credentials")
	}

	return nil
}

func (m *ProfileManager) loadConfiguration() error {
	data, err := os.ReadFile(m.ConfigurationFile)
	if err != nil {
		// The config file doesn't exist - just create an empty config
		if os.IsNotExist(err) {
			m.Configuration = &configuration{Profiles: make(map[string]*Profile)}
			return nil
		}

		return fmt.Errorf("could not read configuration file from '%s': %w", m.ConfigurationFile, err)
	}

	var config configuration
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("could not unmarshal spacectl config from '%s': %w", m.ConfigurationFile, err)
	}

	m.Configuration = &config

	return nil
}

func (m *ProfileManager) writeConfigurationToFile() error {
	file, err := os.OpenFile(m.ConfigurationFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("could not create config file at '%s': %w", m.ConfigurationFile, err)
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(m.Configuration); err != nil {
		return fmt.Errorf("could not write config file at '%s': %w", m.ConfigurationFile, err)
	}

	if err := file.Close(); err != nil {
		return fmt.Errorf("could not close the config file at '%s': %w", m.ConfigurationFile, err)
	}

	return nil
}
