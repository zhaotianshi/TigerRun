//go:build windows

package systemproxy

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows/registry"
)

const internetSettingsPath = `Software\Microsoft\Windows\CurrentVersion\Internet Settings`

func Enable(proxyAddr string) error {
	key, err := registry.OpenKey(registry.CURRENT_USER, internetSettingsPath, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()

	if err := key.SetDWordValue("ProxyEnable", 1); err != nil {
		return err
	}
	if err := key.SetStringValue("ProxyServer", fmt.Sprintf("http=%s;https=%s", proxyAddr, proxyAddr)); err != nil {
		return err
	}
	if err := key.SetStringValue("ProxyOverride", "<local>"); err != nil {
		return err
	}
	notifyInternetSettingsChanged()
	return nil
}

func Disable() error {
	key, err := registry.OpenKey(registry.CURRENT_USER, internetSettingsPath, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()

	if err := key.SetDWordValue("ProxyEnable", 0); err != nil {
		return err
	}
	notifyInternetSettingsChanged()
	return nil
}

func notifyInternetSettingsChanged() {
	wininet := syscall.NewLazyDLL("wininet.dll")
	internetSetOption := wininet.NewProc("InternetSetOptionW")
	const (
		internetOptionSettingsChanged = 39
		internetOptionRefresh         = 37
	)
	_, _, _ = internetSetOption.Call(0, internetOptionSettingsChanged, 0, 0)
	_, _, _ = internetSetOption.Call(0, internetOptionRefresh, uintptr(unsafe.Pointer(nil)), 0)
}
