// Copyright (c) 2021 Ryo Ota
// Released under the MIT License

// Copyright (c) 2020 Jaime Pillora <dev@jpillora.com>
// Released under the MIT License
// https://github.com/jpillora/sshd-lite/tree/master#mit-license

package handy_sshd

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/binary"
	"encoding/pem"
	"fmt"
	"github.com/mattn/go-shellwords"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/exp/slog"
	"io"
	"net"
	"os"
	"os/exec"
	"strconv"
	"sync"
)

type Server struct {
	Logger *slog.Logger

	// Permissions
	AllowTcpipForward bool
	AllowDirectTcpip  bool
	AllowExecute      bool // this should not be split into "allow-exec" and "allow-pty-req" for now because "pty-req" can be used not for shell execution.
	AllowSftp         bool

	// TODO: DNS server ?
}

type exitStatusMsg struct {
	Status uint32
}

func (s *Server) HandleChannels(shell string, chans <-chan ssh.NewChannel) {
	// Service the incoming Channel channel in go routine
	for newChannel := range chans {
		go s.handleChannel(shell, newChannel)
	}
}

func (s *Server) handleChannel(shell string, newChannel ssh.NewChannel) {
	switch newChannel.ChannelType() {
	case "session":
		s.handleSession(shell, newChannel)
	case "direct-tcpip":
		if !s.AllowDirectTcpip {
			newChannel.Reject(ssh.Prohibited, "direct-tcpip not allowed")
			break
		}
		s.handleDirectTcpip(newChannel)
	default:
		newChannel.Reject(ssh.UnknownChannelType, fmt.Sprintf("unknown channel type: %s", newChannel.ChannelType()))
	}
}

func (s *Server) handleSession(shell string, newChannel ssh.NewChannel) {
	// At this point, we have the opportunity to reject the client's
	// request for another logical connection
	connection, requests, err := newChannel.Accept()
	if err != nil {
		s.Logger.Info("Could not accept channel", "err", err)
		return
	}

	var shf *os.File = nil

	for req := range requests {
		switch req.Type {
		case "exec":
			if !s.AllowExecute {
				s.Logger.Info("execution not allowed (exec)")
				req.Reply(false, nil)
				break
			}
			s.handleExecRequest(req, connection)
		case "shell":
			// We only accept the default shell
			// (i.e. no command in the Payload)
			if len(req.Payload) == 0 {
				req.Reply(true, nil)
			}
		case "pty-req":
			if !s.AllowExecute {
				s.Logger.Info("execution not allowed (pty-req)")
				req.Reply(false, nil)
				break
			}
			termLen := req.Payload[3]
			w, h := parseDims(req.Payload[termLen+4:])
			shf, err = s.createPty(shell, connection)
			if err != nil {
				req.Reply(false, nil)
				return
			}
			setWinsize(shf, w, h)
			// Responding true (OK) here will let the client
			// know we have a pty ready for input
			req.Reply(true, nil)
		case "window-change":
			w, h := parseDims(req.Payload)
			if shf != nil {
				setWinsize(shf, w, h)
			}
		case "subsystem":
			s.handleSessionSubSystem(req, connection)
		}
	}
}

func (s *Server) handleExecRequest(req *ssh.Request, connection ssh.Channel) {
	var msg struct {
		Command string
	}
	if err := ssh.Unmarshal(req.Payload, &msg); err != nil {
		s.Logger.Info("failed to parse message in exec", "err", err)
		return
	}
	cmdSlice, err := shellwords.Parse(msg.Command)
	if err != nil {
		return
	}
	cmd := exec.Command(cmdSlice[0], cmdSlice[1:]...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return
	}
	go io.Copy(stdin, connection)
	go io.Copy(connection, stdout)
	go io.Copy(connection, stderr)
	req.Reply(true, nil)
	var exitCode int
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
	}
	connection.SendRequest("exit-status", false, ssh.Marshal(exitStatusMsg{
		Status: uint32(exitCode),
	}))
	connection.Close()
}

func (s *Server) handleSessionSubSystem(req *ssh.Request, connection ssh.Channel) {
	// https://github.com/pkg/sftp/blob/42e9800606febe03f9cdf1d1283719af4a5e6456/examples/go-sftp-server/main.go#L111
	if string(req.Payload[4:]) != "sftp" {
		req.Reply(false, nil)
		return
	}
	if !s.AllowSftp {
		s.Logger.Info("sftp not allowed")
		req.Reply(false, nil)
		return
	}

	req.Reply(true, nil)
	serverOptions := []sftp.ServerOption{
		sftp.WithDebug(os.Stderr),
	}
	sftpServer, err := sftp.NewServer(connection, serverOptions...)
	if err != nil {
		s.Logger.Info("failed to create sftp server", "err", err)
		return
	}
	if err := sftpServer.Serve(); err == io.EOF {
		sftpServer.Close()
	} else if err != nil {
		s.Logger.Info("failed to serve sftp server", "err", err)
		return
	}
}

