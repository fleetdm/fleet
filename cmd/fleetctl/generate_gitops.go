package main

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"unicode"

	"github.com/fleetdm/fleet/v4/server/fleet"
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
type client interface {
	GetAppConfig() (*fleet.EnrichedAppConfig, error)
	GetEnrollSecretSpec() (*fleet.EnrollSecretSpec, error)
	ListTeams(query string) ([]fleet.Team, error)
	ListScripts(query string) ([]*fleet.Script, error)
	GetScriptContents(scriptID uint) ([]byte, error)
}

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
	Client       client
	CLI          *cli.Context
	Messages     Messages
	FilesToWrite map[string]interface{}
	Comments     []Comment
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
		},
	}
}

func createGenerateGitopsAction(fleetClient client) func(*cli.Context) error {
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
		}
		return cmd.Run()
	}
}

func (cmd *GenerateGitopsCommand) Run() error {
	fmt.Println("Generating GitOps configuration files...")

	appConfig, err := cmd.Client.GetAppConfig()
	if err != nil {
		return err
	}

	// Get the list of teams.
	teams, err := cmd.Client.ListTeams("")
	if err != nil {
		fmt.Fprintf(cmd.CLI.App.ErrWriter, "Error getting teams: %s\n", err)
		return ErrGeneric
	}
	// Add in a fake team to use for generating global control and software settings.
	teams = append(teams, fleet.Team{
		ID: 0,
	})
	for _, team := range teams {
		var fileName string
		// If it's a real team, start the filename with the team name.
		if team.ID != 0 {
			fileName = "teams/" + generateTeamFilename(team.Name)
			cmd.FilesToWrite[fileName] = map[string]interface{}{}
		} else {
			fileName = "default.yml"
		}

		// Generate org settings.
		if team.ID == 0 {
			orgSettings, err := cmd.generateOrgSettings(appConfig)
			if err != nil {
				fmt.Fprintf(cmd.CLI.App.ErrWriter, "Error generating org settings: %s\n", err)
				return ErrGeneric
			}

			cmd.FilesToWrite["default.yml"] = map[string]interface{}{
				"org_settings": orgSettings,
			}
		}

		// Generate controls.
		controls, err := cmd.generateControls(fileName, team.ID, team.Name)
		if err != nil {
			fmt.Fprintf(cmd.CLI.App.ErrWriter, "Error generating controls for team %s: %s\n", team.Name, err)
			return ErrGeneric
		}
		cmd.FilesToWrite[fileName].(map[string]interface{})["controls"] = controls

		// Generate software.
		software, err := cmd.generateSoftware(fileName, team.ID)
		if err != nil {
			fmt.Fprintf(cmd.CLI.App.ErrWriter, "Error generating software for team %s: %s\n", team.Name, err)
			return ErrGeneric
		}
		cmd.FilesToWrite[fileName].(map[string]interface{})["software"] = software

		// Generate policies.
		policies, err := cmd.generatePolicies(fileName, team.ID)
		if err != nil {
			fmt.Fprintf(cmd.CLI.App.ErrWriter, "Error generating policies for team %s: %s\n", team.Name, err)
			return ErrGeneric
		}
		cmd.FilesToWrite[fileName].(map[string]interface{})["policies"] = policies

		// Generate queries.
		queries, err := cmd.generateQueries(fileName, team.ID)
		if err != nil {
			fmt.Fprintf(cmd.CLI.App.ErrWriter, "Error generating queries for team %s: %s\n", team.Name, err)
			return ErrGeneric
		}
		cmd.FilesToWrite[fileName].(map[string]interface{})["queries"] = queries

		// Generate agent options.
		agentOptions, err := cmd.generateAgentOptions(fileName, team.ID)
		if err != nil {
			fmt.Fprintf(cmd.CLI.App.ErrWriter, "Error generating agent options for team %s: %s\n", team.Name, err)
			return ErrGeneric
		}
		cmd.FilesToWrite[fileName].(map[string]interface{})["agent_options"] = agentOptions

		if team.ID != 0 {
			// Generate team settings
			teamSettings, err := cmd.generateTeamSettings(fileName, team.ID)
			if err != nil {
				fmt.Fprintf(cmd.CLI.App.ErrWriter, "Error generating team settings for team %s: %s\n", team.Name, err)
				return ErrGeneric
			}
			cmd.FilesToWrite[fileName].(map[string]interface{})["team_settings"] = teamSettings
		}
	}

	if cmd.CLI.String("key") != "" {
		// Marshal and ummarshal the data to standardize the keys.
		b, err := yaml.Marshal(cmd.FilesToWrite["default.yml"])
		if err != nil {
			fmt.Fprintf(cmd.CLI.App.ErrWriter, "Error marshaling org settings: %s\n", err)
			return ErrGeneric
		}
		var data map[string]interface{}
		if err := yaml.Unmarshal(b, &data); err != nil {
			fmt.Fprintf(cmd.CLI.App.ErrWriter, "Error unmarshaling org settings: %s\n", err)
			return ErrGeneric
		}
		value, ok := getValueAtKey(data, cmd.CLI.String("key"))
		if !ok {
			fmt.Fprintf(cmd.CLI.App.ErrWriter, "Key %s not found in org settings\n", cmd.CLI.String("key"))
			return ErrGeneric
		}
		b, err = yaml.Marshal(value)
		if err != nil {
			fmt.Fprintf(cmd.CLI.App.ErrWriter, "Error marshaling value: %s\n", err)
			return ErrGeneric
		}
		fmt.Fprintf(cmd.CLI.App.Writer, "%s", string(b))
		return nil
	}

	// Add comments to the result.
	for path, fileToWrite := range cmd.FilesToWrite {
		// If the filename ends in .yml, marshal it to YAML.
		if strings.HasSuffix(path, ".yml") {
			b, err := yaml.Marshal(fileToWrite)
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
			fmt.Fprintf(cmd.CLI.App.Writer, "%s\n----------------------\n%+v\n", path, string(b))
		} else {
			fmt.Fprintf(cmd.CLI.App.Writer, "%s\n----------------------\n%+v\n", path, fileToWrite)
		}
	}

	return nil
}

