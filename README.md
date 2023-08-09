# handy-sshd
Portable SSH Server

## Install on Ubuntu/Debian

```bash
wget https://github.com/nwtgck/handy-sshd/releases/download/v0.1.0/handy-sshd-0.1.0-linux-amd64.deb
sudo dpkg -i handy-sshd-0.1.0-linux-amd64.deb 
```

## Install on Mac

```bash
brew install nwtgck/handy-sshd/handy-sshd
```

Get more executables in [the releases](https://github.com/nwtgck/handy-sshd/releases).

## Examples

```bash
# Listen on 2222 and accept user name "john" with password "mypassword"
handy-sshd -p 2222 --user "john:mypassword"
```

```bash
# Listen on 2222 and accept user name "john" without password
handy-sshd -p 2222 --user "john:"
```

```bash
# Listen on 2222 and accept users "john" and "alice" without password
handy-sshd -p 2222 --user "john:" --user "alice:"
```

```bash
# Listen on unix domain socket
handy-sshd --unix-socket /tmp/my-unix-socket --user "john:"
```

## --help

```bash
Portable SSH server

Usage:
  handy-sshd [flags]

Flags:
  -h, --help                 help for handy-sshd
      --host string          SSH server host (e.g. 127.0.0.1)
  -p, --port uint16          SSH server port (default 2222)
      --shell string         Shell
      --unix-socket string   Unix-domain socket
      --user stringArray     SSH user name (e.g. "john:mypassword")
  -v, --version              show version
```
