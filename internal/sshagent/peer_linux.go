//go:build linux

package sshagent

import (
	"net"
	"os"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

// peerProcess identifies the process on the other end of a Unix socket.
func peerProcess(conn net.Conn) (pid int, name string) {
	unixConn, ok := conn.(*net.UnixConn)
	if !ok {
		return 0, ""
	}
	raw, err := unixConn.SyscallConn()
	if err != nil {
		return 0, ""
	}
	var cred *unix.Ucred
	raw.Control(func(fd uintptr) {
		cred, err = unix.GetsockoptUcred(int(fd), unix.SOL_SOCKET, unix.SO_PEERCRED)
	})
	if err != nil || cred == nil || cred.Pid <= 0 {
		return 0, ""
	}
	pid = int(cred.Pid)
	return pid, processName(pid)
}

func processName(pid int) string {
	exe, err := os.Readlink("/proc/" + strconv.Itoa(pid) + "/exe")
	if err == nil {
		return exe
	}
	comm, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/comm")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(comm))
}
