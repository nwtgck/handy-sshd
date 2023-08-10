package cmd

import (
	"bytes"
	"github.com/pkg/sftp"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/ssh"
	"io"
	"net"
	"os/exec"
	"strconv"
	"testing"
	"time"
)

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

func assertExec(t *testing.T, client *ssh.Client) {
	session, err := client.NewSession()
	assert.NoError(t, err)
	defer session.Close()
	whoamiBytes, err := session.Output("whoami")
	assert.NoError(t, err)
	expectedWhoamiBytes, err := exec.Command("whoami").Output()
	assert.NoError(t, err)
	assert.Equal(t, string(expectedWhoamiBytes), string(whoamiBytes))
}

func assertNoExec(t *testing.T, client *ssh.Client) {
	session, err := client.NewSession()
	assert.NoError(t, err)
	defer session.Close()
	_, err = session.Output("whoami")
	assert.Error(t, err)
	assert.Equal(t, "ssh: command whoami failed", err.Error())
}

func assertPtyTerminal(t *testing.T, client *ssh.Client) {
	session, err := client.NewSession()
	assert.NoError(t, err)
	defer session.Close()

	err = session.RequestPty("xterm", 100, 200, ssh.TerminalModes{})
	assert.NoError(t, err)
	stdin, err := session.StdinPipe()
	assert.NoError(t, err)
	_, err = stdin.Write([]byte("echo helloworldviapty\r"))
	assert.NoError(t, err)
	stdout, err := session.StdoutPipe()
	assert.NoError(t, err)
	stdoutBytesChan := make(chan []byte)
	go func() {
		var buff bytes.Buffer
		_, err := io.Copy(&buff, stdout)
		assert.NoError(t, err)
		stdoutBytesChan <- buff.Bytes()
	}()
	err = session.Shell()
	assert.NoError(t, err)
	time.Sleep(1 * time.Second)
	session.Close()
	stdoutBytes := <-stdoutBytesChan
	assert.Contains(t, string(stdoutBytes), "helloworldviapty")
}

func assertNoPtyTerminal(t *testing.T, client *ssh.Client) {
	session, err := client.NewSession()
	assert.NoError(t, err)
	defer session.Close()
	err = session.RequestPty("xterm", 100, 200, ssh.TerminalModes{})
	assert.Error(t, err)
	assert.Equal(t, "ssh: pty-req failed", err.Error())
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

func assertNoLocalPortForwarding(t *testing.T, client *ssh.Client) {
	raddr := &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 1234}
	_, err := client.DialTCP("tcp", nil, raddr)
	assert.Error(t, err)
	assert.Equal(t, "ssh: rejected: administratively prohibited (direct-tcpip not allowed)", err.Error())
}

func assertRemotePortForwardingTODO(t *testing.T, client *ssh.Client) {
	remotePort := getAvailableTcpPort()
	acceptedConnChan := make(chan net.Conn)
	var _ = acceptedConnChan
	ln, err := client.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(remotePort)))
	var _ = ln
	assert.NoError(t, err)
	go func() {
		//conn, err := ln.Accept()
		//assert.NoError(t, err)
		//acceptedConnChan <- conn
	}()

	conn, err := net.Dial("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(remotePort)))
	assert.NoError(t, err)
	defer conn.Close()

	// FIXME: implement but the following suspends
	//acceptedConn := <-acceptedConnChan
	//defer acceptedConn.Close()
	// TODO: conn <--> acceptedConn communication
}

func assertNoRemotePortForwarding(t *testing.T, client *ssh.Client) {
	_, err := client.Listen("tcp", "127.0.0.1:5678")
	assert.Error(t, err)
	assert.Equal(t, "ssh: tcpip-forward request denied by peer", err.Error())
}

func assertSftp(t *testing.T, client *ssh.Client) {
	sftpClient, err := sftp.NewClient(client)
	assert.NoError(t, err)
	_, err = sftpClient.Getwd()
	assert.NoError(t, err)
}

func assertNoSftp(t *testing.T, client *ssh.Client) {
	_, err := sftp.NewClient(client)
	assert.Error(t, err)
	assert.Equal(t, "ssh: subsystem request failed", err.Error())
}
