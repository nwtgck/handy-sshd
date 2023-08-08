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
