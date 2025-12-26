package utils

import (
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"syscall"
)

// 這是我們消除 "root 髒檔案" 的關鍵武器
func RunCommandAsUser(username string, name string, args ...string) error {
	// 1. 查找 User 的 UID/GID
	u, _ := user.Lookup(username)
	uid, _ := strconv.Atoi(u.Uid)
	gid, _ := strconv.Atoi(u.Gid)

	cmd := exec.Command(name, args...)

	// 2. 魔法時刻：告訴 Kernel 這個子進程要用凡人身份跑
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{Uid: uint32(uid), Gid: uint32(gid)},
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
