# handy-sshd
Portable SSH Server 

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
