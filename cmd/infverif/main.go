package main

import (
	"fmt"
	"infverif/pkg/verifier"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const appName = "InfVerif (Go)"
const appVersion = "0.3.0"

func main() {
	args := os.Args[1:]

	opts := verifier.Options{}
	var files []string
	var showHelp bool
	var errorCodeHelp int
	var showHDCRules bool
	var showExceptions bool

	for i := 0; i < len(args); i++ {
		arg := args[i]
		// Strip leading - or /
		flag := arg
		if strings.HasPrefix(flag, "/") || strings.HasPrefix(flag, "-") {
			flag = flag[1:]
		} else {
			files = append(files, arg)
			continue
		}

		switch strings.ToLower(flag) {
		case "v", "verbose":
			opts.Verbose = true
		case "c", "configurable":
			opts.Mode = verifier.ModeConfigurable
		case "u", "universal":
			opts.Mode = verifier.ModeUniversal
		case "w", "wcos":
			opts.Mode = verifier.ModeWindows
		case "k":
			opts.Mode = verifier.ModeSubmission
		case "h":
			opts.Mode = verifier.ModeSignatureRequirements
		case "msft":
			opts.Mode = verifier.ModeMSFT
			opts.MSFT = true
		case "info":
			opts.Mode = verifier.ModeInfo
		case "depends":
			opts.Mode = verifier.ModeDepends
		case "api":
			opts.Mode = verifier.ModeAPI
		case "syntax":
			opts.Mode = verifier.ModeSyntax
		case "stampinf":
			opts.StampInf = true
		case "msbuild":
			opts.MSBuild = true
		case "werror":
			opts.WError = true
		case "levelsort":
			opts.LevelSort = true
		case "inbox":
			opts.Inbox = true
		case "append":
			opts.Append = true
		case "recurse":
			opts.Recurse = true
		case "debug":
			opts.Debug = true
		case "noexceptions":
			opts.NoExceptions = true
		case "attestation":
			opts.Attestation = true
		case "logging":
			opts.Logging = true
		case "verboseparams":
			opts.VerboseParams = true
		case "samples":
			opts.Samples = true
		case "wdk":
			opts.WDK = true
		case "hdcrules":
			showHDCRules = true
		case "showexceptions":
			showExceptions = true
		case "code":
			i++
			if i < len(args) {
				fmt.Sscanf(args[i], "%d", &errorCodeHelp)
			}
		case "rulever":
			i++
			if i < len(args) {
				rv, ok := verifier.ParseRuleVersion(args[i])
				if ok {
					opts.RuleVer = &rv
				} else {
					fmt.Fprintf(os.Stderr, "Invalid /rulever value: %s\n", args[i])
					os.Exit(87)
				}
			}
		case "provider":
			i++
			if i < len(args) {
				opts.Provider = args[i]
			}
		case "dll":
			i++
			if i < len(args) {
				opts.DLLPath = args[i]
			}
		case "errorlist":
			i++
			if i < len(args) {
				opts.ErrorListFile = args[i]
			}
		case "errorlevel":
			i++
			if i < len(args) {
				fmt.Sscanf(args[i], "%d", &opts.ErrorLevel)
			}
		case "csv":
			i++
			if i < len(args) {
				opts.CSVFile = args[i]
			}
		case "l", "logout":
			i++
			if i < len(args) {
				opts.LogPath = args[i]
			}
		case "osver":
			i++
			if i < len(args) {
				opts.OsVer = args[i]
			}
		case "wbuild":
			i++
			if i < len(args) {
				opts.WBuild = args[i]
			}
		case "product":
			i++
			if i < len(args) {
				opts.ProductFile = args[i]
			}
		case "exclude":
			i++
			if i < len(args) {
				opts.ExcludeFile = args[i]
			}
		case "fileroot":
			i++
			if i < len(args) {
				opts.FileRoot = args[i]
			}
		case "?", "help":
			showHelp = true
		default:
			// Unknown flag - treat as file
			files = append(files, arg)
		}
	}

	if showHelp || len(files) == 0 {
		// Handle immediate-exit commands first (don't need files)
		if showHDCRules {
			verifier.PrintHDCRules()
			os.Exit(0)
		}
		if showExceptions {
			verifier.PrintExceptions()
			os.Exit(0)
		}
		if errorCodeHelp > 0 {
			verifier.PrintErrorCodeHelp(errorCodeHelp)
			os.Exit(0)
		}

		printUsage()
		if len(files) == 0 && !showHelp {
			os.Exit(87)
		}
		os.Exit(0)
	}

	// Handle immediate-exit commands (even with files specified)
	if showHDCRules {
		verifier.PrintHDCRules()
		os.Exit(0)
	}
	if showExceptions {
		verifier.PrintExceptions()
		os.Exit(0)
	}
	if errorCodeHelp > 0 {
		verifier.PrintErrorCodeHelp(errorCodeHelp)
		os.Exit(0)
	}

	// Parameter validation: /noexceptions requires /h
	if opts.NoExceptions && opts.Mode != verifier.ModeSignatureRequirements {
		fmt.Fprintln(os.Stderr, "Error: /noexceptions requires /h mode.")
		printUsage()
		os.Exit(87)
	}

	// Expand wildcards and recurse
	var expandedFiles []string
	for _, pattern := range files {
		if opts.Recurse {
			expandedFiles = append(expandedFiles, expandRecursive(pattern)...)
		} else if strings.Contains(pattern, "*") || strings.Contains(pattern, "?") {
			matches, err := filepath.Glob(pattern)
			if err != nil || len(matches) == 0 {
				fmt.Fprintf(os.Stderr, "No files matching pattern '%s'\n", pattern)
				continue
			}
			expandedFiles = append(expandedFiles, matches...)
		} else {
			expandedFiles = append(expandedFiles, pattern)
		}
	}

	// Apply exclusion list
	if opts.ExcludeFile != "" {
		excludes := loadExcludeList(opts.ExcludeFile)
		if len(excludes) > 0 {
			var filtered []string
			for _, f := range expandedFiles {
				baseName := strings.ToLower(filepath.Base(f))
				if !excludes[baseName] {
					filtered = append(filtered, f)
				}
			}
			expandedFiles = filtered
		}
	}

	if len(expandedFiles) == 0 {
		fmt.Println("No INF files specified.")
		os.Exit(87)
	}

	// Print verbose mode info
	if opts.Verbose {
		printVerboseModeInfo(opts)
	}

	start := time.Now()
	exitCode := 0
	fileCount := 0

	// Open CSV file if requested
	var csvFile *os.File
	if opts.CSVFile != "" {
		flags := os.O_CREATE | os.O_WRONLY
		if opts.Append {
			flags |= os.O_APPEND
		} else {
			flags |= os.O_TRUNC
		}
		var err error
		csvFile, err = os.OpenFile(opts.CSVFile, flags, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Cannot open CSV file '%s': %v\n", opts.CSVFile, err)
			os.Exit(87)
		}
		defer csvFile.Close()
		if !opts.Append {
			fmt.Fprintln(csvFile, verifier.CSVHeader())
		}
	}

	for _, file := range expandedFiles {
		// Validate file extension
		if !strings.EqualFold(filepath.Ext(file), ".inf") {
			fmt.Fprintf(os.Stderr, "File '%s' does not have .inf extension, skipping.\n", file)
			continue
		}

		fileCount++

		switch opts.Mode {
		case verifier.ModeInfo:
			result := verifier.GetInfo(file)
			printInfo(file, result)
			if result.Err != nil {
				exitCode = 1
			}

		case verifier.ModeDepends:
			depInfo, err := verifier.GetDepends(file)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error parsing '%s': %v\n", file, err)
				exitCode = 1
			} else {
				printDepends(file, depInfo)
			}

		case verifier.ModeSyntax:
			entries, err := verifier.CollectSyntax(file)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error parsing '%s': %v\n", file, err)
				exitCode = 1
			} else {
				printSyntax(file, entries)
			}

		default:
			// Validation mode
			result := verifier.Verify(file, opts)
			printResult(file, result, opts)

			// Write CSV if requested
			if csvFile != nil {
				for _, issue := range result.Issues {
					fmt.Fprintln(csvFile, verifier.FormatCSVRow(filepath.Base(file), issue))
				}
			}

			if !result.Valid {
				exitCode = 1
			}
		}
	}

	elapsed := time.Since(start)
	if opts.Verbose || fileCount > 0 {
		if opts.Mode != verifier.ModeInfo && opts.Mode != verifier.ModeDepends {
			minutes := int(elapsed.Minutes())
			seconds := int(elapsed.Seconds()) % 60
			millis := int(elapsed.Milliseconds()) % 1000
			fmt.Printf("\nChecked %d INF(s) in %d m %d s %d ms\n", fileCount, minutes, seconds, millis)
		}
	}

	os.Exit(exitCode)
}

