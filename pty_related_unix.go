//go:build !windows
// +build !windows

// NOTE: pty.Start() is not supported in Windows

package handy_sshd

import (
	"github.com/creack/pty"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
	"io"
	"os"
	"os/exec"
	"sync"
)

func (s *Server) createPty(shell string, connection ssh.Channel) (*os.File, error) {
	if shell == "" {
		shell = os.Getenv("SHELL")
	}
	if shell == "" {
		shell = "sh"
	}
	// Fire up bash for this session
	sh := exec.Command(shell)

	// Prepare teardown function
	closer := func() {
		connection.SendRequest("exit-status", false, ssh.Marshal(exitStatusMsg{
			Status: 0,
		}))
		connection.Close()
		if sh.Process != nil {
			_, err := sh.Process.Wait()
			if err != nil {
				s.Logger.Info("failed to exit shell", err)
			}
		}
		s.Logger.Info("session closed")
	}

	// Allocate a terminal for this channel
	s.Logger.Info("creating pty...")
	shf, err := pty.Start(sh)
	if err != nil {
		s.Logger.Info("failed to start pty", "err", err)
		closer()
		return nil, errors.Errorf("could not start pty (%s)", err)
	}

	// pipe session to bash and visa-versa
	var once sync.Once
	go func() {
		io.Copy(connection, shf)
		once.Do(closer)
	}()
	go func() {
		io.Copy(shf, connection)
		once.Do(closer)
	}()
	return shf, nil
}

// setWinsize sets the size of the given pty.
func setWinsize(t *os.File, w, h uint32) error {
	return pty.Setsize(t, &pty.Winsize{Rows: uint16(h), Cols: uint16(w)})
}
