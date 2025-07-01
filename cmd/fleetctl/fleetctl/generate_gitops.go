package fleetctl

import (
	"bytes"
	"fmt"
	"os"
	pathUtils "path"
	"reflect"
	"regexp"
	"strings"
	"unicode"

	"github.com/fleetdm/fleet/v4/pkg/spec"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/ghodss/yaml"
	"github.com/urfave/cli/v2"
)

type SecretWarning struct {
	Filename string
	Key      string
}

type Note struct {
	Filename string
	Note     string
}

type Messages struct {
	SecretWarnings []SecretWarning
	Notes          []Note
}

type Comment struct {
	Filename string
	Comment  string
	Token    string
}

type FileToWrite struct {
	Path    string
	Content map[string]interface{}
}

type Software struct {
	Hash       string
	AppStoreId string
	Comment    string
}

type teamToProcess struct {
	ID   *uint
	Team *fleet.Team
}

type generateGitopsClient interface {
	GetAppConfig() (*fleet.EnrichedAppConfig, error)
	GetEnrollSecretSpec() (*fleet.EnrollSecretSpec, error)
	ListTeams(query string) ([]fleet.Team, error)
	ListScripts(query string) ([]*fleet.Script, error)
	ListConfigurationProfiles(teamID *uint) ([]*fleet.MDMConfigProfilePayload, error)
	GetScriptContents(scriptID uint) ([]byte, error)
	GetProfileContents(profileID string) ([]byte, error)
	GetEULAMetadata() (*fleet.MDMEULA, error)
	GetEULAContent(token string) ([]byte, error)
	GetTeam(teamID uint) (*fleet.Team, error)
	ListSoftwareTitles(query string) ([]fleet.SoftwareTitleListResult, error)
	GetSoftwareTitleByID(ID uint, teamID *uint) (*fleet.SoftwareTitle, error)
	GetPolicies(teamID *uint) ([]*fleet.Policy, error)
	GetQueries(teamID *uint, name *string) ([]fleet.Query, error)
	GetLabels() ([]*fleet.LabelSpec, error)
	Me() (*fleet.User, error)
}

// Given a struct type and a field name, return the JSON field name.
func jsonFieldName(t reflect.Type, fieldName string) string {
	field, ok := t.FieldByName(fieldName)
	if !ok {
		panic(fieldName + " not found in " + t.Name())
	}
	tag := field.Tag.Get("json")
	parts := strings.Split(tag, ",")
	name := parts[0]

	if name == "-" || name == "" {
		panic(field.Name + " has no json tag")
	}
	return name
}

// Given a dot-separated path, return the value at that key in a map.
func getValueAtKey(data map[string]interface{}, path string) (interface{}, bool) {
	// Split the path into parts.
	parts := strings.Split(path, ".")
	var cur interface{} = data

	// Keep traversing the map using the keys in the path.
	for _, key := range parts {
		mp, ok := cur.(map[string]interface{})
		if !ok {
			return nil, false
		}
		cur, ok = mp[key]
		if !ok {
			return nil, false
		}
	}
	return cur, true
}

type GenerateGitopsCommand struct {
	Client       generateGitopsClient
	CLI          *cli.Context
	Messages     Messages
	FilesToWrite map[string]interface{}
	Comments     []Comment
	AppConfig    *fleet.EnrichedAppConfig
	SoftwareList map[uint]Software
	ScriptList   map[uint]string
}

func generateGitopsCommand() *cli.Command {
	return &cli.Command{
		Name:        "generate-gitops",
		Usage:       "Generate GitOps configuration files for Fleet.",
		Description: "This command generates GitOps configuration files for Fleet.",
		Action:      createGenerateGitopsAction(nil),
		Flags: []cli.Flag{
			configFlag(),
			contextFlag(),
			debugFlag(),
			&cli.BoolFlag{
				Name:  "insecure",
				Usage: "Output sensitive information in plaintext.",
				Value: false,
			},
			&cli.StringFlag{
				Name:  "key",
				Usage: "A key to output the config value for.",
			},
			&cli.StringFlag{
				Name:  "team",
				Usage: "(Premium only) The team to output configuration for.  Omit to export all configuration.  Use 'global' to export global settings, or 'no-team' to export settings for No Team.",
			},
			&cli.StringFlag{
				Name:  "dir",
				Usage: "The root directory to write the files to.",
			},
			&cli.BoolFlag{
				Name:  "print",
				Usage: "Output to stdout instead of the specified directory.",
			},
			&cli.BoolFlag{
				Name:  "force",
				Usage: "Overwrite existing files.",
			},
		},
	}
}

// Create the action for the generate-gitops command, using a provided fleetClient.
func createGenerateGitopsAction(fleetClient generateGitopsClient) func(*cli.Context) error {
	return func(c *cli.Context) error {
		var err error
		if fleetClient == nil {
			fleetClient, err = clientFromCLI(c)
			if err != nil {
				return err
			}
		}
		cmd := &GenerateGitopsCommand{
			Client:       fleetClient,
			CLI:          c,
			Messages:     Messages{},
			FilesToWrite: make(map[string]interface{}),
			SoftwareList: make(map[uint]Software),
			ScriptList:   make(map[uint]string),
		}
		return cmd.Run()
	}
}

