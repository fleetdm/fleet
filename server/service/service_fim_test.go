package service

import (
	"context"
	"testing"

	"github.com/kolide/fleet/server/kolide"
	"github.com/kolide/fleet/server/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetFIMService(t *testing.T) {
	fileAccessesString := "[\"etc\", \"home\", \"cassandra\"]"
	fileAccessStringValue := []string{"etc", "home", "cassandra"}
	fimIntervalTestValue := 500 //300 is the default value

	ds := &mock.Store{
		AppConfigStore: mock.AppConfigStore{
			AppConfigFunc: func() (*kolide.AppConfig, error) {
				config := &kolide.AppConfig{
					FIMInterval: fimIntervalTestValue,
					FIMFileAccesses: fileAccessesString,
				}
				return config, nil
			},
		},
		FileIntegrityMonitoringStore: mock.FileIntegrityMonitoringStore{
			FIMSectionsFunc: func() (kolide.FIMSections, error) {
				result := kolide.FIMSections{
					"etc": []string{
						"/etc/config/%%",
						"/etc/zipp",
					},
				}
				return result, nil
			},
		},
	}
	svc := service{
		ds: ds,
	}
	resp, err := svc.GetFIM(context.Background())
	require.Nil(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, resp.Interval, uint(fimIntervalTestValue))
	assert.Equal(t, resp.FileAccesses, fileAccessStringValue)
	paths, ok := resp.FilePaths["etc"]
	require.True(t, ok)
	assert.Len(t, paths, 2)
}

func TestUpdateFIM(t *testing.T) {
	fileAccessesString := "[\"etc\", \"home\", \"cassandra\"]"
	fileAccessStringValue := []string{"etc", "home", "cassandra"}
	fimIntervalTestValue := 500 //300 is the default value

	ds := &mock.Store{
		AppConfigStore: mock.AppConfigStore{
			AppConfigFunc: func() (*kolide.AppConfig, error) {
				config := &kolide.AppConfig{
					FIMInterval: fimIntervalTestValue,
					FIMFileAccesses: fileAccessesString,
				}
				return config, nil
			},
			SaveAppConfigFunc: func(_ *kolide.AppConfig) error {
				return nil
			},
		},
		FileIntegrityMonitoringStore: mock.FileIntegrityMonitoringStore{
			ClearFIMSectionsFunc: func() error {
				return nil
			},
			NewFIMSectionFunc: func(fs *kolide.FIMSection, _ ...kolide.OptionalArg) (*kolide.FIMSection, error) {
				fs.ID = 1
				return fs, nil
			},
		},
	}
	svc := service{
		ds: ds,
	}
	fim := kolide.FIMConfig{
		Interval: uint(fimIntervalTestValue),
		FileAccesses: fileAccessStringValue,
		FilePaths: kolide.FIMSections{
			"etc": []string{
				"/etc/config/%%",
				"/etc/zipp",
			},
		},
	}
	err := svc.ModifyFIM(context.Background(), fim)
	require.Nil(t, err)
	assert.True(t, ds.NewFIMSectionFuncInvoked)
	assert.True(t, ds.ClearFIMSectionsFuncInvoked)
	assert.True(t, ds.AppConfigFuncInvoked)
	assert.True(t, ds.SaveAppConfigFuncInvoked)

}
