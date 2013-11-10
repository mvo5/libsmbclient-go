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

Limitation:
-----------

The C libsmbclient from samba is not thread safe, so all go
smbclient.Client/smbclient.File operations are serialized (i.e. there
can only be one operation at a time for each Client/File). As a 
workaround you should create one smbclient.Client per goroutine.


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
