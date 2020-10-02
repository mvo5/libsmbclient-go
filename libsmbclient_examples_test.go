package libsmbclient_test

import (
	"fmt"
	"log"

	"github.com/mvo5/libsmbclient-go"
)

func ExampleNew() {
	client := libsmbclient.New()
	dh, err := client.Opendir("smb://localhost")
	if err != nil {
		log.Fatal(err)
	}
	defer dh.Closedir()
	for {
		dirent, err := dh.Readdir()
		if err != nil {
			break
		}
		fmt.Println(dirent)
	}
}
