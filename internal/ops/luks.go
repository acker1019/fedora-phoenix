package ops

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/acker1019/fedora-phoenix/internal/logging"
)

var luksLog = logging.WithSource("ops/luks")

// UnlockLuks unlocks the device using the provided password string.
// Updated signature: accepts 'password' as the 3rd argument.
func UnlockLuks(devicePath, mapperName, password string) error {
	// Idempotency check: if /dev/mapper/xxx exists, we are good.
	mapperPath := fmt.Sprintf("/dev/mapper/%s", mapperName)
	if _, err := os.Stat(mapperPath); err == nil {
		luksLog.Infof("Device %s is already unlocked. Skipping.", mapperName)
		return nil
	}

	luksLog.Infof("Unlocking %s with injected credentials...", devicePath)

	// Command: cryptsetup open <device> <name> --type luks -
	cmd := exec.Command("cryptsetup", "open", devicePath, mapperName, "--type", "luks")

	// Security: Pipe password to stdin
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to open stdin pipe: %w", err)
	}

	// Capture output
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start cryptsetup: %w", err)
	}

	// Write password and close pipe
	go func() {
		defer stdin.Close()
		io.WriteString(stdin, password)
	}()

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("cryptsetup failed (invalid password?): %w", err)
	}

	luksLog.Info("LUKS unlocked successfully")
	return nil
}

// MountDevice mounts the unlocked mapper device to the target path.
func MountDevice(mapperName, mountPoint string) error {
	// Construct full device path from mapper name
	devicePath := fmt.Sprintf("/dev/mapper/%s", mapperName)

	// Check if already mounted
	// Using `mountpoint -q` is the easiest way in shell, usually safe to exec.
	if err := exec.Command("mountpoint", "-q", mountPoint).Run(); err == nil {
		luksLog.Infof("%s is already mounted. Skipping.", mountPoint)
		return nil
	}

	// Ensure directory exists
	if err := os.MkdirAll(mountPoint, 0755); err != nil {
		return fmt.Errorf("failed to mkdir %s: %w", mountPoint, err)
	}

	luksLog.Infof("Mounting %s -> %s", devicePath, mountPoint)
	cmd := exec.Command("mount", devicePath, mountPoint)
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("mount failed: %w", err)
	}

	luksLog.Info("Mount completed successfully")
	return nil
}
