package card

import "testing"

func TestParseFeatures(t *testing.T) {
	tests := []struct {
		name     string
		title    string
		desc     string
		expected Features
	}{
		// ---- Autograph ----
		{
			name:  "Autograph detected",
			title: "Signed Messi Rookie Card",
			desc:  "",
			expected: Features{
				Autograph: true,
				Patch:     false,
				Numbering: Numbering{IsNumbered: false},
				Grading:   Grading{Graded: false},
			},
		},
		{
			name:  "Autograph negative",
			title: "Unsigned Ronaldo Card",
			desc:  "",
			expected: Features{
				Autograph: false,
				Patch:     false,
				Numbering: Numbering{IsNumbered: false},
				Grading:   Grading{Graded: false},
			},
		},

		// ---- Patch ----
		{
			name:  "Patch detected",
			title: "Patch Card Neymar",
			desc:  "",
			expected: Features{
				Autograph: false,
				Patch:     true,
				Numbering: Numbering{IsNumbered: false},
				Grading:   Grading{Graded: false},
			},
		},
		{
			name:  "No Patch negative",
			title: "Base Card - No Patch",
			desc:  "",
			expected: Features{
				Autograph: false,
				Patch:     false,
				Numbering: Numbering{IsNumbered: false},
				Grading:   Grading{Graded: false},
			},
		},

		// ---- Numbering ----
		{
			name:  "Numbered /50",
			title: "Auto /50 Messi",
			desc:  "",
			expected: Features{
				Autograph: true,
				Patch:     false,
				Numbering: Numbering{IsNumbered: true, NumberedTo: 50},
				Grading:   Grading{Graded: false},
			},
		},
		{
			name:  "Numbered 1 of 25",
			title: "Patch Card 1 of 25",
			desc:  "",
			expected: Features{
				Autograph: false,
				Patch:     true,
				Numbering: Numbering{IsNumbered: true, NumberedTo: 25},
				Grading:   Grading{Graded: false},
			},
		},

		// ---- Grading ----
		{
			name:  "PSA 10 grading",
			title: "Messi Rookie PSA 10",
			desc:  "",
			expected: Features{
				Autograph: false,
				Patch:     false,
				Numbering: Numbering{IsNumbered: false},
				Grading:   Grading{Graded: true, GradingCompany: "PSA", Grade: "10"},
			},
		},
		{
			name:  "BGS 9.5 grading",
			title: "Ronaldo BGS 9.5 Auto",
			desc:  "",
			expected: Features{
				Autograph: true,
				Patch:     false,
				Numbering: Numbering{IsNumbered: false},
				Grading:   Grading{Graded: true, GradingCompany: "BGS", Grade: "9.5"},
			},
		},
		{
			name:  "SGC 10 grading with patch",
			title: "Patch Card SGC 10",
			desc:  "",
			expected: Features{
				Autograph: false,
				Patch:     true,
				Numbering: Numbering{IsNumbered: false},
				Grading:   Grading{Graded: true, GradingCompany: "SGC", Grade: "10"},
			},
		},

		// ---- Combined features ----
		{
			name:  "All features combined",
			title: "Kylian Mbappé Auto Patch /25 PSA 10",
			desc:  "",
			expected: Features{
				Autograph: true,
				Patch:     true,
				Numbering: Numbering{IsNumbered: true, NumberedTo: 25},
				Grading:   Grading{Graded: true, GradingCompany: "PSA", Grade: "10"},
			},
		},

		// ---- Unicode / formatting ----
		{
			name:  "Unicode and apostrophes",
			title: "O’Brien Auto Patch 1 of 10",
			desc:  "",
			expected: Features{
				Autograph: true,
				Patch:     true,
				Numbering: Numbering{IsNumbered: true, NumberedTo: 10},
				Grading:   Grading{Graded: false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseFeatures(tt.title, tt.desc)
			if got.Autograph != tt.expected.Autograph {
				t.Errorf("Autograph = %v, expected %v", got.Autograph, tt.expected.Autograph)
			}
			if got.Patch != tt.expected.Patch {
				t.Errorf("Patch = %v, expected %v", got.Patch, tt.expected.Patch)
			}
			if got.Numbering.IsNumbered != tt.expected.Numbering.IsNumbered ||
				got.Numbering.NumberedTo != tt.expected.Numbering.NumberedTo {
				t.Errorf("Numbering = %+v, expected %+v", got.Numbering, tt.expected.Numbering)
			}
			if got.Grading.Graded != tt.expected.Grading.Graded ||
				got.Grading.GradingCompany != tt.expected.Grading.GradingCompany ||
				got.Grading.Grade != tt.expected.Grading.Grade {
				t.Errorf("Grading = %+v, expected %+v", got.Grading, tt.expected.Grading)
			}
		})
	}
}
