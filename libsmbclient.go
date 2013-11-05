package libsmbclient

import (
	"errors"
	"unsafe"
)

/*
#cgo LDFLAGS: -lsmbclient

#include <libsmbclient.h>
#include <stdlib.h>
#include <unistd.h>

SMBCFILE* my_smbc_opendir(SMBCCTX *c, const char *fname) {
  smbc_opendir_fn fn = smbc_getFunctionOpendir(c);
  return fn(c, fname);
}

int my_smbc_closedir(SMBCCTX *c, SMBCFILE *dir) {
  smbc_closedir_fn fn = smbc_getFunctionClosedir(c);
  return fn(c, dir);
}

struct smbc_dirent* my_smbc_readdir(SMBCCTX *c, SMBCFILE *dir) {
  smbc_readdir_fn fn = smbc_getFunctionReaddir(c);
  return fn(c, dir);
}

SMBCFILE* my_smbc_open(SMBCCTX *c, const char *fname, int flags, mode_t mode) {
  smbc_open_fn fn = smbc_getFunctionOpen(c);
  return fn(c, fname, flags, mode);
}

ssize_t my_smbc_read(SMBCCTX *c, SMBCFILE *file, void *buf, size_t count) {
  smbc_read_fn fn = smbc_getFunctionRead(c);
  return fn(c, file, buf, count);
}

void my_smbc_close(SMBCCTX *c, SMBCFILE *f) {
  smbc_close_fn fn = smbc_getFunctionClose(c);
  fn(c, f);
}

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

func New() *Client {
	c := Client{}
	c.ctx = C.smbc_new_context()
	C.smbc_init_context(c.ctx)
	return &c
}

// client interface
type Client struct {
	ctx *C.SMBCCTX
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
