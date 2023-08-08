package cmd

import (
	"fmt"
	"github.com/nwtgck/handy-sshd"
	"github.com/nwtgck/handy-sshd/version"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
	"golang.org/x/exp/slog"
	"net"
	"os"
	"strconv"
	"strings"
)

var flag struct {
	//dnsServer    string
	showsVersion  bool
	sshHost       string
	sshPort       uint16
	sshUnixSocket string
	sshShell      string
	sshUsers      []string
}

type sshUser struct {
	name     string
	password string
}

func init() {
	cobra.OnInitialize()
	RootCmd.PersistentFlags().BoolVarP(&flag.showsVersion, "version", "v", false, "show version")
	RootCmd.PersistentFlags().StringVarP(&flag.sshHost, "host", "", "", "SSH server host (e.g. 127.0.0.1)")
	RootCmd.PersistentFlags().Uint16VarP(&flag.sshPort, "port", "p", 2222, "SSH server port")
	// NOTE: long name 'unix-socket' is from curl (ref: https://curl.se/docs/manpage.html)
	RootCmd.PersistentFlags().StringVarP(&flag.sshUnixSocket, "unix-socket", "", "", "Unix-domain socket")
	RootCmd.PersistentFlags().StringVarP(&flag.sshShell, "shell", "", "", "Shell")
	//RootCmd.PersistentFlags().StringVar(&flag.dnsServer, "dns-server", "", "DNS server (e.g. 1.1.1.1:53)")
	RootCmd.PersistentFlags().StringArrayVarP(&flag.sshUsers, "user", "", nil, `SSH user name (e.g. "john:mypassword")`)
}

var RootCmd = &cobra.Command{
	Use:          os.Args[0],
	Short:        "handy-sshd",
	Long:         "Portable SSH server",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if flag.showsVersion {
			fmt.Println(version.Version)
			return nil
		}
		logger := slog.Default()
		sshServer := &handy_sshd.Server{
			Logger: logger,
		}
		var sshUsers []sshUser
		for _, u := range flag.sshUsers {
			splits := strings.SplitN(u, ":", 2)
			if len(splits) != 2 {
				return fmt.Errorf("invalid user format: %s", u)
			}
			sshUsers = append(sshUsers, sshUser{name: splits[0], password: splits[1]})
		}

		// (base: https://gist.github.com/jpillora/b480fde82bff51a06238)
		sshConfig := &ssh.ServerConfig{
			//Define a function to run when a client attempts a password login
			PasswordCallback: func(metadata ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
				for _, user := range sshUsers {
					// No auth required
					if user.name == metadata.User() && user.password == string(pass) {
						return nil, nil
					}
				}
				return nil, fmt.Errorf("password rejected for %q", metadata.User())
			},
			NoClientAuth: true,
			NoClientAuthCallback: func(metadata ssh.ConnMetadata) (*ssh.Permissions, error) {
				for _, user := range sshUsers {
					// No auth required
					if user.name == metadata.User() && user.password == "" {
						return nil, nil
					}
				}
				return nil, fmt.Errorf("%s auth required", metadata.User())
			},
		}
		// TODO: specify priv_key by flags
		pri, err := ssh.ParsePrivateKey([]byte(defaultHostKeyPem))
		if err != nil {
			return err
		}
		sshConfig.AddHostKey(pri)

		var ln net.Listener
		if flag.sshUnixSocket == "" {
			address := net.JoinHostPort(flag.sshHost, strconv.Itoa(int(flag.sshPort)))
			ln, err = net.Listen("tcp", address)
			if err != nil {
				return err
			}
			logger.Info(fmt.Sprintf("listening on %s...", address))
		} else {
			ln, err = net.Listen("unix", flag.sshUnixSocket)
			if err != nil {
				return err
			}
			logger.Info(fmt.Sprintf("listening on %s...", flag.sshUnixSocket))
		}
		defer ln.Close()
		for {
			conn, err := ln.Accept()
			if err != nil {
				logger.Error("failed to accept TCP connection", "err", err)
				continue
			}
			sshConn, chans, reqs, err := ssh.NewServerConn(conn, sshConfig)
			if err != nil {
				logger.Info("failed to handshake", "err", err)
				conn.Close()
				continue
			}
			logger.Info("new SSH connection", "client_version", string(sshConn.ClientVersion()))
			go sshServer.HandleGlobalRequests(sshConn, reqs)
			go sshServer.HandleChannels(flag.sshShell, chans)
		}
	},
}
