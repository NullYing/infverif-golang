package verifier

import (
	"fmt"
	"strings"
)

// ExceptionType represents the type of a static exception.
type ExceptionType int

const (
	ExceptionFile     ExceptionType = 0
	ExceptionRegistry ExceptionType = 1
)

// Exception represents a static exception entry in the /h mode database.
type Exception struct {
	Type          ExceptionType
	Root          string // Registry root (HKLM/HKCR) or DIRID for files
	Path          string // Subkey path or file subpath
	RemoveVersion string // OS version when this exception is removed (empty = no planned removal)
}

// RegistryExceptions contains the 41 registry path exceptions for /h mode.
var RegistryExceptions = []Exception{
	{ExceptionRegistry, "HKLM", `SYSTEM\CurrentControlSet`, "10.0.26100"},
	{ExceptionRegistry, "HKLM", `SOFTWARE\Classes`, ""},
	{ExceptionRegistry, "HKLM", `SOFTWARE\Khronos`, ""},
	{ExceptionRegistry, "HKLM", `SOFTWARE\Microsoft\Analog\Providers`, ""},
	{ExceptionRegistry, "HKLM", `SOFTWARE\Microsoft\Cellular\MVSettings\DeviceSpecific\CellUX`, ""},
	{ExceptionRegistry, "HKLM", `SOFTWARE\Microsoft\Cryptography\Calais\Readers`, ""},
	{ExceptionRegistry, "HKLM", `SOFTWARE\Microsoft\Cryptography\Calais\SmartCards`, ""},
	{ExceptionRegistry, "HKLM", `SOFTWARE\Microsoft\Cryptography\DRM_RNG`, ""},
	{ExceptionRegistry, "HKLM", `SOFTWARE\Microsoft\EAPOL`, ""},
	{ExceptionRegistry, "HKLM", `SOFTWARE\Microsoft\Palm\DelayManipulationDuration`, ""},
	{ExceptionRegistry, "HKLM", `SOFTWARE\Microsoft\Shell\OEM\QuickActions\ColorProfileQuickAction`, "10.0.26100"},
	{ExceptionRegistry, "HKLM", `SOFTWARE\Microsoft\Speech_OneCore\AudioInput`, "10.0.26100"},
	{ExceptionRegistry, "HKLM", `SOFTWARE\Microsoft\Windows Media Foundation`, ""},
	{ExceptionRegistry, "HKLM", `SOFTWARE\Microsoft\Windows NT\CurrentVersion\AdaptiveDisplayBrightness`, ""},
	{ExceptionRegistry, "HKLM", `SOFTWARE\Microsoft\Windows NT\CurrentVersion\drivers.desc`, "10.0.26100"},
	{ExceptionRegistry, "HKLM", `SOFTWARE\Microsoft\Windows NT\CurrentVersion\Drivers32`, "10.0.26100"},
	{ExceptionRegistry, "HKLM", `SOFTWARE\Microsoft\Windows NT\CurrentVersion\ICM`, "10.0.26100"},
	{ExceptionRegistry, "HKLM", `SOFTWARE\Microsoft\Windows NT\CurrentVersion\OpenGlDrivers`, ""},
	{ExceptionRegistry, "HKLM", `SOFTWARE\Microsoft\Windows NT\CurrentVersion\Winlogon\Notify\ScCertProp`, ""},
	{ExceptionRegistry, "HKLM", `SOFTWARE\Microsoft\Windows\CurrentVersion\App Paths`, ""},
	{ExceptionRegistry, "HKLM", `SOFTWARE\Microsoft\Windows\CurrentVersion\Audio`, "10.0.26100"},
	{ExceptionRegistry, "HKLM", `SOFTWARE\Microsoft\Windows\CurrentVersion\Authentication`, ""},
	{ExceptionRegistry, "HKLM", `SOFTWARE\Microsoft\Windows\CurrentVersion\Control Panel`, ""},
	{ExceptionRegistry, "HKLM", `SOFTWARE\Microsoft\Windows\CurrentVersion\Controls Folder`, ""},
	{ExceptionRegistry, "HKLM", `SOFTWARE\Microsoft\Windows\CurrentVersion\Explorer`, ""},
	{ExceptionRegistry, "HKLM", `SOFTWARE\Microsoft\Windows\CurrentVersion\Reliability\UserDefined`, "10.0.26100"},
	{ExceptionRegistry, "HKLM", `SOFTWARE\Microsoft\Windows\CurrentVersion\Run`, ""},
	{ExceptionRegistry, "HKLM", `SOFTWARE\Microsoft\Windows\CurrentVersion\RunOnce`, ""},
	{ExceptionRegistry, "HKLM", `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`, ""},
	{ExceptionRegistry, "HKLM", `SOFTWARE\Wow6432Node\Microsoft\Windows Media Foundation`, ""},
	{ExceptionRegistry, "HKLM", `SOFTWARE\Wow6432Node\Khronos`, ""},
	{ExceptionRegistry, "HKLM", `SOFTWARE\WowAA32Node\Microsoft\Windows Media Foundation`, ""},
	{ExceptionRegistry, "HKLM", `SOFTWARE\WowAA32Node\Khronos`, ""},
	{ExceptionRegistry, "HKCR", "", ""},
}

