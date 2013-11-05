// +build ignore

package main

import (
	"."
	"log"
	"fmt"
	"flag"
	"os"
	"bufio"
	"strings"
)

func openSmbdir(duri string) {
	client := libsmbclient.New()
	//client.SetDebug(99)

	dh, err := client.Opendir(duri)
	if err != nil {
		log.Fatal(err)
	}
	for {
		dirent, err := client.Readdir(dh)
		if err != nil {
			break
		}
		fmt.Println(dirent)
	}
	client.Closedir(dh)
}

func openSmbfile(furi string) {
	client := libsmbclient.New()

	f, err := client.Open(furi, 0, 0)
	if err != nil {
		log.Fatal(err)
	}
	buf := make([]byte, 1024)
	for {
		n, err := client.Read(f, buf)
		if err != nil {
			log.Fatal(err)
		}
		if n == 0 {
			break
		}
		fmt.Print(string(buf))
	}
	client.Close(f)
}

func main() {
	var duri, furi string
	flag.StringVar(&duri, "show-dir", "", "smb://path/to/dir style directory")
	flag.StringVar(&furi, "show-file", "", "smb://path/to/file style file")
	flag.Parse()

	libsmbclient.Global_auth_callback = func(server_name, share_name string)(domain, username, password string) {
		fmt.Printf("auth for %s %s: ", server_name, share_name)
		// read pw from stdin
		bio := bufio.NewReader(os.Stdin)
		pw, _, _ := bio.ReadLine()
		return "URT", "vogtm", strings.TrimSpace(string(pw))
	}

	if duri != "" {
		openSmbdir(duri)
	} else if furi != "" {
		openSmbfile(furi)
	}



}
