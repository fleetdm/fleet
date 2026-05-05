//go:build windows
// +build windows

package cisaudit

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"sync"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/rs/zerolog/log"
	"golang.org/x/sys/windows"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
	"gopkg.in/ini.v1"
)

var (
	// CIS items commands initialization
	commandsInit sync.Once

	// Map to store command handlers
	commandHandlers map[string]CommandHandler
)

// CommandHandler is a function type that returns the value for a CIS item
type CommandHandler func() (string, error)

// Audit items stores data from auditpol utility
type AuditItem struct {
	Subcategory string
	GUID        string
	NoAuditing  bool
	Success     bool
	Failure     bool
	Raw         string
}

// SeceditData stores data from secedit utility
type SeceditData struct {
	Unicode struct {
		Unicode bool
	}
	SystemAccess struct {
		MinimumPasswordAge           string
		MaximumPasswordAge           string
		MinimumPasswordLength        string
		PasswordComplexity           string
		PasswordHistorySize          string
		LockoutBadCount              string
		ResetLockoutCount            string
		LockoutDuration              string
		RequireLogonToChangePassword string
		ForceLogoffWhenHourExpire    string
		NewAdministratorName         string
		NewGuestName                 string
		ClearTextPassword            string
		LSAAnonymousNameLookup       string
		EnableAdminAccount           string
		EnableGuestAccount           string
	}
	EventAudit struct {
		AuditSystemEvents    string
		AuditLogonEvents     string
		AuditObjectAccess    string
		AuditPrivilegeUse    string
		AuditPolicyChange    string
		AuditAccountManage   string
		AuditProcessTracking string
		AuditDSAccess        string
		AuditAccountLogon    string
	}
	PrivilegeRights struct {
		SeNetworkLogonRight                       string
		SeBackupPrivilege                         string
		SeChangeNotifyPrivilege                   string
		SeSystemtimePrivilege                     string
		SeCreatePagefilePrivilege                 string
		SeDebugPrivilege                          string
		SeRemoteShutdownPrivilege                 string
		SeAuditPrivilege                          string
		SeIncreaseQuotaPrivilege                  string
		SeIncreaseBasePriorityPrivilege           string
		SeLoadDriverPrivilege                     string
		SeDenyBatchLogonRight                     string
		SeDenyServiceLogonRight                   string
		SeBatchLogonRight                         string
		SeServiceLogonRight                       string
		SeInteractiveLogonRight                   string
		SeSecurityPrivilege                       string
		SeSystemEnvironmentPrivilege              string
		SeProfileSingleProcessPrivilege           string
		SeSystemProfilePrivilege                  string
		SeAssignPrimaryTokenPrivilege             string
		SeRestorePrivilege                        string
		SeShutdownPrivilege                       string
		SeTakeOwnershipPrivilege                  string
		SeDenyNetworkLogonRight                   string
		SeDenyInteractiveLogonRight               string
		SeUndockPrivilege                         string
		SeManageVolumePrivilege                   string
		SeRemoteInteractiveLogonRight             string
		SeImpersonatePrivilege                    string
		SeCreateGlobalPrivilege                   string
		SeIncreaseWorkingSetPrivilege             string
		SeTimeZonePrivilege                       string
		SeCreateSymbolicLinkPrivilege             string
		SeDelegateSessionUserImpersonatePrivilege string
	}
}

// Columns is the schema of the table
func Columns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.TextColumn("item"),
		table.TextColumn("value"),
	}
}

// Generate is called to return the results for the table at query time.
// Constraints for generating can be retrieved from the queryContext.
func Generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	// input item query constraint
	var inputItem string

	// item output value
	var inputValue string

	// error handling
	var err error

	// one-time commands handlers initialization
	registerCommandsHandlers()

	// checking if 'item' is in the where clause
	if constraintList, present := queryContext.Constraints["item"]; present {
		for _, constraint := range constraintList.Constraints {
			if constraint.Operator == table.OperatorEquals {
				inputItem = constraint.Expression // this input as to be kept as-is and returned on the same input column due to a sqlite requirement
				log.Debug().Msgf("cis_audit input item requested: %s\n", inputItem)
			}
		}
	}

	// Getting the input value if supported
	if len(inputItem) > 0 {
		inputValue, err = getValueCisItem(inputItem)
		if err != nil {
			return nil, err
		}
	}

	// returning item and its value
	return []map[string]string{
		{
			"item":  inputItem,
			"value": inputValue,
		},
	}, nil
}

