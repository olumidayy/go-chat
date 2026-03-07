package websocket

import (
	_ "embed"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const defaultGameDuration = time.Minute

//go:embed words.txt
var embeddedWords string

var fallbackWords = []string{
	"adventure",
	"advent",
	"venture",
	"avenue",
	"nature",
	"trade",
	"rated",
	"tread",
	"read",
	"dear",
	"dare",
	"near",
	"earn",
	"turn",
	"rune",
	"tune",
	"ant",
	"tan",
	"art",
	"rat",
	"tar",
	"notebook",
	"book",
	"note",
	"tone",
	"took",
	"token",
	"knot",
	"bone",
	"bent",
	"tone",
	"stone",
	"ones",
	"nose",
	"soon",
	"golang",
	"socket",
	"server",
	"planet",
	"stream",
	"binary",
	"puzzle",
	"garden",
	"rocket",
	"thunder",
	"journey",
	"picture",
	"village",
	"capture",
	"kingdom",
	"rainbow",
}

var fallbackLetterSources = []string{
	"adventure",
	"notebook",
	"paintwork",
	"triangle",
	"developer",
	"backyards",
}

type GameRound struct {
	Letters string
	Scores  []LeaderboardEntry
}

type LeaderboardEntry struct {
	Name  string `json:"name"`
	Score int    `json:"score"`
	Words int    `json:"words"`
}

var scrabbleLetterScores = map[rune]int{
	'a': 1, 'b': 3, 'c': 3, 'd': 2, 'e': 1, 'f': 4, 'g': 2,
	'h': 4, 'i': 1, 'j': 8, 'k': 5, 'l': 1, 'm': 3, 'n': 1,
	'o': 1, 'p': 3, 'q': 10, 'r': 1, 's': 1, 't': 1, 'u': 1,
	'v': 4, 'w': 4, 'x': 8, 'y': 4, 'z': 10,
}

func loadWords() []string {
	fields := strings.Fields(embeddedWords)
	words := make([]string, 0, len(fields))

	for _, field := range fields {
		word := normalizeWord(field)
		if len([]rune(word)) < 2 {
			continue
		}

		words = append(words, word)
	}

	if len(words) == 0 {
		return append(words, fallbackWords...)
	}

	return words
}

func buildWordSet(words []string) map[string]struct{} {
	set := make(map[string]struct{}, len(words))
	for _, word := range words {
		set[word] = struct{}{}
	}

	return set
}

func buildLetterSources(words []string) []string {
	sources := make([]string, 0)
	for _, word := range words {
		length := len([]rune(word))
		if length >= 8 && length <= 10 {
			sources = append(sources, word)
		}
	}

	if len(sources) == 0 {
		return append(sources, fallbackLetterSources...)
	}

	return sources
}

func pickLetterSource(words []string) string {
	if len(words) == 0 {
		return fallbackLetterSources[rand.Intn(len(fallbackLetterSources))]
	}

	return words[rand.Intn(len(words))]
}

func shuffleLetters(word string) string {
	letters := []rune(strings.ToUpper(word))
	if len(letters) < 2 {
		return strings.ToUpper(word)
	}

	scrambled := append([]rune(nil), letters...)
	for attempt := 0; attempt < 8; attempt++ {
		rand.Shuffle(len(scrambled), func(i, j int) {
			scrambled[i], scrambled[j] = scrambled[j], scrambled[i]
		})

		if string(scrambled) != strings.ToUpper(word) {
			return string(scrambled)
		}
	}

	return string(append(scrambled[1:], scrambled[0]))
}

func spacedLetters(word string) string {
	letters := []rune(strings.ToUpper(word))
	parts := make([]string, 0, len(letters))
	for _, letter := range letters {
		parts = append(parts, string(letter))
	}

	return strings.Join(parts, " ")
}

func normalizeWord(word string) string {
	word = strings.TrimSpace(strings.ToLower(word))
	if word == "" {
		return ""
	}

	var builder strings.Builder
	for _, r := range word {
		if !unicode.IsLetter(r) {
			return ""
		}
		builder.WriteRune(unicode.ToLower(r))
	}

	return builder.String()
}

func canBuildWord(word, letters string) bool {
	available := letterCounts(letters)
	for _, r := range word {
		available[r]--
		if available[r] < 0 {
			return false
		}
	}

	return true
}

func letterCounts(word string) map[rune]int {
	counts := make(map[rune]int)
	for _, r := range strings.ToLower(word) {
		counts[r]++
	}

	return counts
}

func scoreWord(word string) int {
	points := 0
	for _, letter := range normalizeWord(word) {
		points += scrabbleLetterScores[letter]
	}

	if points == 0 {
		return 1
	}

	return points
}

func buildLeaderboard(scores, wordCounts map[string]int) []LeaderboardEntry {
	entries := make([]LeaderboardEntry, 0, len(scores))
	for name, score := range scores {
		entries = append(entries, LeaderboardEntry{
			Name:  name,
			Score: score,
			Words: wordCounts[name],
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Score != entries[j].Score {
			return entries[i].Score > entries[j].Score
		}

		if entries[i].Words != entries[j].Words {
			return entries[i].Words > entries[j].Words
		}

		return entries[i].Name < entries[j].Name
	})

	return entries
}

func formatLeaderboard(entries []LeaderboardEntry, limit int) string {
	if len(entries) == 0 {
		return ""
	}

	if limit > 0 && len(entries) > limit {
		entries = entries[:limit]
	}

	parts := make([]string, 0, len(entries))
	for index, entry := range entries {
		parts = append(parts, itoa(index+1)+". "+entry.Name+" — "+itoa(entry.Score)+" pts / "+itoa(entry.Words)+" word"+pluralSuffix(entry.Words))
	}

	return strings.Join(parts, "\n")
}

func formatWinnerLines(winners []string, bestScore int) string {
	if len(winners) == 0 {
		return "Winners:\n- None"
	}

	lines := make([]string, 0, len(winners))
	for _, winner := range winners {
		lines = append(lines, "- "+winner+" — "+itoa(bestScore)+" pts")
	}

	title := "Winner:"
	if len(winners) > 1 {
		title = "Winners:"
	}

	return title + "\n" + strings.Join(lines, "\n")
}

func itoa(value int) string {
	return strconv.Itoa(value)
}

func determineWinners(scores map[string]int) ([]string, int) {
	if len(scores) == 0 {
		return nil, 0
	}

	bestScore := 0
	winners := make([]string, 0)
	for name, score := range scores {
		switch {
		case score > bestScore:
			bestScore = score
			winners = []string{name}
		case score == bestScore:
			winners = append(winners, name)
		}
	}

	sort.Strings(winners)
	return winners, bestScore
}
