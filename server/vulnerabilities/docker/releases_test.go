package docker

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func getFixtureServer(t *testing.T) *httptest.Server {
	fixturePath := filepath.Join("..", "testdata", "docker", "docker_desktop_current_release_notes.html")
	fixture, err := ioutil.ReadFile(fixturePath)
	require.NoError(t, err)

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == releaseURLs[0] {
			w.WriteHeader(http.StatusNotModified)
			_, err := w.Write(fixture)
			require.NoError(t, err)
		}
	}))
}

func TestReleaseClient(t *testing.T) {
	t.Run("#GetReleases", func(t *testing.T) {
		t.Run("current release", func(t *testing.T) {
			server := getFixtureServer(t)
			t.Cleanup(server.Close)

			expected := []DesktopRelease{
				{
					Version:   "4.16.3",
					Date:      "2023-01-30",
					Platforms: []string{"Windows"},
				},
				{
					Version:   "4.16.2",
					Date:      "2023-01-19",
					Platforms: []string{"Windows", "Mac with Intel chip", "Mac with Apple chip", "Debian", "RPM", "Arch package"},
				},
				{
					Version:   "4.16.1",
					Date:      "2023-01-13",
					Platforms: []string{"Windows", "Mac with Intel chip", "Mac with Apple chip", "Debian", "RPM", "Arch package"},
				},
				{
					Version:   "4.16.0",
					Date:      "2023-01-12",
					Platforms: []string{"Windows", "Mac with Intel chip", "Mac with Apple chip", "Debian", "RPM", "Arch package"},
					Updates: []Update{
						{
							ProductName:    "Compose",
							ProductVersion: "v2.15.1",
						},
						{
							ProductName:    "Containerd",
							ProductVersion: "v1.6.14",
						},
						{
							ProductName:    "Docker Engine",
							ProductVersion: "v20.10.22",
						},
						{
							ProductName:    "Buildx",
							ProductVersion: "v0.10.0",
						},
						{
							ProductName:    "Docker Scan",
							ProductVersion: "v0.23.0",
						},
						{
							ProductName:    "Go",
							ProductVersion: "1.19.4",
						},
					},
				},
				{
					Version:   "4.15.0",
					Date:      "2022-12-01",
					Platforms: []string{"Windows", "Mac with Intel chip", "Mac with Apple chip", "Debian", "RPM", "Arch package"},
					Updates: []Update{
						{
							ProductName:    "Compose",
							ProductVersion: "v2.13.0",
						},
						{
							ProductName:    "Containerd",
							ProductVersion: "v1.6.10",
						},
						{
							ProductName:    "Docker Hub Tool",
							ProductVersion: "v0.4.5",
						},
						{
							ProductName:    "Docker Scan",
							ProductVersion: "v0.22.0",
						},
					},
				},
				{
					Version:   "4.14.1",
					Date:      "2022-11-17",
					Platforms: []string{"Windows", "Mac with Intel chip", "Mac with Apple chip", "Debian", "RPM", "Arch package"},
				},
				{
					Version:         "4.14.0",
					Date:            "2022-11-10",
					Platforms:       []string{"Windows", "Mac with Intel chip", "Mac with Apple chip", "Debian", "RPM", "Arch package"},
					Vulnerabilities: []string{"CVE-2022-39253"},
					Updates: []Update{
						{
							ProductName:    "Docker Engine",
							ProductVersion: "v20.10.21",
						},
						{
							ProductName:    "Docker Compose",
							ProductVersion: "v2.12.2",
						},
						{
							ProductName:    "Containerd",
							ProductVersion: "v1.6.9",
						},
						{
							ProductName:    "Go",
							ProductVersion: "1.19.4",
						},
					},
				},
				{
					Version:   "4.13.1",
					Date:      "2022-10-31",
					Platforms: []string{"Windows", "Mac with Intel chip", "Mac with Apple chip", "Debian", "RPM", "Arch package"},
					Updates: []Update{
						{
							ProductName:    "Docker Compose",
							ProductVersion: "v2.12.1",
						},
					},
				},
				{
					Version:         "4.13.0",
					Date:            "2022-10-19",
					Platforms:       []string{"Windows", "Mac with Intel chip", "Mac with Apple chip", "Debian", "RPM", "Arch package"},
					Vulnerabilities: []string{"CVE-2022-2879", "CVE-2022-2880", "CVE-2022-41715", "CVE-2022-39253", "CVE-2022-36109"},
					Updates: []Update{
						{
							ProductName:    "Docker Scan",
							ProductVersion: "v0.21.0",
						},
						{
							ProductName:    "Go",
							ProductVersion: "1.19.2",
						},
						{
							ProductName:    "Docker Engine",
							ProductVersion: "v20.10.20",
						},
						{
							ProductName:    "Docker Credential Helpers",
							ProductVersion: "v0.7.0",
						},
						{
							ProductName:    "Docker Compose",
							ProductVersion: "v2.12.0",
						},
						{
							ProductName:    "Kubernetes",
							ProductVersion: "v1.25.2",
						},
						{
							ProductName:    "Qemu",
							ProductVersion: "7.0.0",
						},
						{
							ProductName:    "Linux kernel",
							ProductVersion: "5.15.49",
						},
					},
				},
				{
					Version:   "4.12.0",
					Date:      "2022-09-01",
					Platforms: []string{"Windows", "Mac with Intel chip", "Mac with Apple chip", "Debian", "RPM", "Arch package"},
					Updates: []Update{
						{
							ProductName:    "Docker Compose",
							ProductVersion: "v2.10.2",
						},
						{
							ProductName:    "Docker Scan",
							ProductVersion: "v0.19.0",
						},
						{
							ProductName:    "Kubernetes",
							ProductVersion: "v1.25.0",
						},
						{
							ProductName:    "Go",
							ProductVersion: "1.19",
						},
						{
							ProductName:    "cri-dockerd",
							ProductVersion: "v0.2.5",
						},
						{
							ProductName:    "Buildx",
							ProductVersion: "v0.9.1",
						},
						{
							ProductName:    "containerd",
							ProductVersion: "v1.6.8",
						},
						{
							ProductName:    "containerd",
							ProductVersion: "v1.6.7",
						},
						{
							ProductName:    "runc ",
							ProductVersion: "v1.1.4",
						},
						{
							ProductName:    "runc ",
							ProductVersion: "v1.1.3",
						},
					},
				},
				{
					Version:   "4.11.1",
					Date:      "2022-08-05",
					Platforms: []string{"Windows", "Mac with Intel chip", "Mac with Apple chip", "Debian", "RPM", "Arch package"},
				},
				{
					Version:   "4.11.0",
					Date:      "2022-07-28",
					Platforms: []string{"Windows", "Mac with Intel chip", "Mac with Apple chip", "Debian", "RPM", "Arch package"},
					Updates: []Update{
						{
							ProductName:    "Docker Compose",
							ProductVersion: "v2.7.0",
						},
						{
							ProductName:    "Docker Compose “Cloud Integrations”",
							ProductVersion: "v1.0.28",
						},
						{
							ProductName:    "Kubernetes",
							ProductVersion: "v1.24.2",
						},
						{
							ProductName:    "Go",
							ProductVersion: "1.18.4",
						},
					},
				},
				{
					Version:   "4.10.0",
					Date:      "2022-06-30",
					Platforms: []string{"Windows", "Mac with Intel chip", "Mac with Apple chip", "Debian", "RPM", "Arch package"},
					Updates: []Update{
						{
							ProductName:    "Docker Engine",
							ProductVersion: "v20.10.17",
						},
						{
							ProductName:    "Docker Compose",
							ProductVersion: "v2.6.1",
						},
						{ProductName: "Kubernetes", ProductVersion: "v1.24.1"},
						{
							ProductName:    "cri-dockerd",
							ProductVersion: "v0.2.1",
						},
						{
							ProductName:    "CNI plugins",
							ProductVersion: "v1.1.1",
						},
						{
							ProductName:    "containerd",
							ProductVersion: "v1.6.6",
						},
						{
							ProductName:    "runc",
							ProductVersion: "v1.1.2",
						},
						{
							ProductName:    "Go",
							ProductVersion: "1.18.3",
						},
					},
				},
				{
					Version:   "4.9.1",
					Date:      "2022-06-16",
					Platforms: []string{"Windows", "Mac with Intel chip", "Mac with Apple chip", "Debian", "RPM", "Arch package"},
				},
				{
					Version:   "4.9.0",
					Date:      "2022-06-02",
					Platforms: []string{"Windows", "Mac with Intel chip", "Mac with Apple chip", "Debian", "RPM", "Arch package"},
					Updates: []Update{
						{
							ProductName:    "Compose",
							ProductVersion: "v2.6.0",
						},
						{
							ProductName:    "Docker Engine",
							ProductVersion: "v20.10.16",
						},
						{
							ProductName:    "containerd",
							ProductVersion: "v1.6.4",
						},
						{
							ProductName:    "runc",
							ProductVersion: "v1.1.1",
						},
						{
							ProductName:    "Go",
							ProductVersion: "1.18.2",
						},
					},
				},
				{
					Version:   "4.8.2",
					Date:      "2022-05-18",
					Platforms: []string{"Windows", "Mac with Intel chip", "Mac with Apple chip", "Debian", "RPM", "Arch package"},
					Updates: []Update{
						{
							ProductName:    "Compose",
							ProductVersion: "v2.5.1",
						},
					},
				},
				{
					Version:   "4.8.1",
					Date:      "2022-05-09",
					Platforms: []string{"Windows", "Mac with Intel chip", "Mac with Apple chip", "Debian", "RPM", "Arch package"},
				},
				{
					Version:   "4.8.0",
					Date:      "2022-05-06",
					Platforms: []string{"Windows", "Mac with Intel chip", "Mac with Apple chip", "Debian", "RPM", "Arch package"},
					Updates: []Update{
						{
							ProductName:    "Compose",
							ProductVersion: "v2.5.0",
						},
						{
							ProductName:    "Go",
							ProductVersion: "1.18.1",
						},
						{
							ProductName:    "Kubernetes",
							ProductVersion: "1.24",
						},
					},
				},
				{
					Version:   "4.7.1",
					Date:      "2022-04-19",
					Platforms: []string{"Windows", "Mac with Intel chip", "Mac with Apple chip"},
				},
				{
					Version:         "4.7.0",
					Date:            "2022-04-07",
					Platforms:       []string{"Windows", "Mac with Intel chip", "Mac with Apple chip"},
					Vulnerabilities: []string{"CVE-2022-24769"},
					Updates: []Update{
						{
							ProductName:    "Docker Engine",
							ProductVersion: "v20.10.14",
						},
						{
							ProductName:    "Compose",
							ProductVersion: "v2.4.1",
						},
						{
							ProductName:    "Buildx",
							ProductVersion: "0.8.2",
						},
						{
							ProductName:    "containerd",
							ProductVersion: "v1.5.11",
						},
						{
							ProductName:    "Go",
							ProductVersion: "1.18",
						},
					},
				},
				{
					Version:   "4.6.1",
					Date:      "2022-03-22",
					Platforms: []string{"Windows", "Mac with Intel chip", "Mac with Apple chip"},
					Updates: []Update{
						{
							ProductName:    "Buildx",
							ProductVersion: "0.8.1",
						},
					},
				},
				{
					Version:         "4.6.0",
					Date:            "2022-03-14",
					Platforms:       []string{"Windows", "Mac with Intel chip", "Mac with Apple chip"},
					Vulnerabilities: []string{"CVE-2022-0847", "CVE-2022-26659:windows"},
					Updates: []Update{
						{
							ProductName:    "Docker Engine",
							ProductVersion: "v20.10.13",
						},
						{
							ProductName:    "Compose",
							ProductVersion: "v2.3.3",
						},
						{
							ProductName:    "Buildx",
							ProductVersion: "0.8.0",
						},
						{
							ProductName:    "containerd",
							ProductVersion: "v1.4.13",
						},
						{
							ProductName:    "runc",
							ProductVersion: "v1.0.3",
						},
						{
							ProductName:    "Go",
							ProductVersion: "1.17.8",
						},
						{
							ProductName:    "Linux kernel",
							ProductVersion: "5.10.104",
						},
						{
							ProductName:    "Buildx",
							ProductVersion: "0.8.1",
						},
						{
							ProductName:    "Qemu",
							ProductVersion: "6.2.0",
						},
					},
				},
				{
					Version:   "4.5.1",
					Date:      "2022-02-15",
					Platforms: []string{"Windows"},
				},
				{
					Version:         "4.5.0",
					Date:            "2022-02-10",
					Platforms:       []string{"Mac with Intel chip", "Mac with Apple chip"},
					Vulnerabilities: []string{"CVE-2021-44719:mac", "CVE-2022-23774:windows"},
					Updates: []Update{
						{
							ProductName:    "Amazon ECR Credential Helper",
							ProductVersion: "v0.6.0",
						},
					},
				},
				{
					Version:   "4.4.4",
					Date:      "2022-01-24",
					Platforms: []string{"Mac with Intel chip", "Mac with Apple chip"},
				},
				{
					Version:   "4.4.3",
					Date:      "2022-01-14",
					Platforms: []string{"Windows"},
				},
				{
					Version:         "4.4.2",
					Date:            "2022-01-13",
					Platforms:       []string{"Windows", "Mac with Intel chip", "Mac with Apple chip"},
					Vulnerabilities: []string{"CVE-2021-45449"},
					Updates: []Update{
						{
							ProductName:    "Docker Engine",
							ProductVersion: "v20.10.12",
						},
						{
							ProductName:    "Compose",
							ProductVersion: "v2.2.3",
						},
						{
							ProductName:    "Kubernetes",
							ProductVersion: "1.22.5",
						},
						{
							ProductName:    "docker scan",
							ProductVersion: "v0.16.0",
						},
					},
				},
				{
					Version:         "4.3.2",
					Date:            "2021-12-21",
					Platforms:       []string{"Windows", "Mac with Intel chip", "Mac with Apple chip"},
					Vulnerabilities: []string{"CVE-2021-45449"},
					Updates: []Update{
						{
							ProductName:    "docker scan",
							ProductVersion: "v0.14.0",
						},
					},
				},
				{
					Version:   "4.3.1",
					Date:      "2021-12-11",
					Platforms: []string{"Windows", "Mac with Intel chip", "Mac with Apple chip"},
					Updates: []Update{
						{
							ProductName:    "docker scan",
							ProductVersion: "v0.11.0",
						},
					},
				},
				{
					Version:   "4.3.0",
					Date:      "2021-12-02",
					Platforms: []string{"Windows", "Mac with Intel chip", "Mac with Apple chip"},
					Updates: []Update{
						{
							ProductName:    "Docker Engine",
							ProductVersion: "v20.10.11",
						},
						{
							ProductName:    "containerd",
							ProductVersion: "v1.4.12",
						},

						{
							ProductName:    "Buildx",
							ProductVersion: "0.7.1",
						},
						{
							ProductName:    "Compose",
							ProductVersion: "v2.2.1",
						},
						{
							ProductName:    "Kubernetes",
							ProductVersion: "1.22.4",
						},
						{
							ProductName:    "Docker Hub Tool",
							ProductVersion: "v0.4.4",
						},
						{
							ProductName:    "Go",
							ProductVersion: "1.17.3",
						},
					},
				},
				{
					Version:   "4.2.0",
					Date:      "2021-11-09",
					Platforms: []string{"Windows", "Mac with Intel chip", "Mac with Apple chip"},
					Updates: []Update{
						{
							ProductName:    "Docker Engine",
							ProductVersion: "v20.10.10",
						},
						{
							ProductName:    "containerd",
							ProductVersion: "v1.4.11",
						},
						{
							ProductName:    "runc",
							ProductVersion: "v1.0.2",
						},
						{
							ProductName:    "Go",
							ProductVersion: "1.17.2",
						},
						{
							ProductName:    "Compose",
							ProductVersion: "v2.1.1",
						},
						{
							ProductName:    "docker-scan",
							ProductVersion: "0.9.0",
						},
					},
				},
				{
					Version:   "4.1.1",
					Date:      "2021-10-12",
					Platforms: []string{"Windows", "Mac with Intel chip", "Mac with Apple chip"},
				},
				{
					Version:   "4.1.0",
					Date:      "2021-09-30",
					Platforms: []string{"Windows", "Mac with Intel chip", "Mac with Apple chip"},
					Updates: []Update{
						{
							ProductName:    "Compose",
							ProductVersion: "V2",
						},
						{
							ProductName:    "Buildx",
							ProductVersion: "0.6.3",
						},
						{
							ProductName:    "Kubernetes",
							ProductVersion: "1.21.5",
						},
						{
							ProductName:    "Go",
							ProductVersion: "1.17.1",
						},
						{
							ProductName:    "Alpine",
							ProductVersion: "3.14",
						},
						{
							ProductName:    "Qemu",
							ProductVersion: "6.1.0",
						},
						{
							ProductName:    "Base distro to debian:bullseye",
							ProductVersion: "",
						},
					},
				},
				{
					Version:   "4.1.0",
					Date:      "2021-09-13",
					Platforms: []string{"Windows", "Mac with Intel chip", "Mac with Apple chip"},
					Updates: []Update{
						{
							ProductName:    "Compose",
							ProductVersion: "V2 RC3",
						},
					},
				},
				{
					Version:   "4.0.0",
					Date:      "2021-08-31",
					Platforms: []string{"Windows", "Mac with Intel chip", "Mac with Apple chip"},
					Updates: []Update{
						{
							ProductName:    "Compose",
							ProductVersion: "V2 RC2",
						},
						{
							ProductName:    "Kubernetes",
							ProductVersion: "1.21.4",
						},
					},
				},
			}

			sut := ReleaseClient{server.Client()}
			actual, err := sut.GetReleases(context.Background())
			require.NoError(t, err)

			require.Subset(t, actual, expected)
		})
	})
}
