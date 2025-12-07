package card

import (
	"card-collectors-uk/database"
	"errors"
	"fmt"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

var (
	reYear = regexp.MustCompile(`(?i)(\d{4})(?:[/\-](\d{2,4}))?`)
)

type CanonicalCard struct {
	StartYear int32
	EndYear   int32
	Set       string
	Player    string
	Features  Features
}

func NewCanonicalCardFromListing(title, description string, allSets []database.SetRow, allPlayers []database.PlayerRow) (CanonicalCard, error) {
	var card CanonicalCard
	text := fmt.Sprintf("%s %s", title, description)

	if matches := reYear.FindStringSubmatch(text); len(matches) >= 2 {
		// matches[1] is always the first year
		startYear, err := strconv.Atoi(matches[1])
		if err == nil {
			card.StartYear = int32(startYear)
		}

		card.EndYear = int32(startYear) + 1
	}

	var found bool
	card.Set, found = detectSet(text, allSets)
	if !found {
		return CanonicalCard{}, errors.New("failed to detect card set")
	}

	card.Player, found = detectPlayer(text, allPlayers)
	if !found {
		return CanonicalCard{}, errors.New("failed to detect card player")
	}

	card.Features = ParseFeatures(title, description)

	return card, nil
}

func (c CanonicalCard) Key() string {
	var parts []string

	if c.StartYear != 0 {
		if c.EndYear != 0 {
			parts = append(parts, fmt.Sprintf("%d/%d", c.StartYear, c.EndYear))
		} else {
			parts = append(parts, fmt.Sprintf("%d", c.StartYear))
		}
	}

	if c.Set != "" {
		parts = append(parts, normaliseString(c.Set))
	}

	if c.Player != "" {
		parts = append(parts, normaliseString(c.Player))
	}

	featureStr := c.Features.BuildFeatureString()
	if len(featureStr) > 0 {
		parts = append(parts, featureStr)
	}

	numberingStr := c.Features.BuildNumberString()
	if len(numberingStr) > 0 {
		parts = append(parts, numberingStr)
	}

	return strings.Join(parts, "-")
}

func ParseKey(key string) (CanonicalCard, error) {
	var c CanonicalCard

	parts := strings.Split(key, "-")
	if len(parts) < 2 {
		return c, errors.New("invalid key, not enough parts")
	}

	yearPart := parts[0]
	yearParts := strings.Split(yearPart, "/")

	if len(yearParts) == 1 {
		start, err := strconv.Atoi(yearParts[0])
		if err != nil {
			return c, fmt.Errorf("invalid start year: %w", err)
		}
		c.StartYear = int32(start)
	} else if len(yearParts) == 2 {
		start, err := strconv.Atoi(yearParts[0])
		if err != nil {
			return c, fmt.Errorf("invalid start year: %w", err)
		}
		c.StartYear = int32(start)

		end, err := strconv.Atoi(yearParts[1])
		if err != nil {
			return c, fmt.Errorf("invalid end year: %w", err)
		}
		c.EndYear = int32(end)
	} else {
		return c, errors.New("invalid year format")
	}

	i := 1

	if i >= len(parts) {
		return c, errors.New("missing set")
	}
	c.Set = parts[i]
	i++

	if i >= len(parts) {
		return c, errors.New("missing player")
	}
	c.Player = parts[i]
	i++

	if i < len(parts) {
		if err := c.Features.ParseFeatureString(parts[i]); err == nil {
			i++
		}
	}

	if i < len(parts) {
		if err := c.Features.ParseNumberString(parts[i]); err == nil {
			i++
		}
	}

	return c, nil
}

func detectSet(text string, sets []database.SetRow) (string, bool) {
	normalisedText := normaliseString(text)
	var foundSet string
	var partsMatched int
	var mostPartsMatched int

	for _, s := range sets {
		var parts []string
		parts = append(parts, strings.Split(s.CompanyName, " ")...)
		parts = append(parts, strings.Split(s.SetName, " ")...)
		parts = append(parts, strings.Split(s.SubsetName, " ")...)
		var filteredParts []string
		for _, p := range parts {
			if strings.TrimSpace(p) != "" {
				filteredParts = append(filteredParts, p)
			}
		}

		for _, part := range filteredParts {
			if strings.Contains(normalisedText, normaliseString(part)) {
				partsMatched++
			}
		}

		if partsMatched > mostPartsMatched {
			foundSet = fmt.Sprintf("%s %s", s.CompanyName, s.SetName)
			if len(s.SubsetName) > 0 {
				foundSet = fmt.Sprintf("%s %s", foundSet, s.SubsetName)
			}

			mostPartsMatched = partsMatched
		}

		partsMatched = 0
	}

	return foundSet, len(foundSet) > 0
}

func detectPlayer(text string, players []database.PlayerRow) (string, bool) {
	normalisedText := normaliseString(text)
	var foundPlayer string
	var partsMatched int
	var mostPartsMatched int

	for _, p := range players {
		// Split first and second names into words
		parts := append(strings.Split(p.FirstName, " "), strings.Split(p.SecondName, " ")...)
		var filteredParts []string
		for _, part := range parts {
			if strings.TrimSpace(part) != "" {
				filteredParts = append(filteredParts, part)
			}
		}

		for _, part := range filteredParts {
			if strings.Contains(normalisedText, normaliseString(part)) {
				partsMatched++
			}
		}

		if partsMatched > mostPartsMatched {
			foundPlayer = fmt.Sprintf("%s %s", p.FirstName, p.SecondName)
			mostPartsMatched = partsMatched
		}

		partsMatched = 0
	}

	return foundPlayer, len(foundPlayer) > 0
}

func normaliseString(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "-", " ")
	s = strings.ReplaceAll(s, "_", " ")
	s = strings.ReplaceAll(s, ".", " ")
	s = strings.ReplaceAll(s, ",", " ")
	s = strings.Join(strings.Fields(s), " ")

	s = strings.Map(func(r rune) rune {
		switch r {
		case ' ', '-', '\'', '’':
			return -1
		default:
			return r
		}
	}, s)

	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	result, _, err := transform.String(t, s)
	if err != nil {
		return s // fallback to original if transform fails
	}

	return result
}