// Execute the actual command.
func (cmd *GenerateGitopsCommand) Run() error {
	// Either "key" or "dir" must be specified.
	if cmd.CLI.String("key") == "" && cmd.CLI.String("dir") == "" {
		fmt.Fprintf(cmd.CLI.App.ErrWriter, "Either --dir or --key must be specified\n")
		return nil
	}
	// But not both.
	if cmd.CLI.String("key") != "" && cmd.CLI.String("dir") != "" {
		fmt.Fprintf(cmd.CLI.App.ErrWriter, "Only one of --dir or --key may be specified\n")
		return nil
	}

	var err error

	// User must be global admin.
	me, err := cmd.Client.Me()
	if err != nil {
		fmt.Fprintf(cmd.CLI.App.ErrWriter, "Error getting user: %s\n", err)
		return ErrGeneric
	}
	if me.GlobalRole != nil && *me.GlobalRole != fleet.RoleAdmin {
		fmt.Fprintf(cmd.CLI.App.ErrWriter, "You are not authorized to run this command.  Please contact your administrator.\n")
		return nil
	}

	// Validate directory is empty (or --force is set).
	if cmd.CLI.String("dir") != "" && !cmd.CLI.Bool("print") {
		dir := cmd.CLI.String("dir")
		_, err := os.Stat(dir)
		if err != nil {
			if !os.IsNotExist(err) {
				fmt.Fprintf(cmd.CLI.App.ErrWriter, "Error checking directory: %s\n", err)
				return ErrGeneric
			}
		} else {
			// Check if the directory is empty.
			entries, err := os.ReadDir(dir)
			if err != nil {
				fmt.Fprintf(cmd.CLI.App.ErrWriter, "Error reading directory: %s\n", err)
				return ErrGeneric
			}
			if len(entries) > 0 && !cmd.CLI.Bool("force") {
				fmt.Fprintf(cmd.CLI.App.ErrWriter, "Directory %s is not empty.  Use --force to overwrite.\n", dir)
				return nil
			}
		}
	}

	fmt.Println("Generating GitOps configuration files...")

	cmd.AppConfig, err = cmd.Client.GetAppConfig()
	if err != nil {
		return err
	}

	// Gather the list of teams to process, which may include some
	// virtual teams (i.e. global and no-team).
	var teamsToProcess []teamToProcess
	globalTeam := teamToProcess{
		ID: nil,
		Team: &fleet.Team{
			Name: "Global",
		},
	}
	noTeam := teamToProcess{
		ID: ptr.Uint(0),
		Team: &fleet.Team{
			ID:   0,
			Name: "No team",
		},
	}
	switch {
	case cmd.CLI.String("team") == "global" || !cmd.AppConfig.License.IsPremium():
		teamsToProcess = []teamToProcess{globalTeam}
	case cmd.CLI.String("team") == "no-team":
		teamsToProcess = []teamToProcess{noTeam}
	default:
		// Get the list of teams.
		teams, err := cmd.Client.ListTeams("")
		if err != nil {
			fmt.Fprintf(cmd.CLI.App.ErrWriter, "Error getting teams: %s\n", err)
			return ErrGeneric
		}
		// If a specific team is requested, find it.
		if cmd.CLI.String("team") != "" {
			transformedSelectedName := generateFilename(cmd.CLI.String("team"))
			for _, team := range teams {
				transformedTeamName := generateFilename(team.Name)
				if transformedSelectedName == transformedTeamName {
					teamsToProcess = []teamToProcess{{
						ID:   &team.ID,
						Team: &team,
					}}
				}
			}
			if len(teamsToProcess) == 0 {
				fmt.Fprintf(cmd.CLI.App.ErrWriter, "Team %s not found\n", cmd.CLI.String("team"))
				return nil
			}
		} else {
			// Otherwise process all teams, including global and no-team.
			teamsToProcess = make([]teamToProcess, len(teams)+2)
			for i, team := range teams {
				teamsToProcess[i] = teamToProcess{
					ID:   &team.ID,
					Team: &team,
				}
			}
			teamsToProcess[len(teams)] = noTeam
			teamsToProcess[len(teams)+1] = globalTeam
		}
	}

	// Iterate over the teams and generate the config files.
	for _, teamToProcess := range teamsToProcess {
		var teamFileName string
		var fileName string
		var team *fleet.Team
		if teamToProcess.ID != nil {
			team = teamToProcess.Team
		}
		// If it's a real team, start the filename with the team name.
		if team != nil {
			teamFileName = generateFilename(team.Name)
			fileName = "teams/" + teamFileName + ".yml"
			cmd.FilesToWrite[fileName] = map[string]interface{}{
				"name": team.Name,
			}
		} else {
			fileName = "default.yml"
		}

		// Set mdm to the global config by default.
		// We'll override this for teams other than no-team.
		mdmConfig := fleet.TeamMDM{
			EnableDiskEncryption: cmd.AppConfig.MDM.EnableDiskEncryption.Value,
			MacOSUpdates:         cmd.AppConfig.MDM.MacOSUpdates,
			IOSUpdates:           cmd.AppConfig.MDM.IOSUpdates,
			IPadOSUpdates:        cmd.AppConfig.MDM.IPadOSUpdates,
			WindowsUpdates:       cmd.AppConfig.MDM.WindowsUpdates,
			MacOSSetup:           cmd.AppConfig.MDM.MacOSSetup,
		}

		if team == nil {
			// Generate org settings, agent options and labels for the global config.
			orgSettings, err := cmd.generateOrgSettings()
			if err != nil {
				fmt.Fprintf(cmd.CLI.App.ErrWriter, "Error generating org settings: %s\n", err)
				return ErrGeneric
			}

			cmd.FilesToWrite["default.yml"] = map[string]interface{}{
				"org_settings": orgSettings,
			}

			cmd.FilesToWrite[fileName].(map[string]interface{})["agent_options"] = cmd.AppConfig.AgentOptions

			// Generate labels.
			labels, err := cmd.generateLabels()
			if err != nil {
				fmt.Fprintf(cmd.CLI.App.ErrWriter, "Error generating labels: %s\n", err)
				return ErrGeneric
			}
			cmd.FilesToWrite[fileName].(map[string]interface{})["labels"] = labels

		} else if team.ID != 0 {
			// Generate team settings and agent options for the team.
			teamSettings, err := cmd.generateTeamSettings(fileName, team)
			if err != nil {
				fmt.Fprintf(cmd.CLI.App.ErrWriter, "Error generating org settings: %s\n", err)
				return ErrGeneric
			}

			cmd.FilesToWrite[fileName].(map[string]interface{})["team_settings"] = teamSettings
			cmd.FilesToWrite[fileName].(map[string]interface{})["agent_options"] = team.Config.AgentOptions

			mdmConfig = team.Config.MDM
		}

		// Generate controls.
		// Only do this on the global team if we're on the free tier.
		if teamToProcess.ID != nil || !cmd.AppConfig.License.IsPremium() {
			controls, err := cmd.generateControls(teamToProcess.ID, teamFileName, &mdmConfig)
			if err != nil {
				fmt.Fprintf(cmd.CLI.App.ErrWriter, "Error generating controls for %s: %s\n", teamFileName, err)
				return ErrGeneric
			}
			cmd.FilesToWrite[fileName].(map[string]interface{})["controls"] = controls
		}

		// Generate software.
		if team != nil {
			software, err := cmd.generateSoftware(fileName, team.ID, teamFileName)
			if err != nil {
				fmt.Fprintf(cmd.CLI.App.ErrWriter, "Error generating software for %s: %s\n", teamFileName, err)
				return ErrGeneric
			}
			if software == nil {
				cmd.FilesToWrite[fileName].(map[string]interface{})["software"] = nil
			} else {
				cmd.FilesToWrite[fileName].(map[string]interface{})["software"] = software
			}
		}

		// Generate policies.
		policies, err := cmd.generatePolicies(teamToProcess.ID, teamFileName)
		if err != nil {
			fmt.Fprintf(cmd.CLI.App.ErrWriter, "Error generating policies for team %s: %s\n", team.Name, err)
			return ErrGeneric
		}
		cmd.FilesToWrite[fileName].(map[string]interface{})["policies"] = policies

		if team == nil || team.ID != 0 {
			// Generate queries (except for on No Team).
			queries, err := cmd.generateQueries(teamToProcess.ID)
			if err != nil {
				fmt.Fprintf(cmd.CLI.App.ErrWriter, "Error generating queries for team %s: %s\n", team.Name, err)
				return ErrGeneric
			}
			cmd.FilesToWrite[fileName].(map[string]interface{})["queries"] = queries
		}
	}

	// If we're just looking to print out a specific key, attempt to do that now.
	if cmd.CLI.String("key") != "" {
		var fileName string
		// If a team is specified, get the file for that team.
		switch cmd.CLI.String("team") {
		case "global":
			fileName = "default.yml"
		case "":
			fileName = "default.yml"
		case "no-team":
			fileName = "teams/no-team.yml"
		default:
			teamFileName := generateFilename(cmd.CLI.String("team"))
			fileName = "teams/" + teamFileName + ".yml"
		}

		// Marshal and ummarshal the data to standardize the keys.
		b, err := yaml.Marshal(cmd.FilesToWrite[fileName])
		if err != nil {
			fmt.Fprintf(cmd.CLI.App.ErrWriter, "Error marshaling settings: %s\n", err)
			return ErrGeneric
		}
		var data map[string]interface{}
		if err := yaml.Unmarshal(b, &data); err != nil {
			fmt.Fprintf(cmd.CLI.App.ErrWriter, "Error unmarshaling settings: %s\n", err)
			return ErrGeneric
		}
		value, ok := getValueAtKey(data, cmd.CLI.String("key"))
		if !ok {
			fmt.Fprintf(cmd.CLI.App.ErrWriter, "Key %s not found in %s\n", cmd.CLI.String("key"), fileName)
			return nil
		}
		b, err = yaml.Marshal(value)
		if err != nil {
			fmt.Fprintf(cmd.CLI.App.ErrWriter, "Error marshaling value: %s\n", err)
			return ErrGeneric
		}
		fmt.Fprintf(cmd.CLI.App.Writer, "%s", string(b))
		return nil
	}

	emptyVal := regexp.MustCompile(`(?m):\s*(null|""|\[\]|\{\})\s*$`)
	// Add comments to the result.
	for path, fileToWrite := range cmd.FilesToWrite {
		fullPath := fmt.Sprintf("%s/%s", cmd.CLI.String("dir"), path)
		var b []byte
		var err error
		// If the filename ends in .yml, marshal it to YAML.
		if strings.HasSuffix(path, ".yml") {
			b, err = yaml.Marshal(fileToWrite)
			if err != nil {
				fmt.Fprintf(cmd.CLI.App.ErrWriter, "Error marshaling file to write: %s\n", err)
				return ErrGeneric
			}
			for _, comment := range cmd.Comments {
				if comment.Filename == path {
					b = bytes.ReplaceAll(b,
						[]byte(comment.Token),
						[]byte("# "+comment.Comment),
					)
				}
			}
			// Replace any empty values with a blank.
			b = emptyVal.ReplaceAll(b, []byte(":"))
		} else {
			b = []byte(fileToWrite.(string))
		}

		// If --print is set, print the file to stdout.
		if cmd.CLI.Bool("print") {
			fmt.Fprintf(cmd.CLI.App.Writer, "------------------------------------------------------------------\n%s\n------------------------------------------------------------------\n\n%+v\n\n", fullPath, string(b))
		} else {
			// Ensure the dir exists
			err = os.MkdirAll(pathUtils.Dir(fullPath), 0o755)
			if err != nil {
				fmt.Fprintf(cmd.CLI.App.ErrWriter, "Error creating dir %s: %s\n\n", fullPath, err)
				return ErrGeneric
			}
			// Write the file to the output directory.
			err = os.WriteFile(fullPath, b, 0o644)
			if err != nil {
				fmt.Fprintf(cmd.CLI.App.ErrWriter, "Error writing file %s: %s\n\n", fullPath, err)
				return ErrGeneric
			}
		}
	}

	fmt.Fprintf(cmd.CLI.App.Writer, "Config generation complete!\n")
	if len(cmd.Messages.SecretWarnings) > 0 {
		fmt.Fprintf(cmd.CLI.App.Writer, "Sensitive information was redacted in the following places, and will need to be replaced:\n")
		for _, secretWarning := range cmd.Messages.SecretWarnings {
			fmt.Fprintf(cmd.CLI.App.Writer, " • %s: %s\n", secretWarning.Filename, secretWarning.Key)
		}
		fmt.Fprintf(cmd.CLI.App.Writer, "\n")
	}

	if cmd.CLI.String("team") == "global" || cmd.CLI.String("team") == "" {
		cmd.Messages.Notes = append(cmd.Messages.Notes, Note{
			Filename: "default.yml",
			Note:     "Warning: YARA rules are not supported by this tool yet. If you have existing YARA rules, add them to the new default.yml file.",
		})
	}

	if cmd.CLI.String("team") != "global" {
		cmd.Messages.Notes = append(cmd.Messages.Notes, Note{
			Note: "Warning: Software categories are not supported by this tool yet. If you have added any categories to software items, add them to the appropriate team .yml file.",
		})
	}

	if len(cmd.Messages.Notes) > 0 {
		fmt.Fprintf(cmd.CLI.App.Writer, "Other notes:\n")
		for _, note := range cmd.Messages.Notes {
			if note.Filename != "" {
				fmt.Fprintf(cmd.CLI.App.Writer, " • %s: %s\n", note.Filename, note.Note)
			} else {
				fmt.Fprintf(cmd.CLI.App.Writer, " • %s\n", note.Note)
			}
		}
	}

	return nil
}

