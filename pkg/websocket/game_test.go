package websocket

import (
	"strings"
	"testing"
)

func TestNormalizeWord_Basic(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Hello", "hello"},
		{"  WORLD  ", "world"},
		{"GoLang", "golang"},
		{"", ""},
		{"   ", ""},
	}
	for _, tt := range tests {
		got := normalizeWord(tt.input)
		if got != tt.want {
			t.Errorf("normalizeWord(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestNormalizeWord_NonLetterReturnsEmpty(t *testing.T) {
	inputs := []string{"hello world", "test123", "foo-bar", "a.b", "go!"}
	for _, input := range inputs {
		got := normalizeWord(input)
		if got != "" {
			t.Errorf("normalizeWord(%q) = %q, want empty", input, got)
		}
	}
}

func TestCanBuildWord(t *testing.T) {
	tests := []struct {
		word    string
		letters string
		want    bool
	}{
		{"art", "ADVENTURE", true},
		{"trade", "ADVENTURE", true},
		{"xyz", "ADVENTURE", false},
		{"aaa", "ADVENTURE", false},
		{"", "ADVENTURE", true},
		{"a", "", false},
	}
	for _, tt := range tests {
		got := canBuildWord(tt.word, tt.letters)
		if got != tt.want {
			t.Errorf("canBuildWord(%q, %q) = %v, want %v", tt.word, tt.letters, got, tt.want)
		}
	}
}

func TestLetterCounts(t *testing.T) {
	counts := letterCounts("hello")
	if counts['h'] != 1 || counts['e'] != 1 || counts['l'] != 2 || counts['o'] != 1 {
		t.Errorf("letterCounts(\"hello\") = %v, unexpected", counts)
	}
}

func TestLetterCounts_Empty(t *testing.T) {
	counts := letterCounts("")
	if len(counts) != 0 {
		t.Errorf("letterCounts empty should be empty, got %v", counts)
	}
}

func TestLetterCounts_CaseInsensitive(t *testing.T) {
	counts := letterCounts("AaBb")
	if counts['a'] != 2 || counts['b'] != 2 {
		t.Errorf("letterCounts(\"AaBb\") = %v, expected a:2 b:2", counts)
	}
}

func TestScoreWord(t *testing.T) {
	tests := []struct {
		word string
		min  int
	}{
		{"a", 1},
		{"z", 10},
		{"q", 10},
		{"art", 1},
		{"jazz", 1},
		{"hello", 1},
	}
	for _, tt := range tests {
		got := scoreWord(tt.word)
		if got < tt.min {
			t.Errorf("scoreWord(%q) = %d, want >= %d", tt.word, got, tt.min)
		}
	}
}

func TestScoreWord_KnownValues(t *testing.T) {
	if got := scoreWord("art"); got != 3 {
		t.Errorf("scoreWord(\"art\") = %d, want 3", got)
	}
	if got := scoreWord("hello"); got != 8 {
		t.Errorf("scoreWord(\"hello\") = %d, want 8", got)
	}
}

func TestShuffleLetters_SameLength(t *testing.T) {
	input := "adventure"
	result := shuffleLetters(input)
	if len([]rune(result)) != len([]rune(input)) {
		t.Errorf("shuffleLetters(%q) changed length: got %d", input, len([]rune(result)))
	}
}

func TestShuffleLetters_UpperCase(t *testing.T) {
	result := shuffleLetters("hello")
	if result != strings.ToUpper(result) {
		t.Errorf("shuffleLetters should return uppercase, got %q", result)
	}
}

func TestShuffleLetters_SameLetters(t *testing.T) {
	input := "adventure"
	result := shuffleLetters(input)
	origCounts := letterCounts(input)
	resCounts := letterCounts(result)
	for k, v := range origCounts {
		if resCounts[k] != v {
			t.Errorf("shuffleLetters changed letter composition: input=%q result=%q", input, result)
			break
		}
	}
}

func TestShuffleLetters_ShortWord(t *testing.T) {
	result := shuffleLetters("a")
	if result != "A" {
		t.Errorf("shuffleLetters(\"a\") = %q, want \"A\"", result)
	}
}

func TestSpacedLetters(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"ABC", "A B C"},
		{"a", "A"},
		{"hello", "H E L L O"},
		{"", ""},
	}
	for _, tt := range tests {
		got := spacedLetters(tt.input)
		if got != tt.want {
			t.Errorf("spacedLetters(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestBuildWordSet(t *testing.T) {
	words := []string{"apple", "banana", "cherry"}
	set := buildWordSet(words)
	for _, w := range words {
		if _, ok := set[w]; !ok {
			t.Errorf("buildWordSet: missing word %q", w)
		}
	}
	if _, ok := set["nothere"]; ok {
		t.Error("buildWordSet: unexpected word found")
	}
}

func TestBuildWordSet_Empty(t *testing.T) {
	set := buildWordSet(nil)
	if len(set) != 0 {
		t.Errorf("buildWordSet(nil) should be empty, got %d entries", len(set))
	}
}

func TestBuildLetterSources_FiltersLength(t *testing.T) {
	words := []string{"hi", "cat", "adventure", "paintwork", "ab", "developer1"}
	sources := buildLetterSources(words)
	for _, s := range sources {
		l := len([]rune(s))
		if l < 8 || l > 10 {
			t.Errorf("buildLetterSources included word %q with length %d", s, l)
		}
	}
}

func TestBuildLetterSources_FallbackOnEmpty(t *testing.T) {
	sources := buildLetterSources([]string{"hi", "cat"})
	if len(sources) == 0 {
		t.Error("buildLetterSources should return fallback when no 8-10 letter words exist")
	}
}

func TestPickLetterSource_FromList(t *testing.T) {
	words := []string{"alpha", "bravo", "charlie"}
	result := pickLetterSource(words)
	found := false
	for _, w := range words {
		if result == w {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("pickLetterSource returned %q which is not in the input", result)
	}
}

func TestPickLetterSource_FallbackOnEmpty(t *testing.T) {
	result := pickLetterSource(nil)
	if result == "" {
		t.Error("pickLetterSource(nil) should return a fallback, got empty")
	}
}

func TestBuildLeaderboard_SortsByScore(t *testing.T) {
	scores := map[string]int{"Alice": 10, "Bob": 20, "Charlie": 15}
	words := map[string]int{"Alice": 3, "Bob": 4, "Charlie": 5}
	lb := buildLeaderboard(scores, words)
	if len(lb) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(lb))
	}
	if lb[0].Name != "Bob" || lb[1].Name != "Charlie" || lb[2].Name != "Alice" {
		t.Errorf("unexpected order: %v", lb)
	}
}

func TestBuildLeaderboard_TiebreakerByWords(t *testing.T) {
	scores := map[string]int{"Alice": 10, "Bob": 10}
	words := map[string]int{"Alice": 3, "Bob": 5}
	lb := buildLeaderboard(scores, words)
	if lb[0].Name != "Bob" {
		t.Errorf("expected Bob first (more words), got %s", lb[0].Name)
	}
}

func TestBuildLeaderboard_TiebreakerByName(t *testing.T) {
	scores := map[string]int{"Charlie": 10, "Alice": 10}
	words := map[string]int{"Charlie": 3, "Alice": 3}
	lb := buildLeaderboard(scores, words)
	if lb[0].Name != "Alice" {
		t.Errorf("expected Alice first (alphabetical), got %s", lb[0].Name)
	}
}

func TestBuildLeaderboard_Empty(t *testing.T) {
	lb := buildLeaderboard(nil, nil)
	if len(lb) != 0 {
		t.Errorf("expected empty leaderboard, got %d entries", len(lb))
	}
}

func TestFormatLeaderboard(t *testing.T) {
	entries := []LeaderboardEntry{
		{Name: "Alice", Score: 20, Words: 5},
		{Name: "Bob", Score: 10, Words: 3},
	}
	result := formatLeaderboard(entries, 0)
	if !strings.Contains(result, "1. Alice") {
		t.Errorf("expected Alice at #1, got:\n%s", result)
	}
	if !strings.Contains(result, "2. Bob") {
		t.Errorf("expected Bob at #2, got:\n%s", result)
	}
}

func TestFormatLeaderboard_WithLimit(t *testing.T) {
	entries := []LeaderboardEntry{
		{Name: "A", Score: 30, Words: 3},
		{Name: "B", Score: 20, Words: 2},
		{Name: "C", Score: 10, Words: 1},
	}
	result := formatLeaderboard(entries, 2)
	lines := strings.Split(result, "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 lines with limit=2, got %d: %q", len(lines), result)
	}
}

func TestFormatLeaderboard_Empty(t *testing.T) {
	result := formatLeaderboard(nil, 0)
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestDetermineWinners_SingleWinner(t *testing.T) {
	scores := map[string]int{"Alice": 30, "Bob": 20}
	winners, best := determineWinners(scores)
	if len(winners) != 1 || winners[0] != "Alice" || best != 30 {
		t.Errorf("expected [Alice] with 30, got %v %d", winners, best)
	}
}

func TestDetermineWinners_Tie(t *testing.T) {
	scores := map[string]int{"Alice": 20, "Bob": 20}
	winners, best := determineWinners(scores)
	if len(winners) != 2 || best != 20 {
		t.Errorf("expected 2 winners with 20, got %v %d", winners, best)
	}
}

func TestDetermineWinners_Empty(t *testing.T) {
	winners, best := determineWinners(nil)
	if winners != nil || best != 0 {
		t.Errorf("expected nil/0, got %v %d", winners, best)
	}
}

func TestFormatWinnerLines_Single(t *testing.T) {
	result := formatWinnerLines([]string{"Alice"}, 30)
	if !strings.HasPrefix(result, "Winner:") {
		t.Errorf("single winner should start with 'Winner:', got %q", result)
	}
	if !strings.Contains(result, "Alice") {
		t.Errorf("expected Alice in output, got %q", result)
	}
}

func TestFormatWinnerLines_Multiple(t *testing.T) {
	result := formatWinnerLines([]string{"Alice", "Bob"}, 20)
	if !strings.HasPrefix(result, "Winners:") {
		t.Errorf("multiple winners should start with 'Winners:', got %q", result)
	}
}

func TestFormatWinnerLines_None(t *testing.T) {
	result := formatWinnerLines(nil, 0)
	if !strings.Contains(result, "None") {
		t.Errorf("no winners should contain 'None', got %q", result)
	}
}

func TestPluralSuffix(t *testing.T) {
	if pluralSuffix(1) != "" {
		t.Error("pluralSuffix(1) should be empty")
	}
	if pluralSuffix(0) != "s" {
		t.Error("pluralSuffix(0) should be 's'")
	}
	if pluralSuffix(5) != "s" {
		t.Error("pluralSuffix(5) should be 's'")
	}
}

func TestLoadWords_ReturnsNonEmpty(t *testing.T) {
	words := loadWords()
	if len(words) == 0 {
		t.Error("loadWords should return a non-empty list")
	}
}

func TestLoadWords_AllLowercase(t *testing.T) {
	words := loadWords()
	for _, w := range words {
		if w != strings.ToLower(w) {
			t.Errorf("loadWords returned non-lowercase word: %q", w)
			break
		}
	}
}

func TestItoa(t *testing.T) {
	if itoa(0) != "0" {
		t.Error("itoa(0) failed")
	}
	if itoa(42) != "42" {
		t.Error("itoa(42) failed")
	}
	if itoa(-1) != "-1" {
		t.Error("itoa(-1) failed")
	}
}
