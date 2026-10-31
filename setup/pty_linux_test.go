//go:build integration && linux

// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package setup_test

import (
	"fmt"
	"os"
	"syscall"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-slogx/setup"
)

func TestIntegrationColorAutoPTY(t *testing.T) {
	t.Parallel()
	master, err := os.OpenFile("/dev/ptmx", os.O_RDWR, 0)
	if err != nil {
		t.Skipf("PTY unavailable: %v", err)
	}
	defer master.Close()
	var unlock int32
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, master.Fd(), syscall.TIOCSPTLCK, uintptr(unsafe.Pointer(&unlock)))
	if errno != 0 {
		t.Skipf("PTY unlock unavailable: %v", errno)
	}
	var number uint32
	_, _, errno = syscall.Syscall(syscall.SYS_IOCTL, master.Fd(), syscall.TIOCGPTN, uintptr(unsafe.Pointer(&number)))
	require.Zero(t, errno)
	slave, err := os.OpenFile(fmt.Sprintf("/dev/pts/%d", number), os.O_RDWR, 0)
	require.NoError(t, err)
	defer slave.Close()
	l, err := setup.New(slave, setup.WithColor(setup.ColorAuto), setup.WithBufferSize(8))
	require.NoError(t, err)
	l.Logger().Info("pty")
	require.NoError(t, l.Shutdown(t.Context()))
	data := make([]byte, 4096)
	n, err := master.Read(data)
	require.NoError(t, err)
	require.Contains(t, string(data[:n]), "\x1b[")
}
