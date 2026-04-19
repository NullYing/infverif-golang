package verifier

import (
	"bufio"
	"fmt"
	"infverif/pkg/infparser"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// ErrorLevel represents the severity of a validation issue.
type ErrorLevel int

const (
	LevelError   ErrorLevel = 1
	LevelWarning ErrorLevel = 2
	LevelInfo    ErrorLevel = 3
)

func (l ErrorLevel) String() string {
	switch l {
	case LevelError:
		return "ERROR"
	case LevelWarning:
		return "WARNING"
	case LevelInfo:
		return "INFO"
	default:
		return "UNKNOWN"
	}
}

// ValidationMode represents the verification mode.
type ValidationMode int

const (
	ModeDefault      ValidationMode = 0
	ModeConfigurable ValidationMode = 1  // /c  - flag 0x01
	ModeUniversal    ValidationMode = 2  // /u  - flag 0x03
	ModeWindows      ValidationMode = 3  // /w  - flag 0x07
	ModeSubmission   ValidationMode = 4  // /k  - flag 0x43
	ModeMSFT         ValidationMode = 5  // /msft - flag 0x23
	ModeInfo         ValidationMode = 10 // /info
	ModeDepends      ValidationMode = 11 // /depends
	ModeAPI          ValidationMode = 12 // /api
)

// ModeFlags returns the internal bit flags for a validation mode,
// matching the original binary's INFVERIF_PARAMS offset 24.
func ModeFlags(m ValidationMode) uint32 {
	switch m {
	case ModeConfigurable:
		return 0x01
	case ModeUniversal:
		return 0x03
	case ModeWindows:
		return 0x07
	case ModeSubmission:
		return 0x43
	case ModeMSFT:
		return 0x23
	default:
		return 0x00
	}
}

// HasConfigurabilityCheck returns true if the mode includes configurability check (Bit 0).
func HasConfigurabilityCheck(m ValidationMode) bool {
	return ModeFlags(m)&0x01 != 0
}

// HasIncludeNeedsCheck returns true if the mode includes Include/Needs check (Bit 1).
func HasIncludeNeedsCheck(m ValidationMode) bool {
	return ModeFlags(m)&0x02 != 0
}

// HasStateSeparationCheck returns true if the mode includes state separation check (Bit 2).
func HasStateSeparationCheck(m ValidationMode) bool {
	return ModeFlags(m)&0x04 != 0
}

// Issue represents a single validation issue found.
type Issue struct {
	Level   ErrorLevel
	Code    int
	Line    int
	File    string
	Message string
}

// InfInfo holds summary information about an INF file.
type InfInfo struct {
	Hash         string
	FamilyID     string
	DriverType   string
	Devices      []DeviceInfo
	Architecture string
}

// DeviceInfo holds per-device information.
type DeviceInfo struct {
	Name        string
	HardwareID  string
	Service     string
	SectionName string
}

// Options holds verifier options.
type Options struct {
	Mode          ValidationMode
	Verbose       bool
	StampInf      bool
	OsVer         string
	WBuild        string
	LogPath       string
	MSBuild       bool
	WError        bool
	ErrorListFile string // path to error list CSV
	ErrorLevel    int    // 0 = no filter, 1 = errors only, 2 = errors+warnings, 3 = all
	CSVFile       string // path for CSV output
	Append        bool   // append to CSV instead of overwriting
	LevelSort     bool   // sort output by error level
	Inbox         bool   // inbox driver validation
	FileRoot      string // file root for env var resolution
	ExcludeFile   string // exclusion list file
	Recurse       bool   // recursive directory search
	ProductFile   string // product definition .ias file
	MSFT          bool   // Microsoft internal mode
	Debug         bool   // wait for debugger
}

// NonSuppressibleError returns true if an error code is in the 1310-1319 range
// and cannot be suppressed by /errorlist.
func NonSuppressibleError(code int) bool {
	return code >= 1310 && code <= 1319
}

// Result holds the verification result for a single INF file.
type Result struct {
	Path   string
	Valid  bool
	Issues []Issue
	Info   *InfInfo
	Err    error
}

// Verify performs validation on an INF file and returns the result.
func Verify(path string, opts Options) Result {
	result := Result{Path: path}

	inf, err := infparser.Parse(path)
	if err != nil {
		result.Err = err
		result.Issues = append(result.Issues, Issue{
			Level:   LevelError,
			Code:    1627,
			File:    path,
			Message: fmt.Sprintf("Failed to parse INF: %v", err),
		})
		return result
	}

	// Resolve strings helper
	resolve := func(s string) string {
		return inf.ResolveString(s)
	}

	// Run basic validation checks
	checkVersion(inf, path, &result)
	checkManufacturer(inf, path, &result, resolve)
	checkInstallSections(inf, path, &result, resolve)
	checkSourceDisks(inf, path, &result)
	checkDestinationDirs(inf, path, &result)
	checkStrings(inf, path, &result)

	// Mode-specific checks based on bit flags
	if HasConfigurabilityCheck(opts.Mode) {
		checkConfigurability(inf, path, &result)
	}
	if HasIncludeNeedsCheck(opts.Mode) {
		checkUniversalRequirements(inf, path, &result)
	}
	if HasStateSeparationCheck(opts.Mode) {
		checkWindowsDriverRequirements(inf, path, &result)
	}

	// Build info
	result.Info = buildInfInfo(inf, path, resolve)

	// Apply /werror: promote warnings to errors
	if opts.WError {
		for i := range result.Issues {
			if result.Issues[i].Level == LevelWarning {
				result.Issues[i].Level = LevelError
			}
		}
	}

	// Apply /errorlist suppression
	if opts.ErrorListFile != "" {
		allowed := LoadErrorList(opts.ErrorListFile)
		if len(allowed) > 0 {
			var filtered []Issue
			for _, issue := range result.Issues {
				if NonSuppressibleError(issue.Code) || !allowed[issue.Code] {
					filtered = append(filtered, issue)
				}
			}
			result.Issues = filtered
		}
	}

	// Apply /errorlevel filter
	if opts.ErrorLevel > 0 {
		var filtered []Issue
		for _, issue := range result.Issues {
			if int(issue.Level) <= opts.ErrorLevel {
				filtered = append(filtered, issue)
			}
		}
		result.Issues = filtered
	}

	// Sort by level if requested
	if opts.LevelSort {
		sortIssuesByLevel(result.Issues)
	}

	// Determine validity
	result.Valid = true
	for _, issue := range result.Issues {
		if issue.Level == LevelError {
			result.Valid = false
			break
		}
	}

	return result
}

// GetInfo returns only information without full validation.
func GetInfo(path string) Result {
	result := Result{Path: path}

	inf, err := infparser.Parse(path)
	if err != nil {
		result.Err = err
		return result
	}

	resolve := func(s string) string {
		return inf.ResolveString(s)
	}

	result.Info = buildInfInfo(inf, path, resolve)
	result.Valid = true
	return result
}

func checkVersion(inf *infparser.INFFile, path string, result *Result) {
	ver := inf.GetSection("version")
	if ver == nil {
		result.Issues = append(result.Issues, Issue{
			Level:   LevelError,
			Code:    1200,
			File:    path,
			Message: "Invalid INF, must contain [Version] section and have signature \"$Windows NT$\".",
		})
		return
	}

	// Check signature
	sig := inf.GetValue("version", "Signature")
	if !strings.EqualFold(sig, "$Windows NT$") {
		result.Issues = append(result.Issues, Issue{
			Level:   LevelError,
			Code:    1200,
			Line:    ver.Line,
			File:    path,
			Message: "Invalid or missing INF signature, expecting \"$Windows NT$\".",
		})
	}

	// Check Class
	class := inf.GetValue("version", "Class")
	classEntry := inf.GetEntry("version", "Class")
	if class == "" {
		result.Issues = append(result.Issues, Issue{
			Level:   LevelError,
			Code:    1282,
			File:    path,
			Message: "Missing Class directive in [Version] section.",
		})
	} else {
		if isReservedClass(class) {
			line := 0
			if len(classEntry) > 0 {
				line = classEntry[0].Line
			}
			result.Issues = append(result.Issues, Issue{
				Level:   LevelError,
				Code:    1284,
				Line:    line,
				File:    path,
				Message: fmt.Sprintf("Class \"%s\" is reserved for use by Microsoft.", class),
			})
		}
	}

	// Check ClassGuid
	classGuid := inf.GetValue("version", "ClassGuid")
	classGuidEntry := inf.GetEntry("version", "ClassGuid")
	if classGuid == "" {
		result.Issues = append(result.Issues, Issue{
			Level:   LevelError,
			Code:    1283,
			File:    path,
			Message: "Missing ClassGuid directive in [Version] section.",
		})
	} else {
		if !isValidGUID(classGuid) {
			line := 0
			if len(classGuidEntry) > 0 {
				line = classGuidEntry[0].Line
			}
			result.Issues = append(result.Issues, Issue{
				Level:   LevelError,
				Code:    1220,
				Line:    line,
				File:    path,
				Message: fmt.Sprintf("Invalid ClassGuid \"%s\", expecting {XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX}.", classGuid),
			})
		}
	}

	// Check Class/ClassGuid match
	if class != "" && classGuid != "" && isValidGUID(classGuid) {
		checkClassGuidMatch(class, classGuid, path, result, classEntry, classGuidEntry)
	}

	// Check Provider
	provider := inf.GetValue("version", "Provider")
	provider = inf.ResolveString(provider)
	if strings.EqualFold(provider, "Microsoft") {
		providerEntry := inf.GetEntry("version", "Provider")
		line := 0
		if len(providerEntry) > 0 {
			line = providerEntry[0].Line
		}
		result.Issues = append(result.Issues, Issue{
			Level:   LevelError,
			Code:    1302,
			Line:    line,
			File:    path,
			Message: "Provider cannot be \"Microsoft\", must be organization who authored INF.",
		})
	}

	// Check DriverVer
	driverVer := inf.GetValue("version", "DriverVer")
	if driverVer != "" {
		checkDriverVer(driverVer, inf, path, result)
	}

	// Check CatalogFile
	catFile := inf.GetValue("version", "CatalogFile")
	if catFile != "" {
		if !strings.HasSuffix(strings.ToLower(catFile), ".cat") {
			catEntry := inf.GetEntry("version", "CatalogFile")
			line := 0
			if len(catEntry) > 0 {
				line = catEntry[0].Line
			}
			result.Issues = append(result.Issues, Issue{
				Level:   LevelError,
				Code:    1264,
				Line:    line,
				File:    path,
				Message: fmt.Sprintf("Invalid catalog file '%s', expecting 'filename.cat'.", catFile),
			})
		}
	}
}

func checkDriverVer(driverVer string, inf *infparser.INFFile, path string, result *Result) {
	entries := inf.GetEntry("version", "DriverVer")
	line := 0
	if len(entries) > 0 {
		line = entries[0].Line
	}

	parts := strings.SplitN(driverVer, ",", 2)
	if len(parts) >= 1 {
		datePart := strings.TrimSpace(parts[0])
		if datePart != "" && !isValidDate(datePart) {
			result.Issues = append(result.Issues, Issue{
				Level:   LevelError,
				Code:    1270,
				Line:    line,
				File:    path,
				Message: fmt.Sprintf("Invalid driver date value %s in DriverVer, expecting MM/DD/YYYY.", datePart),
			})
		}
	}
	if len(parts) >= 2 {
		versionPart := strings.TrimSpace(parts[1])
		if versionPart != "" && !isValidVersion(versionPart) {
			result.Issues = append(result.Issues, Issue{
				Level:   LevelError,
				Code:    1268,
				Line:    line,
				File:    path,
				Message: fmt.Sprintf("Invalid driver version (%s), expecting w.x.y.z, where each segment is between 0-65536.", versionPart),
			})
		}
	}
}

func checkManufacturer(inf *infparser.INFFile, path string, result *Result, resolve func(string) string) {
	mfr := inf.GetSection("manufacturer")
	if mfr == nil {
		// Not always required (e.g. DefaultInstall INFs)
		return
	}

	for _, entry := range mfr.Entries {
		if len(entry.Values) == 0 {
			continue
		}
		// entry.Values[0] is the models section name, entry.Values[1:] are decorations
		modelsSectionBase := entry.Values[0]
		if len(entry.Values) > 1 {
			// Check decorated sections
			for _, decoration := range entry.Values[1:] {
				fullName := modelsSectionBase + "." + decoration
				if inf.GetSection(fullName) == nil {
					result.Issues = append(result.Issues, Issue{
						Level:   LevelError,
						Code:    1301,
						Line:    entry.Line,
						File:    path,
						Message: fmt.Sprintf("Missing models section. Section = [%s], Line = %d", fullName, entry.Line),
					})
				} else {
					checkModelsSection(inf, fullName, decoration, path, result, resolve)
				}
			}
		} else {
			// Undecorated models section
			if inf.GetSection(modelsSectionBase) == nil {
				result.Issues = append(result.Issues, Issue{
					Level:   LevelError,
					Code:    1301,
					Line:    entry.Line,
					File:    path,
					Message: fmt.Sprintf("Missing models section. Section = [%s], Line = %d", modelsSectionBase, entry.Line),
				})
			} else {
				checkModelsSection(inf, modelsSectionBase, "", path, result, resolve)
			}
		}
	}
}

func checkModelsSection(inf *infparser.INFFile, sectionName, decoration string, path string, result *Result, resolve func(string) string) {
	sec := inf.GetSection(sectionName)
	if sec == nil {
		return
	}

	for _, entry := range sec.Entries {
		if len(entry.Values) < 2 {
			continue
		}
		installSection := entry.Values[0]
		hwID := entry.Values[1]

		// Check install section exists
		actualSection := installSection
		if decoration != "" {
			// Try decorated first
			decorated := installSection + "." + getArchFromDecoration(decoration)
			if inf.GetSection(decorated) != nil {
				actualSection = decorated
			}
		}

		if inf.GetSection(actualSection) == nil && inf.GetSection(installSection) == nil {
			result.Issues = append(result.Issues, Issue{
				Level:   LevelWarning,
				Code:    1330,
				Line:    entry.Line,
				File:    path,
				Message: fmt.Sprintf("Section [%s] not found.", installSection),
			})
		}

		// Check services section
		svcSection := actualSection + ".Services"
		if inf.GetSection(svcSection) == nil {
			svcSection = installSection + ".Services"
		}
		if inf.GetSection(svcSection) == nil {
			// Hardware without associated service
			_ = hwID // used for potential future validation
		}
	}
}

func checkInstallSections(inf *infparser.INFFile, path string, result *Result, resolve func(string) string) {
	// Find all DDInstall.Services sections and validate service installs
	for _, secName := range inf.Order {
		if !strings.HasSuffix(secName, ".services") {
			continue
		}

		sec := inf.Sections[secName]
		for _, entry := range sec.Entries {
			if !strings.EqualFold(entry.Key, "AddService") {
				continue
			}
			if len(entry.Values) < 3 {
				result.Issues = append(result.Issues, Issue{
					Level:   LevelWarning,
					Code:    1240,
					Line:    entry.Line,
					File:    path,
					Message: fmt.Sprintf("Skipping directive 'AddService' without a service name in section [%s].", sec.Name),
				})
				continue
			}

			serviceName := entry.Values[0]
			serviceInstallSection := ""
			if len(entry.Values) >= 3 {
				serviceInstallSection = entry.Values[2]
			}

			if serviceName == "" {
				continue
			}

			// Check reserved service names
			if isReservedServiceName(serviceName) {
				result.Issues = append(result.Issues, Issue{
					Level:   LevelError,
					Code:    1242,
					Line:    entry.Line,
					File:    path,
					Message: fmt.Sprintf("Service name %s is reserved for internal use only.", serviceName),
				})
			}

			// Check service install section
			if serviceInstallSection != "" {
				checkServiceInstallSection(inf, serviceInstallSection, serviceName, path, result)
			}
		}
	}
}

func checkServiceInstallSection(inf *infparser.INFFile, sectionName, serviceName, path string, result *Result) {
	sec := inf.GetSection(sectionName)
	if sec == nil {
		return
	}

	svcType := inf.GetValue(sectionName, "ServiceType")
	startType := inf.GetValue(sectionName, "StartType")
	svcBinary := inf.GetValue(sectionName, "ServiceBinary")

	// Check service type
	if svcType == "" {
		result.Issues = append(result.Issues, Issue{
			Level:   LevelError,
			Code:    1230,
			Line:    sec.Line,
			File:    path,
			Message: fmt.Sprintf("Missing service type in section [%s].", sectionName),
		})
	} else {
		svcTypeInt, err := parseIntValue(svcType)
		if err == nil {
			// SERVICE_WIN32 (0x10|0x20) and SERVICE_DRIVER (0x1|0x2) cannot be combined
			isWin32 := (svcTypeInt & 0x30) != 0
			isDriver := (svcTypeInt & 0x03) != 0
			if isWin32 && isDriver {
				result.Issues = append(result.Issues, Issue{
					Level:   LevelError,
					Code:    1231,
					Line:    sec.Line,
					File:    path,
					Message: "Invalid service type, cannot use both SERVICE_WIN32 and SERVICE_DRIVER.",
				})
			}
		}
	}

	// Check disabled service with ASSOCSERVICE
	if startType != "" {
		startTypeInt, err := parseIntValue(startType)
		if err == nil && startTypeInt == 4 {
			// This service is disabled - check if ASSOCSERVICE flag is used
			// The flag 0x2 is SPSVCINST_ASSOCSERVICE
			// We check in the AddService directive, not here
		}
	}

	// Check ServiceBinary
	if svcBinary == "" && svcType != "" {
		svcTypeInt, err := parseIntValue(svcType)
		if err == nil && (svcTypeInt&0x03) != 0 {
			// Driver service needs a binary
			result.Issues = append(result.Issues, Issue{
				Level:   LevelWarning,
				Code:    1244,
				Line:    sec.Line,
				File:    path,
				Message: fmt.Sprintf("Invalid service image path for service '%s'.", serviceName),
			})
		}
	}
}

func checkSourceDisks(inf *infparser.INFFile, path string, result *Result) {
	sdf := inf.GetSection("sourcedisksfiles")
	sdn := inf.GetSection("sourcedisksnames")

	// Check all CopyFiles references have source disk entries
	for _, secName := range inf.Order {
		sec := inf.Sections[secName]
		for _, entry := range sec.Entries {
			if !strings.EqualFold(entry.Key, "CopyFiles") {
				continue
			}
			for _, val := range entry.Values {
				if strings.HasPrefix(val, "@") {
					// Direct file copy
					fileName := val[1:]
					checkFileInSourceDisks(fileName, sdf, entry.Line, sec.Name, path, result)
				} else {
					// Section reference
					copySection := inf.GetSection(val)
					if copySection != nil {
						for _, fileEntry := range copySection.Entries {
							fileName := fileEntry.Key
							if fileName != "" {
								checkFileInSourceDisks(fileName, sdf, fileEntry.Line, val, path, result)
							}
						}
					}
				}
			}
		}
	}

	// Check SourceDisksFiles references valid SourceDisksNames
	if sdf != nil && sdn != nil {
		validDisks := make(map[string]bool)
		for _, entry := range sdn.Entries {
			validDisks[entry.Key] = true
		}

		for _, entry := range sdf.Entries {
			if len(entry.Values) > 0 {
				diskID := entry.Values[0]
				if diskID != "" && !validDisks[diskID] {
					result.Issues = append(result.Issues, Issue{
						Level:   LevelError,
						Code:    1260,
						Line:    entry.Line,
						File:    path,
						Message: fmt.Sprintf("Source file \"%s\" uses disk id %s, which is not listed under [SourceDisksNames].", entry.Key, diskID),
					})
				}
			}
		}
	}
}

func checkFileInSourceDisks(fileName string, sdf *infparser.Section, line int, sectionName, path string, result *Result) {
	if sdf == nil {
		result.Issues = append(result.Issues, Issue{
			Level:   LevelError,
			Code:    1258,
			Line:    line,
			File:    path,
			Message: fmt.Sprintf("Missing file '%s' under [SourceDisksFiles] section.", fileName),
		})
		return
	}

	fileNameLower := strings.ToLower(fileName)
	found := false
	for _, entry := range sdf.Entries {
		if strings.ToLower(entry.Key) == fileNameLower {
			found = true
			break
		}
	}
	if !found {
		result.Issues = append(result.Issues, Issue{
			Level:   LevelError,
			Code:    1258,
			Line:    line,
			File:    path,
			Message: fmt.Sprintf("Missing file '%s' under [SourceDisksFiles] section.", fileName),
		})
	}
}

func checkDestinationDirs(inf *infparser.INFFile, path string, result *Result) {
	destDirs := inf.GetSection("destinationdirs")

	// Find all CopyFiles section references and check they're in DestinationDirs
	for _, secName := range inf.Order {
		sec := inf.Sections[secName]
		for _, entry := range sec.Entries {
			if !strings.EqualFold(entry.Key, "CopyFiles") {
				continue
			}
			for _, val := range entry.Values {
				if strings.HasPrefix(val, "@") {
					continue // Direct file copy uses DefaultDestDir
				}
				if destDirs == nil {
					result.Issues = append(result.Issues, Issue{
						Level:   LevelWarning,
						Code:    1290,
						Line:    entry.Line,
						File:    path,
						Message: fmt.Sprintf("CopyFiles '%s' is not listed in [DestinationDirs].", val),
					})
					continue
				}
				valLower := strings.ToLower(val)
				found := false
				for _, dEntry := range destDirs.Entries {
					if strings.ToLower(dEntry.Key) == valLower {
						found = true
						break
					}
				}
				// Also check DefaultDestDir
				if !found {
					for _, dEntry := range destDirs.Entries {
						if strings.EqualFold(dEntry.Key, "DefaultDestDir") {
							found = true
							break
						}
					}
				}
				if !found {
					result.Issues = append(result.Issues, Issue{
						Level:   LevelWarning,
						Code:    1290,
						Line:    entry.Line,
						File:    path,
						Message: fmt.Sprintf("CopyFiles '%s' is not listed in [DestinationDirs].", val),
					})
				}
			}
		}
	}
}

func checkStrings(inf *infparser.INFFile, path string, result *Result) {
	stringsSection := inf.GetSection("strings")

	// Scan all lines for %token% references
	tokenRegex := regexp.MustCompile(`%([^%]+)%`)
	for _, line := range inf.Lines {
		stripped := strings.TrimSpace(line)
		if stripped == "" || strings.HasPrefix(stripped, ";") || strings.HasPrefix(stripped, "[") {
			continue
		}

		matches := tokenRegex.FindAllStringSubmatch(stripped, -1)
		for _, match := range matches {
			token := match[1]
			// Skip DIRID-like tokens (numeric)
			if _, err := strconv.Atoi(token); err == nil {
				continue
			}
			// Skip system tokens
			if strings.EqualFold(token, "SystemRoot") || strings.EqualFold(token, "DriverData") {
				continue
			}
			if stringsSection != nil {
				tokenLower := strings.ToLower(token)
				found := false
				for _, entry := range stringsSection.Entries {
					if strings.ToLower(entry.Key) == tokenLower {
						found = true
						break
					}
				}
				if !found {
					// This is only a warning; the string might be defined elsewhere
				}
			}
		}
	}
}

func checkUniversalRequirements(inf *infparser.INFFile, path string, result *Result) {
	// Check for registry operations not isolated to HKR
	for _, secName := range inf.Order {
		sec := inf.Sections[secName]
		for _, entry := range sec.Entries {
			if strings.EqualFold(entry.Key, "AddReg") || strings.EqualFold(entry.Key, "DelReg") {
				for _, regSection := range entry.Values {
					checkRegIsolation(inf, regSection, path, result)
				}
			}
		}
	}
}

func checkRegIsolation(inf *infparser.INFFile, sectionName, path string, result *Result) {
	sec := inf.GetSection(sectionName)
	if sec == nil {
		return
	}

	for _, entry := range sec.Entries {
		if len(entry.Values) == 0 {
			continue
		}
		root := entry.Values[0]
		if root != "" && !strings.EqualFold(root, "HKR") {
			subkey := ""
			if len(entry.Values) > 1 {
				subkey = entry.Values[1]
			}
			result.Issues = append(result.Issues, Issue{
				Level:   LevelWarning,
				Code:    2100,
				Line:    entry.Line,
				File:    path,
				Message: fmt.Sprintf("Registry root '%s\\%s' is not isolated to HKR.", root, subkey),
			})
		}
	}
}

func checkWindowsDriverRequirements(inf *infparser.INFFile, path string, result *Result) {
	// Check PnpLockdown
	pnpLockdown := inf.GetValue("version", "PnpLockdown")
	if pnpLockdown != "1" {
		result.Issues = append(result.Issues, Issue{
			Level:   LevelWarning,
			Code:    2150,
			File:    path,
			Message: "[Version] section should specify PnpLockdown=1 to prevent external apps from modifying installed driver files.",
		})
	}

	// Check file destination isolation (DIRID 13)
	destDirs := inf.GetSection("destinationdirs")
	if destDirs != nil {
		for _, entry := range destDirs.Entries {
			if strings.EqualFold(entry.Key, "DefaultDestDir") {
				continue
			}
			if len(entry.Values) > 0 {
				dirid, err := parseIntValue(entry.Values[0])
				if err == nil && dirid != 13 && dirid != 10 {
					result.Issues = append(result.Issues, Issue{
						Level:   LevelWarning,
						Code:    2120,
						Line:    entry.Line,
						File:    path,
						Message: fmt.Sprintf("Destination file path for '%s' is not isolated to DIRID 13.", entry.Key),
					})
				}
			}
		}
	}
}

func buildInfInfo(inf *infparser.INFFile, path string, resolve func(string) string) *InfInfo {
	info := &InfInfo{}

	// Hash - compute a simple hash from the file content
	info.Hash = computeInfHash(inf)

	// Family ID
	provider := resolve(inf.GetValue("version", "Provider"))
	baseName := filepath.Base(path)
	info.FamilyID = provider + "-" + baseName

	// Driver Type
	info.DriverType = determineDriverType(inf)

	// Architecture
	info.Architecture = detectArchitecture(inf)

	// Devices
	mfr := inf.GetSection("manufacturer")
	if mfr != nil {
		for _, entry := range mfr.Entries {
			if len(entry.Values) == 0 {
				continue
			}
			modelsSectionBase := entry.Values[0]
			decorations := entry.Values[1:]

			if len(decorations) > 0 {
				for _, dec := range decorations {
					fullName := modelsSectionBase + "." + dec
					extractDevices(inf, fullName, &info.Devices, resolve)
				}
			} else {
				extractDevices(inf, modelsSectionBase, &info.Devices, resolve)
			}
		}
	}

	return info
}

func extractDevices(inf *infparser.INFFile, modelsSection string, devices *[]DeviceInfo, resolve func(string) string) {
	sec := inf.GetSection(modelsSection)
	if sec == nil {
		return
	}

	for _, entry := range sec.Entries {
		if len(entry.Values) < 2 {
			continue
		}
		dev := DeviceInfo{
			Name:        resolve(entry.Key),
			SectionName: entry.Values[0],
			HardwareID:  entry.Values[1],
		}

		// Find service from .Services section
		svcSectionName := entry.Values[0] + ".Services"
		svcSec := inf.GetSection(svcSectionName)
		if svcSec != nil {
			for _, svcEntry := range svcSec.Entries {
				if strings.EqualFold(svcEntry.Key, "AddService") && len(svcEntry.Values) > 0 {
					dev.Service = svcEntry.Values[0]
					break
				}
			}
		}

		*devices = append(*devices, dev)
	}
}

func determineDriverType(inf *infparser.INFFile) string {
	// Check for DefaultInstall (primitive driver)
	if inf.GetSection("defaultinstall") != nil {
		return "Primitive"
	}
	// Check for Manufacturer (device driver)
	if inf.GetSection("manufacturer") != nil {
		return "Device"
	}
	return "Legacy"
}

func detectArchitecture(inf *infparser.INFFile) string {
	mfr := inf.GetSection("manufacturer")
	if mfr == nil {
		return ""
	}

	archSet := make(map[string]bool)
	for _, entry := range mfr.Entries {
		for i := 1; i < len(entry.Values); i++ {
			dec := strings.ToLower(entry.Values[i])
			arch := getArchFromDecoration(dec)
			if arch != "" {
				archSet[arch] = true
			}
		}
	}

	var archs []string
	for a := range archSet {
		archs = append(archs, a)
	}
	if len(archs) == 0 {
		return "all"
	}
	return strings.Join(archs, ", ")
}

func getArchFromDecoration(decoration string) string {
	dec := strings.ToLower(decoration)
	// Remove version suffix
	parts := strings.SplitN(dec, ".", 2)
	base := parts[0]

	switch {
	case strings.Contains(base, "ntamd64") || strings.Contains(base, "amd64"):
		return "amd64"
	case strings.Contains(base, "ntx86") || strings.Contains(base, "x86"):
		return "x86"
	case strings.Contains(base, "ntarm64") || strings.Contains(base, "arm64"):
		return "arm64"
	case strings.Contains(base, "ntarm") || strings.Contains(base, "arm"):
		return "arm"
	case strings.Contains(base, "ntia64") || strings.Contains(base, "ia64"):
		return "ia64"
	case base == "nt":
		return "all"
	}
	return ""
}

func computeInfHash(inf *infparser.INFFile) string {
	// Simple FNV-like hash matching infverif behavior
	var hash uint64 = 0xcbf29ce484222325
	for _, line := range inf.Lines {
		for _, c := range line {
			hash ^= uint64(c)
			hash *= 0x100000001b3
		}
	}
	return fmt.Sprintf("%016x", hash)
}

// Helper functions

var guidRegex = regexp.MustCompile(`^\{[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}\}$`)

func isValidGUID(s string) bool {
	return guidRegex.MatchString(s)
}

func isValidDate(s string) bool {
	parts := strings.Split(s, "/")
	if len(parts) != 3 {
		return false
	}
	month, err1 := strconv.Atoi(parts[0])
	day, err2 := strconv.Atoi(parts[1])
	year, err3 := strconv.Atoi(parts[2])
	if err1 != nil || err2 != nil || err3 != nil {
		return false
	}
	if month < 1 || month > 12 || day < 1 || day > 31 || year < 1601 || year > 9999 {
		return false
	}
	return true
}

func isValidVersion(s string) bool {
	parts := strings.Split(s, ".")
	if len(parts) != 4 {
		return false
	}
	for _, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 || n > 65536 {
			return false
		}
	}
	return true
}