// getPreProcessedFileContent returns an UTF-16 byte array
// This is useful when reading data from MS-Windows systems that generate UTF-16BE files
func getPreProcessedFileContent(path string) ([]byte, error) {
	// Read the file into a []byte:
	raw, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// make an tranformer that converts MS-Win default to UTF8
	win16be := unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM)

	// make a transformer that is like win16be, but abides by BOM
	utf16bom := unicode.BOMOverride(win16be.NewDecoder())

	// make a Reader that uses utf16bom
	unicodeReader := transform.NewReader(bytes.NewReader(raw), utf16bom)

	// decode and return data
	decoded, err := ioutil.ReadAll(unicodeReader)
	if err != nil {
		return []byte(""), err
	}

	// replace newlines with unix style
	fileContent := strings.ReplaceAll(string(decoded), "\r\n", "\n")

	return []byte(fileContent), nil
}

// getSystem32Dir returns the path to the "system32" directory on Windows
func getSystem32Dir() (string, error) {
	system32Path, err := windows.GetSystemDirectory()
	if err != nil {
		return "", err
	}

	return system32Path, nil
}

// getSeceditData returns data from the "secedit.exe" utility
func getSeceditData() (SeceditData, error) {
	var data SeceditData

	// Get the path to the system32 directory
	system32Dir, err := getSystem32Dir()
	if err != nil {
		return data, fmt.Errorf("path to system32 could not be determined: %w", err)
	}

	// Build the fullpath to the "secedit.exe" executable
	seceditPath := filepath.Join(system32Dir, "secedit.exe")

	// Get temporary directory
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return data, fmt.Errorf("get UserCacheDir failed: %s", err)
	}

	// Create temporary directory
	tempDir, err := ioutil.TempDir(cacheDir, "secedit-")
	if err != nil {
		return data, fmt.Errorf("failed to create temporary directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Execute "secedit.exe" to export the current security configuration
	outputInfPath := filepath.Join(tempDir, "output.inf")
	cmd := exec.Command(seceditPath, "/export", "/cfg", outputInfPath)
	if err := cmd.Run(); err != nil {
		return data, fmt.Errorf("failed to execute secedit.exe: %w", err)
	}

	// Read the exported file
	fileContent, err := getPreProcessedFileContent(outputInfPath)
	if err != nil {
		return data, fmt.Errorf("failed to preprocess .inf file: %w", err)
	}

	// Load the .inf file content
	cfg, err := ini.Load(fileContent)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return data, err
	}

	// Parse System Access section
	if systemAccessSection := cfg.Section("System Access"); systemAccessSection != nil {
		data.SystemAccess.MinimumPasswordAge = systemAccessSection.Key("MinimumPasswordAge").String()
		data.SystemAccess.MaximumPasswordAge = systemAccessSection.Key("MaximumPasswordAge").String()
		data.SystemAccess.MinimumPasswordLength = systemAccessSection.Key("MinimumPasswordLength").String()
		data.SystemAccess.PasswordComplexity = systemAccessSection.Key("PasswordComplexity").String()
		data.SystemAccess.PasswordHistorySize = systemAccessSection.Key("PasswordHistorySize").String()
		data.SystemAccess.LockoutBadCount = systemAccessSection.Key("LockoutBadCount").String()
		data.SystemAccess.ResetLockoutCount = systemAccessSection.Key("ResetLockoutCount").String()
		data.SystemAccess.LockoutDuration = systemAccessSection.Key("LockoutDuration").String()
		data.SystemAccess.RequireLogonToChangePassword = systemAccessSection.Key("RequireLogonToChangePassword").String()
		data.SystemAccess.ForceLogoffWhenHourExpire = systemAccessSection.Key("ForceLogoffWhenHourExpire").String()
		data.SystemAccess.NewAdministratorName = systemAccessSection.Key("NewAdministratorName").String()
		data.SystemAccess.NewGuestName = systemAccessSection.Key("NewGuestName").String()
		data.SystemAccess.ClearTextPassword = systemAccessSection.Key("ClearTextPassword").String()
		data.SystemAccess.LSAAnonymousNameLookup = systemAccessSection.Key("LSAAnonymousNameLookup").String()
		data.SystemAccess.EnableAdminAccount = systemAccessSection.Key("EnableAdminAccount").String()
		data.SystemAccess.EnableGuestAccount = systemAccessSection.Key("EnableGuestAccount").String()
	}

	// Parse Event Audit section
	if eventAuditSection := cfg.Section("Event Audit"); eventAuditSection != nil {
		data.EventAudit.AuditSystemEvents = eventAuditSection.Key("AuditSystemEvents").String()
		data.EventAudit.AuditLogonEvents = eventAuditSection.Key("AuditLogonEvents").String()
		data.EventAudit.AuditObjectAccess = eventAuditSection.Key("AuditObjectAccess").String()
		data.EventAudit.AuditPrivilegeUse = eventAuditSection.Key("AuditPrivilegeUse").String()
		data.EventAudit.AuditPolicyChange = eventAuditSection.Key("AuditPolicyChange").String()
		data.EventAudit.AuditAccountManage = eventAuditSection.Key("AuditAccountManage").String()
		data.EventAudit.AuditProcessTracking = eventAuditSection.Key("AuditProcessTracking").String()
		data.EventAudit.AuditDSAccess = eventAuditSection.Key("AuditDSAccess").String()
		data.EventAudit.AuditAccountLogon = eventAuditSection.Key("AuditAccountLogon").String()
	}

	// Parse Privilege Rights section
	if privilegeRightsSection := cfg.Section("Privilege Rights"); privilegeRightsSection != nil {
		data.PrivilegeRights.SeNetworkLogonRight = getGroupNames(privilegeRightsSection.Key("SeNetworkLogonRight").String())
		data.PrivilegeRights.SeBackupPrivilege = getGroupNames(privilegeRightsSection.Key("SeBackupPrivilege").String())
		data.PrivilegeRights.SeChangeNotifyPrivilege = getGroupNames(privilegeRightsSection.Key("SeChangeNotifyPrivilege").String())
		data.PrivilegeRights.SeSystemtimePrivilege = getGroupNames(privilegeRightsSection.Key("SeSystemtimePrivilege").String())
		data.PrivilegeRights.SeCreatePagefilePrivilege = getGroupNames(privilegeRightsSection.Key("SeCreatePagefilePrivilege").String())
		data.PrivilegeRights.SeDebugPrivilege = getGroupNames(privilegeRightsSection.Key("SeDebugPrivilege").String())
		data.PrivilegeRights.SeRemoteShutdownPrivilege = getGroupNames(privilegeRightsSection.Key("SeRemoteShutdownPrivilege").String())
		data.PrivilegeRights.SeAuditPrivilege = getGroupNames(privilegeRightsSection.Key("SeAuditPrivilege").String())
		data.PrivilegeRights.SeIncreaseQuotaPrivilege = getGroupNames(privilegeRightsSection.Key("SeIncreaseQuotaPrivilege").String())
		data.PrivilegeRights.SeIncreaseBasePriorityPrivilege = getGroupNames(privilegeRightsSection.Key("SeIncreaseBasePriorityPrivilege").String())
		data.PrivilegeRights.SeLoadDriverPrivilege = getGroupNames(privilegeRightsSection.Key("SeLoadDriverPrivilege").String())
		data.PrivilegeRights.SeDenyBatchLogonRight = getGroupNames(privilegeRightsSection.Key("SeDenyBatchLogonRight").String())
		data.PrivilegeRights.SeDenyServiceLogonRight = getGroupNames(privilegeRightsSection.Key("SeDenyServiceLogonRight").String())
		data.PrivilegeRights.SeBatchLogonRight = getGroupNames(privilegeRightsSection.Key("SeBatchLogonRight").String())
		data.PrivilegeRights.SeServiceLogonRight = getGroupNames(privilegeRightsSection.Key("SeServiceLogonRight").String())
		data.PrivilegeRights.SeInteractiveLogonRight = getGroupNames(privilegeRightsSection.Key("SeInteractiveLogonRight").String())
		data.PrivilegeRights.SeSecurityPrivilege = getGroupNames(privilegeRightsSection.Key("SeSecurityPrivilege").String())
		data.PrivilegeRights.SeSystemEnvironmentPrivilege = getGroupNames(privilegeRightsSection.Key("SeSystemEnvironmentPrivilege").String())
		data.PrivilegeRights.SeProfileSingleProcessPrivilege = getGroupNames(privilegeRightsSection.Key("SeProfileSingleProcessPrivilege").String())
		data.PrivilegeRights.SeSystemProfilePrivilege = getGroupNames(privilegeRightsSection.Key("SeSystemProfilePrivilege").String())
		data.PrivilegeRights.SeAssignPrimaryTokenPrivilege = getGroupNames(privilegeRightsSection.Key("SeAssignPrimaryTokenPrivilege").String())
		data.PrivilegeRights.SeRestorePrivilege = getGroupNames(privilegeRightsSection.Key("SeRestorePrivilege").String())
		data.PrivilegeRights.SeShutdownPrivilege = getGroupNames(privilegeRightsSection.Key("SeShutdownPrivilege").String())
		data.PrivilegeRights.SeTakeOwnershipPrivilege = getGroupNames(privilegeRightsSection.Key("SeTakeOwnershipPrivilege").String())
		data.PrivilegeRights.SeDenyNetworkLogonRight = getGroupNames(privilegeRightsSection.Key("SeDenyNetworkLogonRight").String())
		data.PrivilegeRights.SeDenyInteractiveLogonRight = getGroupNames(privilegeRightsSection.Key("SeDenyInteractiveLogonRight").String())
		data.PrivilegeRights.SeUndockPrivilege = getGroupNames(privilegeRightsSection.Key("SeUndockPrivilege").String())
		data.PrivilegeRights.SeManageVolumePrivilege = getGroupNames(privilegeRightsSection.Key("SeManageVolumePrivilege").String())
		data.PrivilegeRights.SeRemoteInteractiveLogonRight = getGroupNames(privilegeRightsSection.Key("SeRemoteInteractiveLogonRight").String())
		data.PrivilegeRights.SeImpersonatePrivilege = getGroupNames(privilegeRightsSection.Key("SeImpersonatePrivilege").String())
		data.PrivilegeRights.SeCreateGlobalPrivilege = getGroupNames(privilegeRightsSection.Key("SeCreateGlobalPrivilege").String())
		data.PrivilegeRights.SeIncreaseWorkingSetPrivilege = getGroupNames(privilegeRightsSection.Key("SeIncreaseWorkingSetPrivilege").String())
		data.PrivilegeRights.SeTimeZonePrivilege = getGroupNames(privilegeRightsSection.Key("SeTimeZonePrivilege").String())
		data.PrivilegeRights.SeCreateSymbolicLinkPrivilege = getGroupNames(privilegeRightsSection.Key("SeCreateSymbolicLinkPrivilege").String())
		data.PrivilegeRights.SeDelegateSessionUserImpersonatePrivilege = getGroupNames(privilegeRightsSection.Key("SeDelegateSessionUserImpersonatePrivilege").String())
	}
	return data, nil
}

