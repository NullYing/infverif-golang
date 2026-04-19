package infparser

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
	"unicode/utf16"
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

// writeTempUTF16LEINF writes UTF-16 LE BOM content to a temp .inf file.
func writeTempUTF16LEINF(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.inf")

	u16 := utf16.Encode([]rune(content))
	data := make([]byte, 2+len(u16)*2)
	data[0] = 0xFF // BOM
	data[1] = 0xFE
	for i, v := range u16 {
		binary.LittleEndian.PutUint16(data[2+i*2:], v)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

// writeTempUTF16BEINF writes UTF-16 BE BOM content to a temp .inf file.
func writeTempUTF16BEINF(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.inf")

	u16 := utf16.Encode([]rune(content))
	data := make([]byte, 2+len(u16)*2)
	data[0] = 0xFE // BOM
	data[1] = 0xFF
	for i, v := range u16 {
		binary.BigEndian.PutUint16(data[2+i*2:], v)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
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

// TestParseUTF8 tests parsing a plain UTF-8 INF file.
func TestParseUTF8(t *testing.T) {
	path := writeTempINF(t, validINF)
	inf, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if inf.GetSection("version") == nil {
		t.Fatal("Expected [Version] section")
	}
	if inf.GetSection("manufacturer") == nil {
		t.Fatal("Expected [Manufacturer] section")
	}
	if inf.GetSection("strings") == nil {
		t.Fatal("Expected [Strings] section")
	}
}

// TestParseUTF16LE tests parsing a UTF-16 LE BOM INF file.
func TestParseUTF16LE(t *testing.T) {
	path := writeTempUTF16LEINF(t, validINF)
	inf, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	sig := inf.GetValue("version", "Signature")
	if sig != "$Windows NT$" {
		t.Errorf("Signature = %q, want %q", sig, "$Windows NT$")
	}
}

// TestParseUTF16BE tests parsing a UTF-16 BE BOM INF file.
func TestParseUTF16BE(t *testing.T) {
	path := writeTempUTF16BEINF(t, validINF)
	inf, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	sig := inf.GetValue("version", "Signature")
	if sig != "$Windows NT$" {
		t.Errorf("Signature = %q, want %q", sig, "$Windows NT$")
	}
}

// TestParseUTF8BOM tests parsing a UTF-8 BOM INF file.
func TestParseUTF8BOM(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.inf")
	data := append([]byte{0xEF, 0xBB, 0xBF}, []byte(validINF)...)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}

	inf, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	sig := inf.GetValue("version", "Signature")
	if sig != "$Windows NT$" {
		t.Errorf("Signature = %q, want %q", sig, "$Windows NT$")
	}
}

// TestGetValueCaseInsensitive tests case-insensitive key lookup.
func TestGetValueCaseInsensitive(t *testing.T) {
	path := writeTempINF(t, validINF)
	inf, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	tests := []struct {
		section, key, want string
	}{
		{"Version", "Signature", "$Windows NT$"},
		{"VERSION", "signature", "$Windows NT$"},
		{"version", "SIGNATURE", "$Windows NT$"},
		{"Version", "Class", "System"},
		{"Version", "ClassGuid", "{4D36E97D-E325-11CE-BFC1-08002BE10318}"},
		{"Version", "Provider", "%ManufacturerName%"},
		{"Version", "CatalogFile", "TestDriver.cat"},
		{"Version", "DriverVer", "01/01/2020"},
		{"Version", "PnpLockdown", "1"},
	}

	for _, tt := range tests {
		got := inf.GetValue(tt.section, tt.key)
		if got != tt.want {
			t.Errorf("GetValue(%q, %q) = %q, want %q", tt.section, tt.key, got, tt.want)
		}
	}
}

// TestGetSectionNil tests that nonexistent sections return nil.
func TestGetSectionNil(t *testing.T) {
	path := writeTempINF(t, validINF)
	inf, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if sec := inf.GetSection("nonexistent"); sec != nil {
		t.Errorf("Expected nil for nonexistent section, got %v", sec)
	}
}

// TestGetValueMissing tests that missing key returns empty string.
func TestGetValueMissing(t *testing.T) {
	path := writeTempINF(t, validINF)
	inf, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if v := inf.GetValue("version", "nonexistent"); v != "" {
		t.Errorf("Expected empty for missing key, got %q", v)
	}
	if v := inf.GetValue("nonexistent", "Signature"); v != "" {
		t.Errorf("Expected empty for missing section, got %q", v)
	}
}

// TestResolveString tests %token% resolution.
func TestResolveString(t *testing.T) {
	path := writeTempINF(t, validINF)
	inf, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	tests := []struct {
		input, want string
	}{
		{"%ManufacturerName%", "TestManufacturer"},
		{"%DeviceName%", "Test Device"},
		{"%ServiceName%", "Test Service"},
		{"%DiskName%", "Test Install Disk"},
		{"%NonExistent%", "%NonExistent%"}, // unresolved
		{"NoPercent", "NoPercent"},         // no % wrapper
	}

	for _, tt := range tests {
		got := inf.ResolveString(tt.input)
		if got != tt.want {
			t.Errorf("ResolveString(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// TestCommentStripping tests that comments are properly stripped.
func TestCommentStripping(t *testing.T) {
	content := `[Version]
Signature = "$Windows NT$" ; this is a comment
Class = System ; another comment
ClassGuid = {4D36E97D-E325-11CE-BFC1-08002BE10318}

[Strings]
Foo = "semi;colon inside quotes"
`
	path := writeTempINF(t, content)
	inf, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	sig := inf.GetValue("version", "Signature")
	if sig != "$Windows NT$" {
		t.Errorf("Signature = %q, want %q (comment should be stripped)", sig, "$Windows NT$")
	}

	cls := inf.GetValue("version", "Class")
	if cls != "System" {
		t.Errorf("Class = %q, want %q", cls, "System")
	}

	// Semicolons inside quotes should be preserved
	foo := inf.GetValue("strings", "Foo")
	if foo != "semi;colon inside quotes" {
		t.Errorf("Foo = %q, want preserved semicolon inside quotes", foo)
	}
}

// TestDuplicateSections tests that entries from duplicate sections are merged.
func TestDuplicateSections(t *testing.T) {
	content := `[Version]
Signature = "$Windows NT$"

[MySection]
Key1 = Value1

[MySection]
Key2 = Value2
`
	path := writeTempINF(t, content)
	inf, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	sec := inf.GetSection("mysection")
	if sec == nil {
		t.Fatal("Expected [MySection] section")
	}
	if len(sec.Entries) != 2 {
		t.Errorf("Expected 2 entries in duplicate section, got %d", len(sec.Entries))
	}
}

// TestMultiValueParsing tests comma-separated value parsing.
func TestMultiValueParsing(t *testing.T) {
	content := `[Version]
Signature = "$Windows NT$"

[Test]
Key = Value1, Value2, Value3
`
	path := writeTempINF(t, content)
	inf, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	all := inf.GetAllValues("test", "Key")
	if len(all) != 1 {
		t.Fatalf("Expected 1 entry for Key, got %d", len(all))
	}
	if len(all[0]) != 3 {
		t.Fatalf("Expected 3 values, got %d", len(all[0]))
	}
	expected := []string{"Value1", "Value2", "Value3"}
	for i, v := range expected {
		if all[0][i] != v {
			t.Errorf("Value[%d] = %q, want %q", i, all[0][i], v)
		}
	}
}

// TestGetAllValues tests getting multiple entries for same key.
func TestGetAllValues(t *testing.T) {
	content := `[Version]
Signature = "$Windows NT$"

[Test]
Key = Value1
Key = Value2
`
	path := writeTempINF(t, content)
	inf, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	all := inf.GetAllValues("test", "Key")
	if len(all) != 2 {
		t.Fatalf("Expected 2 entries for Key, got %d", len(all))
	}
}

// TestGetEntry tests entry retrieval.
func TestGetEntry(t *testing.T) {
	content := `[Version]
Signature = "$Windows NT$"

[Test]
Key1 = Val1
Key2 = Val2
Key1 = Val3
`
	path := writeTempINF(t, content)
	inf, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	entries := inf.GetEntry("test", "Key1")
	if len(entries) != 2 {
		t.Fatalf("Expected 2 entries for Key1, got %d", len(entries))
	}
	if entries[0].Values[0] != "Val1" {
		t.Errorf("First entry = %q, want %q", entries[0].Values[0], "Val1")
	}
	if entries[1].Values[0] != "Val3" {
		t.Errorf("Second entry = %q, want %q", entries[1].Values[0], "Val3")
	}
}

// TestValueOnlyEntries tests lines without key= (value-only, like CopyFiles target lists).
func TestValueOnlyEntries(t *testing.T) {
	content := `[Version]
Signature = "$Windows NT$"

[FileList]
file1.sys
file2.dll
file3.inf
`
	path := writeTempINF(t, content)
	inf, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	sec := inf.GetSection("filelist")
	if sec == nil {
		t.Fatal("Expected [FileList] section")
	}
	if len(sec.Entries) != 3 {
		t.Fatalf("Expected 3 entries, got %d", len(sec.Entries))
	}
	// Value-only entries: key = trimmed line
	if sec.Entries[0].Key != "file1.sys" {
		t.Errorf("Entry[0].Key = %q, want %q", sec.Entries[0].Key, "file1.sys")
	}
}

// TestEmptyFile tests parsing an empty file.
func TestEmptyFile(t *testing.T) {
	path := writeTempINF(t, "")
	inf, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(inf.Sections) != 0 {
		t.Errorf("Expected 0 sections, got %d", len(inf.Sections))
	}
}

// TestNonexistentFile tests parsing a nonexistent file.
func TestNonexistentFile(t *testing.T) {
	_, err := Parse("/nonexistent/path/test.inf")
	if err == nil {
		t.Fatal("Expected error for nonexistent file")
	}
}

// TestSectionLineNumbers tests that line numbers are tracked correctly.
func TestSectionLineNumbers(t *testing.T) {
	content := `[Version]
Signature = "$Windows NT$"

[Section2]
Key = Value
`
	path := writeTempINF(t, content)
	inf, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	sec := inf.GetSection("version")
	if sec == nil || sec.Line != 1 {
		t.Errorf("Version section line = %d, want 1", sec.Line)
	}
	sec2 := inf.GetSection("section2")
	if sec2 == nil || sec2.Line != 4 {
		t.Errorf("Section2 section line = %d, want 4", sec2.Line)
	}
}

// TestSectionOrder tests that section order is preserved.
func TestSectionOrder(t *testing.T) {
	content := `[First]
A = 1

[Second]
B = 2

[Third]
C = 3
`
	path := writeTempINF(t, content)
	inf, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	expected := []string{"first", "second", "third"}
	if len(inf.Order) != len(expected) {
		t.Fatalf("Order length = %d, want %d", len(inf.Order), len(expected))
	}
	for i, name := range expected {
		if inf.Order[i] != name {
			t.Errorf("Order[%d] = %q, want %q", i, inf.Order[i], name)
		}
	}
}
