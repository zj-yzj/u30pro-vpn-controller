//go:build windows

package manager

import (
	"os"
	"os/exec"
	"sync"
)

var testLocks sync.Map

func ownedRoot(os.FileInfo) bool          { return true }
func directorySyncUnsupported(error) bool { return true }
func acquireLock(path string, wait bool) (func(), error) {
	value, _ := testLocks.LoadOrStore(path, &sync.Mutex{})
	lock := value.(*sync.Mutex)
	if wait {
		lock.Lock()
	} else if !lock.TryLock() {
		return nil, fail("busy")
	}
	return lock.Unlock, nil
}
func detach(*exec.Cmd) {}
