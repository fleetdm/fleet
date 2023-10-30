package dataflatten

import (
	"bufio"
	"bytes"
	"strings"
)

type dataFunc func(data []byte, opts ...FlattenOpts) ([]Row, error)

type recordSplittingStrategy int

const (
	None recordSplittingStrategy = iota
	DuplicateKeys
)

func StringDelimitedFunc(kVDelimiter string, splittingStrategy recordSplittingStrategy) dataFunc {
	switch splittingStrategy {
	case None:
		return singleRecordFunc(kVDelimiter)
	case DuplicateKeys:
		return duplicateKeyFunc(kVDelimiter)
	default:
		panic("Unknown record splitting strategy")
	}
}

// duplicateKeyFunc returns a function that conforms to the interface expected
// by dataflattentable.Table's execDataFunc property. properties are grouped
// into a single record based on 'duplicate key' strategy: If a key/value pair
// is encountered, and the record being built already has a value for that key,
// then that record is considered 'complete'. The record is stored in the
// collection, and a new record is started. This strategy is only suitable if
// properties for a single record are grouped together, and there is at least
// one field that appears for every record before any sparse data.
func duplicateKeyFunc(kVDelimiter string) dataFunc {
	return func(rawdata []byte, opts ...FlattenOpts) ([]Row, error) {
		results := []interface{}{}
		scanner := bufio.NewScanner(bytes.NewReader(rawdata))
		row := map[string]interface{}{}
		for scanner.Scan() {
			line := scanner.Text()
			parts := strings.SplitN(line, kVDelimiter, 2)
			if len(parts) < 2 {
				continue
			}
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			if _, ok := row[key]; ok { // this key already exists, so we want to start a new record.
				results = append(results, row) // store the 'finished' record in the collection
				row = map[string]interface{}{} // reset the record
			}
			row[key] = value
		}
		results = append(results, row) // store the final record

		return Flatten(results, opts...)
	}
}

// singleRecordFunc returns an execData function that assumes 'rawdata'
// only holds key-value pairs for a single record. Additionally, each k/v pair
// must be on its own line. Useful for output that can be easily separated into
// separate records before 'flattening'
func singleRecordFunc(kVDelimiter string) dataFunc {
	return func(rawdata []byte, opts ...FlattenOpts) ([]Row, error) {
		results := []interface{}{}
		scanner := bufio.NewScanner(bytes.NewReader(rawdata))
		for scanner.Scan() {
			line := scanner.Text()
			parts := strings.SplitN(line, kVDelimiter, 2)
			if len(parts) < 2 {
				continue
			}
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			results = append(results, map[string]interface{}{key: value})
		}

		return Flatten(results, opts...)
	}
}