// Best effor helper to extract the group names from an input string
func getGroupNames(input string) string {
	var output string

	// remove global occurences of * character
	input = strings.ReplaceAll(input, "*", "")

	// split input by comma
	groups := strings.Split(input, ",")
	for _, group := range groups {
		userGroup, _ := user.LookupGroupId(group)
		if userGroup != nil && len(userGroup.Name) > 0 {
			output += userGroup.Name + ","
		} else {
			output += group + ","
		}
	}

	return output
}

// containsAny checks if any of the given substrings are present in the input string
// It returns true if at least one substring is found, otherwise it returns false
func containsAny(input string, substrings []string) bool {
	for _, substring := range substrings {
		if strings.Contains(input, substring) {
			return true
		}
	}
	return false
}

// contains checks if the given substring is present in the input string.
// It returns true if the substring is found, otherwise it returns false.
func contains(input string, substring string) bool {
	return containsAny(input, []string{substring})
}

// ParseAuditOutput parses the output of the auditpol.exe command
func parseAuditOutput(input string) ([]AuditItem, error) {
	// expected items per line
	const expectedItemsPerLine = 6

	// parse the CSV string into a slice of AuditItem structs
	reader := csv.NewReader(strings.NewReader(input))
	reader.FieldsPerRecord = expectedItemsPerLine

	// read all lines
	lines, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	// parse the CSV lines into AuditItem structs
	var auditItems []AuditItem
	for i, line := range lines {

		// Check if the line has the expected number of items
		if len(line) < expectedItemsPerLine {
			return nil, fmt.Errorf("invalid line at index %d", i)
		}

		// Skip header
		if i == 0 {
			continue
		}

		// Parse the line
		item := AuditItem{
			Subcategory: line[2],
			GUID:        line[3],
			NoAuditing:  contains(line[4], "No Auditing"),
			Success:     contains(line[4], "Success"),
			Failure:     contains(line[4], "Failure"),
			Raw:         line[4],
		}

		auditItems = append(auditItems, item)
	}

	return auditItems, nil
}

