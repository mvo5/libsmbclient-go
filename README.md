Go Bindings for libsmbclient
============================

[![PkgGoDev](https://pkg.go.dev/badge/github.com/mvo5/libsmbclient-go)](https://pkg.go.dev/github.com/mvo5/libsmbclient-go)
[![Tests report](https://github.com/mvo5/libsmbclient-go/workflows/Test/badge.svg)](https://github.com//mvo5/libsmbclient-go/actions?query=workflow%3ATest)
[![Go Report Card](https://goreportcard.com/badge/github.com/mvo5/libsmbclient-go)](https://goreportcard.com/report/github.com/mvo5/libsmbclient-go)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](https://github.com/mvo5/libsmbclient-go/blob/master/COPYING)


Bindings (read-only for now) for the libsmbclient library from
samba. To compile on debian/ubuntu install the "libsmbclient-dev"
package.

Build it with:
```
$ go build ./cmd/smb
$ ./smb -show-dir smb://localhost
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
       "fmt"

       "github.com/mvo5/libsmbclient-go"
)

client := smbclient.New()
dh, err := client.Opendir("smb://localhost")
if err != nil {
    return err
}
defer dh.Closedir()
for {
    dirent, err := dh.Readdir()
    if err != nil {
       break
    }
    fmt.Println(dirent)
}
```
