package cpi

import (
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestVersionCreateBuildMeta(t *testing.T) {
	tests := []struct {
		name       string
		buildOs    string
		buildArch  string
		build      string
		expected   string
		expectGOOS bool
	}{
		{
			name:      "all values provided",
			buildOs:   "linux",
			buildArch: "amd64",
			build:     "12345",
			expected:  "linux.amd64.12345",
		},
		{
			name:      "without build number",
			buildOs:   "darwin",
			buildArch: "arm64",
			build:     "",
			expected:  "darwin.arm64",
		},
		{
			name:       "empty values default to runtime",
			buildOs:    "",
			buildArch:  "",
			build:      "",
			expected:   fmt.Sprintf("%s.%s", runtime.GOOS, runtime.GOARCH),
			expectGOOS: true,
		},
		{
			name:      "with whitespace",
			buildOs:   "  linux  ",
			buildArch: "  amd64  ",
			build:     "  12345  ",
			expected:  "linux.amd64.12345",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := createBuildMeta(tt.buildOs, tt.buildArch, tt.build)

			if tt.expectGOOS {
				if result != tt.expected {
					t.Errorf("createBuildMeta(%q, %q, %q) = %q, want %q",
						tt.buildOs, tt.buildArch, tt.build, result, tt.expected)
				}
			} else {
				if result != tt.expected {
					t.Errorf("createBuildMeta(%q, %q, %q) = %q, want %q",
						tt.buildOs, tt.buildArch, tt.build, result, tt.expected)
				}
			}
		})
	}
}

func TestVersionCreateSemVer(t *testing.T) {
	tests := []struct {
		name       string
		major      string
		minor      string
		patch      string
		prerelease string
		build      string
		expected   string
	}{
		{
			name:       "complete semantic version",
			major:      "1",
			minor:      "2",
			patch:      "3",
			prerelease: "alpha",
			build:      "12345",
			expected:   "1.2.3-alpha+12345",
		},
		{
			name:       "without build metadata",
			major:      "2",
			minor:      "0",
			patch:      "1",
			prerelease: "beta",
			build:      "",
			expected:   "2.0.1-beta",
		},
		{
			name:       "without prerelease",
			major:      "3",
			minor:      "1",
			patch:      "4",
			prerelease: "",
			build:      "67890",
			expected:   "3.1.4+67890",
		},
		{
			name:       "base version only",
			major:      "1",
			minor:      "0",
			patch:      "0",
			prerelease: "",
			build:      "",
			expected:   "1.0.0",
		},
		{
			name:       "empty values default to zeros",
			major:      "",
			minor:      "",
			patch:      "",
			prerelease: "",
			build:      "",
			expected:   "0.0.0",
		},
		{
			name:       "with whitespace",
			major:      "  4  ",
			minor:      "  5  ",
			patch:      "  6  ",
			prerelease: "  rc1  ",
			build:      "  54321  ",
			expected:   "4.5.6-rc1+54321",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := createSemVer(tt.major, tt.minor, tt.patch, tt.prerelease, tt.build)
			if result != tt.expected {
				t.Errorf("createSemVer(%q, %q, %q, %q, %q) = %q, want %q",
					tt.major, tt.minor, tt.patch, tt.prerelease, tt.build, result, tt.expected)
			}
		})
	}
}

func TestVersionGetCpiVersion(t *testing.T) {
	// Save original values
	origMajor, origMinor, origPatch := SemVerMajor, SemVerMinor, SemVerPatch
	origPrerelease, origBuild := SemVerPrerelease, SemVerBuild
	origGoOs, origGoArch := BuildGoOs, BuildGoArch

	// Restore after test
	defer func() {
		SemVerMajor, SemVerMinor, SemVerPatch = origMajor, origMinor, origPatch
		SemVerPrerelease, SemVerBuild = origPrerelease, origBuild
		BuildGoOs, BuildGoArch = origGoOs, origGoArch
	}()

	// Test case 1: With all values set
	SemVerMajor, SemVerMinor, SemVerPatch = "1", "2", "3"
	SemVerPrerelease = "alpha"
	SemVerBuild = "12345"
	BuildGoOs = "linux"
	BuildGoArch = "amd64"

	expected := "1.2.3-alpha+linux.amd64.12345"
	result := getCpiVersion()

	if result != expected {
		t.Errorf("getCpiVersion() = %q, want %q", result, expected)
	}

	// Test case 2: With empty values (should use defaults)
	SemVerMajor, SemVerMinor, SemVerPatch = "", "", ""
	SemVerPrerelease = ""
	SemVerBuild = ""
	BuildGoOs = ""
	BuildGoArch = ""

	expected = fmt.Sprintf("0.0.0+%s.%s", runtime.GOOS, runtime.GOARCH)
	result = getCpiVersion()

	if result != expected {
		t.Errorf("getCpiVersion() = %q, want %q", result, expected)
	}
}

