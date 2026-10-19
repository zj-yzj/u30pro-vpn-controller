//go:build windows

package nativebridge

func LoadTrust() (Trust, error)      { return Trust{}, errUnavailable }
func HashAPK(string) (string, error) { return "", errUnavailable }
func Start() error                   { return errUnavailable }
func Serve() error                   { return errUnavailable }
func Stop() error                    { return errUnavailable }
