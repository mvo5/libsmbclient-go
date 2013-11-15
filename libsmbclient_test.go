package libsmbclient

import (
	"fmt"
	"log"
	"testing"
	"os"
	"os/exec"
	"path/filepath"
	"io"
	_ "time"
	"text/template"
)

var teardown []func()

func generateSmbdConf() string {
	tempdir, _ := filepath.Abs("./tmp/samba")
	teardown = append(teardown, func() {
		os.RemoveAll(tempdir)
	})

	paths := [...]string{
		tempdir, 
		filepath.Join(tempdir, "samaba", "private"),
		filepath.Join(tempdir, "samba", "public"),
	}
	for _, d := range(paths) {
		err := os.MkdirAll(d, 0755)
		if err != nil {
			log.Fatal(err)
		}
	}
	os.Mkdir(filepath.Join(tempdir, "private"), 0755)
	os.Mkdir(filepath.Join(tempdir, "public"), 0755)
	f, err := os.Create(filepath.Join(tempdir, "smbd.conf"))
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	templateText := `[global]
workgroup = TESTGROUP
interfaces = lo 127.0.0.0/8
smb ports = 1445
log level = 2
map to guest = Bad User
passdb backend = smbpasswd
smb passwd file = {{.Tempdir}}/smbpasswd
lock directory = {{.Tempdir}}/intern
state directory = {{.Tempdir}}/intern
cache directory = {{.Tempdir}}/intern
pid directory = {{.Tempdir}}/intern
private dir = {{.Tempdir}}/intern
ncalrpc dir = {{.Tempdir}}/intern

[public]
path = {{.Tempdir}}/public
guest ok = yes

[private]
path = {{.Tempdir}}/private
read only = no
`	
	type Dir struct {
		Tempdir string
	}
	t, err := template.New("smb-conf").Parse(templateText)
	if err != nil {
		log.Fatal(err)
	}
	t.Execute(f, Dir{tempdir})
	return f.Name()
}

func startSmbd() io.Reader {
	// thanks pitti :)
	os.Setenv("LIBSMB_PROG", "nc localhost 1445")
	smb_conf := generateSmbdConf()
	cmd := exec.Command("smbd", "-iFS", "-s", smb_conf)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	cmd.Start()
	teardown = append(teardown, func() {
		cmd.Process.Kill()
		cmd.Process.Wait()
	})
	return stdout
}

func setUp() {
	stdout := startSmbd()
	//time.Sleep(100 * time.Millisecond)
	_ = stdout
	//fmt.Println(stdout)
	//go io.Copy(os.Stdout, stdout)
}

func tearDown() {
	// cleanup
	for _,f := range teardown {
		f()
	}
	// cleanup atexit
	teardown = []func(){}
}

func sliceContains(slice []string, needle string) bool {
	for _, d := range(slice) {
		if d == needle {
			return true
		}
	}
	return false
}

func TestLibsmbclientBindings(t *testing.T) {
	fmt.Println("libsmbclient opendir test")

	setUp()

	// open client
	client := New()
	d, err := client.Opendir("smb://localhost")
	if err != nil {
		t.Error("failed to open localhost ", err)
	}
	
	// collect dirs
	foundSmbDirs := make([]string, 10)
	for {
		dirent, err := d.Readdir()
		if err != nil {
			break
		}
		foundSmbDirs = append(foundSmbDirs, dirent.Name)
	}
	// check for expected data
	for _, needle := range([]string{"private", "public"}) {
		if !sliceContains(foundSmbDirs, needle) {
			t.Error("missing 'public' folder (%v)", foundSmbDirs)
		}
	}

	tearDown()
}
