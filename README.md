Go Bindings for libsmbclient
============================

Bindings for the libsmbclient library from samba. To compile on
debian/ubuntu install the "libsmbclient-dev" package.

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
    dirent, err := client.Readdir(dh)
    if err != nil {
       break
    }
    fmt.Println(dirent)   
}
client.Closedir(dh)
```
