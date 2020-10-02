package libsmbclient

import (
	"fmt"
	"io"
	"sync"
	"unsafe"
)

/*
#cgo LDFLAGS: -lsmbclient
#include <stdlib.h>
#include <string.h>
#include <unistd.h>


#include <samba-4.0/libsmbclient.h>

SMBCFILE* my_smbc_opendir(SMBCCTX *c, const char *fname);
int my_smbc_closedir(SMBCCTX *c, SMBCFILE *dir);
struct smbc_dirent* my_smbc_readdir(SMBCCTX *c, SMBCFILE *dir);
SMBCFILE* my_smbc_open(SMBCCTX *c, const char *fname, int flags, mode_t mode);
ssize_t my_smbc_read(SMBCCTX *c, SMBCFILE *file, void *buf, size_t count);
void my_smbc_close(SMBCCTX *c, SMBCFILE *f);
void my_smbc_auth_callback(SMBCCTX *context,
               const char *server_name, const char *share_name,
	       char *domain_out, int domainmaxlen,
	       char *username_out, int unmaxlen,
	       char *password_out, int pwmaxlen);
void my_smbc_init_auth_callback(SMBCCTX *context, void *go_fn);
off_t my_smbc_lseek(SMBCCTX *c, SMBCFILE * file, off_t offset, int whence);
*/
import "C" // DO NOT CHANGE THE POSITION OF THIS IMPORT

// SmbType
type SmbcType int

const (
	SMBC_WORKGROUP     SmbcType = C.SMBC_WORKGROUP
	SMBC_FILE_SHARE             = C.SMBC_FILE_SHARE
	SMBC_PRINTER_SHARE          = C.SMBC_PRINTER_SHARE
	SMBC_COMMS_SHARE            = C.SMBC_COMMS_SHARE
	SMBC_IPC_SHARE              = C.SMBC_IPC_SHARE
	SMBC_DIR                    = C.SMBC_DIR
	SMBC_FILE                   = C.SMBC_FILE
	SMBC_LINK                   = C.SMBC_LINK
)

type Dirent struct {
	Type    SmbcType
	Comment string
	Name    string
}

// *sigh* even with libsmbclient-4.0 the library is not MT safe,
// e.g. smbc_init_context from multiple threads crashes
var smbMu = sync.Mutex{}

// client interface
type Client struct {
	ctx          *C.SMBCCTX
	authCallback *func(string, string) (string, string, string)
	// libsmbclient is not thread safe
	smbMu *sync.Mutex
}

// File wrapper
type File struct {
	client   *Client
	smbcfile *C.SMBCFILE
}

func New() *Client {
	smbMu.Lock()
	defer smbMu.Unlock()

	c := &Client{ctx: C.smbc_new_context(),
		smbMu: &smbMu}
	C.smbc_init_context(c.ctx)
	// this does not work reliable, see TestLibsmbclientThreaded test
	/*
		runtime.SetFinalizer(c, func(c2 *Client) {
			fmt.Println(fmt.Sprintf("d: %v", c2))
			c2.Close()
		})
	*/

	return c
}

func (c *Client) Destroy() error {
	return c.Close()
}

func (c *Client) Close() error {
	// FIXME: is there a more elegant way for this c.lock.Lock() that
	//        needs to be part of every function? python decorator to
	//        the rescue :)
	c.smbMu.Lock()
	defer c.smbMu.Unlock()

	var err error
	if c.ctx != nil {
		// 1 would mean we force the destroy
		_, err = C.smbc_free_context(c.ctx, C.int(1))
		c.ctx = nil
	}
	return err
}

// authentication callback, this expects a go callback function
// with the signature:
//  func(serverName, shareName) (domain, username, password)
func (c *Client) SetAuthCallback(fn func(string, string) (string, string, string)) {
	c.smbMu.Lock()
	defer c.smbMu.Unlock()

	C.my_smbc_init_auth_callback(c.ctx, unsafe.Pointer(&fn))
	// we need to store it in the Client struct to ensure its not garbage
	// collected later (I think)
	c.authCallback = &fn
}

// options

// SetUseKerberos enable krb5 integration for authentication
func (c *Client) SetUseKerberos() {
	c.smbMu.Lock()
	defer c.smbMu.Unlock()

	C.smbc_setOptionUseKerberos(c.ctx, C.int(1))
}

func (c *Client) GetDebug() int {
	c.smbMu.Lock()
	defer c.smbMu.Unlock()

	return int(C.smbc_getDebug(c.ctx))
}

func (c *Client) SetDebug(level int) {
	c.smbMu.Lock()
	defer c.smbMu.Unlock()

	C.smbc_setDebug(c.ctx, C.int(level))
}