func printVerboseModeInfo(opts verifier.Options) {
	fmt.Println("Running in Verbose")
	flags := verifier.ModeFlags(opts.Mode)
	// /h mode
	if opts.Mode == verifier.ModeSignatureRequirements {
		fmt.Println("Running signature requirements check")
		rv := verifier.GetEffectiveRuleVersion(opts)
		fmt.Printf("Using rules from OS build: %s\n", rv.String())
	}
	// DLL info
	if opts.DLLPath != "" {
		fmt.Printf("Using validation DLL: %s\n", opts.DLLPath)
	}
	// Print checks in reverse order (most specific first) matching original behavior
	if flags&0x04 != 0 {
		fmt.Println("Running state separation check")
	}
	if flags&0x02 != 0 {
		fmt.Println("Running include/needs check")
	}
	if flags&0x01 != 0 {
		fmt.Println("Running configurability check")
	}
	switch opts.Mode {
	case verifier.ModeSubmission:
		fmt.Println("Running Declarative Driver requirements check")
	case verifier.ModeMSFT:
		fmt.Println("Running in MSFT driver mode")
	}
	// /verboseparams
	if opts.VerboseParams {
		allFlags := verifier.ModeFlags(opts.Mode)
		// Add other option flags
		if opts.Inbox {
			allFlags |= 0x10
		}
		if opts.MSFT {
			allFlags |= 0x20
		}
		if opts.NoExceptions {
			allFlags |= 0x800
		}
		if opts.Attestation {
			allFlags |= 0x8000
		}
		if opts.Recurse {
			allFlags |= 0x100
		}
		if opts.LevelSort {
			allFlags |= 0x2000
		}
		if opts.WError {
			allFlags |= 0x800000
		}
		if opts.ErrorListFile != "" {
			allFlags |= 0x1000000
		}
		fmt.Printf("InfVerif Flags: 0x%08X\n", allFlags)
	}
}

