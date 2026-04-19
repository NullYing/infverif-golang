package infparser

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

// Section represents an INF file section with its entries.
type Section struct {
	Name    string
	Entries []Entry
	Line    int // line number where section starts
}

// Entry represents a single line in an INF section.
type Entry struct {
	Key    string
	Values []string
	Raw    string
	Line   int
}

// INFFile represents a parsed INF file.
type INFFile struct {
	Path     string
	Sections map[string]*Section // key is lowercase section name
	Order    []string            // section order
	Lines    []string            // all raw lines
}

// Parse reads and parses an INF file.
func Parse(path string) (*INFFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot open INF file: %w", err)
	}

	text := decodeFile(data)
	reader := bufio.NewReader(strings.NewReader(text))

	inf := &INFFile{
		Path:     path,
		Sections: make(map[string]*Section),
	}

	var currentSection *Section
	lineNum := 0

	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return nil, fmt.Errorf("error reading line %d: %w", lineNum+1, err)
		}

		lineNum++
		line = strings.TrimRight(line, "\r\n")
		inf.Lines = append(inf.Lines, line)

		// Strip comments (but not inside strings)
		stripped := stripComment(line)
		trimmed := strings.TrimSpace(stripped)

		if trimmed == "" {
			if err == io.EOF {
				break
			}
			continue
		}

		// Check for section header
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			name := trimmed[1 : len(trimmed)-1]
			nameLower := strings.ToLower(name)
			section := &Section{
				Name: name,
				Line: lineNum,
			}
			if _, exists := inf.Sections[nameLower]; !exists {
				inf.Sections[nameLower] = section
				inf.Order = append(inf.Order, nameLower)
			} else {
				// Duplicate section - append entries to existing
				section = inf.Sections[nameLower]
			}
			currentSection = section
		} else if currentSection != nil {
			entry := parseLine(trimmed, lineNum)
			currentSection.Entries = append(currentSection.Entries, entry)
		}

		if err == io.EOF {
			break
		}
	}

	return inf, nil
}

// GetSection returns a section by name (case-insensitive).
func (inf *INFFile) GetSection(name string) *Section {
	return inf.Sections[strings.ToLower(name)]
}

// GetValue returns the first value for a key in a section.
func (inf *INFFile) GetValue(sectionName, key string) string {
	sec := inf.GetSection(sectionName)
	if sec == nil {
		return ""
	}
	keyLower := strings.ToLower(key)
	for _, e := range sec.Entries {
		if strings.ToLower(e.Key) == keyLower {
			if len(e.Values) > 0 {
				return e.Values[0]
			}
			return ""
		}
	}
	return ""
}

// GetAllValues returns all values for a key in a section.
func (inf *INFFile) GetAllValues(sectionName, key string) [][]string {
	sec := inf.GetSection(sectionName)
	if sec == nil {
		return nil
	}
	keyLower := strings.ToLower(key)
	var result [][]string
	for _, e := range sec.Entries {
		if strings.ToLower(e.Key) == keyLower {
			result = append(result, e.Values)
		}
	}
	return result
}

// GetEntry returns all entries for a key in a section.
func (inf *INFFile) GetEntry(sectionName, key string) []Entry {
	sec := inf.GetSection(sectionName)
	if sec == nil {
		return nil
	}
	keyLower := strings.ToLower(key)
	var result []Entry
	for _, e := range sec.Entries {
		if strings.ToLower(e.Key) == keyLower {
			result = append(result, e)
		}
	}
	return result
}

// ResolveString resolves a %StringToken% reference.
func (inf *INFFile) ResolveString(token string) string {
	if !strings.HasPrefix(token, "%") || !strings.HasSuffix(token, "%") {
		return token
	}
	key := token[1 : len(token)-1]
	val := inf.GetValue("strings", key)
	if val == "" {
		return token
	}
	// Remove surrounding quotes if present
	if len(val) >= 2 && val[0] == '"' && val[len(val)-1] == '"' {
		val = val[1 : len(val)-1]
	}
	return val
}

func decodeFile(data []byte) string {
	// Check for UTF-16 LE BOM
	if len(data) >= 2 && data[0] == 0xFF && data[1] == 0xFE {
		return decodeUTF16LE(data[2:])
	}
	// Check for UTF-16 BE BOM
	if len(data) >= 2 && data[0] == 0xFE && data[1] == 0xFF {
		return decodeUTF16BE(data[2:])
	}
	// Check for UTF-8 BOM
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		return string(data[3:])
	}
	// Try to detect UTF-16 LE without BOM
	if len(data) >= 4 && !utf8.Valid(data) {
		if data[1] == 0 || data[3] == 0 {
			return decodeUTF16LE(data)
		}
	}
	return string(data)
}

func decodeUTF16LE(data []byte) string {
	if len(data)%2 != 0 {
		data = data[:len(data)-1]
	}
	u16s := make([]uint16, len(data)/2)
	for i := 0; i < len(data); i += 2 {
		u16s[i/2] = uint16(data[i]) | uint16(data[i+1])<<8
	}
	return string(utf16.Decode(u16s))
}

func decodeUTF16BE(data []byte) string {
	if len(data)%2 != 0 {
		data = data[:len(data)-1]
	}
	u16s := make([]uint16, len(data)/2)
	for i := 0; i < len(data); i += 2 {
		u16s[i/2] = uint16(data[i])<<8 | uint16(data[i+1])
	}
	return string(utf16.Decode(u16s))
}

func stripComment(line string) string {
	inQuote := false
	for i, c := range line {
		if c == '"' {
			inQuote = !inQuote
		} else if c == ';' && !inQuote {
			return line[:i]
		}
	}
	return line
}

func parseLine(line string, lineNum int) Entry {
	entry := Entry{
		Raw:  line,
		Line: lineNum,
	}

	// Split on '=' for key=value
	eqIdx := strings.Index(line, "=")
	if eqIdx >= 0 {
		entry.Key = strings.TrimSpace(line[:eqIdx])
		valPart := strings.TrimSpace(line[eqIdx+1:])
		entry.Values = splitValues(valPart)
	} else {
		// No key, treat as a value-only entry (like file list entries)
		entry.Key = strings.TrimSpace(line)
		entry.Values = splitValues(line)
	}

	return entry
}

func splitValues(s string) []string {
	var values []string
	var current strings.Builder
	inQuote := false

	for _, c := range s {
		if c == '"' {
			inQuote = !inQuote
			current.WriteRune(c)
		} else if c == ',' && !inQuote {
			val := strings.TrimSpace(current.String())
			val = trimQuotes(val)
			values = append(values, val)
			current.Reset()
		} else {
			current.WriteRune(c)
		}
	}

	if current.Len() > 0 {
		val := strings.TrimSpace(current.String())
		val = trimQuotes(val)
		values = append(values, val)
	}

	return values
}

func trimQuotes(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}
