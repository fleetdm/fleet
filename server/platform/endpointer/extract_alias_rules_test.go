package endpointer

import (
	"encoding/json"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type extractAliasRulesSuite struct {
	suite.Suite
}

func TestExtractAliasRules(t *testing.T) {
	suite.Run(t, new(extractAliasRulesSuite))
}

// SetupTest runs before every Test* method, clearing the global caches.
func (s *extractAliasRulesSuite) SetupTest() {
	aliasRulesCache.Range(func(key, _ any) bool {
		aliasRulesCache.Delete(key)
		return true
	})
	relativeRulesCache.Range(func(key, _ any) bool {
		relativeRulesCache.Delete(key)
		return true
	})
}

func (s *extractAliasRulesSuite) TestNilInput() {
	rules := ExtractAliasRules(nil)
	require.Nil(s.T(), rules)
}

func (s *extractAliasRulesSuite) TestNonStructInput() {
	rules := ExtractAliasRules("hello")
	require.Nil(s.T(), rules)

	rules = ExtractAliasRules(42)
	require.Nil(s.T(), rules)
}

func (s *extractAliasRulesSuite) TestStructWithNoRenametoTags() {
	type plain struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
	rules := ExtractAliasRules(plain{})
	require.Empty(s.T(), rules)
}

func (s *extractAliasRulesSuite) TestSingleRenametoTag() {
	type singleAlias struct {
		TeamID uint `json:"team_id" renameto:"group_id"`
	}
	rules := ExtractAliasRules(singleAlias{})
	s.Require().Equal([]AliasRule{{OldKey: "team_id", NewKey: "group_id", Scoped: true}}, rules)
}

func (s *extractAliasRulesSuite) TestMultipleRenametoTags() {
	type multiAlias struct {
		TeamID   uint   `json:"team_id" renameto:"group_id"`
		TeamName string `json:"team_name" renameto:"group_name"`
		Other    string `json:"other"`
	}
	rules := ExtractAliasRules(multiAlias{})
	s.Require().Equal([]AliasRule{
		{OldKey: "team_id", NewKey: "group_id", Scoped: true},
		{OldKey: "team_name", NewKey: "group_name", Scoped: true},
	}, rules)
}

func (s *extractAliasRulesSuite) TestJsonTagWithOmitempty() {
	type omitemptyAlias struct {
		TeamID uint `json:"team_id,omitempty" renameto:"group_id"`
	}
	rules := ExtractAliasRules(omitemptyAlias{})
	s.Require().Equal([]AliasRule{{OldKey: "team_id", NewKey: "group_id", Scoped: true}}, rules)
}

func (s *extractAliasRulesSuite) TestJsonTagDashIsSkipped() {
	type dashJSON struct {
		Secret string `json:"-" renameto:"new_secret"`
	}
	rules := ExtractAliasRules(dashJSON{})
	require.Empty(s.T(), rules)
}

func (s *extractAliasRulesSuite) TestNoJsonTagWithRenametoIsSkipped() {
	type noJSON struct {
		Field string `renameto:"new_field"`
	}
	rules := ExtractAliasRules(noJSON{})
	require.Empty(s.T(), rules)
}

func (s *extractAliasRulesSuite) TestEmptyRenametoIsSkipped() {
	type emptyRenameTo struct {
		Field string `json:"field" renameto:""`
	}
	rules := ExtractAliasRules(emptyRenameTo{})
	require.Empty(s.T(), rules)
}

func (s *extractAliasRulesSuite) TestNestedStruct() {
	type inner struct {
		InnerField string `json:"inner_field" renameto:"new_inner"`
	}
	type outer struct {
		OuterField string `json:"outer_field" renameto:"new_outer"`
		Nested     inner
	}
	rules := ExtractAliasRules(outer{})
	s.Require().Equal([]AliasRule{
		{OldKey: "outer_field", NewKey: "new_outer", Scoped: true},
		// Nested has no json tag, so its JSON key is the Go field name "Nested".
		{OldKey: "inner_field", NewKey: "new_inner", Path: "Nested", Scoped: true},
	}, rules)
}

func (s *extractAliasRulesSuite) TestDeeplyNestedStructs() {
	type level2 struct {
		Deep string `json:"deep" renameto:"new_deep"`
	}
	type level1 struct {
		Mid    string `json:"mid" renameto:"new_mid"`
		Nested level2
	}
	type level0 struct {
		Top    string `json:"top" renameto:"new_top"`
		Nested level1
	}
	rules := ExtractAliasRules(level0{})
	s.Require().Equal([]AliasRule{
		{OldKey: "top", NewKey: "new_top", Scoped: true},
		// The nested structs are reached through fields named "Nested" (no
		// json tag), so each level adds a "Nested" path segment.
		{OldKey: "mid", NewKey: "new_mid", Path: "Nested", Scoped: true},
		{OldKey: "deep", NewKey: "new_deep", Path: "Nested" + pathSep + "Nested", Scoped: true},
	}, rules)
}

func (s *extractAliasRulesSuite) TestDeduplicationAcrossNestedStructs() {
	type shared struct {
		TeamID uint `json:"team_id" renameto:"group_id"`
	}
	type parent struct {
		TeamID uint `json:"team_id" renameto:"group_id"`
		Child  shared
	}
	rules := ExtractAliasRules(parent{})
	// The same OldKey→NewKey pair at different paths yields one scoped rule
	// per path: once at the root, once under the "Child" object.
	s.Require().Equal([]AliasRule{
		{OldKey: "team_id", NewKey: "group_id", Scoped: true},
		{OldKey: "team_id", NewKey: "group_id", Path: "Child", Scoped: true},
	}, rules)
}

func (s *extractAliasRulesSuite) TestPointerToStructInput() {
	type ptrInput struct {
		Field string `json:"field" renameto:"new_field"`
	}
	rules := ExtractAliasRules(&ptrInput{})
	s.Require().Equal([]AliasRule{{OldKey: "field", NewKey: "new_field", Scoped: true}}, rules)
}

func (s *extractAliasRulesSuite) TestNestedThroughPointerField() {
	type pointed struct {
		Inner string `json:"inner" renameto:"new_inner"`
	}
	type wrapper struct {
		Ptr *pointed
	}
	rules := ExtractAliasRules(wrapper{})
	// Reached through the "Ptr" field (no json tag → Go field name).
	s.Require().Equal([]AliasRule{{OldKey: "inner", NewKey: "new_inner", Path: "Ptr", Scoped: true}}, rules)
}

func (s *extractAliasRulesSuite) TestNestedThroughSliceField() {
	type elem struct {
		Val string `json:"val" renameto:"new_val"`
	}
	type sliceWrapper struct {
		Items []elem
	}
	rules := ExtractAliasRules(sliceWrapper{})
	// Reached through the "Items" field (no json tag → Go field name).
	s.Require().Equal([]AliasRule{{OldKey: "val", NewKey: "new_val", Path: "Items", Scoped: true}}, rules)
}

func (s *extractAliasRulesSuite) TestNestedThroughMapField() {
	type mapVal struct {
		Key string `json:"key" renameto:"new_key"`
	}
	type mapWrapper struct {
		Data map[string]mapVal
	}
	rules := ExtractAliasRules(mapWrapper{})
	// Reached through the "Data" field (no json tag → Go field name). Note
	// that map values nest under a dynamic map key at runtime, so a rename
	// behind a map cannot be matched by the scoped rewriter; the spec types do
	// not place renames behind maps.
	s.Require().Equal([]AliasRule{{OldKey: "key", NewKey: "new_key", Path: "Data", Scoped: true}}, rules)
}

func (s *extractAliasRulesSuite) TestTeamSpecSetupExperienceScoping() {
	// Regression: the `macos_setup`→`setup_experience` rename (under mdm) must
	// not clobber the unrelated per-software `setup_experience` install flag
	// when a team spec is rewritten from new to old key names.
	rules := ExtractAliasRules(fleet.TeamSpec{})

	var found bool
	for _, r := range rules {
		if r.OldKey == "macos_setup" && r.NewKey == "setup_experience" {
			found = true
			s.True(r.Scoped, "rename rule must be scoped")
			s.Equal("mdm", r.Path, "macos_setup lives under mdm")
		}
	}
	s.Require().True(found, "expected the macos_setup→setup_experience rule")

	spec := []byte(`{
		"name": "Workstations",
		"mdm": {"setup_experience": {"enable_end_user_authentication": true}},
		"software": {"app_store_apps": [{"app_store_id": "1", "setup_experience": true}]}
	}`)
	out, _, err := RewriteDeprecatedKeys(spec, rules)
	s.Require().NoError(err)

	var result map[string]any
	s.Require().NoError(json.Unmarshal(out, &result))

	mdm := result["mdm"].(map[string]any)
	s.Contains(mdm, "macos_setup")
	s.NotContains(mdm, "setup_experience")

	app := result["software"].(map[string]any)["app_store_apps"].([]any)[0].(map[string]any)
	s.Equal(true, app["setup_experience"], "per-software setup_experience must survive")
	s.NotContains(app, "macos_setup")
}

func (s *extractAliasRulesSuite) TestDeduplicationSameStructViaMultiplePaths() {
	type common struct {
		ID string `json:"old_id" renameto:"new_id"`
	}
	type branch1 struct {
		C common
	}
	type branch2 struct {
		C common
	}
	type root struct {
		B1 branch1
		B2 branch2
	}
	rules := ExtractAliasRules(root{})
	// common is reachable via two distinct paths (B1.C and B2.C), so it yields
	// one scoped rule per path. Fields with no json tag use their Go field name
	// as the JSON key.
	s.Require().Equal([]AliasRule{
		{OldKey: "old_id", NewKey: "new_id", Path: "B1" + pathSep + "C", Scoped: true},
		{OldKey: "old_id", NewKey: "new_id", Path: "B2" + pathSep + "C", Scoped: true},
	}, rules)
}
