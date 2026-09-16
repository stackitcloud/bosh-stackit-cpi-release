package cpi

import (
	"encoding/json"
	"runtime"
	"strings"
	"time"
)

var (
	SemVerMajor      string
	SemVerMinor      string
	SemVerPatch      string
	SemVerPrerelease string
	SemVerBuild      string
	BuildProject     string
	BuildDate        string
	BuildVcsUrl      string
	BuildVcsId       string
	BuildVcsIdDate   string
	BuildGoArch      string
	BuildGoOs        string
)

func createBuildMeta(buildOs, buildArch, build string) string {
	p1 := strings.TrimSpace(buildOs)
	p2 := strings.TrimSpace(buildArch)
	p3 := strings.TrimSpace(build)

	// Use default values if they're empty (for testing)
	if p1 == "" {
		p1 = runtime.GOOS
	}

	if p2 == "" {
		p2 = runtime.GOARCH
	}

	b := strings.Join([]string{p1, p2}, ".")
	if p3 != "" {
		b += "." + p3
	}
	return b
}

func createSemVer(major, minor, patch, prerelease, build string) string {
	p1 := strings.TrimSpace(major)
	p2 := strings.TrimSpace(minor)
	p3 := strings.TrimSpace(patch)
	p4 := strings.TrimSpace(prerelease)
	p5 := strings.TrimSpace(build)

	// Use default values if they're empty (for testing)
	if p1 == "" {
		p1 = "0"
	}

	if p2 == "" {
		p2 = "0"
	}

	if p3 == "" {
		p3 = "0"
	}

	sv := strings.Join([]string{p1, p2, p3}, ".")
	if p4 != "" {
		sv += "-" + p4
	}
	if p5 != "" {
		sv += "+" + p5
	}
	return sv
}

// Version returns the version information of the application.
// It includes the version string, build date, and Go runtime version.

func getCpiVersion() string {
	bm := createBuildMeta(BuildGoOs, BuildGoArch, SemVerBuild)
	sv := createSemVer(SemVerMajor, SemVerMinor, SemVerPatch, SemVerPrerelease, bm)
	return sv
}

func getCpiGoVersion() string {
	return runtime.Version()
}

// NOTE: From golang docs
//       RFC3339, RFC822, RFC822Z, RFC1123, and RFC1123Z are useful for formatting;
//       when used with time.Parse they do not accept all the time formats permitted
//       by the RFCs and they do accept time formats not formally defined.
//
//       So used a custom format that should support ISO8601 formatting where it matters
//       and used RFC3339 where it does not.
//

func getCpiZuluTimeStamp(tss string) string {
	// If the timestamp is empty, use current time (for testing)
	if tss == "" {
		return time.Now().UTC().Format(time.RFC3339)
	}

	zuluFormat := "2006-01-02T15:04:05Z07:00"
	ts, err := time.Parse(zuluFormat, tss)
	if err != nil {
		// Use current time if parsing fails
		return time.Now().UTC().Format(time.RFC3339)
	}
	return ts.Format(zuluFormat)
}

func (c *CPI) Version() string {
	versionData := map[string]string{
		"Project":     BuildProject,
		"Version":     getCpiVersion(),
		"Build Date":  getCpiZuluTimeStamp(BuildDate),
		"VCS Url":     BuildVcsUrl,
		"VCS Id":      BuildVcsId,
		"VCS Id Date": getCpiZuluTimeStamp(BuildVcsIdDate),
		"Go Version":  getCpiGoVersion(),
	}
	bytes, _ := json.Marshal(versionData)

	return string(bytes)
}
