package gh

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// MinGHVersion is the lowest gh CLI version aitinerary supports.
const MinGHVersion = "2.0.0"

var ghVersionRe = regexp.MustCompile(`(\d+)\.(\d+)\.(\d+)`)

func parseGHVersion(out string) string {
	return ghVersionRe.FindString(out)
}

func ghVersionAtLeast(have, min string) bool {
	if strings.TrimSpace(have) == "" {
		return true
	}
	return compareDottedVersions(have, min) >= 0
}

func compareDottedVersions(a, b string) int {
	as := strings.Split(a, ".")
	bs := strings.Split(b, ".")
	n := len(as)
	if len(bs) > n {
		n = len(bs)
	}
	for i := 0; i < n; i++ {
		ai, bi := 0, 0
		if i < len(as) {
			ai, _ = strconv.Atoi(as[i])
		}
		if i < len(bs) {
			bi, _ = strconv.Atoi(bs[i])
		}
		switch {
		case ai < bi:
			return -1
		case ai > bi:
			return 1
		}
	}
	return 0
}

func checkGHVersion() error {
	out, err := exec.Command("gh", "--version").CombinedOutput()
	if err != nil {
		return nil
	}
	have := parseGHVersion(string(out))
	if !ghVersionAtLeast(have, MinGHVersion) {
		return fmt.Errorf("gh %s is older than the required %s; upgrade (e.g. `brew upgrade gh`) or reinstall from https://cli.github.com", have, MinGHVersion)
	}
	return nil
}