// Add a comment to a file.  The comment is added as a token in the map, which
// is replaced with the comment when the file is written.
func (cmd *GenerateGitopsCommand) AddComment(filename, comment string) string {
	token := fmt.Sprintf("___GITOPS_COMMENT_%d___", len(cmd.Comments))
	cmd.Comments = append(cmd.Comments, Comment{
		Filename: filename,
		Comment:  comment,
		Token:    token,
	})
	return token
}

// Given a name, generate a filename by replacing spaces with dashes and
// removing any non-alphanumeric characters.
func generateFilename(name string) string {
	fileName := strings.Map(func(r rune) rune {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			return unicode.ToLower(r)
		case unicode.IsSpace(r):
			return '-'
		default:
			return -1
		}
	}, name)
	// Strip any leading/trailing dashes using regex.
	fileName = strings.Trim(fileName, "-")
	return fileName
}

var isJSON = regexp.MustCompile(`^\s*\{`)

// Generate a filename for a profile based on its name and contents.
func generateProfileFilename(profile *fleet.MDMConfigProfilePayload, profileContentsString string) string {
	fileName := generateFilename(profile.Name)
	if profile.Platform == "darwin" {
		if isJSON.MatchString(profileContentsString) {
			fileName += ".json"
		} else {
			fileName += ".mobileconfig"
		}
	} else {
		fileName += ".xml"
	}
	return fileName
}

