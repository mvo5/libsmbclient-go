// +build ignore

package main

import (
	"."
	"log"
	"fmt"
	"flag"
	"os"
	"os/exec"
	"bufio"
	"strings"
	"io"
)

func openSmbdir(client *libsmbclient.Client, duri string) {
	dh, err := client.Opendir(duri)
	if err != nil {
		log.Fatal(err)
	}
	for {
		dirent, err := dh.Readdir()
		if err != nil {
			break
		}
		fmt.Println(dirent)
	}
	dh.Closedir()
}

func openSmbfile(client *libsmbclient.Client, furi string) {
	f, err := client.Open(furi, 0, 0)
	if err != nil {
		log.Fatal(err)
	}
	buf := make([]byte, 1024)
	for {
		_, err := f.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		fmt.Print(string(buf))
	}
	f.Close()
}

func setEcho(terminal_echo_enabled bool) {
	var cmd *exec.Cmd
	if terminal_echo_enabled {
		cmd = exec.Command("stty",  "-F", "/dev/tty", "echo")
	} else  {
		cmd = exec.Command("stty",  "-F", "/dev/tty", "-echo")
	}
	cmd.Run()
}

func main() {
	var duri, furi string
	flag.StringVar(&duri, "show-dir", "", "smb://path/to/dir style directory")
	flag.StringVar(&furi, "show-file", "", "smb://path/to/file style file")
	flag.Parse()

	client := libsmbclient.New()
	//client.SetDebug(99)

	fn := func(server_name, share_name string)(domain, username, password string) {
		fmt.Printf("auth for %s %s: ", server_name, share_name)
		// read pw from stdin
		setEcho(false)
		bio := bufio.NewReader(os.Stdin)
		pw, _, _ := bio.ReadLine()
		setEcho(true)
		return "URT", "vogtm", strings.TrimSpace(string(pw))
	}
	client.SetAuthCallback(fn)

	if duri != "" {
		openSmbdir(client, duri)
	} else if furi != "" {
		openSmbfile(client, furi)
	}



}
