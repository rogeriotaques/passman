//go:build !darwin && !linux

package sshagent

import "net"

// peerProcess has no portable implementation; callers treat pid 0 as unknown.
func peerProcess(net.Conn) (pid int, name string) {
	return 0, ""
}
