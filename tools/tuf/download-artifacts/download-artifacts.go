package main

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/google/go-github/v37/github"
	"github.com/urfave/cli/v2"
)

func main() {
	app := cli.NewApp()
	app.Name = "download-artifacts"
	app.Usage = "CLI to download TUF artifacts from Github Actions"
	app.Commands = []*cli.Command{
		orbitCommand(),
		desktopCommand(),
		osquerydCommand(),
	}
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stdout, "Error: %+v\n", err)
		os.Exit(1)
	}
}

func orbitCommand() *cli.Command {
	var (
		gitTag          string
		outputDirectory string
		githubUsername  string
		githubAPIToken  string
		retry           bool
	)
	return &cli.Command{
		Name:  "orbit",
		Usage: "Fetch orbit executables from the goreleaser-orbit.yaml action",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "git-tag",
				EnvVars:     []string{"DOWNLOAD_ARTIFACTS_GIT_TAG"},
				Required:    true,
				Destination: &gitTag,
				Usage:       "git tag generated for the orbit release",
			},
			&cli.StringFlag{
				Name:        "output-directory",
				EnvVars:     []string{"DOWNLOAD_ARTIFACTS_OUTPUT_DIRECTORY"},
				Required:    true,
				Destination: &outputDirectory,
				Usage:       "name of the output directory to create and download the orbit executables",
			},
			&cli.StringFlag{
				Name:        "github-username",
				EnvVars:     []string{"DOWNLOAD_ARTIFACTS_GITHUB_USERNAME"},
				Required:    true,
				Destination: &githubUsername,
				Usage:       "Github username",
			},
			&cli.StringFlag{
				Name:        "github-api-token",
				EnvVars:     []string{"DOWNLOAD_ARTIFACTS_GITHUB_API_TOKEN"},
				Required:    true,
				Destination: &githubAPIToken,
				Usage:       "Github API token (https://github.com/settings/tokens)",
			},
			&cli.BoolFlag{
				Name:        "retry",
				EnvVars:     []string{"DOWNLOAD_ARTIFACTS_RETRY"},
				Destination: &retry,
				Usage:       "Whether to retry if the artifact doesn't exist yet",
			},
		},
		Action: func(c *cli.Context) error {
			return downloadComponents("goreleaser-orbit.yaml", gitTag, map[string]string{
				"macos":       "orbit-macos",
				"linux":       "orbit-linux",
				"linux-arm64": "orbit-linux-arm64",
				"windows":     "orbit-windows",
			}, outputDirectory, githubUsername, githubAPIToken, retry)
		},
	}
}

func desktopCommand() *cli.Command {
	var (
		gitBranch       string
		outputDirectory string
		githubUsername  string
		githubAPIToken  string
		retry           bool
	)
	return &cli.Command{
		Name:  "desktop",
		Usage: "Fetch Fleet Desktop executables from the generate-desktop-targets.yml action",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "git-branch",
				EnvVars:     []string{"DOWNLOAD_ARTIFACTS_GIT_BRANCH"},
				Required:    true,
				Destination: &gitBranch,
				Usage:       "branch name used to bump the Fleet Desktop version",
			},
			&cli.StringFlag{
				Name:        "output-directory",
				EnvVars:     []string{"DOWNLOAD_ARTIFACTS_OUTPUT_DIRECTORY"},
				Required:    true,
				Destination: &outputDirectory,
				Usage:       "name of the output directory to create and download the Fleet Desktop executables",
			},
			&cli.StringFlag{
				Name:        "github-username",
				EnvVars:     []string{"DOWNLOAD_ARTIFACTS_GITHUB_USERNAME"},
				Required:    true,
				Destination: &githubUsername,
				Usage:       "Github username",
			},
			&cli.StringFlag{
				Name:        "github-api-token",
				EnvVars:     []string{"DOWNLOAD_ARTIFACTS_GITHUB_API_TOKEN"},
				Required:    true,
				Destination: &githubAPIToken,
				Usage:       "Github API token (https://github.com/settings/tokens)",
			},
			&cli.BoolFlag{
				Name:        "retry",
				EnvVars:     []string{"DOWNLOAD_ARTIFACTS_RETRY"},
				Destination: &retry,
				Usage:       "Whether to retry if the artifact doesn't exist yet",
			},
		},
		Action: func(c *cli.Context) error {
			return downloadComponents("generate-desktop-targets.yml", gitBranch, map[string]string{
				"macos":       "desktop.app.tar.gz",
				"linux":       "desktop.tar.gz",
				"linux-arm64": "desktop-arm64.tar.gz",
				"windows":     "fleet-desktop.exe",
			}, outputDirectory, githubUsername, githubAPIToken, retry)
		},
	}
}

