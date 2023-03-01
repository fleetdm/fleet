package fleet

import (
	"encoding/json"
	"testing"

	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPack_EditablePackType(t *testing.T) {
	type fields struct {
		UpdateCreateTimestamps UpdateCreateTimestamps
		ID                     uint
		Name                   string
		Description            string
		Platform               string
		Disabled               bool
		Type                   *string
		LabelIDs               []uint
		HostIDs                []uint
		TeamIDs                []uint
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "default",
			fields: fields{
				ID:          0,
				Name:        "",
				Description: "",
				Platform:    "",
				Disabled:    false,
				Type:        nil,
			},
			want: true,
		},
		{
			name: "type is empty string",
			fields: fields{
				ID:          0,
				Name:        "",
				Description: "",
				Platform:    "",
				Disabled:    false,
				Type:        ptr.String(""),
			},
			want: true,
		},
		{
			name: "type is not empty",
			fields: fields{
				ID:          0,
				Name:        "Global",
				Description: "Global Desc",
				Platform:    "",
				Disabled:    false,
				Type:        ptr.String("global"),
			},
			want: false,
		},
		{
			name: "type is not empty",
			fields: fields{
				ID:          0,
				Name:        "team-1",
				Description: "team-1 pack",
				Platform:    "",
				Disabled:    false,
				Type:        ptr.String("team-1"),
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Pack{
				UpdateCreateTimestamps: tt.fields.UpdateCreateTimestamps,
				ID:                     tt.fields.ID,
				Name:                   tt.fields.Name,
				Description:            tt.fields.Description,
				Platform:               tt.fields.Platform,
				Disabled:               tt.fields.Disabled,
				Type:                   tt.fields.Type,
				LabelIDs:               tt.fields.LabelIDs,
				HostIDs:                tt.fields.HostIDs,
				TeamIDs:                tt.fields.TeamIDs,
			}
			if got := p.EditablePackType(); got != tt.want {
				t.Errorf("EditablePackType() = %v, want %v", got, tt.want)
			}
		})
	}
}

// See #2778.
func TestPack_Marshal(t *testing.T) {
	b, err := json.Marshal(&Pack{})
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]interface{}
	err = json.Unmarshal(b, &m)
	require.NoError(t, err)
	require.NotNil(t, m["disabled"], "marshalled pack does not contain disabled field: %s", string(b))
}

func TestPack_TeamPack(t *testing.T) {
	p := Pack{
		ID:   13,
		Name: "team-foobar",
		Type: ptr.String("team-27"),
	}
	id, err := p.teamPack()
	require.NoError(t, err)
	assert.Equal(t, uint(27), *id)

	p.Type = ptr.String("other")
	id, err = p.teamPack()
	assert.NoError(t, err)
	assert.Nil(t, id)

	p.Type = nil
	id, err = p.teamPack()
	assert.NoError(t, err)
	assert.Nil(t, id)

	p.Type = ptr.String("team-foobar")
	_, err = p.teamPack()
	assert.Error(t, err)
}