func (cmd *GenerateGitopsCommand) generateOrgSettings() (orgSettings map[string]interface{}, err error) {
	t := reflect.TypeOf(fleet.EnrichedAppConfig{})
	orgSettings = map[string]interface{}{
		jsonFieldName(t, "Features"):           cmd.AppConfig.Features,
		jsonFieldName(t, "FleetDesktop"):       cmd.AppConfig.FleetDesktop,
		jsonFieldName(t, "HostExpirySettings"): cmd.AppConfig.HostExpirySettings,
		jsonFieldName(t, "OrgInfo"):            cmd.AppConfig.OrgInfo,
		jsonFieldName(t, "ServerSettings"):     cmd.AppConfig.ServerSettings,
		jsonFieldName(t, "WebhookSettings"):    cmd.AppConfig.WebhookSettings,
	}
	integrations, err := cmd.generateIntegrations("default.yml", &GlobalOrTeamIntegrations{GlobalIntegrations: &cmd.AppConfig.Integrations})
	if err != nil {
		return nil, err
	}
	orgSettings[jsonFieldName(t, "Integrations")] = integrations
	mdm, err := cmd.generateMDM(&cmd.AppConfig.MDM)
	if err != nil {
		return nil, err
	}
	orgSettings[jsonFieldName(t, "MDM")] = mdm
	yaraRules, err := cmd.generateYaraRules(cmd.AppConfig.YaraRules)
	if err != nil {
		return nil, err
	}
	orgSettings[jsonFieldName(t, "YaraRules")] = yaraRules

	// If --insecure is set, add real secrets.
	if cmd.CLI.Bool("insecure") {
		enrollSecrets, err := cmd.Client.GetEnrollSecretSpec()
		if err != nil {
			return nil, err
		}
		secrets := make([]map[string]string, len(enrollSecrets.Secrets))
		for i, spec := range enrollSecrets.Secrets {
			secrets[i] = map[string]string{"secret": spec.Secret}
		}
		orgSettings["secrets"] = secrets
	} else {
		orgSettings["secrets"] = []map[string]string{{"secret": cmd.AddComment("default.yml", "TODO: Add your enroll secrets here")}}
		cmd.Messages.SecretWarnings = append(cmd.Messages.SecretWarnings, SecretWarning{
			Filename: "default.yml",
			Key:      "org_settings.secrets",
		})
	}

	if (orgSettings)[jsonFieldName(t, "SSOSettings")], err = cmd.generateSSOSettings(cmd.AppConfig.SSOSettings); err != nil {
		return nil, err
	}
	return orgSettings, nil
}

func (cmd *GenerateGitopsCommand) generateSSOSettings(ssoSettings *fleet.SSOSettings) (map[string]interface{}, error) {
	t := reflect.TypeOf(fleet.SSOSettings{})
	result := map[string]interface{}{
		jsonFieldName(t, "EnableSSO"):         ssoSettings.EnableSSO,
		jsonFieldName(t, "IDPName"):           ssoSettings.IDPName,
		jsonFieldName(t, "IDPImageURL"):       ssoSettings.IDPImageURL,
		jsonFieldName(t, "EntityID"):          ssoSettings.EntityID,
		jsonFieldName(t, "Metadata"):          ssoSettings.Metadata,
		jsonFieldName(t, "MetadataURL"):       ssoSettings.MetadataURL,
		jsonFieldName(t, "EnableSSOIdPLogin"): ssoSettings.EnableSSOIdPLogin,
	}
	if cmd.AppConfig.License.IsPremium() {
		result[jsonFieldName(t, "EnableJITProvisioning")] = ssoSettings.EnableJITProvisioning
	}
	if !cmd.CLI.Bool("insecure") {
		if ssoSettings.Metadata != "" {
			result[jsonFieldName(t, "Metadata")] = cmd.AddComment("default.yml", "TODO: Add your SSO metadata here")
			cmd.Messages.SecretWarnings = append(cmd.Messages.SecretWarnings, SecretWarning{
				Filename: "default.yml",
				Key:      "org_settings.sso_settings.metadata",
			})

		}
		if ssoSettings.MetadataURL != "" {
			result[jsonFieldName(t, "MetadataURL")] = cmd.AddComment("default.yml", "TODO: Add your SSO metadata URL here")
			cmd.Messages.SecretWarnings = append(cmd.Messages.SecretWarnings, SecretWarning{
				Filename: "default.yml",
				Key:      "org_settings.sso_settings.metadata_url",
			})
		}
	}
	return result, nil
}

type GlobalOrTeamIntegrations struct {
	GlobalIntegrations *fleet.Integrations     `json:"global_integrations,omitempty"`
	TeamIntegrations   *fleet.TeamIntegrations `json:"team_integrations,omitempty"`
}

func (cmd *GenerateGitopsCommand) generateIntegrations(filePath string, integrations *GlobalOrTeamIntegrations) (map[string]interface{}, error) {
	// Rather than crawling through the whole struct, we'll marshall/unmarshall it
	// to get the keys we want.
	b, err := yaml.Marshal(integrations)
	if err != nil {
		fmt.Fprintf(cmd.CLI.App.ErrWriter, "Error marshaling integrations: %s\n", err)
		return nil, err
	}
	var result map[string]interface{}
	if err := yaml.Unmarshal(b, &result); err != nil {
		fmt.Fprintf(cmd.CLI.App.ErrWriter, "Error unmarshaling integrations: %s\n", err)
		return nil, err
	}
	if result["global_integrations"] != nil {
		result = result["global_integrations"].(map[string]interface{})
	} else {
		result = result["team_integrations"].(map[string]interface{})

		// We currently don't support configuring Jira and Zendesk integrations on the team.
		delete(result, "jira")
		delete(result, "zendesk")

		// Team integrations don't have secrets right now, so just return as-is.
		return result, nil
	}
	// Obfuscate secrets if not in insecure mode.
	if !cmd.CLI.Bool("insecure") {
		if googleCalendar, ok := result["google_calendar"]; ok && googleCalendar != nil {
			for _, intg := range googleCalendar.([]interface{}) {
				if apiKeyJson, ok := intg.(map[string]interface{})["api_key_json"]; ok {
					apiKeyJson.(map[string]interface{})["private_key"] = cmd.AddComment(filePath, "TODO: Add your Google Calendar API key JSON here")
					cmd.Messages.SecretWarnings = append(cmd.Messages.SecretWarnings, SecretWarning{
						Filename: "default.yml",
						Key:      "integrations.google_calendar.api_key_json.private_key",
					})
				}
			}
		}
		if jira, ok := result["jira"]; ok && jira != nil {
			for _, intg := range jira.([]interface{}) {
				intg.(map[string]interface{})["api_token"] = cmd.AddComment(filePath, "TODO: Add your Jira API token here")
				cmd.Messages.SecretWarnings = append(cmd.Messages.SecretWarnings, SecretWarning{
					Filename: "default.yml",
					Key:      "integrations.jira.api_token",
				})
			}
		}
		if zendesk, ok := result["zendesk"]; ok && zendesk != nil {
			for _, intg := range zendesk.([]interface{}) {
				intg.(map[string]interface{})["api_token"] = cmd.AddComment(filePath, "TODO: Add your Zendesk API token here")
				cmd.Messages.SecretWarnings = append(cmd.Messages.SecretWarnings, SecretWarning{
					Filename: "default.yml",
					Key:      "integrations.zendesk.api_token",
				})
			}
		}
		if digicert, ok := result["digicert"]; ok && digicert != nil {
			for _, intg := range digicert.([]interface{}) {
				intg.(map[string]interface{})["api_token"] = cmd.AddComment(filePath, "TODO: Add your Digicert API token here")
				cmd.Messages.SecretWarnings = append(cmd.Messages.SecretWarnings, SecretWarning{
					Filename: "default.yml",
					Key:      "integrations.digicert.api_token",
				})
			}
		}
		if ndes_scep_proxy, ok := result["ndes_scep_proxy"]; ok && ndes_scep_proxy != nil {
			ndes_scep_proxy.(map[string]interface{})["password"] = cmd.AddComment(filePath, "TODO: Add your NDES SCEP proxy password here")
			cmd.Messages.SecretWarnings = append(cmd.Messages.SecretWarnings, SecretWarning{
				Filename: "default.yml",
				Key:      "integrations.ndes_scep_proxy.password",
			})
		}
		if custom_scep_proxy, ok := result["custom_scep_proxy"]; ok && custom_scep_proxy != nil {
			for _, intg := range custom_scep_proxy.([]interface{}) {
				intg.(map[string]interface{})["challenge"] = cmd.AddComment(filePath, "TODO: Add your custom SCEP proxy challenge here")
				cmd.Messages.SecretWarnings = append(cmd.Messages.SecretWarnings, SecretWarning{
					Filename: "default.yml",
					Key:      "integrations.custom_scep_proxy.challenge",
				})
			}
		}
	}

	return result, nil
}

