package scim

import (
	"testing"

	"github.com/scim2/filter-parser/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePath(t *testing.T) {
	path, err := filter.ParsePath([]byte("emails[type eq \"work\"].primary"))
	require.NoError(t, err)
	assert.Equal(t, "emails", path.AttributePath.String())
	attrExpression, ok := path.ValueExpression.(*filter.AttributeExpression)
	require.True(t, ok)
	assert.Equal(t, "type", attrExpression.AttributePath.String())
	assert.Equal(t, filter.EQ, attrExpression.Operator)
	assert.Equal(t, "work", attrExpression.CompareValue)
	require.NotNil(t, path.SubAttribute)
	assert.Equal(t, "primary", *path.SubAttribute)
}
