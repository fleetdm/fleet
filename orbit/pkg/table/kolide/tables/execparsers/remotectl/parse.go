//go:build darwin
// +build darwin

package remotectl

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
)

// parseDumpstate parses the result of /usr/libexec/remotectl into a map that can be flattened by dataflatten.
// We expect results in the following format, with empty newlines separating devices:
//
//	<device name>
//			Key: Value
//			Properties: {
//				Key => Value
//			}
//			Services:
//				service1
//					Version: 1
//					Properties: {
//						Key => Value
//					}
//				service2
//				service3
//			Local Services:
//				service4
func (p *parser) parseDumpstate(reader io.Reader) (any, error) {
	results := make(map[string]map[string]interface{})

	p.scanner = bufio.NewScanner(reader)
	for p.scanner.Scan() {
		p.lastReadLine = p.scanner.Text()

		// Process each device
		if p.isDeviceName() {
			currentDeviceName := p.extractDeviceName()
			currentDeviceResults, err := p.parseDevice()
			if err != nil {
				return nil, err
			}
			results[currentDeviceName] = currentDeviceResults
			continue
		}

		return nil, fmt.Errorf("no device name(s) given in remotectl dumpstate output")
	}

	return results, nil
}

func (p *parser) isDeviceName() bool {
	// If the line is not indented (i.e. top-level), we have a device name
	return !strings.HasPrefix(p.lastReadLine, "\t")
}

func (p *parser) extractDeviceName() string {
	// Devices (besides "Local device") are identified as `Found <name> (<type>)` -- strip "Found"
	return strings.TrimSpace(strings.TrimPrefix(p.lastReadLine, "Found"))
}

func (p *parser) parseDevice() (map[string]interface{}, error) {
	deviceResults := make(map[string]interface{})

	for p.scanner.Scan() {
		p.lastReadLine = p.scanner.Text()

		if strings.HasPrefix(strings.TrimSpace(p.lastReadLine), "Heartbeat:") {
			heartbeats, err := p.parseStringArray()
			if err != nil {
				return nil, err
			}
			deviceResults["Heartbeat"] = heartbeats

			// Proceed with parsing p.lastReadLine, since arrays do not have ending delimiters.
		}

		if strings.HasPrefix(strings.TrimSpace(p.lastReadLine), "Services:") {
			services, eof, err := p.parseObjectArray()
			if err != nil {
				return nil, err
			}
			deviceResults["Services"] = services

			// If we've reached the end of the command output ("Services" is sometimes the end), then return.
			// Otherwise, proceed with parsing p.lastReadLine, since arrays do not have ending delimiters.
			if eof {
				return deviceResults, nil
			}
		}

		// We handle Local Services separately from the above, to allow us to go directly from Services into Local Services
		if strings.HasPrefix(strings.TrimSpace(p.lastReadLine), "Local Services:") {
			localServices, eof, err := p.parseObjectArray()
			if err != nil {
				return nil, err
			}
			deviceResults["Local Services"] = localServices

			// If we've reached the end of the command output ("Local Services" is sometimes the end), then return.
			// Otherwise, proceed with parsing p.lastReadLine, since arrays do not have ending delimiters.
			if eof {
				return deviceResults, nil
			}
		}

		if strings.HasPrefix(strings.TrimSpace(p.lastReadLine), "Properties:") {
			deviceProperties, err := p.parseDictionary()
			if err != nil {
				return nil, err
			}
			deviceResults["Properties"] = deviceProperties
			continue
		}

		if p.isDeviceDelimiter() {
			return deviceResults, nil
		}

		// We have a top-level key with a value we should extract to store in `results`
		propertyKey, propertyValue, err := extractTopLevelKeyValue(p.lastReadLine)
		if err != nil {
			return nil, err
		}
		deviceResults[propertyKey] = propertyValue
	}

	return deviceResults, nil
}

