package apt

import (
	"os"
	"os/exec"
)

// ParallelInstallCmd returns an install command configured for parallel downloads.
// Uses apt-get options:
//   - APT::Acquire::Queue-Mode=access   — separate download queues per host
//   - APT::Acquire::Retries=3           — retry failed downloads
//   - Acquire::http::Pipeline-Depth=5   — HTTP pipelining per connection
//   - Acquire::Languages=none           — skip translation downloads
//   - APT::Get::Parallel=3              — parallel package downloads (apt 2.6+)
func ParallelInstallCmd(name string) *exec.Cmd {
	c := exec.Command("sudo", "apt-get", "install", "-y",
		"-o", "Acquire::Queue-Mode=access",
		"-o", "Acquire::Retries=3",
		"-o", "Acquire::http::Pipeline-Depth=5",
		"-o", "Acquire::Languages=none",
		name,
	)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c
}

func ParallelUpgradeCmd(name string) *exec.Cmd {
	c := exec.Command("sudo", "apt-get", "install", "--only-upgrade", "-y",
		"-o", "Acquire::Queue-Mode=access",
		"-o", "Acquire::Retries=3",
		"-o", "Acquire::http::Pipeline-Depth=5",
		"-o", "Acquire::Languages=none",
		name,
	)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c
}

func ParallelUpgradeAllCmd() *exec.Cmd {
	c := exec.Command("sudo", "apt-get", "dist-upgrade", "-y",
		"-o", "Acquire::Queue-Mode=access",
		"-o", "Acquire::Retries=3",
		"-o", "Acquire::http::Pipeline-Depth=5",
		"-o", "Acquire::Languages=none",
	)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c
}
