package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/s3"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/urfave/cli/v2"
)

func main() {
	app := cli.NewApp()
	app.Name = "installerstore"
	app.Usage = "Utility to upload pre-built installers to a file storage (AWS S3, MinIO, etc.)"
	app.UsageText = "installerstore --enroll-secret=xyz --bucket=installers ~/path/to/file.pkg"
	app.ExitErrHandler = exitErrHandler
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:    "fleet-desktop",
			Usage:   "Whether or not the installer includes Fleet Desktop",
			EnvVars: []string{"INSTALLER_FLEET_DESKTOP"},
		},
		&cli.StringFlag{
			Name:     "enroll-secret",
			Usage:    "Enroll secret associated with the installer",
			EnvVars:  []string{"INSTALLER_ENROLL_SECRET"},
			Required: true,
		},
		&cli.StringFlag{
			Name:     "bucket",
			Usage:    "Bucket where to store installers",
			EnvVars:  []string{"INSTALLER_BUCKET"},
			Required: true,
		},
		&cli.StringFlag{
			Name:    "prefix",
			Usage:   "Prefix under which installers are stored",
			EnvVars: []string{"INSTALLER_PREFIX"},
		},
		&cli.StringFlag{
			Name:    "region",
			Usage:   "AWS Region (if blank region is derived)",
			EnvVars: []string{"INSTALLER_REGION"},
		},
		&cli.StringFlag{
			Name:    "endpoint-url",
			Usage:   "AWS Service Endpoint to use (leave blank for default service endpoints)",
			EnvVars: []string{"INSTALLER_ENDPOINT_URL"},
		},
		&cli.StringFlag{
			Name:    "access-key-id",
			Usage:   "Access Key ID for AWS authentication",
			EnvVars: []string{"INSTALLER_ACCESS_KEY_ID"},
		},
		&cli.StringFlag{
			Name:    "secret-access-key",
			Usage:   "Secret Access Key for AWS authentication",
			EnvVars: []string{"INSTALLER_SECRET_ACCESS_KEY"},
		},
		&cli.StringFlag{
			Name:    "sts-assume-role-arn",
			Usage:   "ARN of role to assume for AWS",
			EnvVars: []string{"INSTALLER_STS_ASSUME_ROLE_ARN"},
		},
		&cli.BoolFlag{
			Name:    "disable-ssl",
			Usage:   "Disable SSL (typically for local testing)",
			EnvVars: []string{"INSTALLER_DISABLE_SSL"},
		},
		&cli.BoolFlag{
			Name:    "force-s3-path-style",
			Usage:   "Set this to true to force path-style addressing, i.e., `http://s3.amazonaws.com/BUCKET/KEY`",
			EnvVars: []string{"INSTALLER_FORCE_S3_PATH_STYLE"},
		},
		&cli.BoolFlag{
			Name:    "create-bucket",
			Usage:   "Set this to true to create the bucket if it doesn't exist. Only recommended for local testing.",
			EnvVars: []string{"INSTALLER_CREATE_BUCKET"},
		},
	}

	app.Action = func(c *cli.Context) error {
		bucket := c.String("bucket")
		store, err := s3.NewInstallerStore(config.S3Config{
			Bucket:           bucket,
			Prefix:           c.String("prefix"),
			Region:           c.String("region"),
			EndpointURL:      c.String("endpoint-url"),
			AccessKeyID:      c.String("access-key-id"),
			SecretAccessKey:  c.String("secret-access-key"),
			StsAssumeRoleArn: c.String("sts-assume-role-arn"),
			DisableSSL:       c.Bool("disable-ssl"),
			ForceS3PathStyle: c.Bool("force-s3-path-style"),
		})
		if err != nil {
			return fmt.Errorf("unable to setup store: %w", err)
		}

		if c.Bool("create-bucket") {
			if err := store.CreateTestBucket(bucket); err != nil {
				return fmt.Errorf("unable to create bucket: %w", err)
			}
		}

		fp := c.Args().Get(0)
		if fp == "" {
			return errors.New("please provide an input file")
		}

		r, err := os.Open(fp)
		if err != nil {
			return fmt.Errorf("there was an error opening %s: %w", fp, err)
		}

		key, err := store.Put(context.Background(), fleet.Installer{
			EnrollSecret: c.String("enroll-secret"),
			Kind:         filepath.Ext(fp)[1:],
			Desktop:      c.Bool("fleet-desktop"),
			Content:      r,
		})
		if err != nil {
			return fmt.Errorf("there was a problem uploading the installer with key %s: %w", key, err)
		}

		fmt.Printf("installer uploaded to bucket '%s' with key '%s'\n", bucket, key)
		return nil
	}

	app.Run(os.Args) //nolint:errcheck
}

func exitErrHandler(c *cli.Context, err error) {
	if err == nil {
		return
	}
	fmt.Fprintf(c.App.ErrWriter, "Error: %+v\n", err)
	cli.OsExiter(1)
}
