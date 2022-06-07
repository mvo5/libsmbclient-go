package libsmbclient

import (
	"fmt"
	"io"
	"sync"
	"unsafe"
)

/*
#cgo pkg-config: smbclient
#include <stdlib.h>
#include <string.h>
#include <unistd.h>

#include <libsmbclient.h>

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
import "C"

// SmbcType is the different type of entity returned by samba.
type SmbcType int

const (
	// SmbcWorkgroup is a workgroup entity.
	SmbcWorkgroup SmbcType = C.SMBC_WORKGROUP
	// SmbcFileShare is a file share.
	SmbcFileShare = C.SMBC_FILE_SHARE
	// SmbcPrinterShare is a printer share.
	SmbcPrinterShare = C.SMBC_PRINTER_SHARE
	// SmbcCommsShare is a communication share.
	SmbcCommsShare = C.SMBC_COMMS_SHARE
	// SmbcIPCShare is an ipc share entity.
	SmbcIPCShare = C.SMBC_IPC_SHARE
	// SmbcDir is a directory.
	SmbcDir = C.SMBC_DIR
	// SmbcFile is a file.
	SmbcFile = C.SMBC_FILE
	// SmbcLink is a symlink.
	SmbcLink = C.SMBC_LINK
)

// *sigh* even with libsmbclient-4.0 the library is not MT safe,
// e.g. smbc_init_context from multiple threads crashes
var smbMu = sync.Mutex{}

// Client is a samba client instance, handling its own context and lock.
type Client struct {
	ctx          *C.SMBCCTX
	authCallback *AuthCallback
	// libsmbclient is not thread safe
	smbMu *sync.Mutex
}

// Dirent represents a samba directory entry.
type Dirent struct {
	Type    SmbcType
	Comment string
	Name    string
}

// File reprends a samba file.
type File struct {
	client   *Client
	smbcfile *C.SMBCFILE
}

// New creates a new samba client.
func New() *Client {
	smbMu.Lock()
	defer smbMu.Unlock()

	c := &Client{
		ctx:   C.smbc_new_context(),
		smbMu: &smbMu,
	}
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

// Destroy closes the current samba client.
func (c *Client) Destroy() error {
	return c.Close()
}

// Close closes the current samba client and release context.
func (c *Client) Close() error {
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

// AuthCallback is the authentication function that will be called during connection with samba.
type AuthCallback = func(serverName, shareName string) (domain, username, password string)

// SetAuthCallback assigns the authentication function that will be called during connection
// with samba.
func (c *Client) SetAuthCallback(f AuthCallback) {
	c.smbMu.Lock()
	defer c.smbMu.Unlock()

	C.my_smbc_init_auth_callback(c.ctx, unsafe.Pointer(&f))
	// we need to store it in the Client struct to ensure its not garbage
	// collected later (I think)
	c.authCallback = &f
}

// SetUseKerberos enable krb5 integration for authentication.
func (c *Client) SetUseKerberos() {
	c.smbMu.Lock()
	defer c.smbMu.Unlock()

	C.smbc_setOptionUseKerberos(c.ctx, C.int(1))
}

// GetDebug returns the debug level.
func (c *Client) GetDebug() int {
	c.smbMu.Lock()
	defer c.smbMu.Unlock()

	return int(C.smbc_getDebug(c.ctx))
}

// SetDebug sets the degug level.
func (c *Client) SetDebug(level int) {
	c.smbMu.Lock()
	defer c.smbMu.Unlock()

	C.smbc_setDebug(c.ctx, C.int(level))
}

// GetUser returns the authenticated user.
func (c *Client) GetUser() string {
	c.smbMu.Lock()
	defer c.smbMu.Unlock()

	return C.GoString(C.smbc_getUser(c.ctx))
}

// SetUser sets the user to use for the session.
func (c *Client) SetUser(user string) {
	c.smbMu.Lock()
	defer c.smbMu.Unlock()

	C.smbc_setUser(c.ctx, C.CString(user))
}

// GetWorkgroup returns the name of the current workgroup.
func (c *Client) GetWorkgroup() string {
	c.smbMu.Lock()
	defer c.smbMu.Unlock()

	return C.GoString(C.smbc_getWorkgroup(c.ctx))
}

// SetWorkgroup sets the work group to use for the session.
func (c *Client) SetWorkgroup(wg string) {
	c.smbMu.Lock()
	defer c.smbMu.Unlock()

	C.smbc_setWorkgroup(c.ctx, C.CString(wg))
}

// Opendir opens a directory and returns a handle on success.
func (c *Client) Opendir(durl string) (File, error) {
	c.smbMu.Lock()
	defer c.smbMu.Unlock()

	d, err := C.my_smbc_opendir(c.ctx, C.CString(durl))
	if d == nil {
		return File{}, fmt.Errorf("cannot open %v: %v", durl, err)
	}
	return File{client: c, smbcfile: d}, nil
}

// Closedir closes current directory.
func (e *File) Closedir() error {
	e.client.smbMu.Lock()
	defer e.client.smbMu.Unlock()

	if e.smbcfile != nil {
		_, err := C.my_smbc_closedir(e.client.ctx, e.smbcfile)
		e.smbcfile = nil
		return err
	}
	return nil
}

// Readdir reads the directory named pointed by File and returned its Dirent.
func (e *File) Readdir() (*Dirent, error) {
	e.client.smbMu.Lock()
	defer e.client.smbMu.Unlock()

	cDirent, err := C.my_smbc_readdir(e.client.ctx, e.smbcfile)
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

// Open opens a file and returns a handle on success.
// FIXME: mode is actually "mode_t mode"
func (c *Client) Open(furl string, flags, mode int) (File, error) {
	c.smbMu.Lock()
	defer c.smbMu.Unlock()

	cs := C.CString(furl)
	sf, err := C.my_smbc_open(c.ctx, cs, C.int(flags), C.mode_t(mode))
	if sf == nil && err != nil {
		return File{}, fmt.Errorf("cannot open %v: %v", furl, err)
	}

	return File{client: c, smbcfile: sf}, nil
}

// Read reads up to len(b) bytes from the File. It returns the number of bytes read and any error encountered.
// At end of file, Read returns 0, io.EOF.
func (e *File) Read(buf []byte) (int, error) {
	e.client.smbMu.Lock()
	defer e.client.smbMu.Unlock()

	cCount, err := C.my_smbc_read(e.client.ctx, e.smbcfile, unsafe.Pointer(&buf[0]), C.size_t(len(buf)))
	c := int(cCount)
	if c == 0 {
		return 0, io.EOF
	} else if c < 0 && err != nil {
		return c, fmt.Errorf("cannot read: %v", err)
	}
	return c, nil
}

// Lseek repositions the file offset of the open file to the argument offset according to the directive whence.
func (e *File) Lseek(offset, whence int) (int, error) {
	e.client.smbMu.Lock()
	defer e.client.smbMu.Unlock()

	newOffset, err := C.my_smbc_lseek(e.client.ctx, e.smbcfile, C.off_t(offset), C.int(whence))
	if int(newOffset) < 0 && err != nil {
		return int(newOffset), fmt.Errorf("cannot seek: %v", err)
	}
	return int(newOffset), nil
}

// Close closes current file and and releases its ressources.
func (e *File) Close() {
	e.client.smbMu.Lock()
	defer e.client.smbMu.Unlock()

	if e.smbcfile != nil {
		C.my_smbc_close(e.client.ctx, e.smbcfile)
		e.smbcfile = nil
	}
}

//export authCallbackHelper
func authCallbackHelper(fn unsafe.Pointer, serverName, shareName, domainOut *C.char, domainLen C.int, usernameOut *C.char, ulen C.int, passwordOut *C.char, pwlen C.int) {
	callback := *(*AuthCallback)(fn)
	domain, user, pw := callback(C.GoString(serverName), C.GoString(shareName))
	C.strncpy(domainOut, C.CString(domain), C.size_t(domainLen))
	C.strncpy(usernameOut, C.CString(user), C.size_t(ulen))
	C.strncpy(passwordOut, C.CString(pw), C.size_t(pwlen))
}
