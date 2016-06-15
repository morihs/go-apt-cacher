package aptcacher

import "testing"

func TestParseIndex(t *testing.T) {
	path, hash, _ := parseIndex("ead1cbf42ed119c50bf3aab28b5b6351          8234934 main/binary-amd64/Packages")
	if path != "main/binary-amd64/Packages" || hash != "ead1cbf42ed119c50bf3aab28b5b6351" {
		t.Error(`Failed to parse an index.`)
	}
}
