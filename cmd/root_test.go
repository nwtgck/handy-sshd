package cmd

import (
	"bytes"
	"context"
	"github.com/nwtgck/handy-sshd/version"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/ssh"
	"net"
	"strconv"
	"testing"
)

func TestVersion(t *testing.T) {
	rootCmd := RootCmd()
	rootCmd.SetArgs([]string{"--version"})
	var stdoutBuf bytes.Buffer
	rootCmd.SetOut(&stdoutBuf)
	assert.NoError(t, rootCmd.Execute())
	assert.Equal(t, version.Version+"\n", stdoutBuf.String())
}

func TestZeroUsers(t *testing.T) {
	rootCmd := RootCmd()
	rootCmd.SetArgs([]string{})
	var stderrBuf bytes.Buffer
	rootCmd.SetErr(&stderrBuf)
	assert.Error(t, rootCmd.Execute())
	assert.Equal(t, `Error: No user specified
e.g. --user "john:mypassword"
e.g. --user "john:"
`, stderrBuf.String())
}

func TestAllPermissionsAllowed(t *testing.T) {
	rootCmd := RootCmd()
	port := getAvailableTcpPort()
	rootCmd.SetArgs([]string{"--port", strconv.Itoa(port), "--user", "john:mypassword"})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		var stderrBuf bytes.Buffer
		rootCmd.SetErr(&stderrBuf)
		rootCmd.ExecuteContext(ctx)
	}()
	waitTCPServer(port)
	sshClientConfig := &ssh.ClientConfig{
		User: "john",
		Auth: []ssh.AuthMethod{ssh.Password("mypassword")},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}
	address := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))

	client, err := ssh.Dial("tcp", address, sshClientConfig)
	assert.NoError(t, err)
	defer client.Close()
	assert.NoError(t, err)
	assertExec(t, client)
	assertLocalPortForwarding(t, client)
	assertRemotePortForwardingTODO(t, client)
	// TODO: pty
	// TODO: sftp
}

func TestEmptyPassword(t *testing.T) {
	rootCmd := RootCmd()
	port := getAvailableTcpPort()
	rootCmd.SetArgs([]string{"--port", strconv.Itoa(port), "--user", "john:"})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		var stderrBuf bytes.Buffer
		rootCmd.SetErr(&stderrBuf)
		rootCmd.ExecuteContext(ctx)
	}()
	waitTCPServer(port)
	sshClientConfig := &ssh.ClientConfig{
		User: "john",
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}
	address := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))

	client, err := ssh.Dial("tcp", address, sshClientConfig)
	assert.NoError(t, err)
	defer client.Close()
}

func TestMultipleUsers(t *testing.T) {
	rootCmd := RootCmd()
	port := getAvailableTcpPort()
	rootCmd.SetArgs([]string{"--port", strconv.Itoa(port), "--user", "john:mypassword1", "--user", "alex:mypassword2"})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		var stderrBuf bytes.Buffer
		rootCmd.SetErr(&stderrBuf)
		rootCmd.ExecuteContext(ctx)
	}()
	waitTCPServer(port)
	address := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))

	for _, user := range []struct {
		name     string
		password string
	}{{name: "john", password: "mypassword1"}, {name: "alex", password: "mypassword2"}} {
		sshClientConfig := &ssh.ClientConfig{
			User: user.name,
			Auth: []ssh.AuthMethod{ssh.Password(user.password)},
			HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
				return nil
			},
		}
		client, err := ssh.Dial("tcp", address, sshClientConfig)
		assert.NoError(t, err)
		defer client.Close()
	}
}

func TestWrongPassword(t *testing.T) {
	rootCmd := RootCmd()
	port := getAvailableTcpPort()
	rootCmd.SetArgs([]string{"--port", strconv.Itoa(port), "--user", "john:mypassword"})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		var stderrBuf bytes.Buffer
		rootCmd.SetErr(&stderrBuf)
		rootCmd.ExecuteContext(ctx)
	}()
	waitTCPServer(port)
	sshClientConfig := &ssh.ClientConfig{
		User: "john",
		Auth: []ssh.AuthMethod{ssh.Password("mywrongpassword")},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}
	address := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	_, err := ssh.Dial("tcp", address, sshClientConfig)
	assert.Error(t, err)
	assert.Equal(t, `ssh: handshake failed: ssh: unable to authenticate, attempted methods [none password], no supported methods remain`, err.Error())
}

func TestAllowExecute(t *testing.T) {
	rootCmd := RootCmd()
	port := getAvailableTcpPort()
	rootCmd.SetArgs([]string{"--port", strconv.Itoa(port), "--user", "john:mypassword", "--allow-execute"})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		var stderrBuf bytes.Buffer
		rootCmd.SetErr(&stderrBuf)
		rootCmd.ExecuteContext(ctx)
	}()
	waitTCPServer(port)
	sshClientConfig := &ssh.ClientConfig{
		User: "john",
		Auth: []ssh.AuthMethod{ssh.Password("mypassword")},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}
	address := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	client, err := ssh.Dial("tcp", address, sshClientConfig)
	assert.NoError(t, err)
	defer client.Close()
	assert.NoError(t, err)
	assertNoRemotePortForwarding(t, client)
	assertNoLocalPortForwarding(t, client)
	assertExec(t, client)
	// TODO: no pty
	// TODO: no sftp
}

func TestAllowTcpipForward(t *testing.T) {
	rootCmd := RootCmd()
	port := getAvailableTcpPort()
	rootCmd.SetArgs([]string{"--port", strconv.Itoa(port), "--user", "john:mypassword", "--allow-tcpip-forward"})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		var stderrBuf bytes.Buffer
		rootCmd.SetErr(&stderrBuf)
		rootCmd.ExecuteContext(ctx)
	}()
	waitTCPServer(port)
	sshClientConfig := &ssh.ClientConfig{
		User: "john",
		Auth: []ssh.AuthMethod{ssh.Password("mypassword")},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}
	address := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	client, err := ssh.Dial("tcp", address, sshClientConfig)
	assert.NoError(t, err)
	defer client.Close()
	assert.NoError(t, err)
	assertRemotePortForwardingTODO(t, client)
	assertNoLocalPortForwarding(t, client)
	assertNoExec(t, client)
	// TODO: no pty
	// TODO: no sftp
}

func TestAllowDirectTcpip(t *testing.T) {
	rootCmd := RootCmd()
	port := getAvailableTcpPort()
	rootCmd.SetArgs([]string{"--port", strconv.Itoa(port), "--user", "john:mypassword", "--allow-direct-tcpip"})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		var stderrBuf bytes.Buffer
		rootCmd.SetErr(&stderrBuf)
		rootCmd.ExecuteContext(ctx)
	}()
	waitTCPServer(port)
	sshClientConfig := &ssh.ClientConfig{
		User: "john",
		Auth: []ssh.AuthMethod{ssh.Password("mypassword")},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}
	address := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	client, err := ssh.Dial("tcp", address, sshClientConfig)
	assert.NoError(t, err)
	defer client.Close()
	assert.NoError(t, err)
	assertNoRemotePortForwarding(t, client)
	assertLocalPortForwarding(t, client)
	assertNoExec(t, client)
	// TODO: no pty
	// TODO: no sftp
}

// TODO: TestAllowSftp
