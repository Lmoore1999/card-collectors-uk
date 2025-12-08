package card

import (
	"testing"
)

func TestCanonicalCardKey(t *testing.T) {
	tests := []struct {
		name     string
		card     CanonicalCard
		expected string
	}{
		{
			name:     "Empty card returns empty string",
			card:     CanonicalCard{},
			expected: "",
		},
		{
			name: "Only StartYear",
			card: CanonicalCard{
				StartYear: 2022,
			},
			expected: "2022",
		},
		{
			name: "StartYear + EndYear",
			card: CanonicalCard{
				StartYear: 2022,
				EndYear:   2023,
			},
			expected: "2022/2023",
		},
		{
			name: "Set only",
			card: CanonicalCard{
				Set: "Topps Chrome",
			},
			expected: "toppschrome",
		},
		{
			name: "Player only",
			card: CanonicalCard{
				Player: "Lionel Messi",
			},
			expected: "lionelmessi",
		},
		{
			name: "Features only - Autograph",
			card: CanonicalCard{
				Features: Features{Autograph: true},
			},
			expected: "Auto",
		},
		{
			name: "Features only - Patch",
			card: CanonicalCard{
				Features: Features{Patch: true},
			},
			expected: "Patch",
		},
		{
			name: "Features Auto + Patch",
			card: CanonicalCard{
				Features: Features{Autograph: true, Patch: true},
			},
			expected: "AutoPatch",
		},
		{
			name: "Features with numbering",
			card: CanonicalCard{
				Features: Features{Numbering: Numbering{IsNumbered: true, NumberedTo: 50}},
			},
			expected: "50",
		},
		{
			name: "Features with grading only",
			card: CanonicalCard{
				Features: Features{Grading: Grading{Graded: true, GradingCompany: "PSA", Grade: "10"}},
			},
			expected: "PSA10",
		},
		{
			name: "Features Auto + Patch + Grading",
			card: CanonicalCard{
				Features: Features{
					Autograph: true,
					Patch:     true,
					Grading:   Grading{Graded: true, GradingCompany: "PSA", Grade: "10"},
				},
			},
			expected: "AutoPatchPSA10",
		},
		{
			name: "Start + Set + Player + Auto + Patch + Numbering + Grading",
			card: CanonicalCard{
				StartYear: 2022,
				EndYear:   2023,
				Set:       "Impeccable Elegance",
				Player:    "Kylian Mbappé",
				Features: Features{
					Autograph: true,
					Patch:     true,
					Numbering: Numbering{IsNumbered: true, NumberedTo: 25},
					Grading:   Grading{Graded: true, GradingCompany: "PSA", Grade: "10"},
				},
			},
			expected: "2022/2023-impeccableelegance-kylianmbappe-AutoPatchPSA10-25",
		},
		{
			name: "Start + Player + Numbering only",
			card: CanonicalCard{
				StartYear: 2021,
				Player:    "Pedri",
				Features:  Features{Numbering: Numbering{IsNumbered: true, NumberedTo: 10}},
			},
			expected: "2021-pedri-10",
		},
		{
			name: "Unicode + apostrophe + dash removal",
			card: CanonicalCard{
				Set:    "Donruss-Optic",
				Player: "O’Brien",
			},
			expected: "donrussoptic-obrien",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.card.Key()
			if got != tt.expected {
				t.Errorf("Key() = %q, expected %q", got, tt.expected)
			}
		})
	}
}

func TestParseKey(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected CanonicalCard
		wantErr  bool
	}{
		{
			name:  "single year basic",
			input: "2020-Topps-Messi",
			expected: CanonicalCard{
				StartYear: 2020,
				EndYear:   0,
				Set:       "Topps",
				Player:    "Messi",
			},
		},
		{
			name:  "year range",
			input: "2020/2021-Topps-Haaland",
			expected: CanonicalCard{
				StartYear: 2020,
				EndYear:   2021,
				Set:       "Topps",
				Player:    "Haaland",
			},
		},
		{
			name:    "invalid missing parts",
			input:   "2020",
			wantErr: true,
		},
		{
			name:    "invalid year format",
			input:   "2020/2021/2022-Topps-Messi",
			wantErr: true,
		},
		{
			name:    "invalid start year",
			input:   "xx-Topps-Messi",
			wantErr: true,
		},
		{
			name:    "invalid end year",
			input:   "2020/xx-Topps-Messi",
			wantErr: true,
		},
		{
			name:    "missing set",
			input:   "2020-",
			wantErr: true,
		},
		{
			name:    "missing player",
			input:   "2020-Topps",
			wantErr: true,
		},
		{
			name:  "with feature (ignored if cannot parse)",
			input: "2020-Topps-Messi-RC",
			expected: CanonicalCard{
				StartYear: 2020,
				Set:       "Topps",
				Player:    "Messi",
				// Adjust Features if required in your implementation
			},
		},
		{
			name:  "with feature and number",
			input: "2020-Topps-Messi-RC-10",
			expected: CanonicalCard{
				StartYear: 2020,
				Set:       "Topps",
				Player:    "Messi",
				// adjust features/number accordingly if your parsing stores them
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseKey(tt.input)

			if tt.wantErr && err == nil {
				t.Errorf("expected error, got nil")
				return
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if tt.wantErr {
				return
			}

			if got.StartYear != tt.expected.StartYear {
				t.Errorf("StartYear got %d, want %d", got.StartYear, tt.expected.StartYear)
			}
			if got.EndYear != tt.expected.EndYear {
				t.Errorf("EndYear got %d, want %d", got.EndYear, tt.expected.EndYear)
			}
			if got.Set != tt.expected.Set {
				t.Errorf("Set got %s, want %s", got.Set, tt.expected.Set)
			}
			if got.Player != tt.expected.Player {
				t.Errorf("Player got %s, want %s", got.Player, tt.expected.Player)
			}

			// If your Features struct needs comparing, do manual field comparisons here
		})
	}
}
