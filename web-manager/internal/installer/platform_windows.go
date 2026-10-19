//go:build windows

package installer

import "os"

func rootOwned(os.FileInfo) bool                    { return false }
func (SystemHost) FreeBytes(string) (uint64, error) { return 0, problem("unsupported_device") }
func (SystemHost) Lock(string) (func(), error)      { return nil, problem("unsupported_device") }
func noReplace(string, string) error                { return problem("unsupported_device") }