// getAuditItems returns a slice of AuditItem structs
func getAuditItems() ([]AuditItem, error) {
	// Get the path to the "system32" directory
	system32Dir, err := getSystem32Dir()
	if err != nil {
		return nil, fmt.Errorf("path to system32 could not be determined: %w", err)
	}

	// Build the fullpath to the "auditpol.exe" executable
	auditpolPath := filepath.Join(system32Dir, "auditpol.exe")

	cmd := exec.Command(auditpolPath, "/get", "/category:*", "/r")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute the auditpol command
	err = cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("command execution failed: %v, %s", err, stderr.String())
	}

	// Parse the output
	auditItems, err := parseAuditOutput(stdout.String())
	if err != nil {
		return nil, fmt.Errorf("parsing output failed: %v", err)
	}

	return auditItems, nil
}

// Register the CIS items command handlers
func registerCommandsHandlers() {
	// initialize the commands handlers map
	commandsInit.Do(func() {
		commandHandlers = make(map[string]CommandHandler)

		registerCommandHandler("1.2.1", handler_cis_1_2_1)
		registerCommandHandler("1.2.2", handler_cis_1_2_2)
		registerCommandHandler("1.2.3", handler_cis_1_2_3)
		registerCommandHandler("2.2.4", handler_cis_2_2_4)
		registerCommandHandler("2.2.6", handler_cis_2_2_6)
		registerCommandHandler("2.2.9", handler_cis_2_2_9)
		registerCommandHandler("2.2.17", handler_cis_2_2_17)
		registerCommandHandler("2.2.18", handler_cis_2_2_18)
		registerCommandHandler("2.2.28", handler_cis_2_2_28)
		registerCommandHandler("2.2.29", handler_cis_2_2_29)
		registerCommandHandler("2.2.33", handler_cis_2_2_33)
		registerCommandHandler("2.2.35", handler_cis_2_2_35)
		registerCommandHandler("2.2.36", handler_cis_2_2_36)
		registerCommandHandler("2.2.38", handler_cis_2_2_38)
		registerCommandHandler("2.3.10.1", handler_cis_2_3_10_1)
		registerCommandHandler("2.3.11.6", handler_cis_2_3_11_6)
		registerCommandHandler("17.5.1", handler_cis_17_5_1)
		registerCommandHandler("17.5.2", handler_cis_17_5_2)
		registerCommandHandler("17.5.3", handler_cis_17_5_3)
		registerCommandHandler("17.5.4", handler_cis_17_5_4)
		registerCommandHandler("17.5.5", handler_cis_17_5_5)
		registerCommandHandler("17.5.6", handler_cis_17_5_6)
	})
}

