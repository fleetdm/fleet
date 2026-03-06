package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	all := flag.Bool("all", false, "Generate packages for all platforms")
	deb := flag.Bool("deb", false, "Generate package for Linux (DEB)")
	rpm := flag.Bool("rpm", false, "Generate package for Linux (RPM)")
	debARM := flag.Bool("deb-arm", false, "Generate package for Linux ARM64 (DEB)")
	rpmARM := flag.Bool("rpm-arm", false, "Generate package for Linux ARM64 (RPM)")
	windows := flag.Bool("windows", false, "Generate package for Windows")
	macos := flag.Bool("macos", false, "Generate package for macOS")

	desktop := flag.Bool("desktop", false, "Generate fleet desktop")

	fleetUrl := flag.String("fleet", "", "Fleet server URL (required)")
	tufUrl := flag.String("tuf", "", "TUF server URL (required)")
	enrollSecret := flag.String("enroll", "", "Enroll secret (required)")

	debug := flag.Bool("debug", true, "Enable debugging in generated package")
	insecure := flag.Bool("insecure", true, "Allow insecure connections")

	fresh := flag.Bool("fresh", false, "Always delete test_tuf directory if it exists")

	flag.Parse()

	if *fleetUrl == "" && *tufUrl == "" && *enrollSecret == "" {
		flag.Usage()
		return
	}

	if *fleetUrl == "" {
		fmt.Println("Missing fleet URL")
		os.Exit(1)
	}
	if *tufUrl == "" {
		fmt.Println("Missing TUF URL")
		os.Exit(1)
	}
	if *enrollSecret == "" {
		fmt.Println("Missing enroll secret")
		os.Exit(1)
	}

	platforms := &Platforms{}

	systems := ""
	if *deb || *rpm || *all {
		systems += "linux "
		platforms.Linux = true
	}
	if *debARM || *rpmARM || *all {
		systems += "linux-arm64 "
		platforms.LinuxARM = true
	}
	if *windows || *all {
		systems += "windows "
		platforms.Windows = true
	}
	if *macos || *all {
		systems += "macos "
		platforms.Darwin = true
	}

	cmd := exec.Command("bash", "./tools/tuf/test/main.sh")
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	copyEnvStd(cmd)

	addEnv(cmd, "SYSTEMS", strings.TrimSpace(systems))
	addEnv(cmd, "ENROLL_SECRET", *enrollSecret)
	if *insecure {
		addEnv(cmd, "INSECURE", "1")
	}
	if *debug {
		addEnv(cmd, "DEBUG", "1")
	}
	if *desktop {
		addEnv(cmd, "FLEET_DESKTOP", "1")
	}

	if *deb || *all {
		addEnv(cmd, "DEB_FLEET_URL", *fleetUrl)
		addEnv(cmd, "DEB_TUF_URL", *tufUrl)
		addEnv(cmd, "GENERATE_DEB", "1")
	}

	if *rpm || *all {
		addEnv(cmd, "RPM_FLEET_URL", *fleetUrl)
		addEnv(cmd, "RPM_TUF_URL", *tufUrl)
		addEnv(cmd, "GENERATE_RPM", "1")
	}

	if *debARM || *all {
		addEnv(cmd, "DEB_FLEET_URL", *fleetUrl)
		addEnv(cmd, "DEB_TUF_URL", *tufUrl)
		addEnv(cmd, "GENERATE_DEB_ARM64", "1")
	}

	if *rpmARM || *all {
		addEnv(cmd, "RPM_FLEET_URL", *fleetUrl)
		addEnv(cmd, "RPM_TUF_URL", *tufUrl)
		addEnv(cmd, "GENERATE_RPM_ARM64", "1")
	}

	if *macos || *all {
		addEnv(cmd, "PKG_FLEET_URL", *fleetUrl)
		addEnv(cmd, "PKG_TUF_URL", *tufUrl)
		addEnv(cmd, "GENERATE_PKG", "1")
	}

	if *windows || *all {
		addEnv(cmd, "MSI_FLEET_URL", *fleetUrl)
		addEnv(cmd, "MSI_TUF_URL", *tufUrl)
		addEnv(cmd, "GENERATE_MSI", "1")
	}

	fmt.Println("Changing to repo root")
	if err := chdirGitRoot(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	scanner := bufio.NewScanner(os.Stdin)

	if _, err := os.Stat("test_tuf"); err == nil {
		if !*fresh {
			fmt.Printf("test_tuf exists, delete? [y/N] ")
		}
		scanner.Scan()
		deleteTuf := len(scanner.Text()) != 0 && (scanner.Text()[0] == 'y' || scanner.Text()[0] == 'Y')

		if *fresh || deleteTuf {
			if err := os.RemoveAll("test_tuf"); err != nil {
				fmt.Printf("failed to remote test_tuf: %s\n", err)
				os.Exit(1)
			}
		}
	}

	if err := cmd.Run(); err != nil {
		fmt.Printf("failed to build repo: %s\n", err)
		os.Exit(1)
	}

	fmt.Println("==== Packages built ====")

	for {
		fmt.Println("\nPress enter to rebuild orbit")
		if !scanner.Scan() {
			fmt.Println("exiting")
			os.Exit(0)
		}
		fmt.Println("Rebuiding orbit")
		if err := pushOrbit(platforms); err != nil {
			fmt.Printf("Failed to rebuild orbit: %s\n", err)
			os.Exit(1)
		}
	}
}