func printResult(file string, result verifier.Result, opts verifier.Options) {
	baseName := filepath.Base(file)
	absPath, _ := filepath.Abs(file)

	fmt.Printf("\nValidating %s\n", baseName)

	for _, issue := range result.Issues {
		if opts.MSBuild {
			// MSBuild format: filename(line): error code: message
			if issue.Line > 0 {
				fmt.Printf("%s(%d): %s %d: %s\n",
					absPath, issue.Line, strings.ToLower(issue.Level.String()), issue.Code, issue.Message)
			} else {
				fmt.Printf("%s: %s %d: %s\n",
					absPath, strings.ToLower(issue.Level.String()), issue.Code, issue.Message)
			}
		} else {
			// Default format: ERROR(CODE) in filepath, line LINE: message
			if issue.Line > 0 {
				fmt.Printf("%s(%d) in %s, line %d: %s\n",
					issue.Level, issue.Code, absPath, issue.Line, issue.Message)
			} else {
				fmt.Printf("%s(%d) in %s: %s\n",
					issue.Level, issue.Code, absPath, issue.Message)
			}
		}
	}

	if result.Valid {
		fmt.Println("INF is VALID")
	} else {
		fmt.Println("INF is NOT VALID")
	}
}

func printInfo(file string, result verifier.Result) {
	baseName := filepath.Base(file)

	fmt.Printf("\n%s Information\n", baseName)

	if result.Info == nil {
		fmt.Println("  Unable to parse INF file.")
		return
	}

	info := result.Info
	fmt.Println()
	fmt.Printf("INF Hash:\t%s\n", info.Hash)
	fmt.Printf("Family ID:\t%s\n", info.FamilyID)
	fmt.Printf("Driver Type:\t%s\n", info.DriverType)

	for _, dev := range info.Devices {
		fmt.Println()
		fmt.Printf("Device:\t\t%s\n", dev.Name)
		fmt.Printf("Hardware ID:\t%s\n", dev.HardwareID)
		if dev.Service != "" {
			fmt.Printf("Service:\t%s\n", dev.Service)
		}
		fmt.Printf("Section Name:\t%s\n", dev.SectionName)
	}

	if info.Architecture != "" {
		fmt.Printf("Architecture:\t%s\n", info.Architecture)
	}
}

