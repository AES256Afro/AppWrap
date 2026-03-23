package util

import (
	"os/exec"
	"runtime"
	"strings"
)

type SystemInfo struct {
	OS            string
	Arch          string
	DockerAvail   bool
	DockerVersion string
}

func GetSystemInfo() SystemInfo {
	info := SystemInfo{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
	}

	// Check Docker availability
	out, err := exec.Command("docker", "version", "--format", "{{.Server.Version}}").Output()
	if err == nil {
		info.DockerAvail = true
		info.DockerVersion = strings.TrimSpace(string(out))
	}

	return info
}
