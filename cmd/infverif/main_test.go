package main

import (
	"encoding/binary"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf16"
)

// buildBinary builds the infverif binary and returns the path.
func buildBinary(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	binPath := filepath.Join(dir, "infverif.exe")
	cmd := exec.Command("go", "build", "-o", binPath, "./")
	cmd.Dir = filepath.Join(getProjectRoot(t), "cmd", "infverif")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build: %s\n%s", err, out)
	}
	return binPath
}

func getProjectRoot(t *testing.T) string {
	t.Helper()
	// Walk up from this file to find go.mod
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("Could not find project root")
		}
		dir = parent
	}
}

// writeUTF16LEFile writes a UTF-16 LE BOM file.
func writeUTF16LEFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	u16 := utf16.Encode([]rune(content))
	data := make([]byte, 2+len(u16)*2)
	data[0] = 0xFF
	data[1] = 0xFE
	for i, v := range u16 {
		binary.LittleEndian.PutUint16(data[2+i*2:], v)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

// writeUTF8File writes a plain UTF-8 file.
func writeUTF8File(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

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

func runBinary(t *testing.T, bin string, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	out, err := cmd.CombinedOutput()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("Failed to run: %v", err)
		}
	}
	return string(out), exitCode
}

// === Integration tests: match original infverif.exe behavior ===

// TestCLI_ValidINF: valid INF → exit 0, output contains "INF is VALID"
func TestCLI_ValidINF(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	inf := writeUTF8File(t, dir, "valid.inf", validINF)

	out, code := runBinary(t, bin, inf)
	if code != 0 {
		t.Errorf("Exit code = %d, want 0\nOutput: %s", code, out)
	}
	if !strings.Contains(out, "INF is VALID") {
		t.Errorf("Output should contain 'INF is VALID':\n%s", out)
	}
}

// TestCLI_ReservedClass: reserved class → exit 1, ERROR(1284)
func TestCLI_ReservedClass(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	inf := writeUTF8File(t, dir, "reserved.inf", reservedClassINF)

	out, code := runBinary(t, bin, inf)
	if code != 1 {
		t.Errorf("Exit code = %d, want 1\nOutput: %s", code, out)
	}
	if !strings.Contains(out, "ERROR(1284)") {
		t.Errorf("Output should contain 'ERROR(1284)':\n%s", out)
	}
	if !strings.Contains(out, "INF is NOT VALID") {
		t.Errorf("Output should contain 'INF is NOT VALID':\n%s", out)
	}
	if !strings.Contains(out, `Class "Sample" is reserved`) {
		t.Errorf("Output should mention reserved class 'Sample':\n%s", out)
	}
}

// TestCLI_UTF16LE: UTF-16 LE INF also validates correctly
func TestCLI_UTF16LE(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	inf := writeUTF16LEFile(t, dir, "utf16.inf", validINF)

	out, code := runBinary(t, bin, inf)
	if code != 0 {
		t.Errorf("Exit code = %d, want 0\nOutput: %s", code, out)
	}
	if !strings.Contains(out, "INF is VALID") {
		t.Errorf("Output should contain 'INF is VALID':\n%s", out)
	}
}

// TestCLI_NoArgs: no arguments → exit 87 (bad parameters)
func TestCLI_NoArgs(t *testing.T) {
	bin := buildBinary(t)
	_, code := runBinary(t, bin)
	if code != 87 {
		t.Errorf("Exit code = %d, want 87 for no args", code)
	}
}

// TestCLI_Help: /? → exit 0, output contains usage
func TestCLI_Help(t *testing.T) {
	bin := buildBinary(t)
	out, code := runBinary(t, bin, "/?")
	if code != 0 {
		t.Errorf("Exit code = %d, want 0 for help", code)
	}
	if !strings.Contains(out, "USAGE:") {
		t.Errorf("Help output should contain 'USAGE:':\n%s", out)
	}
	if !strings.Contains(out, "InfVerif (Go)") {
		t.Errorf("Help output should contain app name:\n%s", out)
	}
}

// TestCLI_VerboseUniversal: /v /u → includes "Running in Verbose" and check messages
func TestCLI_VerboseUniversal(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	inf := writeUTF8File(t, dir, "test.inf", validINF)

	out, _ := runBinary(t, bin, "/v", "/u", inf)

	expected := []string{
		"Running in Verbose",
		"Running include/needs check",
		"Running configurability check",
		"INF is VALID",
	}
	for _, s := range expected {
		if !strings.Contains(out, s) {
			t.Errorf("Output missing %q:\n%s", s, out)
		}
	}
}