func (cmd *GenerateGitopsCommand) AddComment(filename, comment string) string {
	token := fmt.Sprintf("___GITOPS_COMMENT_%d___", len(cmd.Comments))
	cmd.Comments = append(cmd.Comments, Comment{
		Filename: filename,
		Comment:  comment,
		Token:    token,
	})
	return token
}

func generateTeamFilename(teamName string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			return unicode.ToLower(r)
		default:
			return '_'
		}
	}, teamName) + ".yml"
}

func (cmd *GenerateGitopsCommand) generateOrgSettings(appConfig *fleet.EnrichedAppConfig) (orgSettings map[string]interface{}, err error) {
	t := reflect.TypeOf(fleet.EnrichedAppConfig{})
	orgSettings = map[string]interface{}{
		jsonFieldName(t, "Features"):           appConfig.Features,
		jsonFieldName(t, "FleetDesktop"):       appConfig.FleetDesktop,
		jsonFieldName(t, "HostExpirySettings"): appConfig.HostExpirySettings,
		jsonFieldName(t, "OrgInfo"):            appConfig.OrgInfo,
		jsonFieldName(t, "ServerSettings"):     appConfig.ServerSettings,
		jsonFieldName(t, "WebhookSettings"):    appConfig.WebhookSettings,
	}
	integrations, err := cmd.generateIntegrations(&appConfig.Integrations)
	if err != nil {
		return nil, err
	}
	orgSettings[jsonFieldName(t, "Integrations")] = integrations
	mdm, err := cmd.generateMDM(&appConfig.MDM)
	if err != nil {
		return nil, err
	}
	orgSettings[jsonFieldName(t, "MDM")] = mdm
	yaraRules, err := cmd.generateYaraRules(appConfig.YaraRules)
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
		(orgSettings)["secrets"] = []map[string]string{{"string": cmd.AddComment("default.yml", "TODO: Add your enrollment secrets here")}}
	}

	if (orgSettings)[jsonFieldName(t, "SSOSettings")], err = cmd.generateSSOSettings(appConfig.SSOSettings); err != nil {
		return nil, err
	}
	return orgSettings, nil
}

func (cmd *GenerateGitopsCommand) generateSSOSettings(ssoSettings *fleet.SSOSettings) (map[string]interface{}, error) {
	t := reflect.TypeOf(fleet.SSOSettings{})
	result := map[string]interface{}{
		jsonFieldName(t, "EnableSSO"):             ssoSettings.EnableSSO,
		jsonFieldName(t, "IDPName"):               ssoSettings.IDPName,
		jsonFieldName(t, "IDPImageURL"):           ssoSettings.IDPImageURL,
		jsonFieldName(t, "EntityID"):              ssoSettings.EntityID,
		jsonFieldName(t, "Metadata"):              ssoSettings.Metadata,
		jsonFieldName(t, "MetadataURL"):           ssoSettings.MetadataURL,
		jsonFieldName(t, "EnableJITProvisioning"): ssoSettings.EnableJITProvisioning,
		jsonFieldName(t, "EnableSSOIdPLogin"):     ssoSettings.EnableSSOIdPLogin,
	}
	if !cmd.CLI.Bool("insecure") {
		if ssoSettings.Metadata != "" {
			result[jsonFieldName(t, "Metadata")] = cmd.AddComment("default.yml", "TODO: Add your SSO metadata here")
		}
		if ssoSettings.MetadataURL != "" {
			result[jsonFieldName(t, "MetadataURL")] = cmd.AddComment("default.yml", "TODO: Add your SSO metadata URL here")
		}
	}
	return result, nil
}

