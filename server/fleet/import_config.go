package fleet

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type ImportConfigService interface {
	// ImportConfig create packs, queries, options etc based on imported
	// osquery configuration.
	ImportConfig(ctx context.Context, cfg *ImportConfig) (*ImportConfigResponse, error)
}

// ImportSection is used to categorize information associated with the import
// of a particular section of an imported osquery configuration file.
type ImportSection string

const (
	OptionsSection    ImportSection = "options"
	PacksSection                    = "packs"
	QueriesSection                  = "queries"
	DecoratorsSection               = "decorators"
	FilePathsSection                = "file_paths"
	YARASigSection                  = "yara_signature_group"
	YARAFileSection                 = "yara_file_group"
)

// WarningType is used to group associated warnings for options, packs etc
// when importing on osquery configuration file.
type WarningType string

const (
	PackDuplicate          WarningType = "duplicate_pack"
	DifferentQuerySameName             = "different_query_same_name"
	OptionAlreadySet                   = "option_already_set"
	OptionReadonly                     = "option_readonly"
	OptionUnknown                      = "option_unknown"
	QueryDuplicate                     = "duplicate_query"
	FIMDuplicate                       = "duplicate_fim"
	YARADuplicate                      = "duplicate_yara"
	Unsupported                        = "unsupported"
)

// ImportStatus contains information pertaining to the import of a section
// of an osquery configuration file.
type ImportStatus struct {
	// Title human readable name of the section of the import file that this
	// status pertains to.
	Title string `json:"title"`
	// ImportCount count of items successfully imported.
	ImportCount int `json:"import_count"`
	// SkipCount count of items that are skipped.  The reasons for the omissions
	// can be found in Warnings.
	SkipCount int `json:"skip_count"`
	// Warnings groups categories of warnings with one or more detail messages.
	Warnings map[WarningType][]string `json:"warnings"`
	// Messages contains an entry for each import attempt.
	Messages []string `json:"messages"`
}

// Warning is used to add a warning message to ImportStatus.
func (is *ImportStatus) Warning(warnType WarningType, fmtMsg string, fmtArgs ...interface{}) {
	is.Warnings[warnType] = append(is.Warnings[warnType], fmt.Sprintf(fmtMsg, fmtArgs...))
}

// Message is used to add a general message to ImportStatus, usually indicating
// what was changed in a successful import.
func (is *ImportStatus) Message(fmtMsg string, args ...interface{}) {
	is.Messages = append(is.Messages, fmt.Sprintf(fmtMsg, args...))
}

// ImportConfigResponse contains information about the import of an osquery
// configuration file.
type ImportConfigResponse struct {
	ImportStatusBySection map[ImportSection]*ImportStatus `json:"import_status"`
}

// Status returns a structure that contains information about the import
// of a particular section of an osquery configuration file.
func (ic *ImportConfigResponse) Status(section ImportSection) (status *ImportStatus) {
	var ok bool
	if status, ok = ic.ImportStatusBySection[section]; !ok {
		status = new(ImportStatus)
		status.Title = strings.Title(string(section))
		status.Warnings = make(map[WarningType][]string)
		ic.ImportStatusBySection[section] = status
	}
	return status
}

const (
	GlobPacks = "*"
	// ImportPackName is a custom pack name used for a pack we create to
	// hold imported scheduled queries.
	ImportPackName = "imported"
)

// QueryDetails represents the query objects used in the packs and the
// schedule section of an osquery configuration.
type QueryDetails struct {
	Query    string           `json:"query"`
	Interval OsQueryConfigInt `json:"interval"`
	// Optional fields
	Removed  *bool             `json:"removed"`
	Platform *string           `json:"platform"`
	Version  *string           `json:"version"`
	Shard    *OsQueryConfigInt `json:"shard"`
	Snapshot *bool             `json:"snapshot"`
}

// PackDetails represents the "packs" section of an osquery configuration
// file.
type PackDetails struct {
	Queries   QueryNameToQueryDetailsMap `json:"queries"`
	Shard     *OsQueryConfigInt          `json:"shard"`
	Version   *string                    `json:"version"`
	Platform  string                     `json:"platform"`
	Discovery []string                   `json:"discovery"`
}

