package ops

import (
	"fmt"
	"io"
	"os"
	"os/exec"
)

// UnlockLuks unlocks the device using the provided password string.
// Updated signature: accepts 'password' as the 3rd argument.
func UnlockLuks(devicePath, mapperName, password string) error {
	// Idempotency check: if /dev/mapper/xxx exists, we are good.
	mapperPath := fmt.Sprintf("/dev/mapper/%s", mapperName)
	if _, err := os.Stat(mapperPath); err == nil {
		fmt.Printf("🔒 [LUKS] Device %s is already unlocked. Skipping.\n", mapperName)
		return nil
	}

	fmt.Printf("🔑 [LUKS] Unlocking %s with injected credentials...\n", devicePath)

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

	fmt.Println("✅ LUKS unlocked successfully.")
	return nil
}

// MountDevice mounts the unlocked mapper device to the target path.
func MountDevice(source, target string) {
	// Check if already mounted
	// Using `mountpoint -q` is the easiest way in shell, usually safe to exec.
	if err := exec.Command("mountpoint", "-q", target).Run(); err == nil {
		fmt.Printf("📂 [Mount] %s is already mounted. Skipping.\n", target)
		return
	}

	// Ensure directory exists
	if err := os.MkdirAll(target, 0755); err != nil {
		panic(fmt.Sprintf("Failed to mkdir %s: %v", target, err))
	}

	fmt.Printf("📂 [Mount] Mounting %s -> %s\n", source, target)
	cmd := exec.Command("mount", source, target)
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		panic(fmt.Sprintf("❌ Mount failed: %v", err))
	}
}