func (cmd *GenerateGitopsCommand) generateEULA() (string, error) {
	// Download the eula metadata for the token.
	eulaMetadata, err := cmd.Client.GetEULAMetadata()
	if err != nil {
		// not found is OK, it means the user has not uploaded a EULA yet.
		if strings.Contains(err.Error(), "Resource Not Found") {
			return "", nil
		}

		fmt.Fprintf(cmd.CLI.App.ErrWriter, "Error getting eula metadata: %s\n", err)
		return "", err
	}

	// now we want the eula contents, which is a PDF.
	eulaContent, err := cmd.Client.GetEULAContent(eulaMetadata.Token)
	if err != nil {
		fmt.Fprintf(cmd.CLI.App.ErrWriter, "Error getting eula contents: %s\n", err)
		return "", err
	}

	fileName := fmt.Sprintf("lib/eula/%s", eulaMetadata.Name)
	cmd.FilesToWrite[fileName] = string(eulaContent)
	path := fmt.Sprintf("./%s", fileName)

	return path, nil
}

// This struct is used to represent the MDM configuration that is used with GitOps.
// It includes an additonal end user license agreement (EULA) field, which is
// not present in the fleet.MDM struct.
type gitopsMDM struct {
	fleet.MDM
	EndUserLicenseAgreement string `json:"end_user_license_agreement,omitempty"`
}

func (cmd *GenerateGitopsCommand) generateMDM(mdm *fleet.MDM) (map[string]interface{}, error) {
	t := reflect.TypeOf(gitopsMDM{})
	result := map[string]interface{}{
		jsonFieldName(t, "AppleServerURL"):        mdm.AppleServerURL,
		jsonFieldName(t, "EndUserAuthentication"): mdm.EndUserAuthentication,
	}
	if cmd.AppConfig.License.IsPremium() {
		result[jsonFieldName(t, "AppleBusinessManager")] = mdm.AppleBusinessManager
		result[jsonFieldName(t, "VolumePurchasingProgram")] = mdm.VolumePurchasingProgram

		eulaPath, err := cmd.generateEULA()
		if err != nil {
			fmt.Fprintf(cmd.CLI.App.ErrWriter, "Error generating EULA: %s\n", err)
			return nil, err
		}
		result[jsonFieldName(t, "EndUserLicenseAgreement")] = eulaPath
	}

	if !cmd.CLI.Bool("insecure") {
		if auth, ok := result[jsonFieldName(t, "EndUserAuthentication")]; ok {
			endUserAuth := auth.(fleet.MDMEndUserAuthentication)
			if endUserAuth.Metadata != "" {
				endUserAuth.Metadata = cmd.AddComment("default.yml", "TODO: Add your MDM end user auth metadata here")
				cmd.Messages.SecretWarnings = append(cmd.Messages.SecretWarnings, SecretWarning{
					Filename: "default.yml",
					Key:      "mdm.end_user_authentication.metadata",
				})
			}
			if endUserAuth.MetadataURL != "" {
				endUserAuth.MetadataURL = cmd.AddComment("default.yml", "TODO: Add your MDM end user auth metadata URL here")
				cmd.Messages.SecretWarnings = append(cmd.Messages.SecretWarnings, SecretWarning{
					Filename: "default.yml",
					Key:      "mdm.end_user_authentication.metadata_url",
				})
			}
			result[jsonFieldName(t, "EndUserAuthentication")] = endUserAuth
		}
	}
	return result, nil
}

func (cmd *GenerateGitopsCommand) generateYaraRules(yaraRules []fleet.YaraRule) (map[string]interface{}, error) {
	// TODC -- come up with a way to export Yara rules.
	return map[string]interface{}{}, nil
}

func (cmd *GenerateGitopsCommand) generateTeamSettings(filePath string, team *fleet.Team) (teamSettings map[string]interface{}, err error) {
	t := reflect.TypeOf(fleet.TeamConfig{})
	teamSettings = map[string]interface{}{
		jsonFieldName(t, "Features"):           team.Config.Features,
		jsonFieldName(t, "HostExpirySettings"): team.Config.HostExpirySettings,
		jsonFieldName(t, "WebhookSettings"):    team.Config.WebhookSettings,
	}
	integrations, err := cmd.generateIntegrations(filePath, &GlobalOrTeamIntegrations{TeamIntegrations: &team.Config.Integrations})
	if err != nil {
		return nil, err
	}
	teamSettings[jsonFieldName(t, "Integrations")] = integrations
	// If --insecure is set, add real secrets.
	if cmd.CLI.Bool("insecure") {
		secrets := make([]map[string]string, len(team.Secrets))
		for i, spec := range team.Secrets {
			secrets[i] = map[string]string{"secret": spec.Secret}
		}
		teamSettings["secrets"] = secrets
	} else {
		teamSettings["secrets"] = []map[string]string{{"secret": cmd.AddComment(filePath, "TODO: Add your enroll secrets here")}}
		cmd.Messages.SecretWarnings = append(cmd.Messages.SecretWarnings, SecretWarning{
			Filename: filePath,
			Key:      "team_settings.secrets",
		})
	}
	return teamSettings, nil
}

