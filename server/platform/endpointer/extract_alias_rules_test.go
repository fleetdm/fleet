package endpointer

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type extractAliasRulesSuite struct {
	suite.Suite
}

func TestExtractAliasRules(t *testing.T) {
	suite.Run(t, new(extractAliasRulesSuite))
}

// SetupTest runs before every Test* method, clearing the global cache.
func (s *extractAliasRulesSuite) SetupTest() {
	aliasRulesCache.Range(func(key, _ any) bool {
		aliasRulesCache.Delete(key)
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
	require.Equal(s.T(), []AliasRule{{OldKey: "team_id", NewKey: "group_id"}}, rules)
}

func (s *extractAliasRulesSuite) TestMultipleRenametoTags() {
	type multiAlias struct {
		TeamID   uint   `json:"team_id" renameto:"group_id"`
		TeamName string `json:"team_name" renameto:"group_name"`
		Other    string `json:"other"`
	}
	rules := ExtractAliasRules(multiAlias{})
	require.Equal(s.T(), []AliasRule{
		{OldKey: "team_id", NewKey: "group_id"},
		{OldKey: "team_name", NewKey: "group_name"},
	}, rules)
}

func (s *extractAliasRulesSuite) TestJsonTagWithOmitempty() {
	type omitemptyAlias struct {
		TeamID uint `json:"team_id,omitempty" renameto:"group_id"`
	}
	rules := ExtractAliasRules(omitemptyAlias{})
	require.Equal(s.T(), []AliasRule{{OldKey: "team_id", NewKey: "group_id"}}, rules)
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
	require.Equal(s.T(), []AliasRule{
		{OldKey: "outer_field", NewKey: "new_outer"},
		{OldKey: "inner_field", NewKey: "new_inner"},
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
	require.Equal(s.T(), []AliasRule{
		{OldKey: "top", NewKey: "new_top"},
		{OldKey: "mid", NewKey: "new_mid"},
		{OldKey: "deep", NewKey: "new_deep"},
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
	// The same OldKey→NewKey pair should appear only once.
	require.Equal(s.T(), []AliasRule{{OldKey: "team_id", NewKey: "group_id"}}, rules)
}

func (s *extractAliasRulesSuite) TestPointerToStructInput() {
	type ptrInput struct {
		Field string `json:"field" renameto:"new_field"`
	}
	rules := ExtractAliasRules(&ptrInput{})
	require.Equal(s.T(), []AliasRule{{OldKey: "field", NewKey: "new_field"}}, rules)
}

func (s *extractAliasRulesSuite) TestNestedThroughPointerField() {
	type pointed struct {
		Inner string `json:"inner" renameto:"new_inner"`
	}
	type wrapper struct {
		Ptr *pointed
	}
	rules := ExtractAliasRules(wrapper{})
	require.Equal(s.T(), []AliasRule{{OldKey: "inner", NewKey: "new_inner"}}, rules)
}

func (s *extractAliasRulesSuite) TestNestedThroughSliceField() {
	type elem struct {
		Val string `json:"val" renameto:"new_val"`
	}
	type sliceWrapper struct {
		Items []elem
	}
	rules := ExtractAliasRules(sliceWrapper{})
	require.Equal(s.T(), []AliasRule{{OldKey: "val", NewKey: "new_val"}}, rules)
}

func (s *extractAliasRulesSuite) TestNestedThroughMapField() {
	type mapVal struct {
		Key string `json:"key" renameto:"new_key"`
	}
	type mapWrapper struct {
		Data map[string]mapVal
	}
	rules := ExtractAliasRules(mapWrapper{})
	require.Equal(s.T(), []AliasRule{{OldKey: "key", NewKey: "new_key"}}, rules)
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
	// common's rule should appear only once even though reachable via two paths.
	require.Equal(s.T(), []AliasRule{{OldKey: "old_id", NewKey: "new_id"}}, rules)
}
