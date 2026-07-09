// Package match is the ticket-key ↔ merged-PR matching engine (PLAN.md §7
// P4.2). It extracts Jira keys from a PR's branch name, title, and body, and
// scores each match's confidence so the actions engine can decide what is safe
// to auto-apply (high) versus what must go to the review queue (ambiguous).
//
// It depends only on the standard library and the jira.Ticket type, so it is
// fully unit-testable without a network.
package match

import (
	"regexp"
	"sort"
	"strings"

	"github.com/madicen/aitinerary/internal/jira"
)

// Confidence rates how sure we are a PR maps to a ticket.
type Confidence int

const (
	// Ambiguous: the key came only from the title/body, or multiple candidate
	// keys were found — never auto-applied.
	Ambiguous Confidence = iota
	// High: a single key found in the branch name — safe to auto-transition.
	High
)

func (c Confidence) String() string {
	if c == High {
		return "high"
	}
	return "ambiguous"
}

// Source names where a key was found, most-trusted first.
type Source string

const (
	SourceBranch Source = "branch"
	SourceTitle  Source = "title"
	SourceBody   Source = "body"
)

// PR is the minimal merged-PR view the matcher needs (populated by internal/gh
// in P4.1).
type PR struct {
	Number   int
	Repo     string // "owner/name"
	Title    string
	Body     string
	HeadRef  string // branch name
	URL      string
}

// Match links a ticket key to a PR with a confidence and the source that gave
// it that confidence.
type Match struct {
	Key        string
	PR         PR
	Confidence Confidence
	Source     Source
}

// KeyRegex builds a case-insensitive regexp matching any configured project key
// followed by -<digits>, anchored so APP-1 does not match inside APP-11 and a
// key is not matched inside a longer alphanumeric token. Returns nil when no
// project keys are given.
func KeyRegex(projectKeys []string) *regexp.Regexp {
	var alts []string
	for _, k := range projectKeys {
		if k = strings.TrimSpace(k); k != "" {
			alts = append(alts, regexp.QuoteMeta(strings.ToUpper(k)))
		}
	}
	if len(alts) == 0 {
		return nil
	}
	// (?i) case-insensitive. \b word boundaries ensure the key is a standalone
	// token: "MAPP-1" won't match APP, and because \d+ is greedy with a trailing
	// \b, "APP-11" yields the key APP-11 (never the prefix APP-1). Go's RE2 has
	// no lookaround, but \b handles both boundaries without consuming neighbours,
	// so adjacent keys ("APP-1 APP-2") both match.
	pat := `(?i)\b((?:` + strings.Join(alts, "|") + `)-\d+)\b`
	return regexp.MustCompile(pat)
}

// ExtractKeys returns the distinct, upper-cased ticket keys found in text,
// preserving first-seen order.
func ExtractKeys(re *regexp.Regexp, text string) []string {
	if re == nil || text == "" {
		return nil
	}
	seen := map[string]bool{}
	var out []string
	for _, m := range re.FindAllStringSubmatch(text, -1) {
		key := strings.ToUpper(m[1])
		if !seen[key] {
			seen[key] = true
			out = append(out, key)
		}
	}
	return out
}

// MatchPR scores one PR against the set of live ticket keys (those not Done).
// The rule (PLAN.md §7 P4.2):
//
//   - exactly one key in the branch name → High (branch source)
//   - otherwise, keys from branch/title/body are Ambiguous
//
// Only keys present in liveKeys are returned, so a PR referencing an already-Done
// or unknown ticket produces nothing.
func MatchPR(re *regexp.Regexp, pr PR, liveKeys map[string]bool) []Match {
	branchKeys := ExtractKeys(re, pr.HeadRef)
	titleKeys := ExtractKeys(re, pr.Title)
	bodyKeys := ExtractKeys(re, pr.Body)

	// Confidence + source per key: branch beats title beats body; a single
	// branch key is High, everything else Ambiguous.
	type info struct {
		src  Source
		conf Confidence
	}
	best := map[string]info{}

	singleBranch := len(branchKeys) == 1
	record := func(keys []string, src Source, conf Confidence) {
		for _, k := range keys {
			if liveKeys != nil && !liveKeys[k] {
				continue
			}
			// Keep the most-trusted source we've seen for this key.
			if cur, ok := best[k]; ok && sourceRank(cur.src) >= sourceRank(src) {
				continue
			}
			best[k] = info{src: src, conf: conf}
		}
	}
	if singleBranch {
		record(branchKeys, SourceBranch, High)
	} else {
		record(branchKeys, SourceBranch, Ambiguous)
	}
	record(titleKeys, SourceTitle, Ambiguous)
	record(bodyKeys, SourceBody, Ambiguous)

	out := make([]Match, 0, len(best))
	for k, in := range best {
		out = append(out, Match{Key: k, PR: pr, Confidence: in.conf, Source: in.src})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

// MatchPRs scores every PR against the live tickets and returns all matches.
// tickets is the set of not-Done tickets to match against.
func MatchPRs(projectKeys []string, prs []PR, tickets []jira.Ticket) []Match {
	re := KeyRegex(projectKeys)
	live := map[string]bool{}
	for _, t := range tickets {
		if !t.IsDone() {
			live[strings.ToUpper(t.Key)] = true
		}
	}
	var all []Match
	for _, pr := range prs {
		all = append(all, MatchPR(re, pr, live)...)
	}
	return all
}

func sourceRank(s Source) int {
	switch s {
	case SourceBranch:
		return 3
	case SourceTitle:
		return 2
	case SourceBody:
		return 1
	default:
		return 0
	}
}
