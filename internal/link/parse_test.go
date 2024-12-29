package link

import (
	"slices"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name      string
		inputURL  string
		inputBody string
		expected  []string
	}{
		{
			name:     "absolute and relative URLs",
			inputURL: "https://starwars.com",
			inputBody: `
<html>
    <body>
        <a href="/characters/luke-skywalker">
            <span>Luke Skywalker</span>
        </a>
        <a href="https://starwars.com/planets/tatooine">
            <span>Tatooine</span>
        </a>
    </body>
</html>
`,
			expected: []string{"https://starwars.com/characters/luke-skywalker", "https://starwars.com/planets/tatooine"},
		},
		{
			name:     "empty body",
			inputURL: "https://starwars.com",
			inputBody: `
<html>
    <body>
    </body>
</html>
`,
			expected: []string{},
		},
		{
			name:     "invalid HTML",
			inputURL: "https://starwars.com",
			inputBody: `
<html>
    <body>
        <a href="/spaceships/millennium-falcon">
        <a href="https://starwars.com/characters/darth-vader">
    </body>
</html>
`,
			expected: []string{"https://starwars.com/spaceships/millennium-falcon", "https://starwars.com/characters/darth-vader"},
		},
		{
			name:     "no links",
			inputURL: "https://starwars.com",
			inputBody: `
<html>
    <body>
        <p>No links here!</p>
    </body>
</html>
`,
			expected: []string{},
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := Parse(tt.inputBody, tt.inputURL)
			if err != nil {
				t.Errorf("Test %v - '%s' FAIL: unexpected error: %v", i, tt.name, err)
				return
			}

			if !slices.Equal(actual, tt.expected) {
				t.Errorf("Test %v - %s FAIL: expected URL: %v, actual: %v", i, tt.name, tt.expected, actual)
			}
		})
	}
}
