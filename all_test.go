package main

import (
	"math/rand"
	"os"
	"testing"
	"time"
)

var gotvForTesting gotv

func init() {
	cacheDir, err := os.MkdirTemp("", "gotv-cache-*")
	if err != nil {
		panic(err)
	}

	gotvForTesting, err = bornWithCacheDir(cacheDir)
	if err != nil {
		panic(err)
	}

	rand.Seed(time.Now().UnixNano())
}

func Test_sortVersions(t *testing.T) {
	var sorted = []string{
		"1",
		"1.1",
		"1.1.1",
		"1.5",
		"1.10",
		"1.11beta1",
		"1.11beta2",
		"1.11rc1",
		"1.11.2",
		"1.11.11",
	}

	for range [100]struct{}{} {
		var vs = append([]string{}, sorted...)
		for range [32]struct{}{} {
			var i, j = rand.Intn(len(vs)), rand.Intn(len(vs))
			vs[i], vs[j] = vs[j], vs[i]
		}
		//t.Log(vs)
		var vs2 = append([]string{}, vs...)
		sortVersions(vs2)
		for i := range sorted {
			if vs2[i] != sorted[i] {
				t.Errorf("wrong sorting result:\n  for %v\n  get %v\n", vs, vs2)
			}
		}
	}
}

func Test_parseGoToolchainVersion(t *testing.T) {
	var cases = []struct {
		v  string
		tv toolchainVersion
	}{
		{"1.19", toolchainVersion{kind_Release, "1.19", false}},
		{"1.19!", toolchainVersion{kind_Release, "1.19", true}},
		{":1.19", toolchainVersion{kind_Alias, "1.19", false}},
		{":1.19!", toolchainVersion{kind_Alias, "1.19", true}},
		{":tip", toolchainVersion{kind_Alias, "tip", false}},
		{":tip!", toolchainVersion{kind_Alias, "tip", true}},
		{"bra:1.19", toolchainVersion{kind_Branch, "1.19", false}},
		{"tag:1.19", toolchainVersion{kind_Tag, "1.19", false}},
		{"rev:12ab", toolchainVersion{kind_Revision, "12ab", false}},
	}

	for _, c := range cases {
		if r := parseGoToolchainVersion(c.v); r != c.tv {
			t.Errorf(`parseGoToolchainVersion("%s") != %v, but %v`, c.v, c.tv, r)
		}
	}
}
