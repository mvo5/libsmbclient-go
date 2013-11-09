Go Bindings for libsmbclient
============================

Bindings (read-only for now) for the libsmbclient library from
samba. To compile on debian/ubuntu install the "libsmbclient-dev"
package.

Build it with:
```
$ go build main.go
$ ./main -show-dir smb://localhost
```

Check main.go for a code example.


Example usage:
--------------

```
import (
       "smbclient"
       "fmt"
)

client := smbclient.New()
dh := client.Opendir("smb://localhost")
for {
    dirent, err := dh.Readdir()
    if err != nil {
       break
    }
    fmt.Println(dirent)   
}
dh.Closedir()
```
