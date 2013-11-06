package libsmbclient

import (
	"errors"
	"unsafe"
)

/*
#cgo LDFLAGS: -lsmbclient
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
void my_smbc_init_auth_callback(SMBCCTX *context);
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

type File struct {
	smbcfile *C.SMBCFILE
}

var Global_auth_callback = func(servername, sharename string)(domain, username, password string) { return "", "", "" }

// client interface
type Client struct {
	ctx *C.SMBCCTX
}

func New() *Client {
	c := Client{}
	c.ctx = C.smbc_new_context()
	C.smbc_init_context(c.ctx)
	// FIXME: move this into a seperate function
	C.my_smbc_init_auth_callback(c.ctx)
	return &c
}

func (c *Client) Destroy() error {
	// 1 would mean we force the destroy
	_, err := C.smbc_free_context(c.ctx, C.int(0))
	return err
}

// debug stuff
func (c *Client) GetDebug() int {
	return int(C.smbc_getDebug(c.ctx))
}

func (c *Client) SetDebug(level int) {
	C.smbc_setDebug(c.ctx, C.int(level))
}

func (c *Client) GetUser() string {
	return C.GoString(C.smbc_getUser(c.ctx))
}

func (c *Client) SetUser(user string) {
	C.smbc_setUser(c.ctx, C.CString(user))
}

func (c *Client) GetWorkgroup() string {
	return C.GoString(C.smbc_getWorkgroup(c.ctx))
}

func (c *Client) SetWorkgroup(wg string) {
	C.smbc_setWorkgroup(c.ctx, C.CString(wg))
}

func (c *Client) Opendir(durl string) (File, error) {
	d, err := C.my_smbc_opendir(c.ctx, C.CString(durl))
	return File{d}, err
}

func (c *Client) Closedir(dir File) error {
	_, err := C.my_smbc_closedir(c.ctx, dir.smbcfile)
	return err
}

func (c *Client) Readdir(dir File) (*Dirent, error) {
	c_dirent, err := C.my_smbc_readdir(c.ctx, dir.smbcfile)
	if c_dirent == nil {
		return nil, errors.New("dirent NULL")
	}
	dirent := Dirent{Type: SmbcType(c_dirent.smbc_type),
		Comment: C.GoStringN(c_dirent.comment, C.int(c_dirent.commentlen)),
		Name:    C.GoStringN(&c_dirent.name[0], C.int(c_dirent.namelen))}
	return &dirent, err
}

// FIXME: mode is actually "mode_t mode"
func (c *Client) Open(furl string, flags int, mode int) (File, error) {
	cs := C.CString(furl)
	sf, err := C.my_smbc_open(c.ctx, cs, C.int(flags), C.mode_t(mode))
	return File{smbcfile: sf}, err
}

func (c *Client) Read(f File, buf []byte) (int, error) {
	c_count, err := C.my_smbc_read(c.ctx, f.smbcfile, unsafe.Pointer(&buf[0]), C.size_t(len(buf)))
	return int(c_count), err
}

func (c *Client) Close(f File) {
	C.my_smbc_close(c.ctx, f.smbcfile)
}


//export GoAuthCallbackHelper
func GoAuthCallbackHelper(server_name, share_name *C.char, domain_out *C.char, domain_len C.int, username_out *C.char, ulen C.int, password_out *C.char, pwlen C.int) {
	domain, user, pw := Global_auth_callback(C.GoString(server_name), C.GoString(share_name))
	C.strncpy(domain_out, C.CString(domain), C.size_t(domain_len))
	C.strncpy(username_out, C.CString(user), C.size_t(ulen))
	C.strncpy(password_out, C.CString(pw), C.size_t(pwlen))
}
