package common_mysql

import "github.com/fleetdm/fleet/v4/server/goose"

// CompareVersions returns any missing or extra elements in v2 with respect to v1
// (v1 or v2 need not be ordered).
func CompareVersions(v1, v2 []int64, knownUnknowns map[int64]struct{}) (missing []int64, unknown []int64, equal bool) {
	v1s := make(map[int64]struct{})
	for _, m := range v1 {
		v1s[m] = struct{}{}
	}
	v2s := make(map[int64]struct{})
	for _, m := range v2 {
		v2s[m] = struct{}{}
	}
	for _, m := range v1 {
		if _, ok := v2s[m]; !ok {
			missing = append(missing, m)
		}
	}
	for _, m := range v2 {
		if _, ok := v1s[m]; !ok {
			unknown = append(unknown, m)
		}
	}
	unknown = unknownUnknowns(unknown, knownUnknowns)
	if len(missing) == 0 && len(unknown) == 0 {
		return nil, nil, true
	}
	return missing, unknown, false
}

func unknownUnknowns(in []int64, knownUnknowns map[int64]struct{}) []int64 {
	var result []int64
	for _, t := range in {
		if _, ok := knownUnknowns[t]; !ok {
			result = append(result, t)
		}
	}
	return result
}

func GetVersionsFromMigrations(migrations goose.Migrations) []int64 {
	versions := make([]int64, len(migrations))
	for i := range migrations {
		versions[i] = migrations[i].Version
	}
	return versions
}