func (c *Client) GetUser() string {
	c.smbMu.Lock()
	defer c.smbMu.Unlock()

	return C.GoString(C.smbc_getUser(c.ctx))
}

func (c *Client) SetUser(user string) {
	c.smbMu.Lock()
	defer c.smbMu.Unlock()

	C.smbc_setUser(c.ctx, C.CString(user))
}

func (c *Client) GetWorkgroup() string {
	c.smbMu.Lock()
	defer c.smbMu.Unlock()

	return C.GoString(C.smbc_getWorkgroup(c.ctx))
}

func (c *Client) SetWorkgroup(wg string) {
	c.smbMu.Lock()
	defer c.smbMu.Unlock()

	C.smbc_setWorkgroup(c.ctx, C.CString(wg))
}

// dir stuff

func (c *Client) Opendir(durl string) (File, error) {
	c.smbMu.Lock()
	defer c.smbMu.Unlock()

	d, err := C.my_smbc_opendir(c.ctx, C.CString(durl))
	if d == nil {
		return File{}, fmt.Errorf("cannot open %v: %v", durl, err)
	}
	return File{client: c, smbcfile: d}, nil
}

func (dir *File) Closedir() error {
	dir.client.smbMu.Lock()
	defer dir.client.smbMu.Unlock()

	if dir.smbcfile != nil {
		_, err := C.my_smbc_closedir(dir.client.ctx, dir.smbcfile)
		dir.smbcfile = nil
		return err
	}
	return nil
}

func (dir *File) Readdir() (*Dirent, error) {
	dir.client.smbMu.Lock()
	defer dir.client.smbMu.Unlock()

	cDirent, err := C.my_smbc_readdir(dir.client.ctx, dir.smbcfile)
	if cDirent == nil && err != nil {
		return nil, fmt.Errorf("cannot readdir: %v", err)
	}
	if cDirent == nil {
		return nil, io.EOF
	}
	dirent := Dirent{Type: SmbcType(cDirent.smbc_type),
		Comment: C.GoStringN(cDirent.comment, C.int(cDirent.commentlen)),
		Name:    C.GoStringN(&cDirent.name[0], C.int(cDirent.namelen))}
	return &dirent, nil
}

// file stuff

// FIXME: mode is actually "mode_t mode"
func (c *Client) Open(furl string, flags int, mode int) (File, error) {
	c.smbMu.Lock()
	defer c.smbMu.Unlock()

	cs := C.CString(furl)
	sf, err := C.my_smbc_open(c.ctx, cs, C.int(flags), C.mode_t(mode))
	if sf == nil && err != nil {
		return File{}, fmt.Errorf("cannot open %v: %v", furl, err)
	}

	return File{client: c, smbcfile: sf}, nil
}

func (f *File) Read(buf []byte) (int, error) {
	f.client.smbMu.Lock()
	defer f.client.smbMu.Unlock()

	cCount, err := C.my_smbc_read(f.client.ctx, f.smbcfile, unsafe.Pointer(&buf[0]), C.size_t(len(buf)))
	c := int(cCount)
	if c == 0 {
		return 0, io.EOF
	} else if c < 0 && err != nil {
		return c, fmt.Errorf("cannot read: %v", err)
	}
	return c, nil
}

func (f *File) Lseek(offset, whence int) (int, error) {
	f.client.smbMu.Lock()
	defer f.client.smbMu.Unlock()

	newOffset, err := C.my_smbc_lseek(f.client.ctx, f.smbcfile, C.off_t(offset), C.int(whence))
	if int(newOffset) < 0 && err != nil {
		return int(newOffset), fmt.Errorf("cannot seek: %v", err)
	}
	return int(newOffset), nil
}

func (f *File) Close() {
	f.client.smbMu.Lock()
	defer f.client.smbMu.Unlock()

	if f.smbcfile != nil {
		C.my_smbc_close(f.client.ctx, f.smbcfile)
		f.smbcfile = nil
	}
}

// INTERNAL use only
//export GoAuthCallbackHelper
func GoAuthCallbackHelper(fn unsafe.Pointer, serverName, shareName *C.char, domainOut *C.char, domainLen C.int, usernameOut *C.char, ulen C.int, passwordOut *C.char, pwlen C.int) {
	callback := *(*func(serverName, shareName string) (string, string, string))(fn)
	domain, user, pw := callback(C.GoString(serverName), C.GoString(shareName))
	C.strncpy(domainOut, C.CString(domain), C.size_t(domainLen))
	C.strncpy(usernameOut, C.CString(user), C.size_t(ulen))
	C.strncpy(passwordOut, C.CString(pw), C.size_t(pwlen))
}
