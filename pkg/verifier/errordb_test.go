package verifier

import (
	"testing"
)

// === ErrorDB tests ===

func TestFindErrorExisting(t *testing.T) {
	tests := []struct {
		code int
		desc string
	}{
		{1200, "Invalid INF: must contain [Version] section"},
		{1203, "Section name too long"},
		{1220, "Invalid GUID format"},
		{1284, "Reserved device class name"},
		{1302, "Provider cannot be"},
		{2100, "Registry operation not isolated to HKR"},
		{2150, "PnpLockdown=1 not specified"},
	}

	for _, tt := range tests {
		entry := FindError(tt.code)
		if entry == nil {
			t.Errorf("FindError(%d) = nil, want non-nil", tt.code)
			continue
		}
		if entry.Code != tt.code {
			t.Errorf("FindError(%d).Code = %d", tt.code, entry.Code)
		}
		// Check description contains expected substring
		if len(tt.desc) > 0 {
			found := false
			if len(entry.Description) >= len(tt.desc) {
				for i := 0; i <= len(entry.Description)-len(tt.desc); i++ {
					if entry.Description[i:i+len(tt.desc)] == tt.desc {
						found = true
						break
					}
				}
			}
			if !found {
				t.Errorf("FindError(%d).Description = %q, want to contain %q", tt.code, entry.Description, tt.desc)
			}
		}
	}
}

func TestFindErrorNotFound(t *testing.T) {
	for _, code := range []int{0, 1, 999, 1199, 9999} {
		entry := FindError(code)
		if entry != nil {
			t.Errorf("FindError(%d) = %+v, want nil", code, entry)
		}
	}
}

func TestFindErrorBoundary(t *testing.T) {
	// First and last entries in the database
	first := FindError(1200)
	if first == nil {
		t.Error("FindError(1200) = nil, want first entry")
	}

	last := FindError(2150)
	if last == nil {
		t.Error("FindError(2150) = nil, want last entry")
	}
}

func TestGetAllErrors(t *testing.T) {
	all := GetAllErrors()
	if len(all) == 0 {
		t.Fatal("GetAllErrors() returned empty slice")
	}

	// Verify sorted order (binary search requires this)
	for i := 1; i < len(all); i++ {
		if all[i].Code <= all[i-1].Code {
			t.Errorf("Database not sorted: entry[%d].Code=%d <= entry[%d].Code=%d",
				i, all[i].Code, i-1, all[i-1].Code)
		}
	}
}

func TestHDCFlags(t *testing.T) {
	// Codes with HDCFlagAllSubmissions
	allSubmCodes := []int{1280, 1281, 1282, 1283, 1284, 1285, 1286, 1301, 1302,
		1310, 1311, 1312, 1313, 1314, 1315, 1316, 1317, 1318, 1319, 1340}
	for _, code := range allSubmCodes {
		entry := FindError(code)
		if entry == nil {
			t.Errorf("FindError(%d) = nil", code)
			continue
		}
		if entry.Flags&HDCFlagAllSubmissions == 0 {
			t.Errorf("Error %d should have HDCFlagAllSubmissions", code)
		}
	}

	// Codes with HDCFlagDownlevelDeclarative
	dlCodes := []int{2100, 2120, 2150}
	for _, code := range dlCodes {
		entry := FindError(code)
		if entry == nil {
			t.Errorf("FindError(%d) = nil", code)
			continue
		}
		if entry.Flags&HDCFlagDownlevelDeclarative == 0 {
			t.Errorf("Error %d should have HDCFlagDownlevelDeclarative", code)
		}
	}

	// Codes without HDC flags
	noHDCCodes := []int{1200, 1203, 1204, 1220, 1230, 1258, 1264}
	for _, code := range noHDCCodes {
		entry := FindError(code)
		if entry == nil {
			continue
		}
		if entry.Flags != 0 {
			t.Errorf("Error %d should have no HDC flags, got 0x%X", code, entry.Flags)
		}
	}
}

// === Exceptions tests ===