// registerCommandHandler registers a new command handler for the given command
func registerCommandHandler(command string, handler CommandHandler) {
	commandHandlers[command] = handler
}

// Helper to access the command handlers map
func getValueCisItem(item string) (string, error) {
	var output string
	var err error

	if handler, exists := commandHandlers[item]; exists {
		output, err = handler()
		if err != nil {
			return "", fmt.Errorf("cis command handler err: %v", err)
		}
	}

	return output, nil
}

// getAuditItem helps to access audit array
func getAuditItem(subcategory string) (string, error) {
	var output string

	// Getting audit items
	items, err := getAuditItems()
	if err != nil {
		return "", err
	}

	// Find the item and save raw content if present
	for _, item := range items {
		if item.Subcategory == subcategory {
			output = item.Raw
			break
		}
	}

	return output, nil
}

// Command handler for CIS item 1.2.1
func handler_cis_1_2_1() (string, error) {
	data, err := getSeceditData()
	if err != nil {
		return "", err
	}

	return data.SystemAccess.LockoutDuration, nil
}

// Command handler for CIS item 1.2.2
func handler_cis_1_2_2() (string, error) {
	data, err := getSeceditData()
	if err != nil {
		return "", err
	}
	return data.SystemAccess.LockoutBadCount, nil
}

// Command handler for CIS item 1.2.3
func handler_cis_1_2_3() (string, error) {
	data, err := getSeceditData()
	if err != nil {
		return "", err
	}
	return data.SystemAccess.ResetLockoutCount, nil
}