// FileExceptions contains the 23 file path exceptions for /h mode.
var FileExceptions = []Exception{
	{ExceptionFile, "10", `Provisioning`, ""},
	{ExceptionFile, "10", `SyChpe32`, ""},
	{ExceptionFile, "10", `SysArm32`, ""},
	{ExceptionFile, "10", `TWAIN_32`, ""},
	{ExceptionFile, "10", `Twain_64`, ""},
	{ExceptionFile, "11", "", ""},
	{ExceptionFile, "12", "", ""},
	{ExceptionFile, "23", "", ""},
	{ExceptionFile, "51", "", ""},
	{ExceptionFile, "52", "", ""},
	{ExceptionFile, "55", "", ""},
	{ExceptionFile, "16422", "", "10.0.26100"},
	{ExceptionFile, "16425", "", ""},
	{ExceptionFile, "16426", "", "10.0.26100"},
	{ExceptionFile, "16427", "", "10.0.26100"},
	{ExceptionFile, "16428", "", "10.0.26100"},
	{ExceptionFile, "66000", "", ""},
	{ExceptionFile, "66001", "", ""},
	{ExceptionFile, "66002", "", "10.0.26100"},
	{ExceptionFile, "66003", "", ""},
	{ExceptionFile, "66004", "", ""},
}

// DIRID display names for readable output
var diridNames = map[string]string{
	"10":    "Windows",
	"11":    `Windows\System32`,
	"12":    `Windows\System32\drivers`,
	"23":    `Windows\System32\spool\drivers\color`,
	"51":    `Windows\System32\spool`,
	"52":    `Windows\System32\spool\drivers`,
	"55":    `Windows\System32\spool\prtprocs`,
	"16422": "Program Files",
	"16425": `Windows\SysWOW64`,
	"16426": "Program Files (x86)",
	"16427": `Program Files\Common Files`,
	"16428": `Program Files (x86)\Common Files`,
	"66000": `Windows\System32\spool\drivers\...\3`,
	"66001": `Windows\System32\spool\prtprocs`,
	"66002": "Windows",
	"66003": `Windows\System32\spool\drivers\color`,
	"66004": `Windows\web\printers`,
}

// RuleVersion represents a parsed rule version (Major.Minor.Build).
type RuleVersion struct {
	Major int
	Minor int
	Build int
}

// String returns the version as "Major.Minor.Build".
func (rv RuleVersion) String() string {
	return fmt.Sprintf("%d.%d.%d", rv.Major, rv.Minor, rv.Build)
}

// DefaultRuleVersion is the default rule version for /h mode.
var DefaultRuleVersion = RuleVersion{Major: 10, Minor: 0, Build: 26200}

// NamedRuleVersions maps named versions to their build numbers.
var NamedRuleVersions = map[string]RuleVersion{
	"vnext":   {10, 0, 99999},
	"vnext_2": {10, 0, 99998},
	"24h2":    {10, 0, 26100},
	"25h2":    {10, 0, 26200},
	"26h2":    {10, 0, 26300},
	"27h2":    {10, 0, 26400},
}

