package main

import (
	"os"
	"path/filepath"

	"github.com/tc-hib/winres"
	"github.com/tc-hib/winres/version"
)

const (
	appName    = "老虎快跑"
	exeName    = "老虎快跑.exe"
	appVersion = "0.1.0"
)

func main() {
	iconFile, err := os.Open(filepath.Join("build", "windows", "icon.ico"))
	must(err)
	defer iconFile.Close()

	icon, err := winres.LoadICO(iconFile)
	must(err)

	rs := winres.ResourceSet{}
	must(rs.SetIcon(winres.ID(3), icon))

	rs.SetManifest(winres.AppManifest{
		DPIAwareness:        winres.DPIPerMonitorV2,
		UseCommonControlsV6: true,
	})

	info := version.Info{
		FileVersion:    [4]uint16{0, 1, 0, 0},
		ProductVersion: [4]uint16{0, 1, 0, 0},
	}
	setVersion(&info, version.ProductName, appName)
	setVersion(&info, version.FileDescription, appName)
	setVersion(&info, version.CompanyName, appName)
	setVersion(&info, version.InternalName, appName)
	setVersion(&info, version.OriginalFilename, exeName)
	setVersion(&info, version.FileVersion, appVersion)
	setVersion(&info, version.ProductVersion, appVersion)
	setVersion(&info, version.Comments, "HTTP/HTTPS debugging proxy")
	rs.SetVersionInfo(info)

	out, err := os.Create("laohukuaipao_windows_amd64.syso")
	must(err)
	defer out.Close()
	must(rs.WriteObject(out, winres.ArchAMD64))
}

func setVersion(info *version.Info, key string, value string) {
	must(info.Set(version.LangDefault, key, value))
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
