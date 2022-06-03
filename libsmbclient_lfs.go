//go:build 386 || amd64p32 || arm || armbe || mips || mips64p32 || mips64p32le || mipsle || ppc || riscv || s390 || sparc
// +build 386 amd64p32 arm armbe mips mips64p32 mips64p32le mipsle ppc riscv s390 sparc

package libsmbclient

/*
#cgo CFLAGS: -D_LARGEFILE_SOURCE -D_FILE_OFFSET_BITS=64
*/
import "C"
