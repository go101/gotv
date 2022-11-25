package main

import (
	"errors"
	"fmt"
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

	// 1         (<=> tag:go1)
	// 1.16.3    (<=> tag:go1.16.3)
	// 1.17      (<=> tag:go1.17)
	// 1.18rc2   (<=> tag:go1.18rc2)
	// 1.19beta1 (<=> tag:go1.19beta1)
	kind_Release

	// :tip  (<=> bra:master)
	// :1.18 (<=> bra:release-branch.go1.18)
	kind_Alias
)

type toolchainVersion struct {
	kind       versionKind
	version    string
	questioned bool
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

	return fmt.Sprintf("{%v, %v}", tv.kind, tv.version) // invalid
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

func parseGoToolchainVersion(arg string) toolchainVersion {
	arg = strings.TrimSpace(arg)
	if len(arg) == 0 {
		return toolchainVersion{kind_Invalid, "version is unspecified", false}
	}

	questioned := strings.HasSuffix(arg, "!")
	if questioned {
		arg = arg[:len(arg)-1]
	}

	if arg == "." {
		return toolchainVersion{kind_Release, arg, questioned}
	}
	if c := arg[0]; '0' <= c && c <= '9' {
		return toolchainVersion{kind_Release, trimTaillingDotZeros(arg), questioned}
	}

	i := strings.IndexByte(arg, ':')
	if i < 0 {
		return toolchainVersion{kind_Invalid, "unrecognized command or invalid version: " + arg, questioned}
	}

	kind, version := arg[:i], arg[i+1:]

	if len(version) == 0 {
		return toolchainVersion{kind_Invalid, "unspecified version for kind (" + kind + ")", questioned}
	}

	switch kind {
	default:
		return toolchainVersion{kind_Invalid, "undetermined version kind: " + kind, questioned}
	case "tag":
		return toolchainVersion{kind_Tag, version, questioned}
	case "bra":
		return toolchainVersion{kind_Branch, version, questioned}
	case "rev":
		return toolchainVersion{kind_Revision, version, questioned}
	case "": // alias verisons
	}

	// validate alias names (roughly)

	if version != "tip" {
		if c := version[0]; c < '0' || c > '9' {
			return toolchainVersion{kind_Invalid, "an alias version must be tip or a go version", questioned}
		}
		version = trimTaillingDotZeros(version)
	}

	return toolchainVersion{kind_Alias, version, questioned}
}

func parseGoToolchainVersions(versions ...string) ([]toolchainVersion, error) {
	var tvs = make([]toolchainVersion, len(versions))
	for i, version := range versions {
		tvs[i] = parseGoToolchainVersion(version)
		if invalid, message := tvs[i].IsInvalid(); invalid {
			return nil, errors.New(message)
		}
	}
	makeAtMostOneQuestioned(tvs)
	return tvs, nil
}

func makeAtMostOneQuestioned(tvs []toolchainVersion) {
	var questioned = false
	for i := range tvs {
		if tvs[i].questioned {
			if questioned {
				tvs[i].questioned = false
			} else {
				questioned = true
				tvs[0].questioned = true
			}
		}
	}
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

// ToDo:
// Return nil for failed to find the steps.
func determineBootstrapToolchainVersion(tv toolchainVersion) *toolchainVersion {

	// Now versions <= 1.5.n are not supported.
	// Version 1.6 - 1.12.n are built with Go 1.15.15.
	// Versions 1.13 - 1.21.n are built with Go 1.17.13.
	// Higher versions are built with Go 1.20.n.

	// If the GOROOT_BOOTSTRAP env var is set, use it.

	// If no cached versions yet, use system go toolchain.
	// The system go toolchain is presented as an invalid version.

	return &toolchainVersion{kind: kind_Invalid}
}
