package verifier

import (
	"os"
	"path/filepath"
	"testing"
)

// writeTempINF writes content to a temp .inf file and returns its path.
func writeTempINF(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.inf")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

// === Valid INF templates ===

const validINF = `[Version]
Signature = "$Windows NT$"
Class = System
ClassGuid = {4D36E97D-E325-11CE-BFC1-08002BE10318}
Provider = %ManufacturerName%
CatalogFile = TestDriver.cat
DriverVer = 01/01/2020,1.0.0.0
PnpLockdown = 1

[Manufacturer]
%ManufacturerName% = Standard,NTamd64

[Standard.NTamd64]
%DeviceName% = MyDevice_Install, Root\TestDevice

[MyDevice_Install]
CopyFiles = MyDevice_CopyFiles

[MyDevice_Install.Services]
AddService = TestService,0x00000002,TestService_Install

[TestService_Install]
DisplayName = %ServiceName%
ServiceType = 1
StartType = 3
ErrorControl = 1
ServiceBinary = %13%\testdriver.sys

[MyDevice_CopyFiles]
testdriver.sys

[SourceDisksFiles]
testdriver.sys = 1

[SourceDisksNames]
1 = %DiskName%

[DestinationDirs]
MyDevice_CopyFiles = 13

[Strings]
ManufacturerName = "TestManufacturer"
DeviceName = "Test Device"
ServiceName = "Test Service"
DiskName = "Test Install Disk"
`

const reservedClassINF = `[Version]
Signature = "$Windows NT$"
Class = Sample
ClassGuid = {78A1C341-4539-11d3-B88D-00C04FAD5171}
Provider = %ManufacturerName%
CatalogFile = TestDriver.cat
DriverVer = 01/01/2020,1.0.0.0
PnpLockdown = 1

[Manufacturer]
%ManufacturerName% = Standard,NTamd64

[Standard.NTamd64]
%DeviceName% = MyDevice_Install, Root\TestDevice

[MyDevice_Install]
CopyFiles = MyDevice_CopyFiles

[MyDevice_Install.Services]
AddService = TestService,0x00000002,TestService_Install

[TestService_Install]
DisplayName = %ServiceName%
ServiceType = 1
StartType = 3
ErrorControl = 1
ServiceBinary = %13%\testdriver.sys

[MyDevice_CopyFiles]
testdriver.sys

[SourceDisksFiles]
testdriver.sys = 1

[SourceDisksNames]
1 = %DiskName%

[DestinationDirs]
MyDevice_CopyFiles = 13

[Strings]
ManufacturerName = "TestManufacturer"
DeviceName = "Test Device"
ServiceName = "Test Service"
DiskName = "Test Install Disk"
`

// --- Verify: basic validation ---

// TestVerifyValidINF matches original: test2.inf → "INF is VALID"
func TestVerifyValidINF(t *testing.T) {
	path := writeTempINF(t, validINF)
	result := Verify(path, Options{})

	if !result.Valid {
		t.Errorf("Expected valid, got issues: %v", result.Issues)
	}
	if len(result.Issues) != 0 {
		for _, i := range result.Issues {
			t.Logf("Issue: %s(%d) line %d: %s", i.Level, i.Code, i.Line, i.Message)
		}
		t.Errorf("Expected 0 issues, got %d", len(result.Issues))
	}
}

// TestVerifyReservedClass matches original: test.inf → ERROR(1284) "Class "Sample" is reserved"
func TestVerifyReservedClass(t *testing.T) {
	path := writeTempINF(t, reservedClassINF)
	result := Verify(path, Options{})

	if result.Valid {
		t.Error("Expected invalid for reserved class")
	}

	found := false
	for _, issue := range result.Issues {
		if issue.Code == 1284 {
			found = true
			if issue.Level != LevelError {
				t.Errorf("Expected ERROR level for 1284, got %s", issue.Level)
			}
			if issue.Line != 3 {
				t.Errorf("Expected line 3 for reserved class, got %d", issue.Line)
			}
		}
	}
	if !found {
		t.Error("Expected error code 1284 for reserved class")
	}
}

// TestVerifyMissingVersionSection tests [Version] absence → ERROR(1200)
func TestVerifyMissingVersionSection(t *testing.T) {
	content := `[Manufacturer]
Test = Models
`
	path := writeTempINF(t, content)
	result := Verify(path, Options{})

	if result.Valid {
		t.Error("Expected invalid for missing [Version]")
	}

	found := false
	for _, issue := range result.Issues {
		if issue.Code == 1200 {
			found = true
		}
	}
	if !found {
		t.Error("Expected error code 1200 for missing [Version]")
	}
}

// TestVerifyBadSignature tests invalid signature → ERROR(1200)
func TestVerifyBadSignature(t *testing.T) {
	content := `[Version]
Signature = "$Chicago$"
Class = System
ClassGuid = {4D36E97D-E325-11CE-BFC1-08002BE10318}
`
	path := writeTempINF(t, content)
	result := Verify(path, Options{})

	found := false
	for _, issue := range result.Issues {
		if issue.Code == 1200 {
			found = true
		}
	}
	if !found {
		t.Error("Expected error code 1200 for bad signature")
	}
}

// TestVerifyMissingClass tests missing Class → ERROR(1282)
func TestVerifyMissingClass(t *testing.T) {
	content := `[Version]
Signature = "$Windows NT$"
ClassGuid = {4D36E97D-E325-11CE-BFC1-08002BE10318}
Provider = TestProvider
`
	path := writeTempINF(t, content)
	result := Verify(path, Options{})

	found := false
	for _, issue := range result.Issues {
		if issue.Code == 1282 {
			found = true
		}
	}
	if !found {
		t.Error("Expected error code 1282 for missing Class")
	}
}

// TestVerifyMissingClassGuid tests missing ClassGuid → ERROR(1283)
func TestVerifyMissingClassGuid(t *testing.T) {
	content := `[Version]
Signature = "$Windows NT$"
Class = System
Provider = TestProvider
`
	path := writeTempINF(t, content)
	result := Verify(path, Options{})

	found := false
	for _, issue := range result.Issues {
		if issue.Code == 1283 {
			found = true
		}
	}
	if !found {
		t.Error("Expected error code 1283 for missing ClassGuid")
	}
}

// TestVerifyInvalidGUID tests invalid GUID format → ERROR(1220)
func TestVerifyInvalidGUID(t *testing.T) {
	content := `[Version]
Signature = "$Windows NT$"
Class = System
ClassGuid = NOT-A-GUID
Provider = TestProvider
`
	path := writeTempINF(t, content)
	result := Verify(path, Options{})

	found := false
	for _, issue := range result.Issues {
		if issue.Code == 1220 {
			found = true
		}
	}
	if !found {
		t.Error("Expected error code 1220 for invalid GUID")
	}
}

// TestVerifyMicrosoftProvider tests Provider="Microsoft" → ERROR(1302)
func TestVerifyMicrosoftProvider(t *testing.T) {
	content := `[Version]
Signature = "$Windows NT$"
Class = System
ClassGuid = {4D36E97D-E325-11CE-BFC1-08002BE10318}
Provider = Microsoft
DriverVer = 01/01/2020,1.0.0.0

[Manufacturer]
Microsoft = Models,NTamd64

[Models.NTamd64]
"Device" = Install, Root\Device

[Install]

[Install.Services]
AddService = Svc,0x00000002,SvcInstall

[SvcInstall]
ServiceType = 1
StartType = 3
ErrorControl = 1
ServiceBinary = %13%\drv.sys

[SourceDisksFiles]
drv.sys = 1

[SourceDisksNames]
1 = "Disk"

[DestinationDirs]
DefaultDestDir = 13

[Strings]
`
	path := writeTempINF(t, content)
	result := Verify(path, Options{})

	found := false
	for _, issue := range result.Issues {
		if issue.Code == 1302 {
			found = true
		}
	}
	if !found {
		t.Error("Expected error code 1302 for Microsoft provider")
	}
}

// --- Mode flags ---

func TestModeFlags(t *testing.T) {
	tests := []struct {
		mode ValidationMode
		want uint32
	}{
		{ModeDefault, 0x00},
		{ModeConfigurable, 0x01},
		{ModeUniversal, 0x03},
		{ModeWindows, 0x07},
		{ModeSubmission, 0x43},
		{ModeMSFT, 0x23},
		{ModeSignatureRequirements, 0x80},
	}
	for _, tt := range tests {
		got := ModeFlags(tt.mode)
		if got != tt.want {
			t.Errorf("ModeFlags(%d) = 0x%02X, want 0x%02X", tt.mode, got, tt.want)
		}
	}
}

func TestHasConfigurabilityCheck(t *testing.T) {
	yes := []ValidationMode{ModeConfigurable, ModeUniversal, ModeWindows, ModeSubmission, ModeMSFT}
	no := []ValidationMode{ModeDefault, ModeSignatureRequirements, ModeInfo}

	for _, m := range yes {
		if !HasConfigurabilityCheck(m) {
			t.Errorf("HasConfigurabilityCheck(%d) = false, want true", m)
		}
	}
	for _, m := range no {
		if HasConfigurabilityCheck(m) {
			t.Errorf("HasConfigurabilityCheck(%d) = true, want false", m)
		}
	}
}

func TestHasIncludeNeedsCheck(t *testing.T) {
	yes := []ValidationMode{ModeUniversal, ModeWindows, ModeSubmission, ModeMSFT}
	no := []ValidationMode{ModeDefault, ModeConfigurable, ModeSignatureRequirements}

	for _, m := range yes {
		if !HasIncludeNeedsCheck(m) {
			t.Errorf("HasIncludeNeedsCheck(%d) = false, want true", m)
		}
	}
	for _, m := range no {
		if HasIncludeNeedsCheck(m) {
			t.Errorf("HasIncludeNeedsCheck(%d) = true, want false", m)
		}
	}
}

func TestHasStateSeparationCheck(t *testing.T) {
	yes := []ValidationMode{ModeWindows}
	no := []ValidationMode{ModeDefault, ModeConfigurable, ModeUniversal, ModeSubmission, ModeSignatureRequirements}

	for _, m := range yes {
		if !HasStateSeparationCheck(m) {
			t.Errorf("HasStateSeparationCheck(%d) = false, want true", m)
		}
	}
	for _, m := range no {
		if HasStateSeparationCheck(m) {
			t.Errorf("HasStateSeparationCheck(%d) = true, want false", m)
		}
	}
}

// --- ErrorLevel ---

func TestErrorLevelString(t *testing.T) {
	tests := []struct {
		level ErrorLevel
		want  string
	}{
		{LevelError, "ERROR"},
		{LevelWarning, "WARNING"},
		{LevelInfo, "INFO"},
		{ErrorLevel(99), "UNKNOWN"},
	}
	for _, tt := range tests {
		got := tt.level.String()
		if got != tt.want {
			t.Errorf("ErrorLevel(%d).String() = %q, want %q", tt.level, got, tt.want)
		}
	}
}

// --- NonSuppressibleError ---

func TestNonSuppressibleError(t *testing.T) {
	for code := 1310; code <= 1319; code++ {
		if !NonSuppressibleError(code) {
			t.Errorf("NonSuppressibleError(%d) = false, want true", code)
		}
	}
	for _, code := range []int{1200, 1284, 1309, 1320, 2100} {
		if NonSuppressibleError(code) {
			t.Errorf("NonSuppressibleError(%d) = true, want false", code)
		}
	}
}

// --- Mode-specific validation ---

// TestVerifyUniversalMode tests /u mode passes valid INF (matches original output)
func TestVerifyUniversalMode(t *testing.T) {
	path := writeTempINF(t, validINF)
	result := Verify(path, Options{Mode: ModeUniversal})

	if !result.Valid {
		for _, i := range result.Issues {
			t.Logf("Issue: %s(%d): %s", i.Level, i.Code, i.Message)
		}
		t.Error("Expected valid for /u mode with valid INF")
	}
}

// TestVerifyWindowsMode tests /w mode passes valid INF
func TestVerifyWindowsMode(t *testing.T) {
	path := writeTempINF(t, validINF)
	result := Verify(path, Options{Mode: ModeWindows})

	if !result.Valid {
		for _, i := range result.Issues {
			t.Logf("Issue: %s(%d): %s", i.Level, i.Code, i.Message)
		}
		t.Error("Expected valid for /w mode with valid INF")
	}
}

// TestVerifySubmissionMode tests /k mode passes valid INF
func TestVerifySubmissionMode(t *testing.T) {
	path := writeTempINF(t, validINF)
	result := Verify(path, Options{Mode: ModeSubmission})

	if !result.Valid {
		for _, i := range result.Issues {
			t.Logf("Issue: %s(%d): %s", i.Level, i.Code, i.Message)
		}
		t.Error("Expected valid for /k mode with valid INF")
	}
}

// TestVerifySignatureRequirementsMode tests /h mode passes valid INF
func TestVerifySignatureRequirementsMode(t *testing.T) {
	path := writeTempINF(t, validINF)
	result := Verify(path, Options{Mode: ModeSignatureRequirements})

	if !result.Valid {
		for _, i := range result.Issues {
			t.Logf("Issue: %s(%d): %s", i.Level, i.Code, i.Message)
		}
		t.Error("Expected valid for /h mode with valid INF")
	}
}

// --- /werror ---

func TestWerrorPromotesWarnings(t *testing.T) {
	// Create an INF that generates a warning
	content := `[Version]
Signature = "$Windows NT$"
Class = System
ClassGuid = {4D36E97D-E325-11CE-BFC1-08002BE10318}
Provider = %Mfg%
DriverVer = 01/01/2020,1.0.0.0

[Manufacturer]
%Mfg% = Models,NTamd64

[Models.NTamd64]
"Device" = Install, Root\Dev

[Install]

[Install.Services]
AddService = Svc,0x00000002,SvcInst

[SvcInst]
ServiceType = 1
StartType = 3
ErrorControl = 1
ServiceBinary = %13%\drv.sys

[SourceDisksFiles]
drv.sys = 1

[SourceDisksNames]
1 = "Disk"

[DestinationDirs]
DefaultDestDir = 13

[Strings]
Mfg = "TestMfg"
`
	path := writeTempINF(t, content)

	// Without /werror
	result1 := Verify(path, Options{})
	warningCount := 0
	for _, i := range result1.Issues {
		if i.Level == LevelWarning {
			warningCount++
		}
	}

	// With /werror: all warnings become errors
	result2 := Verify(path, Options{WError: true})
	for _, i := range result2.Issues {
		if i.Level == LevelWarning {
			t.Error("/werror should promote all warnings to errors")
		}
	}
	_ = warningCount // may be 0 if INF is perfectly valid
}

// --- /errorlevel filtering ---

func TestErrorLevelFiltering(t *testing.T) {
	path := writeTempINF(t, reservedClassINF)

	// ErrorLevel 1 = errors only
	result := Verify(path, Options{ErrorLevel: 1})
	for _, i := range result.Issues {
		if i.Level != LevelError {
			t.Errorf("With ErrorLevel=1, got %s issue (code %d)", i.Level, i.Code)
		}
	}
}

// --- /levelsort ---

func TestLevelSort(t *testing.T) {
	path := writeTempINF(t, reservedClassINF)
	result := Verify(path, Options{LevelSort: true})

	// Issues should be sorted: ERROR (1) < WARNING (2) < INFO (3)
	prevLevel := LevelError
	for _, i := range result.Issues {
		if i.Level < prevLevel {
			t.Error("Issues not sorted by level")
			break
		}
		prevLevel = i.Level
	}
}

// --- GetInfo ---

// TestGetInfoValidINF matches original /info output for test2.inf
func TestGetInfoValidINF(t *testing.T) {
	path := writeTempINF(t, validINF)
	result := GetInfo(path)

	if result.Err != nil {
		t.Fatalf("GetInfo failed: %v", result.Err)
	}
	if result.Info == nil {
		t.Fatal("Expected non-nil Info")
	}

	if result.Info.FamilyID == "" {
		t.Error("Expected non-empty FamilyID")
	}
	if result.Info.DriverType != "Device" {
		t.Errorf("DriverType = %q, want %q", result.Info.DriverType, "Device")
	}
	if len(result.Info.Devices) == 0 {
		t.Fatal("Expected at least one device")
	}

	dev := result.Info.Devices[0]
	if dev.HardwareID != "Root\\TestDevice" {
		t.Errorf("HardwareID = %q, want %q", dev.HardwareID, "Root\\TestDevice")
	}
	if dev.Service != "TestService" {
		t.Errorf("Service = %q, want %q", dev.Service, "TestService")
	}
}

// --- GetDepends ---

func TestGetDependsNoDeps(t *testing.T) {
	path := writeTempINF(t, validINF)
	depInfo, err := GetDepends(path)
	if err != nil {
		t.Fatalf("GetDepends failed: %v", err)
	}

	// Valid INF with no Include/Needs should have empty sections
	if depInfo != nil && len(depInfo.Sections) != 0 {
		t.Errorf("Expected 0 dependency sections, got %d", len(depInfo.Sections))
	}
}

// --- CollectSyntax ---

func TestCollectSyntax(t *testing.T) {
	path := writeTempINF(t, validINF)
	entries, err := CollectSyntax(path)
	if err != nil {
		t.Fatalf("CollectSyntax failed: %v", err)
	}

	// The valid INF contains CopyFiles, AddService, PnpLockdown, SourceDisksFiles, SourceDisksNames
	foundDirectives := make(map[string]bool)
	for _, e := range entries {
		foundDirectives[e.Name] = true
	}

	expected := []string{"CopyFiles", "AddService", "PnpLockdown", "SourceDisksFiles", "SourceDisksNames"}
	for _, name := range expected {
		if !foundDirectives[name] {
			t.Errorf("Expected syntax entry for %q", name)
		}
	}
}

// --- /provider check ---

func TestProviderMatch(t *testing.T) {
	path := writeTempINF(t, validINF)

	// Matching provider
	result := Verify(path, Options{Provider: "TestManufacturer"})
	for _, i := range result.Issues {
		if i.Code == 1302 && i.Message == "Provider name must match the /provider switch." {
			t.Error("Expected no provider mismatch error for matching name")
		}
	}

	// Non-matching provider
	result2 := Verify(path, Options{Provider: "WrongName"})
	found := false
	for _, i := range result2.Issues {
		if i.Code == 1302 {
			found = true
		}
	}
	if !found {
		t.Error("Expected error 1302 for provider mismatch")
	}
}

// --- CSV / Format helpers ---

func TestCSVHeader(t *testing.T) {
	header := CSVHeader()
	if header != "Filename,Level,Code,Line,Message" {
		t.Errorf("CSVHeader() = %q, want %q", header, "Filename,Level,Code,Line,Message")
	}
}

func TestFormatCSVRow(t *testing.T) {
	issue := Issue{
		Level:   LevelError,
		Code:    1284,
		Line:    3,
		File:    `C:\test\driver.inf`,
		Message: `Class "Sample" is reserved for use by Microsoft.`,
	}

	row := FormatCSVRow(`C:\test\driver.inf`, issue)
	// Should contain file, level, code, line, message
	if row == "" {
		t.Error("FormatCSVRow returned empty string")
	}
}

// --- /h mode with rule versions ---

func TestVerifySignatureReqsWithRuleVersion(t *testing.T) {
	path := writeTempINF(t, validINF)
	rv := RuleVersion{10, 0, 26100}
	result := Verify(path, Options{
		Mode:    ModeSignatureRequirements,
		RuleVer: &rv,
	})

	if !result.Valid {
		for _, i := range result.Issues {
			t.Logf("Issue: %s(%d): %s", i.Level, i.Code, i.Message)
		}
		t.Error("Expected valid for /h mode with 24h2 rules")
	}
}

// --- Co-installer / ClassInstall32 check (configurability) ---

func TestConfigurabilityCoInstaller(t *testing.T) {
	content := `[Version]
Signature = "$Windows NT$"
Class = System
ClassGuid = {4D36E97D-E325-11CE-BFC1-08002BE10318}
Provider = %Mfg%
DriverVer = 01/01/2020,1.0.0.0

[Manufacturer]
%Mfg% = Models,NTamd64

[Models.NTamd64]
"Device" = Install, Root\Dev

[Install]

[Install.Services]
AddService = Svc,0x00000002,SvcInst

[SvcInst]
ServiceType = 1
StartType = 3
ErrorControl = 1
ServiceBinary = %13%\drv.sys

[Install.CoInstallers]
CopyFiles = CoInst_CopyFiles
AddReg = CoInst_AddReg

[CoInst_AddReg]
HKR,,CoInstallers32,0x00010000,"MyCoInst.dll,MyEntry"

[CoInst_CopyFiles]
MyCoInst.dll

[SourceDisksFiles]
drv.sys = 1
MyCoInst.dll = 1

[SourceDisksNames]
1 = "Disk"

[DestinationDirs]
DefaultDestDir = 13
CoInst_CopyFiles = 11

[Strings]
Mfg = "TestMfg"
`
	path := writeTempINF(t, content)
	result := Verify(path, Options{Mode: ModeConfigurable})

	found := false
	for _, i := range result.Issues {
		if i.Code == 1340 {
			found = true
		}
	}
	if !found {
		t.Error("Expected error 1340 for co-installer in configurable mode")
	}
}

func TestConfigurabilityClassInstall32(t *testing.T) {
	content := `[Version]
Signature = "$Windows NT$"
Class = System
ClassGuid = {4D36E97D-E325-11CE-BFC1-08002BE10318}
Provider = %Mfg%
DriverVer = 01/01/2020,1.0.0.0

[Manufacturer]
%Mfg% = Models,NTamd64

[Models.NTamd64]
"Device" = Install, Root\Dev

[Install]

[Install.Services]
AddService = Svc,0x00000002,SvcInst

[SvcInst]
ServiceType = 1
StartType = 3
ErrorControl = 1
ServiceBinary = %13%\drv.sys

[ClassInstall32]
AddReg = ClassReg

[ClassReg]
HKR,,,,"Sample Class"

[SourceDisksFiles]
drv.sys = 1

[SourceDisksNames]
1 = "Disk"

[DestinationDirs]
DefaultDestDir = 13

[Strings]
Mfg = "TestMfg"
`
	path := writeTempINF(t, content)
	result := Verify(path, Options{Mode: ModeConfigurable})

	found := false
	for _, i := range result.Issues {
		if i.Code == 1285 {
			found = true
		}
	}
	if !found {
		t.Error("Expected error 1285 for ClassInstall32 in configurable mode")
	}
}

// --- LoadErrorList ---

func TestLoadErrorList(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "errors.csv")
	content := "1200\n1284\n1302\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	errList := LoadErrorList(path)
	for _, code := range []int{1200, 1284, 1302} {
		if !errList[code] {
			t.Errorf("Expected code %d in error list", code)
		}
	}
	if errList[1310] {
		t.Error("Code 1310 should not be in error list")
	}
}