// Command handler for CIS item 2.2.4
func handler_cis_2_2_4() (string, error) {
	data, err := getSeceditData()
	if err != nil {
		return "", err
	}
	return data.PrivilegeRights.SeIncreaseQuotaPrivilege, nil
}

// Command handler for CIS item 2.2.6
func handler_cis_2_2_6() (string, error) {
	data, err := getSeceditData()
	if err != nil {
		return "", err
	}
	return data.PrivilegeRights.SeRemoteInteractiveLogonRight, nil
}

// Command handler for CIS item 2.2.9
func handler_cis_2_2_9() (string, error) {
	data, err := getSeceditData()
	if err != nil {
		return "", err
	}
	return data.PrivilegeRights.SeTimeZonePrivilege, nil
}

// Command handler for CIS item 2.2.17
func handler_cis_2_2_17() (string, error) {
	data, err := getSeceditData()
	if err != nil {
		return "", err
	}
	return data.PrivilegeRights.SeDenyBatchLogonRight, nil
}

// Command handler for CIS item 2.2.18
func handler_cis_2_2_18() (string, error) {
	data, err := getSeceditData()
	if err != nil {
		return "", err
	}
	return data.PrivilegeRights.SeDenyServiceLogonRight, nil
}

// Command handler for CIS item 2.2.28
func handler_cis_2_2_28() (string, error) {
	data, err := getSeceditData()
	if err != nil {
		return "", err
	}
	return data.PrivilegeRights.SeBatchLogonRight, nil
}

// Command handler for CIS item 2.2.29
func handler_cis_2_2_29() (string, error) {
	data, err := getSeceditData()
	if err != nil {
		return "", err
	}
	return data.PrivilegeRights.SeServiceLogonRight, nil
}

// Command handler for CIS item 2.2.33
func handler_cis_2_2_33() (string, error) {
	data, err := getSeceditData()
	if err != nil {
		return "", err
	}
	return data.PrivilegeRights.SeManageVolumePrivilege, nil
}

// Command handler for CIS item 2.2.35
func handler_cis_2_2_35() (string, error) {
	data, err := getSeceditData()
	if err != nil {
		return "", err
	}
	return data.PrivilegeRights.SeSystemProfilePrivilege, nil
}

// Command handler for CIS item 2.2.36
func handler_cis_2_2_36() (string, error) {
	data, err := getSeceditData()
	if err != nil {
		return "", err
	}
	return data.PrivilegeRights.SeAssignPrimaryTokenPrivilege, nil
}

// Command handler for CIS item 2.2.38
func handler_cis_2_2_38() (string, error) {
	data, err := getSeceditData()
	if err != nil {
		return "", err
	}
	return data.PrivilegeRights.SeShutdownPrivilege, nil
}

// Command handler for CIS item 2.3.10.1
func handler_cis_2_3_10_1() (string, error) {
	data, err := getSeceditData()
	if err != nil {
		return "", err
	}
	return data.SystemAccess.LSAAnonymousNameLookup, nil
}

// Command handler for CIS item 2.3.11.6
func handler_cis_2_3_11_6() (string, error) {
	data, err := getSeceditData()
	if err != nil {
		return "", err
	}
	return data.SystemAccess.ForceLogoffWhenHourExpire, nil
}

// Command handler for CIS item 17.5.1
func handler_cis_17_5_1() (string, error) {
	output, err := getAuditItem("Account Lockout")
	if err != nil {
		return "", err
	}

	return output, nil
}

// Command handler for CIS item 17.5.2
func handler_cis_17_5_2() (string, error) {
	output, err := getAuditItem("Group Membership")
	if err != nil {
		return "", err
	}

	return output, nil
}

// Command handler for CIS item 17.5.3
func handler_cis_17_5_3() (string, error) {
	output, err := getAuditItem("Logoff")
	if err != nil {
		return "", err
	}

	return output, nil
}

// Command handler for CIS item 17.5.4
func handler_cis_17_5_4() (string, error) {
	output, err := getAuditItem("Logon")
	if err != nil {
		return "", err
	}

	return output, nil
}

// Command handler for CIS item 17.5.5
func handler_cis_17_5_5() (string, error) {
	output, err := getAuditItem("Other Logon/Logoff Events")
	if err != nil {
		return "", err
	}

	return output, nil
}

// Command handler for CIS item 17.5.6
func handler_cis_17_5_6() (string, error) {
	output, err := getAuditItem("Special Logon")
	if err != nil {
		return "", err
	}

	return output, nil
}
