package link

import (
	"testing"
)

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		name     string
		inputURL string
		expected string
		wantErr  bool
	}{
		{
			name:     "Standardize protocol",
			inputURL: "HTTP://starwars.fandom.com",
			expected: "starwars.fandom.com",
			wantErr:  false,
		},
		{
			name:     "Remove trailing slash",
			inputURL: "https://starwars.fandom.com/",
			expected: "starwars.fandom.com",
			wantErr:  false,
		},
		{
			name:     "Handle uppercase characters",
			inputURL: "https://StarWars.fandom.com",
			expected: "starwars.fandom.com",
			wantErr:  false,
		},
		{
			name:     "Normalize paths",
			inputURL: "https://starwars.fandom.com/wiki/../wiki/Darth_Vader",
			expected: "starwars.fandom.com/wiki/Darth_Vader",
			wantErr:  false,
		},
		{
			name:     "Remove query parameters",
			inputURL: "https://starwars.fandom.com/wiki/Darth_Vader?b=2&a=1",
			expected: "starwars.fandom.com/wiki/Darth_Vader",
			wantErr:  false,
		},
		{
			name:     "Remove hash fragments",
			inputURL: "https://starwars.fandom.com/wiki/Darth_Vader#Biography",
			expected: "starwars.fandom.com/wiki/Darth_Vader",
			wantErr:  false,
		},
		{
			name:     "Invalid URL",
			inputURL: "ht@tp://invalid-url",
			expected: "",
			wantErr:  true,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, _, err := NormalizeURL(tt.inputURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("Test %v - '%s' FAIL: unexpected error: %v", i, tt.name, err)
				return
			}
			if actual != tt.expected {
				t.Errorf("Test %v - %s FAIL: expected URL: %v, actual: %v", i, tt.name, tt.expected, actual)
			}
		})
	}
}