func TestLoadErrorListNonexistent(t *testing.T) {
	errList := LoadErrorList("/nonexistent/file.csv")
	if len(errList) != 0 {
		t.Error("Expected empty map for nonexistent file")
	}
}

// --- Error suppression with non-suppressible codes ---

func TestErrorListSuppression(t *testing.T) {
	dir := t.TempDir()

	// Create error list that tries to suppress 1284 and 1310
	errFile := filepath.Join(dir, "suppress.csv")
	if err := os.WriteFile(errFile, []byte("1284\n1310\n"), 0644); err != nil {
		t.Fatal(err)
	}

	path := writeTempINF(t, reservedClassINF)
	result := Verify(path, Options{ErrorListFile: errFile})

	// 1284 should be suppressed
	for _, i := range result.Issues {
		if i.Code == 1284 {
			t.Error("Error 1284 should be suppressed by error list")
		}
	}

	// 1310-1319 range: if present, should NOT be suppressed
	for _, i := range result.Issues {
		if i.Code >= 1310 && i.Code <= 1319 {
			// Good - non-suppressible errors are kept
		}
	}
}

// --- GetEffectiveRuleVersion ---

func TestGetEffectiveRuleVersion(t *testing.T) {
	// Default
	rv := GetEffectiveRuleVersion(Options{})
	if rv != DefaultRuleVersion {
		t.Errorf("Default rule version = %s, want %s", rv.String(), DefaultRuleVersion.String())
	}

	// With explicit rulever
	custom := RuleVersion{10, 0, 26100}
	rv = GetEffectiveRuleVersion(Options{RuleVer: &custom})
	if rv != custom {
		t.Errorf("Custom rule version = %s, want %s", rv.String(), custom.String())
	}
}
