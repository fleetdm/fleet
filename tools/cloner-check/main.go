// Command cloner-check is used to verify that each item stored in the
// cached_mysql in-memory cache properly implements the fleet.Cloner interface.
//
// There are two ways to use this command, the first one is typically used by
// CI and checks that the cacheable items don't have any changes when compared
// to the current generated files. That is, if a field is added or modified in
// a cacheable struct, and those changes haven't been reflected in the
// generated files yet, it will raise an error, ensuring that the developer
// takes those changes into account in the custom Clone implementation.
//
// The --check flag runs this check mode scenario, but it is optional as running
// in check mode is the default:
//
//	$ go run ./tools/cloner-check/main.go [--check]
//
// (or alternatively "make check-go-cloner")
//
// The second way to use this command is with the --update flag, which is used
// to update the generated files with the current version of the cacheable
// items (i.e. the current struct definition). Use this when you've
// double-checked that the custom Clone implementation is up-to-date and
// correct.
//
//	$ go run ./tools/cloner-check/main.go --update
//
// (or atternatively "make update-go-cloner")
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/pmezard/go-difflib/difflib"
)

// If you add a new cacheable struct, make sure to add it to this list.
var cacheableItems = []fleet.Cloner{
	&fleet.AppConfig{},
	&fleet.Pack{},
	&fleet.ScheduledQuery{},
	&fleet.Features{},
	&fleet.TeamMDM{},
	&fleet.Query{},
	&fleet.MDMProfileSpec{},
	&fleet.MDMConfigAsset{},
	// TeamAgentOptions is not in the list because it is a json.RawMessage, no fields can change.
	// Same for ResultCountForQuery, it's just an int.
}

func main() {
	flagCheck := flag.Bool("check", false, "Run in check mode (default if no flag is provided)")
	flagUpdate := flag.Bool("update", false, "Update the generated files with the current cacheable items")
	flag.Parse()

	// make sure this is run from the root of the repository
	if _, err := os.Stat(filepath.Join("tools", "cloner-check", "main.go")); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "error: this command must be run from the root of the repository\n")
		os.Exit(1)
	}

	if *flagUpdate {
		updateCacheableItems()
		return
	}

	if *flagCheck || !*flagUpdate {
		if !checkCacheableItems() {
			fmt.Fprintf(os.Stderr, `
Some cacheable items failed the check, ensure you do the following:

1. Verify the Cloner implementation for that type, make sure it takes the new/updated field(s) into account if necessary.
2. Run "go run ./tools/cloner-check/main.go --update" (or "make update-go-cloner") to update the generated files and fix this check.
`)
			os.Exit(1)
		}
	}
}

func checkCacheableItems() bool {
	ok := true

	for _, item := range cacheableItems {
		itemType, _ := getUnderlyingStructType(reflect.TypeOf(item))
		filename := typeToFilename(itemType)

		want, err := os.ReadFile(filepath.Join("tools", "cloner-check", "generated_files", filename))
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: error reading file %s: %v\n", itemType, filename, err)
			ok = false
			continue
		}

		var sb strings.Builder
		if err := generateFieldsList(&sb, item); err != nil {
			fmt.Fprintf(os.Stderr, "%s: error generating field list: %v\n", itemType, err)
			ok = false
			continue
		}

		diff := difflib.UnifiedDiff{
			A:        difflib.SplitLines(string(want)),
			B:        difflib.SplitLines(sb.String()),
			FromFile: filename,
			ToFile:   "current",
			Context:  2,
		}
		text, err := difflib.GetUnifiedDiffString(diff)
		if err != nil {
			panic(err)
		}
		if len(text) != 0 {
			fmt.Fprintf(os.Stderr, "%s: fields mismatch vs file %s:\n%s", itemType, filename, text)
			ok = false
			continue
		}
	}

	return ok
}