func (p *parser) isDeviceDelimiter() bool {
	// A newline indicates a new device's information is coming next
	return strings.TrimSpace(p.lastReadLine) == ""
}

func (p *parser) parseDictionary() (map[string]interface{}, error) {
	dictionaryResults := make(map[string]interface{})

	for p.scanner.Scan() {
		p.lastReadLine = strings.TrimSpace(p.scanner.Text())

		// Exiting dictionary
		if p.lastReadLine == "}" {
			return dictionaryResults, nil
		}

		propertyKey, propertyValue, err := extractPropertyKeyValue(p.lastReadLine)
		if err != nil {
			return nil, err
		}

		dictionaryResults[propertyKey] = propertyValue
	}

	return nil, errors.New("unexpected end to dictionary")
}

func (p *parser) parseStringArray() ([]string, error) {
	arrayResults := make([]string, 0)

	startingIndentationLevel := p.getCurrentIndentationLevel()
	for p.scanner.Scan() {
		p.lastReadLine = p.scanner.Text()

		currentIndentationLevel := p.getCurrentIndentationLevel()

		// Exiting array
		if currentIndentationLevel <= startingIndentationLevel {
			return arrayResults, nil
		}

		// Ignore everything not at the top level
		if currentIndentationLevel == startingIndentationLevel+1 {
			arrayResults = append(arrayResults, strings.TrimSpace(p.lastReadLine))
		}
	}

	return arrayResults, nil
}

func (p *parser) parseObjectArray() ([]map[string]interface{}, bool, error) {
	arrayResults := make([]map[string]interface{}, 0)
	eof := false

	startingIndentationLevel := p.getCurrentIndentationLevel()
	arrayItemIndentationLevel := startingIndentationLevel + 1
	arrayItemPropertyIndentationLevel := arrayItemIndentationLevel + 1

	for p.scanner.Scan() {
		p.lastReadLine = p.scanner.Text()

		currentIndentationLevel := p.getCurrentIndentationLevel()

		// Exiting array
		if currentIndentationLevel <= startingIndentationLevel {
			return arrayResults, eof, nil
		}

		// Process items
		if currentIndentationLevel == arrayItemIndentationLevel {
			item := make(map[string]interface{})
			// Create artificial key "Name" to hold the name of the item
			item["Name"] = strings.TrimSpace(p.lastReadLine)
			arrayResults = append(arrayResults, item)
			continue
		}

		// One more level indented -- we have properties attached to the item we processed last. Extract them.
		if currentIndentationLevel >= arrayItemPropertyIndentationLevel {
			lastProcessedItem := arrayResults[len(arrayResults)-1]

			if strings.HasPrefix(strings.TrimSpace(p.lastReadLine), "Properties:") {
				itemProperties, err := p.parseDictionary()
				if err != nil {
					return nil, eof, err
				}
				lastProcessedItem["Properties"] = itemProperties
				continue
			}

			propertyKey, propertyValue, err := extractTopLevelKeyValue(p.lastReadLine)
			if err != nil {
				return nil, eof, err
			}
			lastProcessedItem[propertyKey] = propertyValue
		}
	}

	eof = true
	return arrayResults, eof, nil
}

func (p *parser) getCurrentIndentationLevel() int {
	return strings.LastIndex(p.lastReadLine, "\t")
}

func extractPropertyKeyValue(line string) (string, string, error) {
	// key-value pairs in the `Properties` dictionary are in the format `key => value`
	return extractKeyValue(line, "=>")
}

func extractTopLevelKeyValue(line string) (string, string, error) {
	// Top-level key-value pairs are in the format `key: value`
	return extractKeyValue(line, ":")
}

func extractKeyValue(line, delimiter string) (string, string, error) {
	extracted := strings.Split(line, delimiter)
	if len(extracted) != 2 {
		return "", "", errors.New("top-level key/value pair in remotectl output is in an unexpected format")
	}

	return strings.TrimSpace(extracted[0]), strings.TrimSpace(extracted[1]), nil
}
