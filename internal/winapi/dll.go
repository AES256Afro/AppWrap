package winapi

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

// ResolveDLLPath finds the full path of a DLL using Windows DLL search order:
// 1. Application directory
// 2. System directory (System32 or SysWOW64)
// 3. Windows directory
// 4. Current directory
// 5. PATH directories
func ResolveDLLPath(dllName, appDir, arch string) string {
	// 1. Application directory
	candidate := filepath.Join(appDir, dllName)
	if fileExists(candidate) {
		return candidate
	}

	winDir := os.Getenv("SystemRoot")
	if winDir == "" {
		winDir = `C:\Windows`
	}

	// 2. System directory
	sys32 := filepath.Join(winDir, "System32")
	sysWOW := filepath.Join(winDir, "SysWOW64")

	if arch == "x86" {
		// 32-bit apps use SysWOW64 on 64-bit Windows
		candidate = filepath.Join(sysWOW, dllName)
		if fileExists(candidate) {
			return candidate
		}
	}
	candidate = filepath.Join(sys32, dllName)
	if fileExists(candidate) {
		return candidate
	}

	// 3. Windows directory
	candidate = filepath.Join(winDir, dllName)
	if fileExists(candidate) {
		return candidate
	}

	// 4. PATH directories
	pathDirs := strings.Split(os.Getenv("PATH"), ";")
	for _, dir := range pathDirs {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			continue
		}
		candidate = filepath.Join(dir, dllName)
		if fileExists(candidate) {
			return candidate
		}
	}

	return ""
}

// GetFileVersion reads the file version string from a PE file's version resource.
func GetFileVersion(path string) string {
	pathUTF16, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return ""
	}

	mod := syscall.MustLoadDLL("version.dll")
	getInfoSize := mod.MustFindProc("GetFileVersionInfoSizeW")
	getInfo := mod.MustFindProc("GetFileVersionInfoW")
	queryValue := mod.MustFindProc("VerQueryValueW")

	size, _, _ := getInfoSize.Call(uintptr(unsafe.Pointer(pathUTF16)), 0)
	if size == 0 {
		return ""
	}

	data := make([]byte, size)
	ret, _, _ := getInfo.Call(
		uintptr(unsafe.Pointer(pathUTF16)),
		0,
		uintptr(size),
		uintptr(unsafe.Pointer(&data[0])),
	)
	if ret == 0 {
		return ""
	}

	subBlock, _ := syscall.UTF16PtrFromString(`\`)
	var info unsafe.Pointer
	var infoLen uint32

	ret, _, _ = queryValue.Call(
		uintptr(unsafe.Pointer(&data[0])),
		uintptr(unsafe.Pointer(subBlock)),
		uintptr(unsafe.Pointer(&info)),
		uintptr(unsafe.Pointer(&infoLen)),
	)
	if ret == 0 || infoLen == 0 {
		return ""
	}

	type VS_FIXEDFILEINFO struct {
		Signature        uint32
		StrucVersion     uint32
		FileVersionMS    uint32
		FileVersionLS    uint32
		ProductVersionMS uint32
		ProductVersionLS uint32
		FileFlagsMask    uint32
		FileFlags        uint32
		FileOS           uint32
		FileType         uint32
		FileSubtype      uint32
		FileDateMS       uint32
		FileDateLS       uint32
	}

	fi := (*VS_FIXEDFILEINFO)(info)
	if fi.Signature != 0xFEEF04BD {
		return ""
	}

	major := fi.FileVersionMS >> 16
	minor := fi.FileVersionMS & 0xFFFF
	patch := fi.FileVersionLS >> 16
	build := fi.FileVersionLS & 0xFFFF

	return strings.TrimRight(
		strings.Join([]string{
			itoa(major), itoa(minor), itoa(patch), itoa(build),
		}, "."),
		".0",
	)
}

func itoa(n uint32) string {
	if n == 0 {
		return "0"
	}
	buf := [10]byte{}
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