func parseIntValue(s string) (int, error) {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		val, err := strconv.ParseInt(s[2:], 16, 64)
		return int(val), err
	}
	val, err := strconv.Atoi(s)
	return val, err
}

func isReservedClass(class string) bool {
	reserved := []string{
		"sample", "legacydriver", "adapter", "apmsupport",
	}
	classLower := strings.ToLower(class)
	for _, r := range reserved {
		if classLower == r {
			return true
		}
	}
	return false
}

func isReservedServiceName(name string) bool {
	reserved := []string{
		"null", "beep", "vga", "rdisk",
	}
	nameLower := strings.ToLower(name)
	for _, r := range reserved {
		if nameLower == r {
			return true
		}
	}
	return false
}

func checkClassGuidMatch(class, classGuid, path string, result *Result, classEntries, classGuidEntries []infparser.Entry) {
	knownClasses := map[string]string{
		"system":            "{4D36E97D-E325-11CE-BFC1-08002BE10318}",
		"net":               "{4D36E972-E325-11CE-BFC1-08002BE10318}",
		"display":           "{4D36E968-E325-11CE-BFC1-08002BE10318}",
		"media":             "{4D36E96C-E325-11CE-BFC1-08002BE10318}",
		"hdc":               "{4D36E96A-E325-11CE-BFC1-08002BE10318}",
		"hidclass":          "{745A17A0-74D3-11D0-B6FE-00A0C90F57DA}",
		"keyboard":          "{4D36E96B-E325-11CE-BFC1-08002BE10318}",
		"mouse":             "{4D36E96F-E325-11CE-BFC1-08002BE10318}",
		"usb":               "{36FC9E60-C465-11CF-8056-444553540000}",
		"ports":             "{4D36E978-E325-11CE-BFC1-08002BE10318}",
		"printer":           "{4D36E979-E325-11CE-BFC1-08002BE10318}",
		"firmware":          "{F2E7DD72-6468-4E36-B6F1-6488F42C1B52}",
		"softwaredevice":    "{62F9C741-B25A-46CE-B54C-9BCCCE08B6F2}",
		"softwarecomponent": "{5C4C3332-344D-483C-8739-259E934C9CC8}",
		"extension":         "{E2F84CE7-8EFA-411C-AA69-97454CA4CB57}",
	}

	classLower := strings.ToLower(class)
	if expected, ok := knownClasses[classLower]; ok {
		if !strings.EqualFold(classGuid, expected) {
			line := 0
			if len(classGuidEntries) > 0 {
				line = classGuidEntries[0].Line
			}
			result.Issues = append(result.Issues, Issue{
				Level:   LevelError,
				Code:    1286,
				Line:    line,
				File:    path,
				Message: fmt.Sprintf("Class name and ClassGuid mismatch, expecting ClassGuid \"%s\" for Class \"%s\".", expected, class),
			})
		}
	}
}

