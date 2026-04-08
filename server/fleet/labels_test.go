package fleet

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectMissingLabels(t *testing.T) {
	labelMap := map[string]uint{"label1": 1, "label2": 2}

	testCases := []struct {
		name     string
		labels   []string
		expected []string
	}{
		{
			name:     "no labels",
			labels:   []string{},
			expected: []string{},
		},
		{
			name:     "empty label",
			labels:   []string{""},
			expected: []string{},
		},
		{
			name:     "one missing label",
			labels:   []string{"iamnotalabel"},
			expected: []string{"iamnotalabel"},
		},
		{
			name:     "missing multiple labels",
			labels:   []string{"mac", "windows"},
			expected: []string{"mac", "windows"},
		},
		{
			name:     "some missing labels and some valid labels",
			labels:   []string{"label1", "mac", "label2", "windows"},
			expected: []string{"mac", "windows"},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			missing := DetectMissingLabels(labelMap, tt.labels)
			if !assert.Equal(t, tt.expected, missing) {
				t.Errorf("Expected [%s], but got [%s]", strings.Join(tt.expected, ", "), strings.Join(missing, ", "))
			}
		})
	}
}

func TestLabelIdentsWithScope_Equal(t *testing.T) {
	t.Run("nil both", func(t *testing.T) {
		var a, b *LabelIdentsWithScope
		require.True(t, a.Equal(b))
	})

	t.Run("nil one", func(t *testing.T) {
		a := &LabelIdentsWithScope{LabelScope: LabelScopeIncludeAny}
		require.False(t, a.Equal(nil))
	})

	t.Run("same scope and labels", func(t *testing.T) {
		a := &LabelIdentsWithScope{
			LabelScope: LabelScopeIncludeAny,
			ByName:     map[string]LabelIdent{"foo": {LabelID: 1, LabelName: "foo"}},
		}
		b := &LabelIdentsWithScope{
			LabelScope: LabelScopeIncludeAny,
			ByName:     map[string]LabelIdent{"foo": {LabelID: 1, LabelName: "foo"}},
		}
		require.True(t, a.Equal(b))
	})

	t.Run("different scope", func(t *testing.T) {
		a := &LabelIdentsWithScope{LabelScope: LabelScopeIncludeAny}
		b := &LabelIdentsWithScope{LabelScope: LabelScopeExcludeAny}
		require.False(t, a.Equal(b))
	})

	t.Run("with ExcludeByName equal", func(t *testing.T) {
		a := &LabelIdentsWithScope{
			LabelScope:    LabelScopeIncludeAny,
			ByName:        map[string]LabelIdent{"foo": {LabelID: 1, LabelName: "foo"}},
			ExcludeByName: map[string]LabelIdent{"bar": {LabelID: 2, LabelName: "bar"}},
		}
		b := &LabelIdentsWithScope{
			LabelScope:    LabelScopeIncludeAny,
			ByName:        map[string]LabelIdent{"foo": {LabelID: 1, LabelName: "foo"}},
			ExcludeByName: map[string]LabelIdent{"bar": {LabelID: 2, LabelName: "bar"}},
		}
		require.True(t, a.Equal(b))
	})

	t.Run("with ExcludeByName different", func(t *testing.T) {
		a := &LabelIdentsWithScope{
			LabelScope:    LabelScopeIncludeAny,
			ByName:        map[string]LabelIdent{"foo": {LabelID: 1, LabelName: "foo"}},
			ExcludeByName: map[string]LabelIdent{"bar": {LabelID: 2, LabelName: "bar"}},
		}
		b := &LabelIdentsWithScope{
			LabelScope: LabelScopeIncludeAny,
			ByName:     map[string]LabelIdent{"foo": {LabelID: 1, LabelName: "foo"}},
		}
		require.False(t, a.Equal(b))
	})
}

func TestLabelIdentsWithScope_HasExcludeLabels(t *testing.T) {
	t.Run("nil receiver", func(t *testing.T) {
		var l *LabelIdentsWithScope
		require.False(t, l.HasExcludeLabels())
	})

	t.Run("no exclude labels", func(t *testing.T) {
		l := &LabelIdentsWithScope{
			LabelScope: LabelScopeIncludeAny,
			ByName:     map[string]LabelIdent{"foo": {LabelID: 1}},
		}
		require.False(t, l.HasExcludeLabels())
	})

	t.Run("has exclude labels", func(t *testing.T) {
		l := &LabelIdentsWithScope{
			LabelScope:    LabelScopeIncludeAny,
			ByName:        map[string]LabelIdent{"foo": {LabelID: 1}},
			ExcludeByName: map[string]LabelIdent{"bar": {LabelID: 2}},
		}
		require.True(t, l.HasExcludeLabels())
	})
}

func TestLabelIdentsWithScope_AllLabelIDs(t *testing.T) {
	t.Run("nil receiver", func(t *testing.T) {
		var l *LabelIdentsWithScope
		require.Nil(t, l.AllLabelIDs())
	})

	t.Run("include only", func(t *testing.T) {
		l := &LabelIdentsWithScope{
			ByName: map[string]LabelIdent{
				"foo": {LabelID: 1},
				"bar": {LabelID: 2},
			},
		}
		ids := l.AllLabelIDs()
		require.Len(t, ids, 2)
		require.ElementsMatch(t, []uint{1, 2}, ids)
	})

	t.Run("combined include and exclude", func(t *testing.T) {
		l := &LabelIdentsWithScope{
			ByName: map[string]LabelIdent{
				"foo": {LabelID: 1},
			},
			ExcludeByName: map[string]LabelIdent{
				"bar": {LabelID: 2},
				"baz": {LabelID: 3},
			},
		}
		ids := l.AllLabelIDs()
		require.Len(t, ids, 3)
		require.ElementsMatch(t, []uint{1, 2, 3}, ids)
	})
}

func TestPolicyPayload_Verify_CombinedLabels(t *testing.T) {
	t.Run("include_any + exclude_any is allowed", func(t *testing.T) {
		p := PolicyPayload{
			Name:             "test",
			Query:            "SELECT 1",
			LabelsIncludeAny: []string{"label1"},
			LabelsExcludeAny: []string{"label2"},
		}
		err := p.Verify()
		require.NoError(t, err)
	})

	t.Run("include_any only is allowed", func(t *testing.T) {
		p := PolicyPayload{
			Name:             "test",
			Query:            "SELECT 1",
			LabelsIncludeAny: []string{"label1"},
		}
		err := p.Verify()
		require.NoError(t, err)
	})

	t.Run("exclude_any only is allowed", func(t *testing.T) {
		p := PolicyPayload{
			Name:             "test",
			Query:            "SELECT 1",
			LabelsExcludeAny: []string{"label1"},
		}
		err := p.Verify()
		require.NoError(t, err)
	})
}
