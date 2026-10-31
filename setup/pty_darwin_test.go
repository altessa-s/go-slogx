//go:build integration && darwin

// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package setup_test

import (
	"bytes"
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
	for _, request := range []uintptr{syscall.TIOCPTYGRANT, syscall.TIOCPTYUNLK} {
		_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, master.Fd(), request, 0)
		if errno != 0 {
			t.Skipf("PTY ioctl unavailable: %v", errno)
		}
	}
	var name [128]byte
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, master.Fd(), syscall.TIOCPTYGNAME, uintptr(unsafe.Pointer(&name[0])))
	require.Zero(t, errno)
	slave, err := os.OpenFile(string(bytes.TrimRight(name[:], "\x00")), os.O_RDWR, 0)
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
