package profile

import "time"

// AppProfile is the central data model representing everything needed
// to containerize a Windows application.
type AppProfile struct {
	SchemaVersion string        `yaml:"schemaVersion" json:"schemaVersion"`
	App           AppInfo       `yaml:"app" json:"app"`
	Binary        BinaryInfo    `yaml:"binary" json:"binary"`
	Dependencies  DependencySet `yaml:"dependencies" json:"dependencies"`
	Registry      []RegistryEntry `yaml:"registry,omitempty" json:"registry,omitempty"`
	Environment   map[string]string `yaml:"environment,omitempty" json:"environment,omitempty"`
	FileSystem    FileSystemReqs `yaml:"filesystem,omitempty" json:"filesystem,omitempty"`
	Network       NetworkReqs    `yaml:"network,omitempty" json:"network,omitempty"`
	Display       DisplayReqs    `yaml:"display,omitempty" json:"display,omitempty"`
	Services      []ServiceDep   `yaml:"services,omitempty" json:"services,omitempty"`
	Packages      []PackageRef   `yaml:"packages,omitempty" json:"packages,omitempty"`
	Security      SecurityConfig `yaml:"security,omitempty" json:"security,omitempty"`
	Build         BuildConfig    `yaml:"build" json:"build"`
	Metadata      Metadata       `yaml:"metadata" json:"metadata"`
}

type AppInfo struct {
	Name      string `yaml:"name" json:"name"`
	Version   string `yaml:"version,omitempty" json:"version,omitempty"`
	Publisher string `yaml:"publisher,omitempty" json:"publisher,omitempty"`
}

type BinaryInfo struct {
	Path       string   `yaml:"path" json:"path"`
	Args       []string `yaml:"args,omitempty" json:"args,omitempty"`
	WorkingDir string   `yaml:"workingDir,omitempty" json:"workingDir,omitempty"`
	Arch       string   `yaml:"arch" json:"arch"`           // x86, x64, arm64
	Subsystem  string   `yaml:"subsystem" json:"subsystem"` // gui, console, service
}

type DependencySet struct {
	DLLs     []DLLDependency `yaml:"dlls" json:"dlls"`
	COM      []COMDependency `yaml:"com,omitempty" json:"com,omitempty"`
	Fonts    []string        `yaml:"fonts,omitempty" json:"fonts,omitempty"`
	VCRedist []string        `yaml:"vcredist,omitempty" json:"vcredist,omitempty"` // e.g. "vc2015", "vc2019"
	DotNet   *DotNetReq      `yaml:"dotnet,omitempty" json:"dotnet,omitempty"`
	DirectX  *DirectXReq     `yaml:"directx,omitempty" json:"directx,omitempty"`
}

type DLLDependency struct {
	Name        string `yaml:"name" json:"name"`
	FullPath    string `yaml:"fullPath,omitempty" json:"fullPath,omitempty"`
	Version     string `yaml:"version,omitempty" json:"version,omitempty"`
	IsSystem    bool   `yaml:"isSystem" json:"isSystem"`
	IsDelayLoad bool   `yaml:"isDelayLoad,omitempty" json:"isDelayLoad,omitempty"`
	Source      string `yaml:"source,omitempty" json:"source,omitempty"` // "static", "runtime", "package"
}