func TestParseRuleVersionNamed(t *testing.T) {
	tests := []struct {
		input string
		want  RuleVersion
		ok    bool
	}{
		{"vnext", RuleVersion{10, 0, 99999}, true},
		{"vnext_2", RuleVersion{10, 0, 99998}, true},
		{"24h2", RuleVersion{10, 0, 26100}, true},
		{"25h2", RuleVersion{10, 0, 26200}, true},
		{"26h2", RuleVersion{10, 0, 26300}, true},
		{"27h2", RuleVersion{10, 0, 26400}, true},
		{"VNEXT", RuleVersion{10, 0, 99999}, true}, // case insensitive
		{"24H2", RuleVersion{10, 0, 26100}, true},  // case insensitive
	}

	for _, tt := range tests {
		got, ok := ParseRuleVersion(tt.input)
		if ok != tt.ok {
			t.Errorf("ParseRuleVersion(%q) ok = %v, want %v", tt.input, ok, tt.ok)
			continue
		}
		if got != tt.want {
			t.Errorf("ParseRuleVersion(%q) = %s, want %s", tt.input, got.String(), tt.want.String())
		}
	}
}

func TestParseRuleVersionNumeric(t *testing.T) {
	tests := []struct {
		input string
		want  RuleVersion
		ok    bool
	}{
		{"10.0.26100", RuleVersion{10, 0, 26100}, true},
		{"10.0.26200", RuleVersion{10, 0, 26200}, true},
		{"10.0.0", RuleVersion{10, 0, 0}, true},
		{"1.2.3", RuleVersion{1, 2, 3}, true},
	}

	for _, tt := range tests {
		got, ok := ParseRuleVersion(tt.input)
		if ok != tt.ok {
			t.Errorf("ParseRuleVersion(%q) ok = %v, want %v", tt.input, ok, tt.ok)
			continue
		}
		if got != tt.want {
			t.Errorf("ParseRuleVersion(%q) = %s, want %s", tt.input, got.String(), tt.want.String())
		}
	}
}

func TestParseRuleVersionInvalid(t *testing.T) {
	invalid := []string{"", "invalid", "10.0", "10", "abc.def.ghi"}
	for _, s := range invalid {
		_, ok := ParseRuleVersion(s)
		if ok {
			t.Errorf("ParseRuleVersion(%q) should fail", s)
		}
	}
}

func TestRuleVersionString(t *testing.T) {
	rv := RuleVersion{10, 0, 26200}
	got := rv.String()
	if got != "10.0.26200" {
		t.Errorf("RuleVersion.String() = %q, want %q", got, "10.0.26200")
	}
}

