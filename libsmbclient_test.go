package libsmbclient

import (
	"fmt"
	"log"
	"testing"
	"os"
	"os/exec"
	"path/filepath"
	"io"
)

type SmbdContext struct {
	smbd *exec.Cmd
}

func generateSmbdConf() string {
	// FIXME: temp race
	tmpdir := "/tmp/samba"
	os.RemoveAll(tmpdir)
	os.Mkdir(tmpdir, 0755)
	os.Mkdir(filepath.Join(tmpdir, "private"), 0755)
	os.Mkdir(filepath.Join(tmpdir, "public"), 0755)
	f, err := os.Create(filepath.Join(tmpdir, "smbd.conf"))
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	// FIXME: generate on the fly via template
	s := `[global]
workgroup = TESTGROUP
interfaces = lo 127.0.0.0/8
smb ports = 1445
log level = 2
map to guest = Bad User
passdb backend = smbpasswd
smb passwd file = /tmp/samba/smbpasswd
lock directory = /tmp/samba/intern
state directory = /tmp/samba/intern
cache directory = /tmp/samba/intern
pid directory = /tmp/samba/intern
private dir = /tmp/samba/intern
ncalrpc dir = /tmp/samba/intern

[public]
path = /tmp/samba/public
guest ok = yes

[private]
path = /tmp/samba/private
read only = no
`
	f.Write([]byte(s))
	return f.Name()
}

func startSmbd() (SmbdContext, io.Reader) {
	// thanks pitti :)
	os.Setenv("LIBSMB_PROG", "nc localhost 1445")
	smb_conf := generateSmbdConf()
	cmd := exec.Command("smbd", "-iFS", "-s", smb_conf)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	smbdContext := SmbdContext{smbd: cmd}
	cmd.Start()
	return smbdContext, stdout
}

func (s *SmbdContext) Stop() {
	s.smbd.Process.Kill()
	s.smbd.Process.Wait()
}

func TestLibsmbclientBindings(t *testing.T) {
	fmt.Println("Start libsmbclient test")
	smbdContext, stdout := startSmbd()

	// keep us updated :)
	//go io.Copy(os.Stdout, stdout)

	client := New()
	d, err := client.Opendir("smb://localhost")
	if err != nil {
		t.Error("failed to open localhost ", err)
	}
	for {
		dirent, err := d.Readdir()
		if err != nil {
			break
		}
		fmt.Println(dirent)
	}

	fmt.Print(stdout)
	smbdContext.Stop()
}
