package aptcacher

import "testing"

type kv struct {
	key   string
	value string
}

var (
	testMatrix = map[string]kv{
		"Origin: Ubuntu": {"Origin", "Ubuntu"},
		"MD5Sum:":        {"Md5Sum", ""},
	}
)

func TestSplitSingleLineField(t *testing.T) {
	t.Parallel()
	key, value, _ := parseDCFField("Origin: Ubuntu")
	if key != "Origin" || value != "Ubuntu" {
		t.Error(`Failed to split a single line field`)
	}
}

func TestSplitMultipleLineField(t *testing.T) {
	t.Parallel()
	key, value, _ := parseDCFField("MD5Sum:")
	if key != "MD5Sum" || value != "" {
		t.Error(`Failed to split a multiple line field`)
	}
}

func TestSplitContinuationLineField(t *testing.T) {
	t.Parallel()
	key, value, _ := parseDCFField(" test")
	if key != "" || value != "test" {
		t.Error(`Failed to split a continuation line field`)
	}
}

func TestParseDCF(t *testing.T) {
	t.Parallel()
	dcf := `Origin: Ubuntu
Label: Ubuntu
Suite: trusty
Version: 14.04
Codename: trusty
Date: Thu, 08 May 2014 14:19:09 UTC
Architectures: amd64 arm64 armhf i386 powerpc ppc64el
Components: main restricted universe multiverse
Description: Ubuntu Trusty 14.04
MD5Sum:
 ead1cbf42ed119c50bf3aab28b5b6351          8234934 main/binary-amd64/Packages
 52d605b4217be64f461751f233dd9a8f               96 main/binary-amd64/Release`
	md5sums := []string{
		"ead1cbf42ed119c50bf3aab28b5b6351          8234934 main/binary-amd64/Packages",
		"52d605b4217be64f461751f233dd9a8f               96 main/binary-amd64/Release",
	}

	release, _ := parseDCF(dcf)
	if release["Label"][0] != "Ubuntu" || len(release["MD5Sum"]) != len(md5sums) || release["MD5Sum"][0] != md5sums[0] || release["MD5Sum"][1] != md5sums[1] {
		t.Error(release)
	}
}
