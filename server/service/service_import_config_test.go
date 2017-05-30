package service

import (
	"testing"

	"github.com/kolide/kolide/server/config"
	"github.com/kolide/kolide/server/datastore/inmem"
	"github.com/kolide/kolide/server/kolide"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createServiceMockForImport(t *testing.T) *service {
	ds, err := inmem.New(config.TestConfig())
	require.Nil(t, err)
	err = ds.MigrateData()
	require.Nil(t, err)
	return &service{
		ds: ds,
	}
}

func TestHashQuery(t *testing.T) {
	q1 := `SELECT * FROM t1 INNER JOIN ON
		t1.id = t2.t1id
		WHERE t1.name = 'foo'  `
	q2 := "SELECT * from t1 INNER JOIN\tON t1.id = t2.t1id	WHERE t1.name = 'foo';"
	h1 := hashQuery("platform", q1)
	h2 := hashQuery("platform", q2)
	assert.Equal(t, h1, h2)
	q2 = "SELECT * from t1 INNER JOIN\tON t1.id = t2.t1id	WHERE t2.name = 'foo';"
	h2 = hashQuery("platform", q2)
	assert.NotEqual(t, h1, h2)

}
func TestImportFilePaths(t *testing.T) {
	cfg := &kolide.ImportConfig{
		FileIntegrityMonitoring: kolide.FIMCategoryToPaths{
			"files1": []string{
				"path1",
				"path2",
			},
			"files2": []string{
				"path3",
			},
		},
		YARA: &kolide.YARAConfig{
			Signatures: map[string][]string{
				"sig1": []string{
					"path4",
					"path5",
				},
				"sig2": []string{
					"path6",
				},
			},
			FilePaths: map[string][]string{
				"files1": []string{
					"sig1",
					"sig2",
				},
				"files2": []string{
					"sig1",
				},
			},
		},
	}
	resp := &kolide.ImportConfigResponse{
		ImportStatusBySection: make(map[kolide.ImportSection]*kolide.ImportStatus),
	}
	svc := createServiceMockForImport(t)
	tx, _ := svc.ds.Begin()
	err := svc.importFIMSections(cfg, resp, tx)
	require.Nil(t, err)
	assert.Equal(t, 2, resp.Status(kolide.FilePathsSection).ImportCount)
	sections, err := svc.ds.FIMSections()
	require.Nil(t, err)
	assert.Len(t, sections, 2)
	yara, err := svc.ds.YARASection()
	require.Nil(t, err)
	assert.Len(t, yara.Signatures, 2)
	assert.Len(t, yara.FilePaths, 2)
}

func TestImportDecorators(t *testing.T) {
	cfg := &kolide.ImportConfig{
		Decorators: &kolide.DecoratorConfig{
			Load: []string{
				"select from foo",
				"select from bar",
			},
			Always: []string{
				"select from always",
			},
			Interval: map[string][]string{
				"100": []string{
					"select from 100",
				},
				"200": []string{
					"select from 200",
				},
			},
		},
	}
	resp := &kolide.ImportConfigResponse{
		ImportStatusBySection: make(map[kolide.ImportSection]*kolide.ImportStatus),
	}
	svc := createServiceMockForImport(t)
	tx, _ := svc.ds.Begin()
	err := svc.importDecorators(cfg, resp, tx)
	require.Nil(t, err)
	assert.Equal(t, 5, resp.Status(kolide.DecoratorsSection).ImportCount)
	dec, err := svc.ds.ListDecorators()
	require.Nil(t, err)
	assert.Len(t, dec, 5)
}

func TestImportScheduledQueries(t *testing.T) {
	cfg := &kolide.ImportConfig{
		Schedule: kolide.QueryNameToQueryDetailsMap{
			"q1": kolide.QueryDetails{
				Query:    "select pid from processes",
				Interval: 60,
				Platform: stringPtr("linux"),
			},
			"q2": kolide.QueryDetails{
				Query:    "select uid from users",
				Interval: 120,
				Platform: stringPtr("linux"),
				Version:  stringPtr("1.0"),
			},
			"q3": kolide.QueryDetails{
				Query:    "select name from os",
				Interval: 240,
				Platform: stringPtr("linux"),
				Snapshot: boolPtr(true),
			},
		},
	}
	resp := &kolide.ImportConfigResponse{
		ImportStatusBySection: make(map[kolide.ImportSection]*kolide.ImportStatus),
	}
	svc := createServiceMockForImport(t)
	user := &kolide.User{
		Username: "bob",
		Password: []byte("secret"),
		Email:    "bob@something.com",
		Admin:    false,
		AdminForcedPasswordReset: false,
	}
	user, err := svc.ds.NewUser(user)
	require.Nil(t, err)
	skipQuery := &kolide.Query{
		Name:        "q3",
		Query:       "select version from os",
		Description: "should be skipped",
		Saved:       true,
		AuthorID:    user.ID,
	}
	_, err = svc.ds.NewQuery(skipQuery)
	require.Nil(t, err)
	noskipQuery := &kolide.Query{
		Name:     "q2",
		Query:    "select uid from users",
		Saved:    true,
		AuthorID: user.ID,
	}
	_, err = svc.ds.NewQuery(noskipQuery)
	require.Nil(t, err)
	tx, _ := svc.ds.Begin()
	err = svc.importScheduledQueries(user.ID, cfg, resp, tx)
	require.Nil(t, err)
	_, ok, err := svc.ds.QueryByName("q1")
	require.Nil(t, err)
	require.True(t, ok)
	_, ok, err = svc.ds.QueryByName("q2")
	require.Nil(t, err)
	require.True(t, ok)
	_, ok, err = svc.ds.QueryByName("q3")
	require.Nil(t, err)
	require.True(t, ok)

}

func TestOptionsImportConfig(t *testing.T) {
	opts := kolide.OptionNameToValueMap{
		"aws_access_key_id": "foo",
	}
	resp := &kolide.ImportConfigResponse{
		ImportStatusBySection: make(map[kolide.ImportSection]*kolide.ImportStatus),
	}
	svc := createServiceMockForImport(t)
	tx, _ := svc.ds.Begin()
	err := svc.importOptions(opts, resp, tx)
	require.Nil(t, err)
	status := resp.Status(kolide.OptionsSection)
	require.NotNil(t, status)
	assert.Equal(t, 1, status.ImportCount)
	opt, err := svc.ds.OptionByName("aws_access_key_id")
	require.Nil(t, err)
	assert.Equal(t, "foo", opt.GetValue())
	require.Len(t, status.Messages, 1)
	assert.Equal(t, "set aws_access_key_id value to foo", status.Messages[0])
}

func TestOptionsImportConfigWithSkips(t *testing.T) {
	opts := kolide.OptionNameToValueMap{
		"aws_access_key_id":     "foo",
		"aws_secret_access_key": "secret",
		// this should be skipped because it's already set
		"aws_firehose_period": 100,
		// these should be skipped because it's read only
		"disable_distributed": false,
		"pack_delimiter":      "x",
		// this should be skipped because it's not an option we know about
		"wombat": "not venomous",
	}
	resp := &kolide.ImportConfigResponse{
		ImportStatusBySection: make(map[kolide.ImportSection]*kolide.ImportStatus),
	}
	svc := createServiceMockForImport(t)
	tx, _ := svc.ds.Begin()
	// set option val, it should be skipped
	opt, err := svc.ds.OptionByName("aws_firehose_period")
	require.Nil(t, err)
	opt.SetValue(23)
	err = svc.ds.SaveOptions([]kolide.Option{*opt})
	require.Nil(t, err)
	err = svc.importOptions(opts, resp, tx)
	require.Nil(t, err)
	status := resp.Status(kolide.OptionsSection)
	require.NotNil(t, status)
	assert.Equal(t, 2, status.ImportCount)
	assert.Equal(t, 4, status.SkipCount)
	assert.Len(t, status.Warnings[kolide.OptionAlreadySet], 1)
	assert.Len(t, status.Warnings[kolide.OptionReadonly], 2)
	assert.Len(t, status.Warnings[kolide.OptionUnknown], 1)
	assert.Len(t, status.Messages, 2)
}

func TestPacksImportConfig(t *testing.T) {
	svc := createServiceMockForImport(t)
	tx, _ := svc.ds.Begin()

	p := &kolide.Pack{
		Name: "dup",
	}
	_, err := svc.ds.NewPack(p)
	require.Nil(t, err)

	q1 := kolide.QueryDetails{
		Query:    "select * from foo",
		Interval: 100,
		Removed:  boolPtr(false),
		Platform: stringPtr("linux"),
		Version:  stringPtr("1.0"),
	}
	q2 := kolide.QueryDetails{
		Query:    "select * from bar",
		Interval: 50,
		Removed:  boolPtr(false),
		Platform: stringPtr("linux"),
		Version:  stringPtr("1.0"),
	}
	q3 := kolide.QueryDetails{
		Query:    "select * from baz",
		Interval: 500,
		Removed:  boolPtr(false),
		Platform: stringPtr("linux"),
		Version:  stringPtr("1.0"),
	}

	importConfig := kolide.ImportConfig{
		Packs: kolide.PackNameMap{
			"ext1": "/home/usr/ext1.json",
			"pack1": kolide.PackDetails{
				Queries: kolide.QueryNameToQueryDetailsMap{
					"q1": q1,
					"q2": q2,
				},
				Discovery: []string{
					"select * from zz",
					"select id, xx from yy",
				},
			},
			"dup": kolide.PackDetails{
				Queries: kolide.QueryNameToQueryDetailsMap{
					"q1": q1,
					"q2": q2,
				},
			},
			"*": "/home/usr/packs/*",
		},
		ExternalPacks: kolide.PackNameToPackDetails{
			"ext1": kolide.PackDetails{
				Queries: kolide.QueryNameToQueryDetailsMap{
					"q1": q1,
				},
				Discovery: []string{
					"select * from zz",
					"select a, b, c from processes",
				},
			},
			"ext2": kolide.PackDetails{
				Queries: kolide.QueryNameToQueryDetailsMap{
					"q3": q3,
				},
			},
		},
		GlobPackNames: []string{"ext2"},
	}
	resp := &kolide.ImportConfigResponse{
		ImportStatusBySection: make(map[kolide.ImportSection]*kolide.ImportStatus),
	}
	user := &kolide.User{
		Username: "bob",
		Password: []byte("secret"),
		Email:    "bob@something.com",
		Admin:    false,
		AdminForcedPasswordReset: false,
	}
	user, err = svc.ds.NewUser(user)
	require.Nil(t, err)

	packs, err := importConfig.CollectPacks()
	require.Nil(t, err)
	assert.Len(t, packs, 4)
	err = svc.importPacks(user.ID, &importConfig, resp, tx)
	require.Nil(t, err)
	queries, err := svc.ds.ListQueries(kolide.ListOptions{})
	require.Nil(t, err)
	assert.Len(t, queries, 3)
	pack, ok, err := svc.ds.PackByName("pack1")
	require.Nil(t, err)
	require.True(t, ok)
	sqs, err := svc.ds.ListScheduledQueriesInPack(pack.ID, kolide.ListOptions{})
	require.Nil(t, err)
	assert.Len(t, sqs, 2)
	labels, err := svc.ds.ListLabels(kolide.ListOptions{})
	require.Nil(t, err)
	assert.Len(t, labels, 8)
	assert.Equal(t, 3, resp.Status(kolide.PacksSection).ImportCount)
	assert.Equal(t, 1, resp.Status(kolide.PacksSection).SkipCount)
	assert.Equal(t, 3, resp.Status(kolide.QueriesSection).ImportCount)
}
