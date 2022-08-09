package main

import (
	"sort"
	"strconv"
	"strings"
)

type versionKind int

// Note: git allows a branch name collides with a revision hash or a tag name.
//       Doing so will cause ambiguousness in "git checkout such-a-branch-name".

const (
	kind_Invalid versionKind = iota // must be 0

	kind_Tag
	kind_Branch
	kind_Revision

	// 1         (<=> name:go1)
	// 1.16.3    (<=> name:go1.16.3)
	// 1.17      (<=> name:go1.17)
	// 1.18rc2   (<=> name:go1.18rc2)
	// 1.19beta1 (<=> name:go1.19beta1)
	kind_Release
	// ToDo:
	// 1.        (<=> name:go1.Latest.Latest)
	// 1.18.     (<=> name:go1.18.Latest)

	// :tip  (<=> name:master)
	// :1.18 (<=> name:release-branch.go1.18)
	kind_Alias
)

type toolchainVersion struct {
	kind    versionKind
	version string
}

func (tv toolchainVersion) IsInvalid() (bool, string) {
	if tv.kind == kind_Invalid {
		return true, tv.version
	}
	return false, ""
}

func (tv toolchainVersion) String() string {
	switch tv.kind {
	case kind_Tag:
		return "tag:" + tv.version
	case kind_Branch:
		return "bra:" + tv.version
	case kind_Revision:
		return "rev:" + tv.version
	case kind_Release:
		return "" + tv.version
	case kind_Alias:
		return ":" + tv.version
	}

	return "" // invalid
}

func (tv toolchainVersion) folderName() string {
	var folder string
	switch tv.kind {
	default:
		panic("bad version: " + tv.String())
	case kind_Tag:
		folder = "tag_" + tv.version
	case kind_Branch:
		folder = "bra_" + tv.version
	case kind_Revision:
		folder = "rev_" + tv.version
	}

	return folder
}

// The second result means "consumed".
func parseGoToolchainVersion(arg string) toolchainVersion {
	arg = strings.TrimSpace(arg)
	if len(arg) == 0 {
		return toolchainVersion{kind_Invalid, "version is unspecified"}
	}

	if c := arg[0]; '0' <= c && c <= '9' {
		return toolchainVersion{kind_Release, trimTaillingDotZeros(arg)}
	}

	i := strings.IndexByte(arg, ':')
	if i < 0 {
		return toolchainVersion{kind_Invalid, "invalid version: " + arg}
	}

	kind, version := arg[:i], arg[i+1:]

	if len(version) == 0 {
		return toolchainVersion{kind_Invalid, "unspecified version for kind (" + kind + ")"}
	}

	switch kind {
	default:
		return toolchainVersion{kind_Invalid, "undetermined version kind: " + kind}
	case "tag":
		return toolchainVersion{kind_Tag, version}
	case "bra":
		return toolchainVersion{kind_Branch, version}
	case "rev":
		return toolchainVersion{kind_Revision, version}
	case "": // alias verisons
	}

	// validate alias names (roughly)

	if version != "tip" {
		if c := version[0]; c < '0' || c > '9' {
			return toolchainVersion{kind_Invalid, "an alias version must be tip or a go version"}
		}
		version = trimTaillingDotZeros(version)
	}

	return toolchainVersion{kind_Alias, version}
}

func trimTaillingDotZeros(version string) string {
	for strings.HasSuffix(version, ".0") {
		version = version[:len(version)-2]
	}
	return version
}

func sortVersions(versions []string) {
	sort.Slice(versions, func(a, b int) bool {
		return compareVersions(versions[a], versions[b])
	})
}

func indexNonDigits(str string) int {
	for i, b := range str {
		if b < '0' || b > '9' {
			return i
		}
	}
	return len(str)
}

func compareVersions(x, y string) bool {
	xs, ys := strings.SplitN(x, ".", -1), strings.SplitN(y, ".", -1)
	var true, false = true, false
	if len(xs) > len(ys) {
		xs, ys = ys, xs
		true, false = false, true
	}
	for i, r := range xs {
		s := ys[i]
		rk := indexNonDigits(r)
		sk := indexNonDigits(s)
		rn, _ := strconv.Atoi(r[:rk])
		sn, _ := strconv.Atoi(s[:sk])
		if rn < sn {
			return true
		}
		if rn > sn {
			return false
		}
		if len(r) == rk {
			if len(s) == sk {
				continue
			}
			return false
		} else if len(s) == sk {
			return true
		} else {
			return r[rk:] <= s[sk:]
		}
	}

	return true
}