type COMDependency struct {
	CLSID       string `yaml:"clsid" json:"clsid"`
	ProgID      string `yaml:"progId,omitempty" json:"progId,omitempty"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
	DLLPath     string `yaml:"dllPath,omitempty" json:"dllPath,omitempty"`
}

type DotNetReq struct {
	Version   string `yaml:"version" json:"version"`     // e.g. "4.8", "6.0", "8.0"
	Framework string `yaml:"framework" json:"framework"` // "framework" or "core"
}

type DirectXReq struct {
	Version string `yaml:"version" json:"version"` // e.g. "9", "11", "12"
}

type RegistryEntry struct {
	Hive      string      `yaml:"hive" json:"hive"`                               // HKLM, HKCU, HKCR
	Path      string      `yaml:"path" json:"path"`
	ValueName string      `yaml:"valueName,omitempty" json:"valueName,omitempty"`
	ValueType string      `yaml:"valueType,omitempty" json:"valueType,omitempty"` // REG_SZ, REG_DWORD, etc.
	Value     interface{} `yaml:"value,omitempty" json:"value,omitempty"`
	Required  bool        `yaml:"required" json:"required"`
}

type FileSystemReqs struct {
	ConfigPaths []string `yaml:"configPaths,omitempty" json:"configPaths,omitempty"`
	DataDirs    []string `yaml:"dataDirs,omitempty" json:"dataDirs,omitempty"`
	TempDirs    []string `yaml:"tempDirs,omitempty" json:"tempDirs,omitempty"`
	ExtraFiles  []string `yaml:"extraFiles,omitempty" json:"extraFiles,omitempty"` // Additional files to copy
}

type NetworkReqs struct {
	Ports     []PortMapping `yaml:"ports,omitempty" json:"ports,omitempty"`
	Protocols []string      `yaml:"protocols,omitempty" json:"protocols,omitempty"` // tcp, udp
}

type PortMapping struct {
	Container int    `yaml:"container" json:"container"`
	Host      int    `yaml:"host,omitempty" json:"host,omitempty"`
	Protocol  string `yaml:"protocol,omitempty" json:"protocol,omitempty"` // tcp, udp
}

type DisplayReqs struct {
	Width      int  `yaml:"width,omitempty" json:"width,omitempty"`
	Height     int  `yaml:"height,omitempty" json:"height,omitempty"`
	ColorDepth int  `yaml:"colorDepth,omitempty" json:"colorDepth,omitempty"`
	GPU        bool `yaml:"gpu,omitempty" json:"gpu,omitempty"`
	Audio      bool `yaml:"audio,omitempty" json:"audio,omitempty"`
}

type ServiceDep struct {
	Name        string `yaml:"name" json:"name"`
	DisplayName string `yaml:"displayName,omitempty" json:"displayName,omitempty"`
	Required    bool   `yaml:"required" json:"required"`
}

type PackageRef struct {
	Manager string `yaml:"manager" json:"manager"` // winget, chocolatey, scoop
	ID      string `yaml:"id" json:"id"`           // package identifier
	Version string `yaml:"version,omitempty" json:"version,omitempty"`
}

// SecurityConfig holds encryption, firewall, and VPN settings.
type SecurityConfig struct {
	Encryption EncryptionConfig `yaml:"encryption,omitempty" json:"encryption,omitempty"`
	Firewall   FirewallConfig   `yaml:"firewall,omitempty" json:"firewall,omitempty"`
	VPN        VPNConfig        `yaml:"vpn,omitempty" json:"vpn,omitempty"`
}

// EncryptionConfig controls Age-based encryption of app files inside the container.
// Files are encrypted at build time and decrypted to tmpfs (RAM) at container startup.
type EncryptionConfig struct {
	Enabled    bool   `yaml:"enabled" json:"enabled"`
	Recipient  string `yaml:"recipient,omitempty" json:"recipient,omitempty"`   // Age public key (age1...)
	KeyFile    string `yaml:"keyFile,omitempty" json:"keyFile,omitempty"`       // Path to Age identity file on host (for build)
	Passphrase bool   `yaml:"passphrase,omitempty" json:"passphrase,omitempty"` // Use passphrase instead of keypair
}

// FirewallConfig controls iptables-based network access inside the container.
type FirewallConfig struct {
	Enabled       bool           `yaml:"enabled" json:"enabled"`
	DefaultPolicy string         `yaml:"defaultPolicy,omitempty" json:"defaultPolicy,omitempty"` // "deny" or "allow" (default: deny)
	AllowRules    []FirewallRule `yaml:"allow,omitempty" json:"allow,omitempty"`
	DenyRules     []FirewallRule `yaml:"deny,omitempty" json:"deny,omitempty"`
	AllowDNS      bool           `yaml:"allowDNS,omitempty" json:"allowDNS,omitempty"`           // Allow DNS (port 53) even when default is deny
	AllowLoopback bool           `yaml:"allowLoopback,omitempty" json:"allowLoopback,omitempty"` // Allow localhost traffic
}

// FirewallRule defines an iptables allow or deny rule.
type FirewallRule struct {
	IP       string `yaml:"ip,omitempty" json:"ip,omitempty"`             // IP address or CIDR (e.g. "1.1.1.1", "10.0.0.0/8")
	Port     int    `yaml:"port,omitempty" json:"port,omitempty"`         // Port number
	PortRange string `yaml:"portRange,omitempty" json:"portRange,omitempty"` // Port range (e.g. "8000:9000")
	Protocol string `yaml:"protocol,omitempty" json:"protocol,omitempty"` // tcp, udp, or both (default: both)
	Comment  string `yaml:"comment,omitempty" json:"comment,omitempty"`   // Human-readable description
}

// VPNConfig controls WireGuard VPN integration inside the container.
type VPNConfig struct {
	Enabled   bool   `yaml:"enabled" json:"enabled"`
	Provider  string `yaml:"provider,omitempty" json:"provider,omitempty"` // "wireguard" (default), future: "openvpn"
	ConfigFile string `yaml:"configFile,omitempty" json:"configFile,omitempty"` // Path to WireGuard .conf on host
	// Inline WireGuard config (alternative to ConfigFile)
	Interface *WGInterface `yaml:"interface,omitempty" json:"interface,omitempty"`
	Peer      *WGPeer      `yaml:"peer,omitempty" json:"peer,omitempty"`
	KillSwitch bool        `yaml:"killSwitch,omitempty" json:"killSwitch,omitempty"` // Block all traffic if VPN drops
}

// WGInterface is the [Interface] section of a WireGuard config.
type WGInterface struct {
	PrivateKey string `yaml:"privateKey,omitempty" json:"privateKey,omitempty"`
	Address    string `yaml:"address,omitempty" json:"address,omitempty"`       // e.g. "10.0.0.2/32"
	DNS        string `yaml:"dns,omitempty" json:"dns,omitempty"`               // e.g. "1.1.1.1"
	ListenPort int    `yaml:"listenPort,omitempty" json:"listenPort,omitempty"`
}

// WGPeer is the [Peer] section of a WireGuard config.
type WGPeer struct {
	PublicKey           string `yaml:"publicKey,omitempty" json:"publicKey,omitempty"`
	Endpoint            string `yaml:"endpoint,omitempty" json:"endpoint,omitempty"`             // e.g. "vpn.example.com:51820"
	AllowedIPs          string `yaml:"allowedIPs,omitempty" json:"allowedIPs,omitempty"`         // e.g. "0.0.0.0/0" for full tunnel
	PersistentKeepalive int    `yaml:"persistentKeepalive,omitempty" json:"persistentKeepalive,omitempty"`
	PresharedKey        string `yaml:"presharedKey,omitempty" json:"presharedKey,omitempty"`
}

type BuildConfig struct {
	Strategy    string `yaml:"strategy" json:"strategy"` // "wine", "windows-servercore", "windows-nanoserver"
	BaseImage   string `yaml:"baseImage,omitempty" json:"baseImage,omitempty"`
	WineVersion string `yaml:"wineVersion,omitempty" json:"wineVersion,omitempty"`
}

type Metadata struct {
	CreatedAt   time.Time `yaml:"createdAt" json:"createdAt"`
	CreatedBy   string    `yaml:"createdBy" json:"createdBy"`
	HostOS      string    `yaml:"hostOS" json:"hostOS"`
	ScanMethods []string  `yaml:"scanMethods" json:"scanMethods"` // "static", "runtime", "package"
	Confidence  string    `yaml:"confidence" json:"confidence"`   // "high", "medium", "low"
}

const CurrentSchemaVersion = "1.0"
