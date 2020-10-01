package libsmbclient_test

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"text/template"
	_ "time"

	. "gopkg.in/check.v1"

	"github.com/mvo5/libsmbclient-go"
)

func Test(t *testing.T) { TestingT(t) }

var SMB_CONF_TEMPLATE = `[global]
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

type smbclientSuite struct {
	smbdCmd *exec.Cmd
}

var _ = Suite(&smbclientSuite{})

func (s *smbclientSuite) SetUpSuite(c *C) {
	s.startSmbd(c)
}

func (s *smbclientSuite) TearDownSuite(c *C) {
	s.smbdCmd.Process.Kill()
	s.smbdCmd.Wait()
	s.smbdCmd = nil
}

func (s *smbclientSuite) generateSmbdConf(c *C) string {
	tempdir := c.MkDir()
	paths := [...]string{
		tempdir,
		filepath.Join(tempdir, "samaba", "private"),
		filepath.Join(tempdir, "samba", "public"),
	}
	for _, d := range paths {
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
	templateText := SMB_CONF_TEMPLATE
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

func (s *smbclientSuite) startSmbd(c *C) {
	// tells smbd to use a port different from "445"
	os.Setenv("LIBSMB_PROG", "nc localhost 1445")
	smb_conf := s.generateSmbdConf(c)
	cmd := exec.Command("smbd", "-FS", "-s", smb_conf)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Start()
	c.Assert(err, IsNil)
	s.smbdCmd = cmd
}

func (s *smbclientSuite) TestLibsmbclientBindings(c *C) {
	// open client
	client := libsmbclient.New()
	d, err := client.Opendir("smb://localhost")
	c.Assert(err, IsNil)

	// collect dirs
	foundSmbDirs := map[string]bool{}
	for {
		dirent, err := d.Readdir()
		if err != nil {
			break
		}
		foundSmbDirs[dirent.Name] = true
	}
	// check for expected data
	if !foundSmbDirs["private"] || !foundSmbDirs["public"] {
		c.Errorf("missing excpected folder in (%v)", foundSmbDirs)
	}
}

func getRandomFileName() string {
	return fmt.Sprintf("%d", rand.Int())
}

func openFile(client *libsmbclient.Client, path string, c chan int) {
	f, err := client.Open(path, 0, 0)
	if err != nil {
		log.Fatal(fmt.Sprintf("%s: %s", path, err))
	}
	defer func() {
		f.Close()
		c <- 1
	}()

	// FIXME: move this into the lib as ioutil.ReadFile()
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
	}
}

func readAllFilesInDir(client *libsmbclient.Client, baseDir string, c chan int) {
	d, err := client.Opendir(baseDir)
	if err != nil {
		log.Fatal(fmt.Sprintf("%s: %s", baseDir, err))
	}
	defer d.Closedir()
	for {
		dirent, err := d.Readdir()
		if err != nil {
			break
		}
		if dirent.Name == "." || dirent.Name == ".." {
			continue
		}
		if dirent.Type == libsmbclient.SMBC_DIR {
			go readAllFilesInDir(client, baseDir+dirent.Name+"/", c)
		}
		if dirent.Type == libsmbclient.SMBC_FILE {
			go openFile(client, baseDir+dirent.Name, c)
		}
	}
}

func (s *smbclientSuite) TestLibsmbclientThreaded(c *C) {
	CLIENTS := 4
	DIRS := 4
	THREADS := 8
	FILE_SIZE := 4 * 1024

	for i := 0; i < DIRS; i++ {
		dirname := fmt.Sprintf("./tmp/samba/public/%d/", i)
		os.MkdirAll(dirname, 0755)

		// create a bunch of test files
		buf := make([]byte, FILE_SIZE)
		for j := 0; j < THREADS; j++ {
			tmpf := dirname + getRandomFileName()
			ioutil.WriteFile(tmpf, buf, 0644)
		}

	}

	// make N clients
	ch := make(chan int)
	for i := 0; i < CLIENTS; i++ {
		baseDir := "smb://localhost/public/"
		client := libsmbclient.New()
		// FIXME: close eventually
		//defer client.Close()
		go readAllFilesInDir(client, baseDir, ch)
	}

	count := 0
	for count < THREADS*DIRS*CLIENTS {
		count += <-ch
	}

	fmt.Println(fmt.Sprintf("done: %d", count))
}