// checkConfigurability checks if the INF meets configurability requirements (/c mode).
func checkConfigurability(inf *infparser.INFFile, path string, result *Result) {
	// Check for co-installer usage (not allowed in configurable drivers)
	for _, secName := range inf.Order {
		sec := inf.Sections[secName]
		if strings.HasSuffix(strings.ToLower(secName), ".coinstallers") {
			for _, entry := range sec.Entries {
				if strings.EqualFold(entry.Key, "AddReg") || strings.EqualFold(entry.Key, "CopyFiles") {
					result.Issues = append(result.Issues, Issue{
						Level:   LevelError,
						Code:    1340,
						Line:    entry.Line,
						File:    path,
						Message: fmt.Sprintf("Co-installer found in section [%s]. Configurable driver packages must not use co-installers.", sec.Name),
					})
				}
			}
		}

		// Check for ClassInstall32 sections (class installers not allowed)
		if strings.HasPrefix(strings.ToLower(secName), "classinstall32") {
			result.Issues = append(result.Issues, Issue{
				Level:   LevelError,
				Code:    1285,
				Line:    sec.Line,
				File:    path,
				Message: fmt.Sprintf("ClassInstall32 section [%s] found. Configurable driver packages must not use class installers.", sec.Name),
			})
		}
	}
}