func printUsage() {
	fmt.Printf("\n%s\nVersion %s\n\n", appName, appVersion)
	fmt.Println("USAGE: infverif [/code <error code>] [/v] [[/h] | [/w] | [/u] | [/k]]")
	fmt.Println("                [/rulever <Major.Minor.Build> | vnext]")
	fmt.Println("                [/wbuild <Major.Minor.Build>] [/info] [/stampinf]")
	fmt.Println("                [/l <path>] [/osver <TargetOSVersion>] [/product <ias file>]")
	fmt.Println("                [/provider <ProviderName>] <files>")
	fmt.Println()
	fmt.Println("/v")
	fmt.Println("\tDisplay verbose file logging details.")
	fmt.Println()
	fmt.Println("/h")
	fmt.Println("\tReports errors using WHQL signature requirements. (mode)")
	fmt.Println()
	fmt.Println("/c")
	fmt.Println("\tReports errors for Configurability requirements. (mode)")
	fmt.Println()
	fmt.Println("/u")
	fmt.Println("\tReports errors for Universal Driver requirements. (mode)")
	fmt.Println()
	fmt.Println("/w")
	fmt.Println("\tReports errors for Windows Driver requirements. (mode)")
	fmt.Println()
	fmt.Println("/k")
	fmt.Println("\tReports errors using Declarative Driver requirements. (mode)")
	fmt.Println()
	fmt.Println("/code <error code>")
	fmt.Println("\tDisplays help for the specified InfVerif error code.")
	fmt.Println()
	fmt.Println("/rulever <Major.Minor.Build> | vnext")
	fmt.Println("\tSpecify the rule version for /h mode. Default is current InfVerif version.")
	fmt.Println("\tNamed versions: vnext, 24h2, 25h2, 26h2, 27h2")
	fmt.Println()
	fmt.Println("/wbuild <Major.Minor.Build>")
	fmt.Println("\tFor Windows Drivers with downlevel support, specifies")
	fmt.Println("\tthe build number where /w should be enforced.")
	fmt.Println("\tDefaults to 10.0.17763")
	fmt.Println()
	fmt.Println("/info")
	fmt.Println("\tDisplays INF summary information.")
	fmt.Println()
	fmt.Println("/stampinf")
	fmt.Println("\tTreat $ARCH$ as a valid architecture, to validate")
	fmt.Println("\tpre-stampinf files.")
	fmt.Println()
	fmt.Println("/l <path>")
	fmt.Println("\tAn inline-annotated HTML version of each INF")
	fmt.Println("\tfile will be placed in the <path>.")
	fmt.Println()
	fmt.Println("/osver <TargetOsVersion>")
	fmt.Println("\tProcess the INF for only a specific target OS.")
	fmt.Println("\tFormatting is the same as a Models section, i.e. NTAMD64.6.0")
	fmt.Println()
	fmt.Println("/product <ias file>")
	fmt.Println("\tValidate Include/Needs against a product definition .ias file.")
	fmt.Println()
	fmt.Println("/provider <ProviderName>")
	fmt.Println("\tReports an error for INFs not using the specified provider name.")
	fmt.Println()
	fmt.Println("<files>")
	fmt.Println("\tA space-separated list of INF files to analyze.")
	fmt.Println("\tAll files must have .inf extension.")
	fmt.Println("\tWildcards (*) may be used.")
	fmt.Println()
	fmt.Println("Only one mode option (/h, /c, /u, /w, /k, /info, /depends, /syntax)")
	fmt.Println("may be passed at a time.")
}

func printDepends(file string, depInfo *verifier.DependencyInfo) {
	baseName := filepath.Base(file)
	fmt.Printf("\n%s dependencies\n", baseName)

	if len(depInfo.Sections) == 0 {
		fmt.Println("  No Include/Needs dependencies found.")
		return
	}

	for _, sec := range depInfo.Sections {
		fmt.Printf("\nSection [%s]\n", sec.Name)
		if len(sec.Includes) > 0 {
			fmt.Println("  INFs:")
			for _, inc := range sec.Includes {
				fmt.Printf("    %s\n", inc)
			}
		}
		if len(sec.Needs) > 0 {
			fmt.Println("  Sections:")
			for _, need := range sec.Needs {
				fmt.Printf("    [%s]\n", need)
			}
		}
	}
}

func printSyntax(file string, entries []verifier.SyntaxEntry) {
	fmt.Println("\nINF Syntax Report")
	fmt.Println()

	if len(entries) == 0 {
		fmt.Println("No syntax found")
		return
	}

	fmt.Printf("%-35s %s\n", "Syntax", "Minimum Supported OS")
	for _, e := range entries {
		name := e.Name
		if len(name) > 35 {
			name = name[:35]
		}
		fmt.Printf("%-35s (%s)\n", name, e.MinVersion.String())
	}
}

func expandRecursive(pattern string) []string {
	var result []string
	dir := filepath.Dir(pattern)
	base := filepath.Base(pattern)

	if dir == "" || dir == "." {
		dir = "."
	}

	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		matched, _ := filepath.Match(strings.ToLower(base), strings.ToLower(filepath.Base(path)))
		if matched {
			result = append(result, path)
		}
		return nil
	})

	return result
}

func loadExcludeList(path string) map[string]bool {
	result := make(map[string]bool)
	data, err := os.ReadFile(path)
	if err != nil {
		return result
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") && !strings.HasPrefix(line, ";") {
			result[strings.ToLower(line)] = true
		}
	}
	return result
}