func TestIsExceptionActive(t *testing.T) {
	tests := []struct {
		name string
		exc  Exception
		rv   RuleVersion
		want bool
	}{
		{
			"no removal - always active",
			Exception{RemoveVersion: ""},
			RuleVersion{10, 0, 99999},
			true,
		},
		{
			"before removal version - active",
			Exception{RemoveVersion: "10.0.26100"},
			RuleVersion{10, 0, 26000},
			true,
		},
		{
			"at removal version - inactive",
			Exception{RemoveVersion: "10.0.26100"},
			RuleVersion{10, 0, 26100},
			false,
		},
		{
			"after removal version - inactive",
			Exception{RemoveVersion: "10.0.26100"},
			RuleVersion{10, 0, 26200},
			false,
		},
		{
			"minor version boundary",
			Exception{RemoveVersion: "10.1.0"},
			RuleVersion{10, 0, 99999},
			true,
		},
		{
			"major version boundary",
			Exception{RemoveVersion: "11.0.0"},
			RuleVersion{10, 0, 99999},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsExceptionActive(tt.exc, tt.rv)
			if got != tt.want {
				t.Errorf("IsExceptionActive() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsRegistryPathExempt(t *testing.T) {
	rv := RuleVersion{10, 0, 26000} // Before 26100 removals

	tests := []struct {
		name         string
		root, subkey string
		noExceptions bool
		want         bool
	}{
		{"HKCR wildcard", "HKCR", `anything\at\all`, false, true},
		{"HKLM Classes exact", "HKLM", `SOFTWARE\Classes`, false, true},
		{"HKLM Classes sub", "HKLM", `SOFTWARE\Classes\MyClass`, false, true},
		{"HKLM Khronos", "HKLM", `SOFTWARE\Khronos`, false, true},
		{"HKLM not matched", "HKLM", `SOFTWARE\Unknown\Path`, false, false},
		{"noExceptions disables all", "HKCR", `anything`, true, false},
		{"HKLM CCS before removal", "HKLM", `SYSTEM\CurrentControlSet`, false, true},
		{"case insensitive root", "hklm", `SOFTWARE\Classes`, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsRegistryPathExempt(tt.root, tt.subkey, rv, tt.noExceptions)
			if got != tt.want {
				t.Errorf("IsRegistryPathExempt(%q, %q) = %v, want %v", tt.root, tt.subkey, got, tt.want)
			}
		})
	}
}

func TestIsRegistryPathExemptVersionRemoval(t *testing.T) {
	// HKLM\SYSTEM\CurrentControlSet is removed at 10.0.26100
	before := RuleVersion{10, 0, 26000}
	at := RuleVersion{10, 0, 26100}
	after := RuleVersion{10, 0, 26200}

	if !IsRegistryPathExempt("HKLM", `SYSTEM\CurrentControlSet`, before, false) {
		t.Error("Expected exempt before removal version")
	}
	if IsRegistryPathExempt("HKLM", `SYSTEM\CurrentControlSet`, at, false) {
		t.Error("Expected NOT exempt at removal version")
	}
	if IsRegistryPathExempt("HKLM", `SYSTEM\CurrentControlSet`, after, false) {
		t.Error("Expected NOT exempt after removal version")
	}
}

func TestIsFilePathExempt(t *testing.T) {
	rv := RuleVersion{10, 0, 26000} // Before 26100 removals

	tests := []struct {
		name         string
		dirid        string
		subpath      string
		noExceptions bool
		want         bool
	}{
		{"System32 (DIRID 11)", "11", "", false, true},
		{"System32 subpath", "11", `foo.dll`, false, true},
		{"drivers (DIRID 12)", "12", "", false, true},
		{"DIRID 13 not exempt", "13", "", false, false},
		{"DIRID 10 Provisioning", "10", "Provisioning", false, true},
		{"DIRID 10 subpath", "10", `Provisioning\something`, false, true},
		{"DIRID 10 no match", "10", "OtherFolder", false, false},
		{"noExceptions disables all", "11", "", true, false},
		{"Program Files before removal", "16422", "", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsFilePathExempt(tt.dirid, tt.subpath, rv, tt.noExceptions)
			if got != tt.want {
				t.Errorf("IsFilePathExempt(%q, %q) = %v, want %v", tt.dirid, tt.subpath, got, tt.want)
			}
		})
	}
}

func TestIsFilePathExemptVersionRemoval(t *testing.T) {
	// DIRID 16422 (Program Files) removed at 10.0.26100
	before := RuleVersion{10, 0, 26000}
	at := RuleVersion{10, 0, 26100}

	if !IsFilePathExempt("16422", "", before, false) {
		t.Error("Expected exempt before removal version")
	}
	if IsFilePathExempt("16422", "", at, false) {
		t.Error("Expected NOT exempt at removal version")
	}
}

func TestExceptionTableCounts(t *testing.T) {
	if len(RegistryExceptions) < 30 {
		t.Errorf("RegistryExceptions has %d entries, expected at least 30", len(RegistryExceptions))
	}
	if len(FileExceptions) < 20 {
		t.Errorf("FileExceptions has %d entries, expected at least 20", len(FileExceptions))
	}
}

func TestDefaultRuleVersion(t *testing.T) {
	if DefaultRuleVersion.Major != 10 || DefaultRuleVersion.Minor != 0 || DefaultRuleVersion.Build != 26200 {
		t.Errorf("DefaultRuleVersion = %s, want 10.0.26200", DefaultRuleVersion.String())
	}
}