func (cmd *GenerateGitopsCommand) generateControls(teamId *uint, teamName string, teamMdm *fleet.TeamMDM) (map[string]interface{}, error) {
	t := reflect.TypeOf(spec.GitOpsControls{})
	result := map[string]interface{}{}

	if teamId != nil {
		scripts, err := cmd.generateScripts(teamId, teamName)
		if err != nil {
			fmt.Fprintf(cmd.CLI.App.ErrWriter, "Error generating scripts: %s\n", err)
			return nil, err
		}
		result[jsonFieldName(t, "Scripts")] = scripts
	}

	profiles, err := cmd.generateProfiles(teamId, teamName)
	if err != nil {
		fmt.Fprintf(cmd.CLI.App.ErrWriter, "Error generating profiles: %s\n", err)
		return nil, err
	}
	if profiles != nil {
		if len(profiles["apple_profiles"].([]map[string]interface{})) > 0 {
			result[jsonFieldName(t, "MacOSSettings")] = map[string]interface{}{
				"custom_settings": profiles["apple_profiles"],
			}
		}
		if len(profiles["windows_profiles"].([]map[string]interface{})) > 0 {
			result[jsonFieldName(t, "WindowsSettings")] = map[string]interface{}{
				"custom_settings": profiles["windows_profiles"],
			}
		}
	}

	if teamMdm != nil && cmd.AppConfig.License.IsPremium() {
		mdmT := reflect.TypeOf(fleet.TeamMDM{})

		result[jsonFieldName(mdmT, "EnableDiskEncryption")] = teamMdm.EnableDiskEncryption
		result[jsonFieldName(mdmT, "MacOSUpdates")] = teamMdm.MacOSUpdates
		result[jsonFieldName(mdmT, "IOSUpdates")] = teamMdm.IOSUpdates
		result[jsonFieldName(mdmT, "IPadOSUpdates")] = teamMdm.IPadOSUpdates
		result[jsonFieldName(mdmT, "WindowsUpdates")] = teamMdm.WindowsUpdates

		if teamId == nil || *teamId == 0 {
			mdmT := reflect.TypeOf(fleet.MDM{})
			result[jsonFieldName(mdmT, "WindowsMigrationEnabled")] = cmd.AppConfig.MDM.WindowsMigrationEnabled
			result[jsonFieldName(mdmT, "MacOSMigration")] = cmd.AppConfig.MDM.MacOSMigration
		}
		if cmd.AppConfig.MDM.WindowsEnabledAndConfigured {
			result["windows_enabled_and_configured"] = cmd.AppConfig.MDM.WindowsEnabledAndConfigured
		}

		// TODO -- add an IsSet() method to MacOSSSetup to encapsulate this logic.
		if teamMdm.MacOSSetup.BootstrapPackage.Value != "" || teamMdm.MacOSSetup.EnableEndUserAuthentication || teamMdm.MacOSSetup.MacOSSetupAssistant.Value != "" || teamMdm.MacOSSetup.Script.Value != "" || (teamMdm.MacOSSetup.Software.Valid && len(teamMdm.MacOSSetup.Software.Value) > 0) {
			result[jsonFieldName(mdmT, "MacOSSetup")] = "TODO: update with your macos_setup configuration"
			cmd.Messages.Notes = append(cmd.Messages.Notes, Note{
				Filename: teamName,
				Note:     "The macos_setup configuration is not supported by this tool yet.  To configure it, please follow the Fleet documentation at https://fleetdm.com/docs/configuration/yaml-files#macos-setup",
			})
		}
	}

	return result, nil
}

func (cmd *GenerateGitopsCommand) generateProfiles(teamId *uint, teamName string) (map[string]interface{}, error) {
	// Get profiles.
	profiles, err := cmd.Client.ListConfigurationProfiles(teamId)
	if err != nil {
		if strings.Contains(err.Error(), fleet.MDMNotConfiguredMessage) {
			return nil, nil
		}

		fmt.Fprintf(cmd.CLI.App.ErrWriter, "Error getting profiles: %v\n", err)
		return nil, err
	}
	if len(profiles) == 0 {
		return nil, nil
	}
	appleProfilesSlice := make([]map[string]interface{}, 0)
	windowsProfilesSlice := make([]map[string]interface{}, 0)
	for _, profile := range profiles {
		profileSpec := map[string]interface{}{}
		// Parse any labels.
		if profile.LabelsIncludeAll != nil {
			labels := make([]string, len(profile.LabelsIncludeAll))
			for i, label := range profile.LabelsIncludeAll {
				labels[i] = label.LabelName
			}
			profileSpec["labels_include_all"] = labels
		}
		if profile.LabelsIncludeAny != nil {
			labels := make([]string, len(profile.LabelsIncludeAny))
			for i, label := range profile.LabelsIncludeAny {
				labels[i] = label.LabelName
			}
			profileSpec["labels_include_any"] = labels
		}
		if profile.LabelsExcludeAny != nil {
			labels := make([]string, len(profile.LabelsExcludeAny))
			for i, label := range profile.LabelsExcludeAny {
				labels[i] = label.LabelName
			}
			profileSpec["labels_exclude_any"] = labels
		}

		// Download the profile contents.
		profileContents, err := cmd.Client.GetProfileContents(profile.ProfileUUID)
		if err != nil {
			fmt.Fprintf(cmd.CLI.App.ErrWriter, "Error getting profile contents: %s\n", err)
			return nil, err
		}
		profileContentsString := string(profileContents)

		fileName := fmt.Sprintf("profiles/%s", generateProfileFilename(profile, profileContentsString))
		if teamId == nil {
			fileName = fmt.Sprintf("lib/%s", fileName)
		} else {
			fileName = fmt.Sprintf("lib/%s/%s", teamName, fileName)
		}

		cmd.FilesToWrite[fileName] = profileContentsString
		var path string
		if teamId == nil {
			path = fmt.Sprintf("./%s", fileName)
		} else {
			path = fmt.Sprintf("../%s", fileName)
		}

		profileSpec["path"] = path

		if profile.Platform == "darwin" {
			appleProfilesSlice = append(appleProfilesSlice, profileSpec)
		} else {
			windowsProfilesSlice = append(windowsProfilesSlice, profileSpec)
		}
	}

	return map[string]interface{}{
		"apple_profiles":   appleProfilesSlice,
		"windows_profiles": windowsProfilesSlice,
	}, nil
}

