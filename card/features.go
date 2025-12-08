package card

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	autographKeywords = []string{
		"auto", "autograph", "signed", "signature", "sig",
	}

	patchKeywords = []string{
		"patch", "jersey", "rpa", "relic", "memorabilia", "kit", "shirt",
	}

	negativeAutograph = []string{
		"no auto", "not auto", "non auto", "no autograph", "not autograph",
	}

	negativePatch = []string{
		"no patch", "non patch", "no jersey", "no relic", "not patch",
	}

	gradingCompanies = []string{"psa", "bgs", "sgc"}

	numberedRegex = []*regexp.Regexp{
		// Most specific: "X of Y" or "X / Y"  -> two captures, use matches[2]
		regexp.MustCompile(`(?i)(\d{1,4})\s*(?:of|/)\s*(\d{1,5})\b`),

		// "#15/150" or "# 15 / 150" -> two captures, use last capture
		regexp.MustCompile(`(?i)#\s*(\d{1,4})\s*/\s*(\d{1,5})\b`),

		// "12/250" (leading number optionally captured only for left part but second is captured)
		// This pattern captures only the second number (works for "12/250")
		regexp.MustCompile(`\b\d{1,4}\s*/\s*(\d{1,5})\b`),

		// "numbered to 50" or "serial to 10" -> single capture (last capture is correct)
		regexp.MustCompile(`(?i)(?:numbered|serial).*?\bto\s*(\d{1,5})\b`),

		// Fallback: "/50" (no left number) -> single capture
		regexp.MustCompile(`/\s*(\d{1,5})\b`),
	}

	gradeRegexes = []*regexp.Regexp{
		regexp.MustCompile(`\b(?:psa|bgs|sgc)\s*(\d{1,2}(?:\.5)?)\b`), // e.g., "PSA 10", "BGS 9.5"
	}
)

type Features struct {
	Autograph bool
	Patch     bool
	Numbering Numbering
	Grading   Grading
}

type Numbering struct {
	IsNumbered bool
	NumberedTo int32
}

type Grading struct {
	Graded         bool
	GradingCompany string
	Grade          string
}

func (f Features) BuildFeatureString() string {
	var out string

	if f.Autograph {
		out += "Auto"
	}

	if f.Patch {
		out += "Patch"
	}

	if f.Grading.Graded {
		out += fmt.Sprintf("%s%s", f.Grading.GradingCompany, f.Grading.Grade)
	}

	return out
}

func (f Features) BuildNumberString() string {
	var out string
	if f.Numbering.IsNumbered {
		if f.Numbering.NumberedTo > 0 {
			out += fmt.Sprintf("%d", f.Numbering.NumberedTo)
		}
	}

	return out
}

func ParseFeatures(title, desc string) Features {
	text := fmt.Sprintf("%s %s", strings.ToLower(title), strings.ToLower(desc))

	autograph := detectWithNegatives(text, autographKeywords, negativeAutograph)
	patch := detectWithNegatives(text, patchKeywords, negativePatch)
	numbering := extractNumbering(text)
	grading := extractGrading(text)

	return Features{
		Autograph: autograph,
		Patch:     patch,
		Numbering: numbering,
		Grading:   grading,
	}
}

func (f Features) ParseFeatureString(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}

	// Autograph
	if strings.Contains(strings.ToLower(s), "auto") {
		f.Autograph = true
	}

	// Patch
	if strings.Contains(strings.ToLower(s), "patch") {
		f.Patch = true
	}

	// Grading - look for PSA, BGS, SGC prefixes
	upper := strings.ToUpper(s)
	for _, company := range gradingCompanies {
		c := strings.ToUpper(company)
		if strings.HasPrefix(upper, c) {
			// everything after the prefix is grade value
			grade := strings.TrimPrefix(upper, c)
			if grade != "" {
				f.Grading = Grading{
					Graded:         true,
					GradingCompany: c,
					Grade:          grade,
				}
			}
			break
		}
	}

	return nil
}

func (f Features) ParseNumberString(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}

	n, err := strconv.Atoi(s)
	if err != nil {
		return fmt.Errorf("invalid numbering: %w", err)
	}

	f.Numbering = Numbering{
		IsNumbered: true,
		NumberedTo: int32(n),
	}

	return nil
}

func detectWithNegatives(text string, positives, negatives []string) bool {
	for _, n := range negatives {
		if containsWord(text, n) {
			return false
		}
	}
	for _, p := range positives {
		if containsWord(text, p) {
			return true
		}
	}
	return false
}

func extractNumbering(text string) Numbering {
	for _, re := range numberedRegex {
		matches := re.FindStringSubmatch(text)

		if len(matches) == 3 {
			num := matches[2]
			n, err := strconv.Atoi(num)
			if err == nil {
				return Numbering{
					IsNumbered: true,
					NumberedTo: int32(n),
				}
			}
		}

		if len(matches) == 2 {
			num := matches[1]
			n, err := strconv.Atoi(num)
			if err == nil {
				return Numbering{
					IsNumbered: true,
					NumberedTo: int32(n),
				}
			}
		}
	}

	return Numbering{IsNumbered: false}
}

func extractGrading(text string) Grading {
	text = strings.ToLower(text)
	for _, re := range gradeRegexes {
		matches := re.FindStringSubmatch(text)
		if len(matches) >= 2 {
			company := ""
			for _, c := range gradingCompanies {
				if strings.Contains(text, c) {
					company = strings.ToUpper(c)
					break
				}
			}

			return Grading{
				Graded:         true,
				GradingCompany: company,
				Grade:          matches[1],
			}
		}
	}
	return Grading{Graded: false}
}

func containsWord(text, word string) bool {
	return regexp.MustCompile(`\b` + regexp.QuoteMeta(word) + `\b`).MatchString(text)
}