// YARAConfig yara configuration maps keys to lists of files.
// See https://osquery.readthedocs.io/en/stable/deployment/yara/
type YARAConfig struct {
	Signatures map[string][]string `json:"signatures"`
	FilePaths  map[string][]string `json:"file_paths"`
}

// Decorator section of osquery config each section contains rows of decorator
// queries.
type DecoratorConfig struct {
	Load   []string `json:"load"`
	Always []string `json:"always"`
	/*
		Interval maps a string representation of a numeric interval to a set
		of decorator queries.
			{
				"interval": {
					"3600": [
						"SELECT total_seconds FROM uptime;"
					]
				}
			}
	*/
	Interval map[string][]string `json:"interval"`
}

type OptionNameToValueMap map[string]interface{}
type QueryNameToQueryDetailsMap map[string]QueryDetails
type PackNameMap map[string]interface{}
type FIMCategoryToPaths map[string][]string
type PackNameToPackDetails map[string]PackDetails

// ImportConfig is a representation of an Osquery configuration. Osquery
// documentation has further details.
// See https://osquery.readthedocs.io/en/stable/deployment/configuration/
type ImportConfig struct {
	// DryRun if true an import will be attempted, and if successful will be completely rolled back
	DryRun bool
	// Options is a map of option name to a value which can be an int,
	// bool, or string.
	Options OptionNameToValueMap `json:"options"`
	// Schedule is a map of query names to details
	Schedule QueryNameToQueryDetailsMap `json:"schedule"`
	// Packs is a map of pack names to either PackDetails, or a string
	// containing a file path with a pack config. If a string, we expect
	// PackDetails to be stored in ExternalPacks.
	Packs PackNameMap `json:"packs"`
	// FileIntegrityMonitoring file integrity monitoring information.
	// See https://osquery.readthedocs.io/en/stable/deployment/file-integrity-monitoring/
	FileIntegrityMonitoring FIMCategoryToPaths `json:"file_paths"`
	// YARA configuration
	YARA       *YARAConfig      `json:"yara"`
	Decorators *DecoratorConfig `json:"decorators"`
	// ExternalPacks are packs referenced when an item in the Packs map references
	// an external file.  The PackName here must match the PackName in the Packs map.
	ExternalPacks PackNameToPackDetails `json:"-"`
	// GlobPackNames lists pack names that are globbed.
	GlobPackNames []string `json:"glob"`
}

func (ic *ImportConfig) fetchGlobPacks(packs *PackNameToPackDetails) error {
	for _, packName := range ic.GlobPackNames {
		pack, ok := ic.ExternalPacks[packName]
		if !ok {
			return fmt.Errorf("glob pack '%s' details not found", packName)
		}
		(*packs)[packName] = pack
	}
	return nil
}

// CollectPacks consolidates packs, globbed packs and external packs.
func (ic *ImportConfig) CollectPacks() (PackNameToPackDetails, error) {
	result := make(PackNameToPackDetails)
	for packName, packContent := range ic.Packs {
		// special case handling for Globbed packs
		if packName == GlobPacks {
			if err := ic.fetchGlobPacks(&result); err != nil {
				return nil, err
			}
			continue
		}
		// content can either be a file path, in which case we expect to find
		// pack in ExternalPacks, or pack details
		switch content := packContent.(type) {
		case string:
			pack, ok := ic.ExternalPacks[packName]
			if !ok {
				return nil, fmt.Errorf("external pack '%s' details not found", packName)
			}
			result[packName] = pack
		case PackDetails:
			result[packName] = content
		default:
			return nil, errors.New("unexpected pack content")
		}
	}
	return result, nil
}

// OsQueryConfigInt is provided becase integers in the osquery config file may
// be represented as strings in the json. If we know a particular field is
// supposed to be an Integer, we convert from string to int if we can.
type OsQueryConfigInt uint

func (c *OsQueryConfigInt) UnmarshalJSON(b []byte) error {
	stripped := bytes.Trim(b, `"`)
	v, err := strconv.ParseUint(string(stripped), 10, 64)
	if err != nil {
		return err
	}
	*c = OsQueryConfigInt(v)
	return nil
}