func addEnv(cmd *exec.Cmd, key string, val string) {
	cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, val))
}

func copyEnv(cmd *exec.Cmd, key string) {
	addEnv(cmd, key, os.Getenv(key))
}

func copyEnvStd(cmd *exec.Cmd) {
	copyEnv(cmd, "PATH")
	copyEnv(cmd, "GOPATH")
	copyEnv(cmd, "HOME")
}

func chdirGitRoot() error {
	topLevel, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return fmt.Errorf("getting repo root: %w\n", err)
	}

	if err := os.Chdir(strings.TrimSpace(string(topLevel))); err != nil {
		return fmt.Errorf("changing to repo root: %w\n", err)
	}

	return nil
}

type Platforms struct {
	Linux    bool
	LinuxARM bool
	Windows  bool
	Darwin   bool
}

func pushOrbit(platforms *Platforms) error {
	if err := chdirGitRoot(); err != nil {
		return fmt.Errorf("changing to repo root to push orbit %w", err)
	}

	// Build the orbits
	if platforms.Linux {
		if err := buildOrbit("orbit-linux", "linux", "amd64"); err != nil {
			return fmt.Errorf("push orbit: %w", err)
		}
	}

	if platforms.LinuxARM {
		if err := buildOrbit("orbit-linux-arm64", "linux", "arm64"); err != nil {
			return fmt.Errorf("push orbit: %w", err)
		}
	}

	if platforms.Windows {
		if err := buildOrbit("orbit-windows.exe", "windows", "amd64"); err != nil {
			return fmt.Errorf("push orbit: %w", err)
		}
	}

	if platforms.Darwin {
		if err := buildOrbit("orbit-darwin", "darwin", "amd64"); err != nil {
			return fmt.Errorf("push orbit: %w", err)
		}
	}

	// Push orbits to the tuf server
	if platforms.Linux {
		if err := pushTarget("linux", "orbit", "orbit-linux", "42"); err != nil {
			return fmt.Errorf("pushing binary: %w", err)
		}
	}

	if platforms.LinuxARM {
		if err := pushTarget("linux-arm64", "orbit", "orbit-linux-arm64", "42"); err != nil {
			return fmt.Errorf("pushing binary: %w", err)
		}
	}

	if platforms.Windows {
		if err := pushTarget("windows", "orbit", "orbit-windows.exe", "42"); err != nil {
			return fmt.Errorf("pushing binary: %w", err)
		}
	}

	if platforms.Darwin {
		if err := pushTarget("darwin", "orbit", "orbit-darwin", "42"); err != nil {
			return fmt.Errorf("pushing binary: %w", err)
		}
	}

	return nil
}

func buildOrbit(output, goos, goarch string) error {
	cmd := exec.Command("go", "build", "-v", "-o", output, "./orbit/cmd/orbit")
	addEnv(cmd, "GOOS", goos)
	addEnv(cmd, "GOARCH", goarch)
	copyEnvStd(cmd)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	fmt.Printf("building orbit %s-%s\n", goos, goarch)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("compiling orbit for %s-%s: %w", goos, goarch, err)
	}

	return nil
}

func pushTarget(platform, component, binary, version string) error {
	cmd := exec.Command("./tools/tuf/test/push_target.sh", platform, component, binary, version)
	copyEnvStd(cmd)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("pushing %s %s %s\n", component, platform, version)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pushing %s %s (%s) with binary %s: %w", component, version, platform, binary, err)
	}

	return nil
}