func TestVersionGetCpiGoVersion(t *testing.T) {
	result := getCpiGoVersion()
	expected := runtime.Version()

	if result != expected {
		t.Errorf("getCpiGoVersion() = %q, want %q", result, expected)
	}
}

func TestGetCpiZuluTimeStamp(t *testing.T) {
	tests := []struct {
		name     string
		timeStr  string
		checkNow bool
	}{
		{
			name:     "valid timestamp",
			timeStr:  "2023-04-15T14:30:45Z",
			checkNow: false,
		},
		{
			name:     "empty timestamp uses current time",
			timeStr:  "",
			checkNow: true,
		},
		{
			name:     "invalid timestamp uses current time",
			timeStr:  "not-a-timestamp",
			checkNow: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			beforeTest := time.Now().UTC().Add(-1 * time.Second) // Allow 1 second before
			result := getCpiZuluTimeStamp(tt.timeStr)
			afterTest := time.Now().UTC().Add(1 * time.Second) // Allow 1 second after

			if tt.checkNow {
				// Parse the result
				resultTime, err := time.Parse(time.RFC3339, result)
				if err != nil {
					t.Errorf("getCpiZuluTimeStamp(%q) returned invalid timestamp: %q", tt.timeStr, result)
					return
				}

				// Check that the result is between beforeTest and afterTest with a more generous time window
				if resultTime.Before(beforeTest) || resultTime.After(afterTest) {
					t.Errorf("getCpiZuluTimeStamp(%q) = %q, expected time between %q and %q",
						tt.timeStr, result, beforeTest.Format(time.RFC3339), afterTest.Format(time.RFC3339))
				}
			} else {
				expected := tt.timeStr
				if result != expected {
					t.Errorf("getCpiZuluTimeStamp(%q) = %q, want %q", tt.timeStr, result, expected)
				}
			}
		})
	}
}

func TestVersionCPIVersion(t *testing.T) {
	// Save original values
	origProject := BuildProject
	origMajor, origMinor, origPatch := SemVerMajor, SemVerMinor, SemVerPatch
	origPrerelease, origBuild := SemVerPrerelease, SemVerBuild
	origBuildDate := BuildDate
	origBuildVcsUrl := BuildVcsUrl
	origBuildVcsId := BuildVcsId
	origBuildVcsIdDate := BuildVcsIdDate
	origGoOs, origGoArch := BuildGoOs, BuildGoArch

	// Restore after test
	defer func() {
		BuildProject = origProject
		SemVerMajor, SemVerMinor, SemVerPatch = origMajor, origMinor, origPatch
		SemVerPrerelease, SemVerBuild = origPrerelease, origBuild
		BuildDate = origBuildDate
		BuildVcsUrl = origBuildVcsUrl
		BuildVcsId = origBuildVcsId
		BuildVcsIdDate = origBuildVcsIdDate
		BuildGoOs, BuildGoArch = origGoOs, origGoArch
	}()

	// Set test values
	BuildProject = "stackit-cpi"
	SemVerMajor, SemVerMinor, SemVerPatch = "1", "0", "0"
	SemVerPrerelease = "beta"
	SemVerBuild = "123"
	BuildGoOs = "linux"
	BuildGoArch = "amd64"
	BuildDate = "2023-04-15T14:30:45Z"
	BuildVcsUrl = "https://github.com/stackitcloud/bosh-stackit-cpi-release"
	BuildVcsId = "abc123"
	BuildVcsIdDate = "2023-04-14T10:20:30Z"

	cpi := &CPI{}
	result := cpi.Version()

	// Check that each expected line is in the result
	expectedLines := []string{
		`"Project":"stackit-cpi"`,
		`"Version":"1.0.0-beta+linux.amd64.123"`,
		`"Build Date":"2023-04-15T14:30:45Z"`,
		`"VCS Url":"https://github.com/stackitcloud/bosh-stackit-cpi-release"`,
		`"VCS Id":"abc123"`,
		`"VCS Id Date":"2023-04-14T10:20:30Z"`,
	}

	for _, line := range expectedLines {
		if !strings.Contains(result, line) {
			t.Errorf("CPI.Version() output missing expected line: %s", line)
		}
	}

	if !strings.Contains(result, `"Go Version":"go`) {
		t.Errorf("CPI.Version() output missing Go version line")
	}
}