func updateCacheableItems() bool {
	ok := true

	for _, item := range cacheableItems {
		itemType, _ := getUnderlyingStructType(reflect.TypeOf(item))
		filename := typeToFilename(itemType)

		var sb strings.Builder
		if err := generateFieldsList(&sb, item); err != nil {
			fmt.Fprintf(os.Stderr, "%s: error generating field list: %v\n", itemType, err)
			ok = false
			continue
		}
		if err := os.WriteFile(filepath.Join("tools", "cloner-check", "generated_files", filename), []byte(sb.String()), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "%s: error generating file %s: %v\n", itemType, filename, err)
			ok = false
			continue
		}
	}

	return ok
}

func typeToFilename(t reflect.Type) string {
	return fmt.Sprintf("%s.txt", strings.ToLower(t.Name()))
}

func generateFieldsList(w io.Writer, item fleet.Cloner) error {
	// keep a map of already-printed types, to avoid printing the same type multiple times
	seenTypes := make(map[string]bool)
	t := reflect.TypeOf(item)
	return generateStructFieldsList(w, t, seenTypes)
}

var basicTypes = map[reflect.Kind]bool{
	reflect.Bool:       true,
	reflect.Int:        true,
	reflect.Int8:       true,
	reflect.Int16:      true,
	reflect.Int32:      true,
	reflect.Int64:      true,
	reflect.Uint:       true,
	reflect.Uint8:      true,
	reflect.Uint16:     true,
	reflect.Uint32:     true,
	reflect.Uint64:     true,
	reflect.Uintptr:    true,
	reflect.Float32:    true,
	reflect.Float64:    true,
	reflect.Complex64:  true,
	reflect.Complex128: true,
	reflect.String:     true,
}

func generateStructFieldsList(w io.Writer, t reflect.Type, seenTypes map[string]bool) error {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// should always be a struct in the end
	if t.Kind() != reflect.Struct {
		panic("generateStructFieldsList called with non-struct type: " + t.String())
	}

	key := fmt.Sprintf("%s/%s", t.PkgPath(), t.Name())

	if seenTypes[key] {
		return nil
	}
	seenTypes[key] = true

	count := t.NumField()
	for i := 0; i < count; i++ {
		field := t.Field(i)

		// if the type is defined as a basic type, add that information to the line.
		if basicTypes[field.Type.Kind()] && field.Type.PkgPath() != "" {
			if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", key, field.Name, field.Type.String(), field.Type.Kind()); err != nil {
				return err
			}
		} else {
			if _, err := fmt.Fprintf(w, "%s\t%s\t%s\n", key, field.Name, field.Type.String()); err != nil {
				return err
			}
		}

		// if the field is (eventually) a struct, print this struct's fields too
		// (this resolves pointers to structs, slices, arrays, and maps, which is
		// why it receives potentially 2 types - map key and value)
		st1, st2 := getUnderlyingStructType(field.Type)
		if st1 != nil {
			if err := generateStructFieldsList(w, st1, seenTypes); err != nil {
				return err
			}
		}
		if st2 != nil {
			if err := generateStructFieldsList(w, st2, seenTypes); err != nil {
				return err
			}
		}
	}
	return nil
}

func getUnderlyingStructType(t reflect.Type) (st1, st2 reflect.Type) {
	for {
		if t.Kind() == reflect.Pointer || t.Kind() == reflect.Slice || t.Kind() == reflect.Array {
			t = t.Elem()
			continue
		}

		if t.Kind() == reflect.Map {
			// a map's key cannot be a map, so we can safely call
			// getUnderlyingStructType again on the map's key type
			st1, _ = getUnderlyingStructType(t.Key())
			// and then do the same for the map's value type
			k, v := getUnderlyingStructType(t.Elem())
			// however, this does not support a map of maps, so if the value has two types (was a map),
			// we panic.
			if k != nil && v != nil {
				panic("unsupported map of maps: " + t.String())
			}
			st2 = k
			return st1, st2
		}

		if t.Kind() == reflect.Struct {
			return t, nil
		}

		// not a pointer, slice, array, map nor struct
		return nil, nil
	}
}
