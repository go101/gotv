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

	kind_Default

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
	kind          versionKind
	version       string
	forceSyncRepo bool
}

func (tv toolchainVersion) IsInvalid() (bool, string) {
	if tv.kind == kind_Invalid {
		return true, tv.version
	} else if tv.kind == kind_Default {
		return true, "bad default version use scenario"
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

	forceSyncRepo := strings.HasSuffix(arg, "!")
	if forceSyncRepo {
		arg = strings.TrimSpace(arg[:len(arg)-1])

		if len(arg) == 0 {
			return toolchainVersion{kind_Invalid, "! should not be used solely as an argument", true}
		}
	}

	if arg == "." {
		return toolchainVersion{kind_Release, arg, forceSyncRepo}
	}
	if c := arg[0]; '0' <= c && c <= '9' {
		return toolchainVersion{kind_Release, trimTaillingDotZeros(arg), forceSyncRepo}
	}

	i := strings.IndexByte(arg, ':')
	if i < 0 {
		if forceSyncRepo {
			return toolchainVersion{kind_Invalid, "unrecognized command or invalid version: " + arg, forceSyncRepo}
		}
		// View arg as a go command.
		return toolchainVersion{kind_Default, "", false} // kind_Default and forceSyncRepo always conflicts.
	}

	kind, version := arg[:i], arg[i+1:]

	if len(version) == 0 {
		return toolchainVersion{kind_Invalid, "unspecified version for kind (" + kind + ")", forceSyncRepo}
	}

	switch kind {
	default:
		return toolchainVersion{kind_Invalid, "undetermined version kind: " + kind, forceSyncRepo}
	case "tag":
		return toolchainVersion{kind_Tag, version, forceSyncRepo}
	case "bra":
		return toolchainVersion{kind_Branch, version, forceSyncRepo}
	case "rev":
		return toolchainVersion{kind_Revision, version, forceSyncRepo}
	case "": // alias verisons
	}

	// validate alias names (roughly)

	if version != "tip" {
		if c := version[0]; c < '0' || c > '9' {
			return toolchainVersion{kind_Invalid, "an alias version must be tip or a go version", forceSyncRepo}
		}
		version = trimTaillingDotZeros(version)
	}

	return toolchainVersion{kind_Alias, version, forceSyncRepo}
}

func parseGoToolchainVersions(versions ...string) ([]toolchainVersion, error) {
	var tvs = make([]toolchainVersion, len(versions))
	for i, version := range versions {
		tvs[i] = parseGoToolchainVersion(version)
		if invalid, message := tvs[i].IsInvalid(); invalid {
			return nil, errors.New(message)
		}
	}
	makeAtMostOneForceSyncRepo(tvs)
	return tvs, nil
}

func makeAtMostOneForceSyncRepo(tvs []toolchainVersion) {
	var forceSyncRepo = false
	for i := range tvs {
		if tvs[i].forceSyncRepo {
			if forceSyncRepo {
				tvs[i].forceSyncRepo = false
			} else {
				forceSyncRepo = true
				tvs[0].forceSyncRepo = true
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

// Return nil for failed to find the steps.
// The input tv should have been normalized,
// so tv.kind may be only tag/brranch/revision.
// Returning an invalid tv means using system Go toolchain installation.
func determineBootstrapToolchainVersion(tv toolchainVersion) *toolchainVersion {
	switch tv.kind {
	case kind_Tag:
		// Now versions <= 1.5.n are not supported.
		// Version 1.6 - 1.12.n are built with Go 1.15.15.
		// Versions 1.13 - 1.21.n are built with Go 1.17.13.
		// Higher versions are built with Go 1.20.n.
		return &toolchainVersion{kind: kind_Invalid}
	case kind_Branch:
		// ToDo: convert to latest tag for this branch.
		return &toolchainVersion{kind: kind_Invalid}
	case kind_Revision:
		// ToDo: read all revisions and find the closest tag, then ...

		// Try to use local toolchain installation.
		return &toolchainVersion{kind: kind_Invalid}
	}

	return nil
}