// TestCLI_VerboseWindows: /v /w → includes state separation check
func TestCLI_VerboseWindows(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	inf := writeUTF8File(t, dir, "test.inf", validINF)

	out, _ := runBinary(t, bin, "/v", "/w", inf)

	expected := []string{
		"Running in Verbose",
		"Running state separation check",
		"Running include/needs check",
		"Running configurability check",
	}
	for _, s := range expected {
		if !strings.Contains(out, s) {
			t.Errorf("Output missing %q:\n%s", s, out)
		}
	}
}

// TestCLI_VerboseH: /v /h → includes "Running signature requirements check"
func TestCLI_VerboseH(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	inf := writeUTF8File(t, dir, "test.inf", validINF)

	out, _ := runBinary(t, bin, "/v", "/h", inf)

	if !strings.Contains(out, "Running signature requirements check") {
		t.Errorf("Output missing signature requirements message:\n%s", out)
	}
	if !strings.Contains(out, "Using rules from OS build:") {
		t.Errorf("Output missing rule version display:\n%s", out)
	}
}

// TestCLI_Info: /info → displays INF information
func TestCLI_Info(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	inf := writeUTF8File(t, dir, "test.inf", validINF)

	out, code := runBinary(t, bin, "/info", inf)
	if code != 0 {
		t.Errorf("Exit code = %d, want 0", code)
	}

	expected := []string{
		"Information",
		"INF Hash:",
		"Family ID:",
		"Driver Type:",
		"Device:",
		"Hardware ID:",
		"Root\\TestDevice",
		"Service:",
		"TestService",
		"Architecture:",
	}
	for _, s := range expected {
		if !strings.Contains(out, s) {
			t.Errorf("Output missing %q:\n%s", s, out)
		}
	}
}

// TestCLI_Depends: /depends → shows dependency info
func TestCLI_Depends(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	inf := writeUTF8File(t, dir, "test.inf", validINF)

	out, code := runBinary(t, bin, "/depends", inf)
	if code != 0 {
		t.Errorf("Exit code = %d, want 0", code)
	}
	if !strings.Contains(out, "dependencies") {
		t.Errorf("Output should contain 'dependencies':\n%s", out)
	}
}

// TestCLI_Syntax: /syntax → shows syntax report
func TestCLI_Syntax(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	inf := writeUTF8File(t, dir, "test.inf", validINF)

	out, code := runBinary(t, bin, "/syntax", inf)
	if code != 0 {
		t.Errorf("Exit code = %d, want 0", code)
	}
	if !strings.Contains(out, "INF Syntax Report") {
		t.Errorf("Output should contain 'INF Syntax Report':\n%s", out)
	}
	// Should report known directives
	if !strings.Contains(out, "CopyFiles") {
		t.Errorf("Syntax report should list CopyFiles:\n%s", out)
	}
	if !strings.Contains(out, "AddService") {
		t.Errorf("Syntax report should list AddService:\n%s", out)
	}
}

// TestCLI_CodeHelp: /code 1203 → displays error description
func TestCLI_CodeHelp(t *testing.T) {
	bin := buildBinary(t)
	out, code := runBinary(t, bin, "/code", "1203")
	if code != 0 {
		t.Errorf("Exit code = %d, want 0", code)
	}
	if !strings.Contains(out, "1203") {
		t.Errorf("Output should contain error code:\n%s", out)
	}
	if !strings.Contains(out, "Section name") {
		t.Errorf("Output should contain error description:\n%s", out)
	}
}

// TestCLI_HDCRules: /hdcrules → displays HDC rules
func TestCLI_HDCRules(t *testing.T) {
	bin := buildBinary(t)
	out, code := runBinary(t, bin, "/hdcrules")
	if code != 0 {
		t.Errorf("Exit code = %d, want 0", code)
	}
	if !strings.Contains(out, "HDC Error Code Rules") {
		t.Errorf("Output should contain 'HDC Error Code Rules':\n%s", out)
	}
	if !strings.Contains(out, "All Submissions") {
		t.Errorf("Output should contain 'All Submissions':\n%s", out)
	}
}

// TestCLI_ShowExceptions: /showexceptions → displays exceptions list
func TestCLI_ShowExceptions(t *testing.T) {
	bin := buildBinary(t)
	out, code := runBinary(t, bin, "/showexceptions")
	if code != 0 {
		t.Errorf("Exit code = %d, want 0", code)
	}
	if !strings.Contains(out, "Release,Source,Type,Path") {
		t.Errorf("Output should contain CSV header:\n%s", out)
	}
	if !strings.Contains(out, "Registry") {
		t.Errorf("Output should contain Registry exceptions:\n%s", out)
	}
}

// TestCLI_MSBuildFormat: /msbuild → MSBuild format errors
func TestCLI_MSBuildFormat(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	inf := writeUTF8File(t, dir, "reserved.inf", reservedClassINF)

	out, _ := runBinary(t, bin, "/msbuild", inf)
	// MSBuild format: file(line): error code: message
	if !strings.Contains(out, "error 1284:") {
		t.Errorf("MSBuild output should contain 'error 1284:':\n%s", out)
	}
}