func (cmd *GenerateGitopsCommand) generateScripts(teamId *uint, teamName string) ([]map[string]interface{}, error) {
	// Get scripts.
	query := ""
	if teamId != nil {
		query = fmt.Sprintf("team_id=%d", *teamId)
	}
	scripts, err := cmd.Client.ListScripts(query)
	if err != nil {
		fmt.Fprintf(cmd.CLI.App.ErrWriter, "Error getting scripts: %s\n", err)
		return nil, err
	}
	if len(scripts) == 0 {
		return nil, nil
	}

	scriptSlice := make([]map[string]interface{}, len(scripts))
	// For each script, get the contents and add a new file for output.
	for i, script := range scripts {
		fileName := fmt.Sprintf("scripts/%s", script.Name)
		if teamId == nil {
			fileName = fmt.Sprintf("lib/%s", fileName)
		} else {
			fileName = fmt.Sprintf("lib/%s/%s", teamName, fileName)
		}
		scriptContents, err := cmd.Client.GetScriptContents(script.ID)
		if err != nil {
			fmt.Fprintf(cmd.CLI.App.ErrWriter, "Error getting script contents: %s\n", err)
			return nil, err
		}
		cmd.FilesToWrite[fileName] = string(scriptContents)
		var path string
		if teamId == nil {
			path = fmt.Sprintf("./%s", fileName)
		} else {
			path = fmt.Sprintf("../%s", fileName)
		}
		scriptSlice[i] = map[string]interface{}{
			"path": path,
		}
		cmd.ScriptList[script.ID] = path
	}
	return scriptSlice, nil
}

func (cmd *GenerateGitopsCommand) generatePolicies(teamId *uint, filePath string) ([]map[string]interface{}, error) {
	policies, err := cmd.Client.GetPolicies(teamId)
	if err != nil {
		fmt.Fprintf(cmd.CLI.App.ErrWriter, "Error getting policies: %s\n", err)
		return nil, err
	}
	if len(policies) == 0 {
		return nil, nil
	}
	t := reflect.TypeOf(fleet.Policy{})
	result := make([]map[string]interface{}, len(policies))
	for i, policy := range policies {
		policySpec := map[string]interface{}{
			jsonFieldName(t, "Name"):                     policy.Name,
			jsonFieldName(t, "Description"):              policy.Description,
			jsonFieldName(t, "Resolution"):               policy.Resolution,
			jsonFieldName(t, "Query"):                    policy.Query,
			jsonFieldName(t, "Platform"):                 policy.Platform,
			jsonFieldName(t, "Critical"):                 policy.Critical,
			jsonFieldName(t, "CalendarEventsEnabled"):    policy.CalendarEventsEnabled,
			jsonFieldName(t, "ConditionalAccessEnabled"): policy.ConditionalAccessEnabled,
		}
		// Handle software automation.
		if policy.InstallSoftware != nil {
			if software, ok := cmd.SoftwareList[policy.InstallSoftware.SoftwareTitleID]; ok {
				policySpec["install_software"] = map[string]interface{}{
					"hash_sha256": software.Hash + " " + software.Comment,
				}
			} else {
				policySpec["install_software"] = map[string]interface{}{
					"hash_sha256": cmd.AddComment(filePath, "TODO: Add your hash_sha256 here"),
				}
				cmd.Messages.Notes = append(cmd.Messages.Notes, Note{
					Filename: filePath,
					Note:     fmt.Sprintf("Warning: policy %s software (install_software) has no hash_sha256.  This is required for GitOps to work.  Please add the hash_sha256 manually.", policy.Name),
				})
			}
		}
		// Handle script automation.
		if policy.RunScript != nil {
			if scriptPath, ok := cmd.ScriptList[policy.RunScript.ID]; ok {
				policySpec["run_script"] = map[string]interface{}{
					"path": scriptPath,
				}
			}
		}
		// Parse any labels.
		if policy.LabelsIncludeAny != nil {
			labels := make([]string, len(policy.LabelsIncludeAny))
			for i, label := range policy.LabelsIncludeAny {
				labels[i] = label.LabelName
			}
			policySpec["labels_include_any"] = labels
		}
		if policy.LabelsExcludeAny != nil {
			labels := make([]string, len(policy.LabelsExcludeAny))
			for i, label := range policy.LabelsExcludeAny {
				labels[i] = label.LabelName
			}
			policySpec["labels_exclude_any"] = labels
		}
		result[i] = policySpec
	}
	return result, nil
}

func (cmd *GenerateGitopsCommand) generateQueries(teamId *uint) ([]map[string]interface{}, error) {
	queries, err := cmd.Client.GetQueries(teamId, nil)
	if err != nil {
		fmt.Fprintf(cmd.CLI.App.ErrWriter, "Error getting queries: %s\n", err)
		return nil, err
	}
	if len(queries) == 0 {
		return nil, nil
	}
	t := reflect.TypeOf(fleet.Query{})
	result := make([]map[string]interface{}, len(queries))
	for i, query := range queries {
		querySpec := map[string]interface{}{
			jsonFieldName(t, "Name"):               query.Name,
			jsonFieldName(t, "Description"):        query.Description,
			jsonFieldName(t, "Query"):              query.Query,
			jsonFieldName(t, "Platform"):           query.Platform,
			jsonFieldName(t, "Interval"):           query.Interval,
			jsonFieldName(t, "ObserverCanRun"):     query.ObserverCanRun,
			jsonFieldName(t, "AutomationsEnabled"): query.AutomationsEnabled,
			jsonFieldName(t, "MinOsqueryVersion"):  query.MinOsqueryVersion,
			jsonFieldName(t, "Logging"):            query.Logging,
			jsonFieldName(t, "DiscardData"):        query.DiscardData,
		}

		// Parse any labels.
		if query.LabelsIncludeAny != nil {
			labels := make([]string, len(query.LabelsIncludeAny))
			for i, label := range query.LabelsIncludeAny {
				labels[i] = label.LabelName
			}
			querySpec["labels_include_any"] = labels
		}

		result[i] = querySpec
	}
	return result, nil
}

