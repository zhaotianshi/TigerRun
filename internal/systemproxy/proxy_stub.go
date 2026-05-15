//go:build !windows

package systemproxy

import "errors"

func Enable(proxyAddr string) error {
	return errors.New("system proxy switching is only implemented on Windows in this build")
}

func Disable() error {
	return errors.New("system proxy switching is only implemented on Windows in this build")
}
