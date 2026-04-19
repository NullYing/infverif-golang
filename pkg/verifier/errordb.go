package verifier

// ErrorEntry represents an entry in the InfVerif error database.
type ErrorEntry struct {
	Code        int
	Flags       uint32 // HDC flags: 0x20000 = All Submissions, 0x80000 = Downlevel Declarative
	Description string
}

// HDC flag constants
const (
	HDCFlagAllSubmissions       uint32 = 0x20000
	HDCFlagDownlevelDeclarative uint32 = 0x80000
)

// errorDatabase contains the known InfVerif error codes with descriptions.
// Based on the 208-entry database at unk_14005F800 in the new binary.
var errorDatabase = []ErrorEntry{
	{1200, 0, "Invalid INF: must contain [Version] section and have signature \"$Windows NT$\"."},
	{1203, 0, "Section name too long or contains invalid characters."},
	{1204, 0, "Directive syntax error."},
	{1205, 0, "Invalid string substitution token."},
	{1206, 0, "Duplicate section name."},
	{1207, 0, "Empty or missing value."},
	{1210, 0, "Unknown directive."},
	{1220, 0, "Invalid GUID format, expecting {XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX}."},
	{1230, 0, "Missing required ServiceType in service install section."},
	{1231, 0, "Invalid service type: cannot combine SERVICE_WIN32 and SERVICE_DRIVER."},
	{1232, 0, "Invalid StartType value."},
	{1233, 0, "Invalid ErrorControl value."},
	{1234, 0, "Service install section missing required directives."},
	{1240, 0, "AddService directive missing required parameters."},
	{1241, 0, "Invalid AddService flags."},
	{1242, 0, "Reserved service name used."},
	{1243, 0, "Disabled service cannot use SPSVCINST_ASSOCSERVICE."},
	{1244, 0, "Invalid service binary path."},
	{1250, 0, "Invalid registry value type."},
	{1251, 0, "Invalid binary data format (expecting hex values 00-FF)."},
	{1252, 0, "Invalid registry root key."},
	{1253, 0, "Registry value data exceeds maximum length."},
	{1254, 0, "Invalid DWORD value."},
	{1258, 0, "File not listed under [SourceDisksFiles] section."},
	{1260, 0, "SourceDisksFiles disk ID not listed under [SourceDisksNames]."},
	{1264, 0, "Invalid catalog file name, expecting 'filename.cat'."},
	{1268, 0, "Invalid driver version format, expecting w.x.y.z (0-65536 per segment)."},
	{1270, 0, "Invalid DriverVer date format, expecting MM/DD/YYYY."},
	{1280, HDCFlagAllSubmissions, "Device class name/GUID mismatch."},
	{1281, HDCFlagAllSubmissions, "Device class GUID mismatch."},
	{1282, HDCFlagAllSubmissions, "Missing Class directive in [Version] section."},
	{1283, HDCFlagAllSubmissions, "Missing ClassGuid directive in [Version] section."},
	{1284, HDCFlagAllSubmissions, "Reserved device class name."},
	{1285, HDCFlagAllSubmissions, "ClassInstall32 section found; configurable driver packages must not use class installers."},
	{1286, HDCFlagAllSubmissions, "Class name and ClassGuid mismatch for known device class."},
	{1290, 0, "CopyFiles section not listed in [DestinationDirs]."},
	{1300, 0, "Invalid Models section reference."},
	{1301, HDCFlagAllSubmissions, "Missing models section."},
	{1302, HDCFlagAllSubmissions, "Provider cannot be \"Microsoft\"."},
	{1303, 0, "Invalid target OS version decoration."},
	{1310, HDCFlagAllSubmissions, "Service reference error (non-suppressible)."},
	{1311, HDCFlagAllSubmissions, "Service configuration reference error (non-suppressible)."},
	{1312, HDCFlagAllSubmissions, "Service reference validation error (non-suppressible)."},
	{1313, HDCFlagAllSubmissions, "Service install error (non-suppressible)."},
	{1314, HDCFlagAllSubmissions, "Service binary reference error (non-suppressible)."},
	{1315, HDCFlagAllSubmissions, "Configuration reference error (non-suppressible)."},
	{1316, HDCFlagAllSubmissions, "Section reference validation error (non-suppressible)."},
	{1317, HDCFlagAllSubmissions, "Section reference error (non-suppressible)."},
	{1318, HDCFlagAllSubmissions, "Missing section reference (non-suppressible)."},
	{1319, HDCFlagAllSubmissions, "Section cross-reference error (non-suppressible)."},
	{1330, 0, "Install section not found for model."},
	{1340, HDCFlagAllSubmissions, "Co-installer found; configurable driver packages must not use co-installers."},
	{2083, 0, "API surface output comment line error."},
	{2100, HDCFlagDownlevelDeclarative, "Registry operation not isolated to HKR."},
	{2120, HDCFlagDownlevelDeclarative, "Destination file path not isolated to DIRID 13."},
	{2150, HDCFlagDownlevelDeclarative, "PnpLockdown=1 not specified in [Version] section."},
}

// FindError searches the error database for a given error code.
func FindError(code int) *ErrorEntry {
	lo, hi := 0, len(errorDatabase)-1
	for lo <= hi {
		mid := (lo + hi) / 2
		if errorDatabase[mid].Code == code {
			return &errorDatabase[mid]
		}
		if errorDatabase[mid].Code < code {
			lo = mid + 1
		} else {
			hi = mid - 1
		}
	}
	return nil
}

// GetAllErrors returns the full error database.
func GetAllErrors() []ErrorEntry {
	return errorDatabase
}