func (cmd *GenerateGitopsCommand) generateSoftware(filePath string, teamId uint, teamFilename string) (map[string]interface{}, error) {
	query := fmt.Sprintf("available_for_install=1&team_id=%d", teamId)
	software, err := cmd.Client.ListSoftwareTitles(query)
	if err != nil {
		fmt.Fprintf(cmd.CLI.App.ErrWriter, "Error getting software: %s\n", err)
		return nil, err
	}
	if len(software) == 0 {
		return nil, nil
	}
	result := make(map[string]interface{})
	packages := make([]map[string]interface{}, 0)
	appStoreApps := make([]map[string]interface{}, 0)
	for _, sw := range software {
		softwareSpec := make(map[string]interface{})
		switch {
		case sw.SoftwarePackage != nil:
			pkgName := ""
			if sw.SoftwarePackage.Name != "" {
				pkgName = fmt.Sprintf(" (%s)", sw.SoftwarePackage.Name)
			}
			comment := cmd.AddComment(filePath, fmt.Sprintf("%s%s version %s", sw.Name, pkgName, sw.SoftwarePackage.Version))
			if sw.HashSHA256 == nil {
				cmd.Messages.Notes = append(cmd.Messages.Notes, Note{
					Filename: filePath,
					Note:     fmt.Sprintf("Warning: software %s has no hash_sha256.  This is required for GitOps to work.  Please add it manually.", sw.Name),
				})
				softwareSpec["hash_sha256"] = cmd.AddComment(filePath, "TODO: Add your hash_sha256 here")
			} else {
				softwareSpec["hash_sha256"] = *sw.HashSHA256 + " " + comment
				cmd.SoftwareList[sw.ID] = Software{
					Hash:    *sw.HashSHA256,
					Comment: comment,
				}
			}
		case sw.AppStoreApp != nil:
			softwareSpec["app_store_id"] = sw.AppStoreApp.AppStoreID
		default:
			fmt.Fprintf(cmd.CLI.App.ErrWriter, "Error: software %s has no software package or app store app\n", sw.Name)
			continue
		}

		softwareTitle, err := cmd.Client.GetSoftwareTitleByID(sw.ID, &teamId)
		if err != nil {
			fmt.Fprintf(cmd.CLI.App.ErrWriter, "Error getting software title %s: %s\n", sw.Name, err)
			return nil, err
		}

		if softwareTitle.SoftwarePackage != nil {
			filenamePrefix := generateFilename(sw.Name) + "-" + sw.SoftwarePackage.Platform
			if softwareTitle.SoftwarePackage.InstallScript != "" {
				script := softwareTitle.SoftwarePackage.InstallScript
				fileName := fmt.Sprintf("lib/%s/scripts/%s", teamFilename, filenamePrefix+"-install")
				path := fmt.Sprintf("../%s", fileName)
				softwareSpec["install_script"] = map[string]interface{}{
					"path": path,
				}
				cmd.FilesToWrite[fileName] = script
			}

			if softwareTitle.SoftwarePackage.PostInstallScript != "" {
				script := softwareTitle.SoftwarePackage.PostInstallScript
				fileName := fmt.Sprintf("lib/%s/scripts/%s", teamFilename, filenamePrefix+"-postinstall")
				path := fmt.Sprintf("../%s", fileName)
				softwareSpec["post_install_script"] = map[string]interface{}{
					"path": path,
				}
				cmd.FilesToWrite[fileName] = script
			}

			if softwareTitle.SoftwarePackage.UninstallScript != "" {
				script := softwareTitle.SoftwarePackage.UninstallScript
				fileName := fmt.Sprintf("lib/%s/scripts/%s", teamFilename, filenamePrefix+"-uninstall")
				path := fmt.Sprintf("../%s", fileName)
				softwareSpec["uninstall_script"] = map[string]interface{}{
					"path": path,
				}
				cmd.FilesToWrite[fileName] = script
			}

			if softwareTitle.SoftwarePackage.PreInstallQuery != "" {
				query := softwareTitle.SoftwarePackage.PreInstallQuery
				fileName := fmt.Sprintf("lib/%s/queries/%s", teamFilename, filenamePrefix+"-preinstallquery.yml")
				path := fmt.Sprintf("../%s", fileName)
				softwareSpec["pre_install_query"] = map[string]interface{}{
					"path": path,
				}
				cmd.FilesToWrite[fileName] = []map[string]interface{}{{
					"query": query,
				}}
			}

			if softwareTitle.SoftwarePackage.SelfService {
				softwareSpec["self_service"] = softwareTitle.SoftwarePackage.SelfService
			}

			if softwareTitle.SoftwarePackage.URL != "" {
				softwareSpec["url"] = softwareTitle.SoftwarePackage.URL
			}
		}

		if cmd.AppConfig.License.IsPremium() {
			var labels []fleet.SoftwareScopeLabel
			var labelKey string
			if softwareTitle.SoftwarePackage != nil {
				if len(softwareTitle.SoftwarePackage.LabelsIncludeAny) > 0 {
					labels = softwareTitle.SoftwarePackage.LabelsIncludeAny
					labelKey = "labels_include_any"
				}
				if len(softwareTitle.SoftwarePackage.LabelsExcludeAny) > 0 {
					labels = softwareTitle.SoftwarePackage.LabelsExcludeAny
					labelKey = "labels_exclude_any"
				}
			} else {
				if len(softwareTitle.AppStoreApp.LabelsIncludeAny) > 0 {
					labels = softwareTitle.AppStoreApp.LabelsIncludeAny
					labelKey = "labels_include_any"
				}
				if len(softwareTitle.AppStoreApp.LabelsExcludeAny) > 0 {
					labels = softwareTitle.AppStoreApp.LabelsExcludeAny
					labelKey = "labels_exclude_any"
				}
			}
			if len(labels) > 0 {
				labelsList := make([]string, len(labels))
				for i, label := range labels {
					labelsList[i] = label.LabelName
				}
				softwareSpec[labelKey] = labelsList
			}
		}

		if sw.SoftwarePackage != nil {
			packages = append(packages, softwareSpec)
		} else {
			appStoreApps = append(appStoreApps, softwareSpec)
		}
	}
	if len(packages) > 0 {
		result["packages"] = packages
	}
	if len(appStoreApps) > 0 {
		result["app_store_apps"] = appStoreApps
	}
	// TODO -- add FMA apps to the result. Currently they will be output using hashes.
	return result, nil
}

func (cmd *GenerateGitopsCommand) generateLabels() ([]map[string]interface{}, error) {
	labels, err := cmd.Client.GetLabels()
	if err != nil {
		fmt.Fprintf(cmd.CLI.App.ErrWriter, "Error getting labels: %s\n", err)
		return nil, err
	}
	if len(labels) == 0 {
		return nil, nil
	}
	t := reflect.TypeOf(fleet.LabelSpec{})
	result := make([]map[string]interface{}, 0)
	for _, label := range labels {
		if label.LabelType != fleet.LabelTypeRegular {
			continue
		}
		labelSpec := map[string]interface{}{
			jsonFieldName(t, "Name"):                label.Name,
			jsonFieldName(t, "Description"):         label.Description,
			jsonFieldName(t, "LabelMembershipType"): label.LabelMembershipType,
		}
		if label.Platform != "" {
			labelSpec[jsonFieldName(t, "Platform")] = label.Platform
		}
		switch label.LabelMembershipType {
		case fleet.LabelMembershipTypeManual:
			labelSpec[jsonFieldName(t, "Hosts")] = label.Hosts
		case fleet.LabelMembershipTypeDynamic:
			labelSpec[jsonFieldName(t, "Query")] = label.Query
		case fleet.LabelMembershipTypeHostVitals:
			labelSpec[jsonFieldName(t, "HostVitalsCriteria")] = label.HostVitalsCriteria
		}

		result = append(result, labelSpec)
	}
	return result, nil
}

var _ generateGitopsClient = (*service.Client)(nil)
