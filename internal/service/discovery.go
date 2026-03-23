package service

import "github.com/theencryptedafro/appwrap/internal/discovery"

// InstalledApp re-exports the discovery type for frontend consumption.
type InstalledApp = discovery.InstalledApp

// ListInstalledApps discovers all installed applications on the system.
func (s *AppService) ListInstalledApps() ([]InstalledApp, error) {
	return discovery.ScanInstalled()
}
