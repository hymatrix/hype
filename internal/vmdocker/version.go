package vmdocker

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
)

var semverTagPattern = regexp.MustCompile(`^v(\d+)\.(\d+)\.(\d+)(?:-([0-9A-Za-z.-]+))?(?:\+([0-9A-Za-z.-]+))?$`)

func LatestSemverTag(lsRemoteOutput string) (string, error) {
	best := ""
	for _, line := range strings.Split(lsRemoteOutput, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		tag := strings.TrimPrefix(parts[len(parts)-1], "refs/tags/")
		if !IsSemverTag(tag) {
			continue
		}
		if best == "" || compareSemverTags(tag, best) > 0 {
			best = tag
		}
	}
	if best == "" {
		return "", errors.New("no semver vmdocker release tags found")
	}
	return best, nil
}

func IsSemverTag(tag string) bool {
	return semverTagPattern.MatchString(strings.TrimSpace(tag))
}

type parsedSemverTag struct {
	major      int
	minor      int
	patch      int
	prerelease string
}

func compareSemverTags(a, b string) int {
	pa := parseSemverTag(a)
	pb := parseSemverTag(b)
	if pa.major != pb.major {
		return compareInt(pa.major, pb.major)
	}
	if pa.minor != pb.minor {
		return compareInt(pa.minor, pb.minor)
	}
	if pa.patch != pb.patch {
		return compareInt(pa.patch, pb.patch)
	}
	switch {
	case pa.prerelease == "" && pb.prerelease != "":
		return 1
	case pa.prerelease != "" && pb.prerelease == "":
		return -1
	default:
		return strings.Compare(pa.prerelease, pb.prerelease)
	}
}

func parseSemverTag(tag string) parsedSemverTag {
	match := semverTagPattern.FindStringSubmatch(strings.TrimSpace(tag))
	if match == nil {
		return parsedSemverTag{}
	}

	major, _ := strconv.Atoi(match[1])
	minor, _ := strconv.Atoi(match[2])
	patch, _ := strconv.Atoi(match[3])
	return parsedSemverTag{
		major:      major,
		minor:      minor,
		patch:      patch,
		prerelease: match[4],
	}
}

func compareInt(a, b int) int {
	switch {
	case a > b:
		return 1
	case a < b:
		return -1
	default:
		return 0
	}
}
