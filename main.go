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

func askAuth(server_name, share_name string)(out_domain, out_username, out_password string) { 
	bio := bufio.NewReader(os.Stdin)
	fmt.Printf("auth for %s %s\n", server_name, share_name)
	// domain
	fmt.Print("Domain: ")
	domain, _, _ := bio.ReadLine()
	// read username
	fmt.Print("Username: ")
	username, _, _ := bio.ReadLine()
	// read pw from stdin
	fmt.Print("Password: ")
	setEcho(false)
	password, _, _ := bio.ReadLine()
	setEcho(true)
	return strings.TrimSpace(string(domain)), strings.TrimSpace(string(username)), strings.TrimSpace(string(password))
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
	var withAuth, withMultithread bool
	flag.StringVar(&duri, "show-dir", "", "smb://path/to/dir style directory")
	flag.StringVar(&furi, "show-file", "", "smb://path/to/file style file")
	flag.BoolVar(&withAuth, "with-auth", false, "ask for auth")
	flag.BoolVar(&withMultithread, "with-multithread", false, "mulithread")
	flag.Parse()

	client := libsmbclient.New()
	//client.SetDebug(99)

	if withAuth {
		client.SetAuthCallback(askAuth)
	}

	var fn func(*libsmbclient.Client, string)
	var uri string
	if duri != "" {
		fn = openSmbdir
		uri = duri
	} else if furi != "" {
		fn = openSmbfile
		uri = furi
	}
	if withMultithread {
		go fn(client, uri)
		go fn(client, uri)
	}
	fn(client, uri)

}