// TestCLI_CSVOutput: /csv → writes CSV file
func TestCLI_CSVOutput(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	inf := writeUTF8File(t, dir, "reserved.inf", reservedClassINF)
	csvPath := filepath.Join(dir, "output.csv")

	_, _ = runBinary(t, bin, "/csv", csvPath, inf)

	data, err := os.ReadFile(csvPath)
	if err != nil {
		t.Fatalf("Failed to read CSV: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "Filename,Level,Code,Line,Message") {
		t.Errorf("CSV should contain header:\n%s", content)
	}
	if !strings.Contains(content, "1284") {
		t.Errorf("CSV should contain error 1284:\n%s", content)
	}
}

// TestCLI_NoExceptionsRequiresH: /noexceptions without /h → exit 87
func TestCLI_NoExceptionsRequiresH(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	inf := writeUTF8File(t, dir, "test.inf", validINF)

	out, code := runBinary(t, bin, "/noexceptions", inf)
	if code != 87 {
		t.Errorf("Exit code = %d, want 87 for /noexceptions without /h\nOutput: %s", code, out)
	}
}

// TestCLI_Provider: /provider with matching name → valid
func TestCLI_ProviderMatch(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	inf := writeUTF8File(t, dir, "test.inf", validINF)

	out, code := runBinary(t, bin, "/provider", "TestManufacturer", inf)
	if code != 0 {
		t.Errorf("Exit code = %d, want 0 for matching provider\nOutput: %s", code, out)
	}
}

// TestCLI_ProviderMismatch: /provider with wrong name → error 1302
func TestCLI_ProviderMismatch(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	inf := writeUTF8File(t, dir, "test.inf", validINF)

	out, code := runBinary(t, bin, "/provider", "WrongName", inf)
	if code != 1 {
		t.Errorf("Exit code = %d, want 1 for provider mismatch\nOutput: %s", code, out)
	}
	if !strings.Contains(out, "1302") {
		t.Errorf("Output should contain error 1302:\n%s", out)
	}
}

// TestCLI_RuleVer: /h /rulever vnext → displays vnext version
func TestCLI_RuleVer(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	inf := writeUTF8File(t, dir, "test.inf", validINF)

	out, _ := runBinary(t, bin, "/v", "/h", "/rulever", "vnext", inf)
	if !strings.Contains(out, "10.0.99999") {
		t.Errorf("Output should contain vnext version 10.0.99999:\n%s", out)
	}
}

// TestCLI_VerboseParams: /verboseparams → displays flags
func TestCLI_VerboseParams(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	inf := writeUTF8File(t, dir, "test.inf", validINF)

	out, _ := runBinary(t, bin, "/v", "/h", "/verboseparams", inf)
	if !strings.Contains(out, "InfVerif Flags: 0x") {
		t.Errorf("Output should contain 'InfVerif Flags:':\n%s", out)
	}
}

// TestCLI_MultipleFiles: validates multiple INFs in one run
func TestCLI_MultipleFiles(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	inf1 := writeUTF8File(t, dir, "valid.inf", validINF)
	inf2 := writeUTF8File(t, dir, "reserved.inf", reservedClassINF)

	out, code := runBinary(t, bin, inf1, inf2)
	// At least one invalid → exit 1
	if code != 1 {
		t.Errorf("Exit code = %d, want 1 (one invalid INF)\nOutput: %s", code, out)
	}
	if !strings.Contains(out, "Checked 2 INF(s)") {
		t.Errorf("Output should mention checking 2 INFs:\n%s", out)
	}
}

// TestCLI_CheckedCount: output includes "Checked N INF(s)"
func TestCLI_CheckedCount(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	inf := writeUTF8File(t, dir, "test.inf", validINF)

	out, _ := runBinary(t, bin, inf)
	if !strings.Contains(out, "Checked 1 INF(s)") {
		t.Errorf("Output should contain 'Checked 1 INF(s)':\n%s", out)
	}
}

// TestCLI_Werror: /werror with reserved class (already ERROR) → still exit 1
func TestCLI_Werror(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	inf := writeUTF8File(t, dir, "test.inf", reservedClassINF)

	_, code := runBinary(t, bin, "/werror", inf)
	if code != 1 {
		t.Errorf("Exit code = %d, want 1 for /werror with error INF", code)
	}
}

// TestCLI_InvalidRuleVer: /rulever with invalid value → exit 87
func TestCLI_InvalidRuleVer(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	inf := writeUTF8File(t, dir, "test.inf", validINF)

	_, code := runBinary(t, bin, "/h", "/rulever", "invalid", inf)
	if code != 87 {
		t.Errorf("Exit code = %d, want 87 for invalid /rulever", code)
	}
}