// ParseRuleVersion parses a rule version string.
// Accepts "Major.Minor.Build" or named versions (vnext, 24h2, etc.)
func ParseRuleVersion(s string) (RuleVersion, bool) {
	// Check named versions
	if rv, ok := NamedRuleVersions[strings.ToLower(s)]; ok {
		return rv, true
	}
	// Parse Major.Minor.Build
	var major, minor, build int
	n, _ := fmt.Sscanf(s, "%d.%d.%d", &major, &minor, &build)
	if n == 3 {
		return RuleVersion{major, minor, build}, true
	}
	return RuleVersion{}, false
}

// IsExceptionActive checks if an exception is still active for a given rule version.
func IsExceptionActive(exc Exception, rv RuleVersion) bool {
	if exc.RemoveVersion == "" {
		return true // No removal planned
	}
	var rmMajor, rmMinor, rmBuild int
	n, _ := fmt.Sscanf(exc.RemoveVersion, "%d.%d.%d", &rmMajor, &rmMinor, &rmBuild)
	if n != 3 {
		return true
	}
	// Active if rule version < removal version
	if rv.Major < rmMajor {
		return true
	}
	if rv.Major == rmMajor && rv.Minor < rmMinor {
		return true
	}
	if rv.Major == rmMajor && rv.Minor == rmMinor && rv.Build < rmBuild {
		return true
	}
	return false
}

// IsRegistryPathExempt checks if a registry path is in the exception list.
func IsRegistryPathExempt(root, subkey string, rv RuleVersion, noExceptions bool) bool {
	if noExceptions {
		return false
	}
	rootUpper := strings.ToUpper(root)
	subkeyLower := strings.ToLower(subkey)

	for _, exc := range RegistryExceptions {
		if !IsExceptionActive(exc, rv) {
			continue
		}
		excRoot := strings.ToUpper(exc.Root)
		if excRoot != rootUpper {
			continue
		}
		if exc.Path == "" {
			// Entire root is exempt (e.g. HKCR)
			return true
		}
		excPath := strings.ToLower(exc.Path)
		if subkeyLower == excPath || strings.HasPrefix(subkeyLower, excPath+`\`) {
			return true
		}
	}
	return false
}

// IsFilePathExempt checks if a DIRID is in the file exception list.
func IsFilePathExempt(dirid string, subpath string, rv RuleVersion, noExceptions bool) bool {
	if noExceptions {
		return false
	}
	subpathLower := strings.ToLower(subpath)

	for _, exc := range FileExceptions {
		if !IsExceptionActive(exc, rv) {
			continue
		}
		if exc.Root != dirid {
			continue
		}
		if exc.Path == "" {
			return true // Entire DIRID is exempt
		}
		excPath := strings.ToLower(exc.Path)
		if subpathLower == excPath || strings.HasPrefix(subpathLower, excPath+`\`) {
			return true
		}
	}
	return false
}

// SyntaxEntry represents a syntax feature found in an INF file.
type SyntaxEntry struct {
	Name       string
	MinVersion RuleVersion
}

// knownSyntaxFeatures maps INF directives/features to their minimum supported OS versions.
var knownSyntaxFeatures = map[string]RuleVersion{
	"AddService":                  {10, 0, 10240},
	"AddReg":                      {10, 0, 10240},
	"CopyFiles":                   {10, 0, 10240},
	"DelReg":                      {10, 0, 10240},
	"Include":                     {10, 0, 10240},
	"Needs":                       {10, 0, 10240},
	"AddFilter":                   {10, 0, 25319},
	"AddEventProvider":            {10, 0, 18362},
	"AddSoftwareDevice":           {10, 0, 22000},
	"PnpLockdown":                 {10, 0, 10240},
	"AddInterface":                {10, 0, 10240},
	"DestinationDirs.DefaultOnly": {10, 0, 10240},
	"SourceDisksFiles":            {10, 0, 10240},
	"SourceDisksNames":            {10, 0, 10240},
}
