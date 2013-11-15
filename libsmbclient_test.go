package libsmbclient

import (
	"fmt"
	"testing"
	"os"
	"os/exec"
	"time"
)

type SmbdContext struct {
	smbd *exec.Cmd
}

func startSmbd() SmbdContext {
	// FIXME: need exact cmd here
	os.Setenv("SAMBA_EXEC", "nc localhost 1445")
	// FIXME: generate on the fly via template
	smb_conf := "./test/smb.conf"
	cmd := exec.Command("smbd", "-iFS", "-s", smb_conf)
	smbdContext := SmbdContext{smbd: cmd}
	cmd.Start()
	return smbdContext
}

func (s *SmbdContext) Stop(t *testing.T) {
	s.smbd.Process.Kill()
	state, err := s.smbd.Process.Wait()
	if !state.Success() || err != nil {
		t.Error(fmt.Sprintf("Smbd failed with state '%s' error: %v", state, err))
	}
}

func TestLibsmbclientBindings(t *testing.T) {
	fmt.Println("Start libsmbclient test")
	smbdContext := startSmbd()
	defer smbdContext.Stop(t)
	// FIXME: suxx use proper wait here
	time.Sleep(1 * time.Second)
}