// GetDepends returns the Include/Needs dependency information for an INF file.
func GetDepends(path string) (*DependencyInfo, error) {
	inf, err := infparser.Parse(path)
	if err != nil {
		return nil, err
	}

	depInfo := &DependencyInfo{
		Sections: make([]SectionDependency, 0),
	}

	for _, secName := range inf.Order {
		sec := inf.Sections[secName]
		var includes []string
		var needs []string

		for _, entry := range sec.Entries {
			if strings.EqualFold(entry.Key, "Include") {
				for _, v := range entry.Values {
					v = strings.TrimSpace(v)
					if v != "" {
						includes = append(includes, v)
					}
				}
			}
			if strings.EqualFold(entry.Key, "Needs") {
				for _, v := range entry.Values {
					v = strings.TrimSpace(v)
					if v != "" {
						needs = append(needs, v)
					}
				}
			}
		}

		if len(includes) > 0 || len(needs) > 0 {
			depInfo.Sections = append(depInfo.Sections, SectionDependency{
				Name:     sec.Name,
				Includes: includes,
				Needs:    needs,
			})
		}
	}

	return depInfo, nil
}

// DependencyInfo holds Include/Needs dependency data for an INF.
type DependencyInfo struct {
	Sections []SectionDependency
}

// SectionDependency holds dependencies for one section.
type SectionDependency struct {
	Name     string
	Includes []string
	Needs    []string
}

// LoadErrorList reads a CSV error list file and returns a map of error codes to suppress.
func LoadErrorList(path string) map[int]bool {
	result := make(map[int]bool)
	f, err := os.Open(path)
	if err != nil {
		return result
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		// Parse comma-separated error codes
		for _, field := range strings.Split(line, ",") {
			field = strings.TrimSpace(field)
			if code, err := strconv.Atoi(field); err == nil {
				result[code] = true
			}
		}
	}
	return result
}

// sortIssuesByLevel sorts issues by error level (ERROR first, then WARNING, then INFO).
func sortIssuesByLevel(issues []Issue) {
	sort.SliceStable(issues, func(i, j int) bool {
		return issues[i].Level < issues[j].Level
	})
}

// FormatCSVRow formats an issue as a CSV row.
func FormatCSVRow(file string, issue Issue) string {
	return fmt.Sprintf("%s,%s,%d,%d,%s",
		file, issue.Level, issue.Code, issue.Line,
		strings.ReplaceAll(issue.Message, ",", ";"))
}

// CSVHeader returns the CSV header line.
func CSVHeader() string {
	return "Filename,Level,Code,Line,Message"
}