// (base: https://github.com/peertechde/zodiac/blob/110fdd2dfd27359546c1cd75a9fec5de2882bf42/pkg/server/server.go#L228)
func (s *Server) handleDirectTcpip(newChannel ssh.NewChannel) {
	var msg struct {
		RemoteAddr string
		RemotePort uint32
		SourceAddr string
		SourcePort uint32
	}
	if err := ssh.Unmarshal(newChannel.ExtraData(), &msg); err != nil {
		s.Logger.Info("failed to parse message", "err", err)
		return
	}
	channel, reqs, err := newChannel.Accept()
	if err != nil {
		s.Logger.Info("failed to accept", "err", err)
		return
	}
	go ssh.DiscardRequests(reqs)
	raddr := net.JoinHostPort(msg.RemoteAddr, strconv.Itoa(int(msg.RemotePort)))
	conn, err := net.Dial("tcp", raddr)
	if err != nil {
		s.Logger.Info("failed to dial", "err", err)
		channel.Close()
		return
	}
	var closeOnce sync.Once
	closer := func() {
		channel.Close()
		conn.Close()
	}
	go func() {
		io.Copy(channel, conn)
		closeOnce.Do(closer)
	}()
	io.Copy(conn, channel)
	closeOnce.Do(closer)
	return
}

// =======================

// parseDims extracts terminal dimensions (width x height) from the provided buffer.
func parseDims(b []byte) (uint32, uint32) {
	w := binary.BigEndian.Uint32(b)
	h := binary.BigEndian.Uint32(b[4:])
	return w, h
}

// ======================

func GenerateKey() ([]byte, error) {
	var r io.Reader
	r = rand.Reader
	priv, err := rsa.GenerateKey(r, 2048)
	if err != nil {
		return nil, err
	}
	err = priv.Validate()
	if err != nil {
		return nil, err
	}
	b := x509.MarshalPKCS1PrivateKey(priv)
	return pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: b}), nil
}

// Borrowed from https://github.com/creack/termios/blob/master/win/win.go

// ======================================================================

func (s *Server) HandleGlobalRequests(sshConn *ssh.ServerConn, reqs <-chan *ssh.Request) {
	for req := range reqs {
		switch req.Type {
		case "tcpip-forward":
			if !s.AllowTcpipForward {
				s.Logger.Info("tcpip-forward not allowed")
				req.Reply(false, nil)
				break
			}
			s.handleTcpipForward(sshConn, req)
		// TODO: support: streamlocal-forward@openssh.com https://github.com/golang/crypto/blob/master/ssh/streamlocal.go
		default:
			// discard
			if req.WantReply {
				req.Reply(false, nil)
			}
			s.Logger.Info("request discarded", "request_type", req.Type)
		}
	}
}

// https://datatracker.ietf.org/doc/html/rfc4254#section-7.1
func (s *Server) handleTcpipForward(sshConn *ssh.ServerConn, req *ssh.Request) {
	var msg struct {
		Addr string
		Port uint32
	}
	if err := ssh.Unmarshal(req.Payload, &msg); err != nil {
		req.Reply(false, nil)
		return
	}
	ln, err := net.Listen("tcp", net.JoinHostPort(msg.Addr, strconv.Itoa(int(msg.Port))))
	if err != nil {
		req.Reply(false, nil)
		return
	}
	go func() {
		sshConn.Wait()
		ln.Close()
		s.Logger.Info("connection closed", "address", ln.Addr().String())
	}()
	for {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		var replyMsg struct {
			Addr           string
			Port           uint32
			OriginatorAddr string
			OriginatorPort uint32
		}
		replyMsg.Addr = msg.Addr
		replyMsg.Port = msg.Port

		go func() {
			channel, reqs, err := sshConn.OpenChannel("forwarded-tcpip", ssh.Marshal(&replyMsg))
			if err != nil {
				req.Reply(false, nil)
				conn.Close()
				return
			}
			go ssh.DiscardRequests(reqs)
			go func() {
				io.Copy(channel, conn)
				conn.Close()
				channel.Close()
			}()
			go func() {
				io.Copy(conn, channel)
				conn.Close()
				channel.Close()
			}()
		}()
	}
}
