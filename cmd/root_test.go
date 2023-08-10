package cmd

import (
	"bytes"
	"github.com/nwtgck/handy-sshd/version"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/ssh"
	"io"
	"net"
	"os/exec"
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
	go func() {
		var stderrBuf bytes.Buffer
		rootCmd.SetErr(&stderrBuf)
		assert.NoError(t, rootCmd.Execute())
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
	defer client.Close()
	assert.NoError(t, err)
	assertExec(t, client)
	assertLocalPortForwarding(t, client)
}

func assertExec(t *testing.T, client *ssh.Client) {
	session, err := client.NewSession()
	assert.NoError(t, err)
	defer session.Close()
	whoamiBytes, err := session.Output("whoami")
	assert.NoError(t, err)
	expectedWhoamiBytes, err := exec.Command("whoami").Output()
	assert.NoError(t, err)
	assert.Equal(t, string(whoamiBytes), string(expectedWhoamiBytes))
}

func assertLocalPortForwarding(t *testing.T, client *ssh.Client) {
	var remoteTcpPort int
	acceptedConnChan := make(chan net.Conn)
	{
		ln, err := net.Listen("tcp", ":0")
		assert.NoError(t, err)
		remoteTcpPort = ln.Addr().(*net.TCPAddr).Port
		go func() {
			conn, err := ln.Accept()
			assert.NoError(t, err)
			acceptedConnChan <- conn
		}()
	}
	raddr := &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: remoteTcpPort}
	conn, err := client.DialTCP("tcp", nil, raddr)
	assert.NoError(t, err)
	defer conn.Close()
	acceptedConn := <-acceptedConnChan
	defer acceptedConn.Close()
	{
		localToRemote := [3]byte{1, 2, 3}
		_, err = conn.Write(localToRemote[:])
		assert.NoError(t, err)
		var buf [len(localToRemote)]byte
		_, err = io.ReadFull(acceptedConn, buf[:])
		assert.NoError(t, err)
		assert.Equal(t, buf, localToRemote)
	}
	{
		remoteToLocal := [4]byte{10, 20, 30, 40}
		_, err = acceptedConn.Write(remoteToLocal[:])
		assert.NoError(t, err)
		var buf [len(remoteToLocal)]byte
		_, err = io.ReadFull(conn, buf[:])
		assert.NoError(t, err)
		assert.Equal(t, buf, remoteToLocal)
	}
}

func getAvailableTcpPort() int {
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		panic(err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

func waitTCPServer(port int) {
	for {
		conn, err := net.Dial("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
		if err == nil {
			conn.Close()
			break
		}
	}
}
