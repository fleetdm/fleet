// toml parsing won't work -- ini files don't quote the string and
// tend to have random spaces. Bummer, since
// https://github.com/pelletier/go-toml/pull/433 was right

// based on github.com/kolide/launcher/pkg/osquery/tables
package dataflatten

import (
	"github.com/go-ini/ini"
)

func IniFile(file string, opts ...FlattenOpts) ([]Row, error) {
	return flattenIni(file, opts...)
}

func Ini(rawdata []byte, opts ...FlattenOpts) ([]Row, error) {
	return flattenIni(rawdata, opts...)
}

// flattenIni uses go-ini to flatten ini data. The underlying library
// accepts both files and []byte via the interface{} type.  It also
// makes heavy use of reflect, so this does some manual iteration to
// extract things.
func flattenIni(in interface{}, opts ...FlattenOpts) ([]Row, error) {

	v := map[string]interface{}{}

	iniFile, err := ini.Load(in)
	if err != nil {
		return nil, err
	}

	for _, section := range iniFile.Sections() {
		// While we can use section.KeysHash() directly, instead we
		// iterate. This allows us to canonicalize the value to handle
		// booleans. Everything else we leave as string
		sectionMap := make(map[string]interface{})
		for _, key := range section.Keys() {
			asBool, ok := iniToBool(key.Value())
			if ok {
				sectionMap[key.Name()] = asBool
			} else {
				sectionMap[key.Name()] = key.Value()
			}
		}
		v[section.Name()] = sectionMap
	}

	return Flatten(v, opts...)
}

// iniToBool attempts to convert an ini value to a boolean. It returns
// the converted value, and ok. The list of strings comes from go-ini
func iniToBool(val string) (bool, bool) {
	switch val {
	case "t", "T", "true", "TRUE", "True", "YES", "yes", "Yes", "y", "ON", "on", "On":
		return true, true
	case "f", "F", "false", "FALSE", "False", "NO", "no", "No", "n", "OFF", "off", "Off":
		return false, true
	}
	return false, false
}
