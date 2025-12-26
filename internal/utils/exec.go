package utils

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"syscall"

	"github.com/acker1019/fedora-phoenix/internal/logging"
)

var execLog = logging.WithSource("utils/exec")

// RunCommandAsUser executes a command as the specified user.
// This is the core engine for Block IV (User Space) operations.
//
// It switches the process context to the target user's UID/GID
// before executing the command, preventing "root-owned files" in user space.
func RunCommandAsUser(username, name string, args ...string) error {
	execLog.Infof("Executing as %s: %s %v", username, name, args)

	// Lookup user information
	u, err := user.Lookup(username)
	if err != nil {
		return fmt.Errorf("failed to lookup user %s: %w", username, err)
	}

	// Parse UID and GID
	uid, err := strconv.Atoi(u.Uid)
	if err != nil {
		return fmt.Errorf("invalid UID for user %s: %w", username, err)
	}

	gid, err := strconv.Atoi(u.Gid)
	if err != nil {
		return fmt.Errorf("invalid GID for user %s: %w", username, err)
	}

	// Create command
	cmd := exec.Command(name, args...)

	// Set process credentials to switch to target user
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{
			Uid: uint32(uid),
			Gid: uint32(gid),
		},
	}

	// Set HOME environment variable for the user
	cmd.Env = append(os.Environ(), fmt.Sprintf("HOME=%s", u.HomeDir))

	// Connect stdout/stderr for visibility
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Execute command
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("command failed: %w", err)
	}

	execLog.Infof("Command executed successfully as %s", username)
	return nil
}
