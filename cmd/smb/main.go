package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/mvo5/libsmbclient-go"
)

func openSmbdir(client *libsmbclient.Client, duri string) {
	dh, err := client.Opendir(duri)
	if err != nil {
		log.Print(err)
		return
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

func openSmbfile(client *libsmbclient.Client, furi string) {
	f, err := client.Open(furi, 0, 0)
	if err != nil {
		log.Print(err)
		return
	}
	buf := make([]byte, 64*1024)
	for {
		_, err := f.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Print(err)
			return
		}
		fmt.Print(string(buf))
	}
	f.Close()
}

func askAuth(serverName, shareName string) (outDomain, outUsername, outPassword string) {
	bio := bufio.NewReader(os.Stdin)
	fmt.Printf("auth for %s %s\n", serverName, shareName)
	// domain
	fmt.Print("Domain: ")
	domain, _ := bio.ReadString('\n')
	// read username
	fmt.Print("Username: ")
	username, _ := bio.ReadString('\n')
	// read pw from stdin
	fmt.Print("Password: ")
	setEcho(false)
	password, _ := bio.ReadString('\n')
	setEcho(true)
	return strings.TrimSpace(domain), strings.TrimSpace(username), strings.TrimSpace(password)
}

func setEcho(terminalEchoEnabled bool) {
	var cmd *exec.Cmd
	if terminalEchoEnabled {
		cmd = exec.Command("stty", "-F", "/dev/tty", "echo")
	} else {
		cmd = exec.Command("stty", "-F", "/dev/tty", "-echo")
	}
	cmd.Run()
}

func multiThreadStressTest(client *libsmbclient.Client, uri string) {
	fmt.Println("m: " + uri)
	dh, err := client.Opendir(uri)
	if err != nil {
		log.Print(err)
		return
	}
	defer dh.Closedir()
	for {
		dirent, err := dh.Readdir()
		if err != nil {
			break
		}
		newUri := uri + "/" + dirent.Name
		switch dirent.Type {
		case libsmbclient.SMBC_DIR, libsmbclient.SMBC_FILE_SHARE:
			fmt.Println("d: " + newUri)
			go multiThreadStressTest(client, newUri)
		case libsmbclient.SMBC_FILE:
			fmt.Println("f: " + newUri)
			go openSmbfile(client, newUri)
		}
	}

	// FIXME: instead of sleep, wait for all threads to exit
	time.Sleep(10 * time.Second)
}

func main() {
	var duri, furi, suri string
	var withAuth, withKrb5 bool
	flag.StringVar(&duri, "show-dir", "", "smb://path/to/dir style directory")
	flag.StringVar(&furi, "show-file", "", "smb://path/to/file style file")
	flag.BoolVar(&withAuth, "with-auth", false, "ask for auth")
	flag.BoolVar(&withKrb5, "with-krb5", false, "use Kerberos for auth")
	flag.StringVar(&suri, "stress-test", "", "run threaded stress test")
	flag.Parse()

	client := libsmbclient.New()
	//client.SetDebug(99)

	if withKrb5 {
		client.SetUseKerberos()
	} else if withAuth {
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
	} else if suri != "" {
		fn = multiThreadStressTest
		uri = suri
	} else {
		flag.Usage()
		return
	}
	fn(client, uri)

}