func downloadAndExtractZip(client *http.Client, githubUsername string, githubAPIToken string, urlPath string, destPath string) error {
	zipFile, err := os.CreateTemp("", "file.zip")
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer zipFile.Close()
	defer os.Remove(zipFile.Name())

	req, err := http.NewRequest(http.MethodGet, urlPath, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(githubUsername, githubAPIToken)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("could not download %s: %w", urlPath, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("could not download %s: received http status code %s", urlPath, resp.Status)
	}
	_, err = io.Copy(zipFile, resp.Body)
	if err != nil {
		return fmt.Errorf("could not write %s: %w", zipFile.Name(), err)
	}

	// Open the downloaded file for reading. With zip, we cannot unzip directly from resp.Body
	zipReader, err := zip.OpenReader(zipFile.Name())
	if err != nil {
		return fmt.Errorf("could not open %s: %w", zipFile.Name(), err)
	}
	defer zipReader.Close()

	err = os.MkdirAll(filepath.Dir(destPath), 0o755)
	if err != nil {
		return fmt.Errorf("could not create directory %s: %w", filepath.Dir(destPath), err)
	}

	// Extract each file in the archive
	for _, archiveReader := range zipReader.File {
		err = extractZipFile(archiveReader, destPath)
		if err != nil {
			return err
		}
	}

	return nil
}

func extractZipFile(archiveReader *zip.File, destPath string) error {
	if archiveReader.FileInfo().Mode()&os.ModeSymlink != 0 {
		// Skip symlinks for security reasons
		return nil
	}

	// Open the file in the archive
	archiveFile, err := archiveReader.Open()
	if err != nil {
		return fmt.Errorf("could not open archive %s: %w", archiveReader.Name, err)
	}
	defer archiveFile.Close()

	// Clean the archive path to prevent extracting files outside the destination.
	archivePath := filepath.Clean(archiveReader.Name)
	if strings.HasPrefix(archivePath, ".."+string(filepath.Separator)) {
		// Skip relative paths for security reasons
		return nil
	}
	// Prepare to write the file
	finalPath := filepath.Join(destPath, archivePath)

	// Check if the file to extract is just a directory
	if archiveReader.FileInfo().IsDir() {
		err = os.MkdirAll(finalPath, 0o755)
		if err != nil {
			return fmt.Errorf("could not create directory %s: %w", finalPath, err)
		}
	} else {
		// Create all needed directories
		if os.MkdirAll(filepath.Dir(finalPath), 0o755) != nil {
			return fmt.Errorf("could not create directory %s: %w", filepath.Dir(finalPath), err)
		}

		// Prepare to write the destination file
		destinationFile, err := os.OpenFile(finalPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, archiveReader.Mode())
		if err != nil {
			return fmt.Errorf("could not open file %s: %w", finalPath, err)
		}
		defer destinationFile.Close()

		// Write the destination file
		// Ignoring gosec's G110 warning as these are artifacts downloaded from Fleet's Github.
		if _, err = io.Copy(destinationFile, archiveFile); err != nil { //nolint:gosec
			return fmt.Errorf("could not write file %s: %w", finalPath, err)
		}
	}
	return nil
}

func downloadComponents(workflowName string, headBranch string, artifactNames map[string]string, outputDirectory string, githubUsername string, githubAPIToken string, retry bool) error {
	if err := os.RemoveAll(outputDirectory); err != nil {
		return err
	}
	for _, osPath := range []string{"macos", "windows", "linux", "linux-arm64"} {
		if err := os.MkdirAll(filepath.Join(outputDirectory, osPath), constant.DefaultDirMode); err != nil {
			return err
		}
	}
	ctx := context.Background()
	var workflowRun *github.WorkflowRun
	gc := github.NewClient(fleethttp.NewClient())
	for {
		workflow, _, err := gc.Actions.GetWorkflowByFileName(ctx, "fleetdm", "fleet", workflowName)
		if err != nil {
			return err
		}
		workflowRuns, _, err := gc.Actions.ListWorkflowRunsByID(ctx, "fleetdm", "fleet", *workflow.ID, nil)
		if err != nil {
			return err
		}
		for _, wr := range workflowRuns.WorkflowRuns {
			if headBranch == *wr.HeadBranch {
				workflowRun = wr
				break
			}
		}
		if workflowRun != nil || !retry {
			break
		}
		fmt.Printf("Workflow not available yet, it might be queued, retrying in 60s...\n")
		time.Sleep(60 * time.Second)
	}
	if workflowRun == nil {
		return fmt.Errorf("workflow with tag %s not found", headBranch)
	}
	var urls map[string]string
	for {
		artifactList, _, err := gc.Actions.ListWorkflowRunArtifacts(ctx, "fleetdm", "fleet", *workflowRun.ID, nil)
		if err != nil {
			return err
		}
		urls = make(map[string]string)
		for _, artifact := range artifactList.Artifacts {
			switch {
			case *artifact.Name == artifactNames["linux"]:
				urls["linux"] = *artifact.ArchiveDownloadURL
			case *artifact.Name == artifactNames["linux-arm64"]:
				urls["linux-arm64"] = *artifact.ArchiveDownloadURL
			case *artifact.Name == artifactNames["macos"]:
				urls["macos"] = *artifact.ArchiveDownloadURL
			case *artifact.Name == artifactNames["windows"]:
				urls["windows"] = *artifact.ArchiveDownloadURL
			default:
				fmt.Printf("skipping artifact name: %q\n", *artifact.Name)
			}
		}
		if len(urls) == 4 || !retry {
			break
		}
		fmt.Printf("All artifacts are not available yet, the workflow might still be running, retrying in 60s...\n")
		time.Sleep(60 * time.Second)
	}
	if len(urls) != 4 {
		return fmt.Errorf("missing some artifact: %+v", urls)
	}
	for osName, downloadURL := range urls {
		outputDir := filepath.Join(outputDirectory, osName)
		fmt.Printf("Downloading and extracting %s into %s...\n", downloadURL, outputDir)
		if err := downloadAndExtractZip(fleethttp.NewClient(), githubUsername, githubAPIToken, downloadURL, outputDir); err != nil {
			return err
		}
	}
	return nil
}

func osquerydCommand() *cli.Command {
	var (
		gitBranch       string
		outputDirectory string
		githubUsername  string
		githubAPIToken  string
		retry           bool
	)
	return &cli.Command{
		Name:  "osqueryd",
		Usage: "Fetch osqueryd executables from the generate-osqueryd-targets.yml action",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "git-branch",
				EnvVars:     []string{"DOWNLOAD_ARTIFACTS_GIT_BRANCH"},
				Required:    true,
				Destination: &gitBranch,
				Usage:       "branch name used to bump the osqueryd version",
			},
			&cli.StringFlag{
				Name:        "output-directory",
				EnvVars:     []string{"DOWNLOAD_ARTIFACTS_OUTPUT_DIRECTORY"},
				Required:    true,
				Destination: &outputDirectory,
				Usage:       "name of the output directory to create and download the osqueryd executables",
			},
			&cli.StringFlag{
				Name:        "github-username",
				EnvVars:     []string{"DOWNLOAD_ARTIFACTS_GITHUB_USERNAME"},
				Required:    true,
				Destination: &githubUsername,
				Usage:       "Github username",
			},
			&cli.StringFlag{
				Name:        "github-api-token",
				EnvVars:     []string{"DOWNLOAD_ARTIFACTS_GITHUB_API_TOKEN"},
				Required:    true,
				Destination: &githubAPIToken,
				Usage:       "Github API token (https://github.com/settings/tokens)",
			},
			&cli.BoolFlag{
				Name:        "retry",
				EnvVars:     []string{"DOWNLOAD_ARTIFACTS_RETRY"},
				Destination: &retry,
				Usage:       "Whether to retry if the artifact doesn't exist yet",
			},
		},
		Action: func(c *cli.Context) error {
			return downloadComponents("generate-osqueryd-targets.yml", gitBranch, map[string]string{
				"macos":       "osqueryd.app.tar.gz",
				"linux":       "osqueryd",
				"linux-arm64": "osqueryd-arm64",
				"windows":     "osqueryd.exe",
			}, outputDirectory, githubUsername, githubAPIToken, retry)
		},
	}
}
