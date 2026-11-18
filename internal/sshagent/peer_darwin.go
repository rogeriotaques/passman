//go:build darwin

package sshagent

import (
	"net"
	"os/exec"
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
	raw.Control(func(fd uintptr) {
		pid, err = unix.GetsockoptInt(int(fd), unix.SOL_LOCAL, unix.LOCAL_PEERPID)
	})
	if err != nil || pid <= 0 {
		return 0, ""
	}
	return pid, processName(pid)
}

func processName(pid int) string {
	out, err := exec.Command("/bin/ps", "-o", "comm=", "-p", strconv.Itoa(pid)).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
