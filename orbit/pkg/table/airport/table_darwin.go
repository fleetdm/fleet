//go:build darwin
// +build darwin

// based on github.com/kolide/launcher/pkg/osquery/tables
package airport

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/fleetdm/fleet/v4/orbit/pkg/dataflatten"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/dataflattentable"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/tablehelpers"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/osquery/osquery-go/plugin/table"
)

var (
	allowedOptions = []string{"getinfo", "scan"}
	airportPaths   = []string{"/System/Library/PrivateFrameworks/Apple80211.framework/Versions/Current/Resources/airport"}
)

type Table struct {
	name   string
	logger log.Logger
}

const tableName = "airport_util"

func TablePlugin(logger log.Logger) *table.Plugin {
	columns := dataflattentable.Columns(
		table.TextColumn("option"),
	)

	t := &Table{
		name:   tableName,
		logger: logger,
	}

	return table.NewPlugin(t.name, columns, t.generate)
}

type airportExecutor struct {
	ctx    context.Context
	logger log.Logger
	paths  []string
}

func (a *airportExecutor) Exec(option string) ([]byte, error) {
	return tablehelpers.Exec(a.ctx, a.logger, 30, airportPaths, []string{"--" + option}, false)
}

type executor interface {
	Exec(string) ([]byte, error)
}

func (t *Table) generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	airportExecutor := &airportExecutor{
		ctx:    ctx,
		logger: t.logger,
		paths:  airportPaths,
	}

	return generateAirportData(queryContext, airportExecutor, t.logger)
}

func generateAirportData(queryContext table.QueryContext, airportExecutor executor, logger log.Logger) ([]map[string]string, error) {
	options := tablehelpers.GetConstraints(queryContext, "option", tablehelpers.WithAllowedValues(allowedOptions))

	if len(options) == 0 {
		return nil, fmt.Errorf("The %s table requires that you specify a constraint for WHERE option. Valid values for option are (%s).", tableName, strings.Join(allowedOptions, ", "))
	}

	var results []map[string]string
	for _, option := range options {
		airportOutput, err := airportExecutor.Exec(option)
		if err != nil {
			level.Debug(logger).Log("msg", "Error execing airport", "option", option, "err", err)
			continue
		}

		optionResult, err := processAirportOutput(bytes.NewReader(airportOutput), option, queryContext, logger)
		if err != nil {
			level.Debug(logger).Log("msg", "Error processing airport output", "option", option, "err", err)
			continue
		}
		results = append(results, optionResult...)
	}

	return results, nil
}

func processAirportOutput(airportOutput io.Reader, option string, queryContext table.QueryContext, logger log.Logger) ([]map[string]string, error) {
	var results []map[string]string

	var unmarshalledOutput []map[string]interface{}

	rowData := map[string]string{"option": option}

	switch option {
	case "getinfo":
		unmarshalledOutput = []map[string]interface{}{unmarshallGetInfoOutput(airportOutput)}
	case "scan":
		unmarshalledOutput = unmarshallScanOuput(airportOutput)
	default:
		return nil, fmt.Errorf("unsupported option %s", option)
	}

	for _, dataQuery := range tablehelpers.GetConstraints(queryContext, "query", tablehelpers.WithDefaults("*")) {

		flattened, err := dataflatten.Flatten(unmarshalledOutput, dataflatten.WithLogger(logger), dataflatten.WithQuery(strings.Split(dataQuery, "/")))
		if err != nil {
			return nil, err
		}
		results = append(results, dataflattentable.ToMap(flattened, dataQuery, rowData)...)
	}

	return results, nil
}

// unmarshallGetInfoOutput parses the output of the airport getinfo command
func unmarshallGetInfoOutput(reader io.Reader) map[string]interface{} {
	/* example output:

	    agrCtlRSSI: -55
	    agrExtRSSI: 0
	   agrCtlNoise: -94
	   agrExtNoise: 0
	         state: running
	       op mode: station
	       ...
	*/
	scanner := bufio.NewScanner(reader)
	scanner.Split(bufio.ScanLines)

	results := make(map[string]interface{})

	for scanner.Scan() {
		line := scanner.Text()

		// since BSSID (a1:a1:a1...) contains ":", we need to split on ": "
		parts := trimSpaces(strings.Split(line, ": "))

		// if we only have a key, move on
		if len(parts) < 2 {
			continue
		}

		results[parts[0]] = parts[1]
	}

	return results
}

// unmarshallScanOuput parses the output of the airport scan command
func unmarshallScanOuput(reader io.Reader) []map[string]interface{} {
	/* example output:

	            SSID BSSID             RSSI CHANNEL HT CC SECURITY (auth/unicast/group)
	   i got spaces! a0:a0:a0:a0:a0:a0 -92  108     Y  US WPA(PSK/AES,TKIP/TKIP) RSN(PSK/AES,TKIP/TKIP)
	       no-spaces b1:b1:b1:b1:b1:b1 -91  116     N  EU RSN(PSK/AES/AES)
	*/

	scanner := bufio.NewScanner(reader)
	scanner.Split(bufio.ScanLines)

	var headers []string
	var headerSeparatorIndexes []int
	var results []map[string]interface{}

	for scanner.Scan() {
		line := scanner.Text()

		if headerSeparatorIndexes == nil {
			headerSeparatorIndexes = scanColumnSeparatorIndexes(line)
			headers = splitAtIndexes(line, headerSeparatorIndexes)
			continue
		}

		rowData := make(map[string]interface{}, len(headers))

		for i, value := range splitAtIndexes(line, headerSeparatorIndexes) {
			rowData[headers[i]] = value
		}

		results = append(results, rowData)
	}

	return results
}

func splitAtIndexes(str string, indexes []int) []string {
	var result []string

	startIndex := 0
	for _, index := range indexes {

		// if the index is out of bounds, return the rest of the string
		if index >= len(str) {
			result = append(result, str[startIndex:])
			return result
		}

		result = append(result, str[startIndex:index])
		startIndex = index
	}

	result = append(result, str[startIndex:])

	return trimSpaces(result)
}

func trimSpaces(strs []string) []string {
	for i, str := range strs {
		strs[i] = strings.TrimSpace(str)
	}
	return strs
}

func scanColumnSeparatorIndexes(line string) []int {
	/* example header row of airport scan:

	        SSID BSSID             RSSI CHANNEL HT CC SECURITY (auth/unicast/group)

	the leading spaces that are tricky
	the first column "SSID" is right aligned
	then the next column "BSSID" is left aligned
	*/

	var indexes []int
	foundFirstWord := false

	for i := 0; i < len(line)-1; i++ {
		if line[i] == ' ' && line[i+1] != ' ' {

			// due to leading spaces, don't record the index until the second time we meed the condition
			if !foundFirstWord {
				foundFirstWord = true
				continue
			}
			indexes = append(indexes, i)
		}
	}

	// since the last column has a space in it, just remove the last index
	if len(indexes) > 0 {
		indexes = indexes[:len(indexes)-1]
	}

	return indexes
}