func (cmd *GenerateGitopsCommand) generateIntegrations(integrations *fleet.Integrations) (map[string]interface{}, error) {
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
	// Obfuscate secrets if not in insecure mode.
	if !cmd.CLI.Bool("insecure") {
		if googleCalendar, ok := result["google_calendar"]; ok {
			for _, intg := range googleCalendar.([]interface{}) {
				if apiKeyJson, ok := intg.(map[string]interface{})["api_key_json"]; ok {
					apiKeyJson.(map[string]interface{})["private_key"] = cmd.AddComment("default.yml", "TODO: Add your Google Calendar API key JSON here")
				}
			}
		}
		if jira, ok := result["jira"]; ok {
			for _, intg := range jira.([]interface{}) {
				intg.(map[string]interface{})["api_token"] = cmd.AddComment("default.yml", "TODO: Add your Jira API token here")
			}
		}
		if zendesk, ok := result["zendesk"]; ok {
			for _, intg := range zendesk.([]interface{}) {
				intg.(map[string]interface{})["api_token"] = cmd.AddComment("default.yml", "TODO: Add your Zendesk API token here")
			}
		}
		if digicert, ok := result["digicert"]; ok && digicert != nil {
			for _, intg := range digicert.([]interface{}) {
				intg.(map[string]interface{})["api_token"] = cmd.AddComment("default.yml", "TODO: Add your Digicert API token here")
			}
		}
		if ndes_scep_proxy, ok := result["ndes_scep_proxy"]; ok && ndes_scep_proxy != nil {
			ndes_scep_proxy.(map[string]interface{})["password"] = cmd.AddComment("default.yml", "TODO: Add your NDES SCEP proxy password here")
		}
		if custom_scep_proxy, ok := result["custom_scep_proxy"]; ok && custom_scep_proxy != nil {
			for _, intg := range custom_scep_proxy.([]interface{}) {
				intg.(map[string]interface{})["challenge"] = cmd.AddComment("default.yml", "TODO: Add your custom SCEP proxy challenge here")
			}
		}
	}

	return result, nil
}

func (cmd *GenerateGitopsCommand) generateMDM(mdm *fleet.MDM) (map[string]interface{}, error) {
	t := reflect.TypeOf(fleet.MDM{})
	result := map[string]interface{}{
		jsonFieldName(t, "AppleBusinessManager"):    mdm.AppleBusinessManager,
		jsonFieldName(t, "VolumePurchasingProgram"): mdm.VolumePurchasingProgram,
		jsonFieldName(t, "AppleServerURL"):          mdm.AppleServerURL,
		jsonFieldName(t, "EndUserAuthentication"):   mdm.EndUserAuthentication,
	}
	if !cmd.CLI.Bool("insecure") {
		if auth, ok := result[jsonFieldName(t, "EndUserAuthentication")]; ok {
			endUserAuth := auth.(fleet.MDMEndUserAuthentication)
			if endUserAuth.Metadata != "" {
				endUserAuth.Metadata = cmd.AddComment("default.yml", "TODO: Add your MDM end user auth metadata here")
			}
			if endUserAuth.MetadataURL != "" {
				endUserAuth.MetadataURL = cmd.AddComment("default.yml", "TODO: Add your MDM end user auth metadata URL here")
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

func (cmd *GenerateGitopsCommand) generateTeamSettings(filePath string, teamID uint) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func (cmd *GenerateGitopsCommand) generateAgentOptions(filePath string, teamId uint) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func (cmd *GenerateGitopsCommand) generateControls(filePath string, teamId uint, teamName string) (map[string]interface{}, error) {
	scripts, err := cmd.Client.ListScripts("")
	if err != nil {
		fmt.Fprintf(cmd.CLI.App.ErrWriter, "Error getting scripts: %s\n", err)
		return nil, err
	}
	// For each script, get the contents and add a new file for output.
	for _, script := range scripts {
		fileName := fmt.Sprintf("scripts/%s.yml", script.Name)
		if teamId == 0 {
			fileName = fmt.Sprintf("lib/%s", fileName)
		} else {
			fileName = fmt.Sprintf("lib/%s/%s", teamName, fileName)
		}
		script, err := cmd.Client.GetScriptContents(scripts[0].ID)
		if err != nil {
			fmt.Fprintf(cmd.CLI.App.ErrWriter, "Error getting script contents: %s\n", err)
			return nil, err
		}
		cmd.FilesToWrite[fileName] = string(script)
	}
	return map[string]interface{}{}, nil
}

func (cmd *GenerateGitopsCommand) generatePolicies(filePath string, teamId uint) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func (cmd *GenerateGitopsCommand) generateQueries(filePath string, teamId uint) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func (cmd *GenerateGitopsCommand) generateSoftware(filePath string, teamId uint) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func (cmd *GenerateGitopsCommand) generateLabels() (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

var _ client = (*service.Client)(nil)
