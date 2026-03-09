package main

import (
	"crypto/rand"
	"encoding/json"
	"math/big"
	"net/http"
	"strings"
)

var funFacts = []string{
	`"Scrabble" was invented in 1938 by an unemployed architect.`,
	`The longest English word without repeating a letter is "uncopyrightable".`,
	`English has over 170,000 words in current use.`,
	`"Set" has the most definitions of any English word (430+).`,
	`A pangram uses every letter of the alphabet at least once.`,
	`"Q" without "U" words: qi, qat, qoph, qanat, qadi.`,
	`The blank tile in Scrabble is worth 0 but can be any letter.`,
	`"Dreamt" is the only English word ending in "mt".`,
	`"Pneumonoultramicroscopicsilicovolcanoconiosis" is 45 letters long.`,
	`The highest possible Scrabble score for a single word is 1778.`,
	`"Typewriter" can be typed using only the top row of a keyboard.`,
	`"Lollipop" is the longest word you can type with just your right hand.`,
	`"Rhythms" is the longest English word without a vowel.`,
	`"Subdermatoglyphic" is the longest isogram (no repeating letters) at 17 letters.`,
	`An ambigram reads the same upside down — like "SWIMS".`,
	`"Bookkeeper" is the only English word with three consecutive double letters.`,
	`"Almost" is the longest common English word in alphabetical order.`,
	`Over 80% of English words with the prefix "un-" are negative.`,
	`Shakespeare invented over 1,700 words including "eyeball" and "bedroom".`,
	`The dot over a lowercase "i" or "j" is called a tittle.`,
	`"Stewardesses" is the longest word typed with only the left hand.`,
	`"Underground" is the only word that begins and ends with "und".`,
	`"Queue" is the only word that sounds the same after removing 4 letters.`,
	`"Z" is worth 10 points in Scrabble — tied with "Q" for the highest.`,
	`"Abstemious" and "Facetious" contain all 5 vowels in order.`,
	`"Queueing" is the only English word with 5 consecutive vowels.`,
	`No English word rhymes with "orange", "purple", "silver", or "month".`,
	`"Strengths" has only one vowel in 9 letters.`,
	`"Therein" contains 13 words without rearranging letters (the, he, her, ere, etc.).`,
	`The first Scrabble game was called "Lexiko", then "Criss-Crosswords".`,
	`"Goodbye" comes from "God be with ye" — a 16th century contraction.`,
	`"Carat", "karat", "caret", and "carrot" are all pronounced the same.`,
	`A sentence using every letter of the alphabet is called a pangram.`,
	`"Eleven plus two" is an anagram of "twelve plus one".`,
	`The word "alphabet" comes from Greek: alpha + beta.`,
	`"Aegilops" (a grass genus) is the longest word with letters in alphabetical order.`,
	`"Racecar" is the same word spelled backwards — a palindrome.`,
	`"Hundred" is the only number whose letters are in alphabetical order.`,
	`An average person's vocabulary is 20,000–35,000 words.`,
	`"Ghost words" are words that appear in dictionaries by mistake.`,
	`"Indivisibility" has only one vowel used 5 times.`,
	`The word "set" takes 60,000 words to define in the Oxford English Dictionary.`,
	`"Jinx" is worth 18 points in Scrabble — one of the highest 4-letter words.`,
	`"Quizzify" would score 41 points in Scrabble if it were allowed.`,
	`"Abracadabra" originally appeared in a 2nd-century medical text.`,
	`The shortest complete sentence in English is "I am." or even "Go."`,
	`"Astronaut" means "star sailor" in Greek.`,
	`"Muscle" comes from Latin "musculus" meaning "little mouse".`,
	`The most common letter in English is "E" — it appears in ~11% of all words.`,
	`"Unsplittable" — only 2% of English words contain all 5 vowels.`,
	`The word "bee" in spelling bee comes from an Old English word for "prayer gathering".`,
	`"Disambiguation" has 14 letters and no repeating ones.`,
	`"Sms" was added to the dictionary in 2014 — language evolves fast.`,
	`"Volcano" comes from Vulcan, the Roman god of fire.`,
	`Dr. Seuss wrote "Green Eggs and Ham" using only 50 words on a dare.`,
	`"Honorificabilitudinity" was used by Shakespeare in Love's Labour's Lost.`,
	`English gains about 1,000 new words every year.`,
	`There are exactly 100 two-letter words legal in Scrabble (English).`,
	`"Oxyphenbutazone" is often cited as the highest-scoring Scrabble word at 1,778 pts.`,
	`A "logophile" is a person who loves words.`,
	`"Cwm" and "crwth" are English words borrowed from Welsh — no vowels needed.`,
	`"Hippopotomonstrosesquippedaliophobia" ironically means fear of long words.`,
	`"Floccinaucinihilipilification" means estimating something as worthless — at 29 letters.`,
	`"Supercalifragilisticexpialidocious" is 34 letters but not in most dictionaries.`,
	`"Peanut" is neither a pea nor a nut — it's a legume.`,
	`"Wifi" doesn't actually stand for "wireless fidelity".`,
	`"CRISPR" is an anagram of "CRISP" with an extra R — fitting for gene editing.`,
	`The chances of getting a blank tile in Scrabble: 2 out of 100 tiles.`,
	`"OK" is the most widely understood word across all languages.`,
	`"Mama" and "Papa" sound similar in nearly every language on earth.`,
	`"Nerd" was first used by Dr. Seuss in "If I Ran the Zoo" (1950).`,
	`In competitive Scrabble, top players memorize 180,000+ words.`,
	`An anagram of "listen" is "silent".`,
	`An anagram of "astronomer" is "moon starer".`,
	`An anagram of "dormitory" is "dirty room".`,
	`An anagram of "the eyes" is "they see".`,
	`An anagram of "a decimal point" is "I'm a dot in place".`,
	`The word "teacher" is an anagram of "cheater".`,
	`An anagram of "slot machines" is "cash lost in me".`,
	`An anagram of "Elvis" is "lives".`,
	`An anagram of "the Morse code" is "here come dots".`,
	`An anagram of "William Shakespeare" is "I am a weakish speller".`,
	`English is the only major language that doesn't have a governing body.`,
	`The word "robot" comes from Czech "robota" meaning "forced labor".`,
	`"Uncopyrightable" at 15 letters is the longest common isogram.`,
	`"Forty" is the only number spelled with letters in alphabetical order.`,
	`"Eggcorn" is when you mishear a phrase — like "for all intensive purposes".`,
	`"Defenestration" means throwing someone out of a window. There's a word for that.`,
	`Word games train pattern recognition, memory, and quick recall.`,
	`A balanced mix of short and long words usually wins tight rounds.`,
	`Finding prefixes first can unlock many valid follow-up guesses.`,
	`Suffixes like -ing, -ed, and -er can turn one root into many words.`,
	`Scanning letters in pairs helps spot hidden words faster than random searching.`,
	`Even a low-point word can protect a lead by claiming a key letter combo.`,
	`High-value letters are strongest when placed in flexible word shapes.`,
	`One reliable strategy is to build around common vowels early.`,
	`Short validation loops often beat waiting for a perfect long word.`,
	`Momentum in word games usually comes from consistent, accurate guesses.`,
	`Practicing anagrams can significantly improve live round performance.`,
	`Top players often prioritize certainty before complexity under time pressure.`,
	`Recognizing letter frequency can guide better guess ordering.`,
	`A calm pace can outperform rushed guessing when the clock gets tight.`,
	`Keeping a mental map of used words avoids accidental repeats.`,
	`Word stems are powerful anchors for finding multiple valid entries.`,
	`Switching perspective from left-to-right to chunk-based scanning can help.`,
	`A strong opening minute sets the tone for the rest of a match.`,
	`Round-to-round consistency usually beats occasional big bursts.`,
	`A good habit is to verify letter counts before sending a guess.`,
	`Players improve fastest when they review missed opportunities after each round.`,
	`Creative guesses still perform best when grounded in known patterns.`,
	`Small point gains compound quickly across multiple rounds.`,
	`Many winning plays come from simple words found early.`,
	`Confidence grows fastest when you build from familiar letter clusters.`,
	`Recognizing common digraphs like th, ch, and sh speeds discovery.`,
	`Repeating high-quality routines can stabilize performance under pressure.`,
	`Strategic patience can be just as valuable as speed.`,
	`Careful observation often reveals words hidden in plain sight.`,
	`Developing a personal search pattern makes guessing more efficient.`,
	`Good rounds often start with quick scans for 3- and 4-letter options.`,
	`Long words are rewarding, but dependable medium words win consistently.`,
	`A clean typing rhythm can reduce avoidable input mistakes.`,
	`In close games, accuracy usually decides the final standings.`,
	`Looking for vowel-consonant balance can unlock more legal combinations.`,
	`Word-building confidence improves with regular short practice sessions.`,
	`Consistent players tend to reuse effective solving patterns.`,
	`Spotting uncommon letter pairs early can create scoring opportunities.`,
	`A quick reset after a missed guess keeps focus high.`,
	`Late-round discipline can protect narrow leads.`,
	`Efficient scanning beats random guessing over longer sessions.`,
	`Most improvement comes from refining fundamentals, not flashy plays.`,
	`Structured search habits help when the timer pressure increases.`,
	`Great rounds combine speed, precision, and steady decision-making.`,
	`Smart guess ordering can surface higher-value words sooner.`,
	`Clear focus and repetition are powerful competitive advantages.`,
	`The best strategy is the one you can execute consistently.`,
	`Strong endings often come from simple, reliable choices.`,
	`Habitual review turns close losses into future wins.`,
	`Every round is a new chance to improve your pattern recognition.`,
}

func randomFunFact() string {
	if len(funFacts) == 0 {
		return "Words can be wildly fun."
	}

	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(funFacts))))
	if err != nil {
		return funFacts[0]
	}

	fact := strings.TrimSpace(funFacts[n.Int64()])
	if fact == "" {
		return funFacts[0]
	}

	return fact
}

func handleFunFact(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"fact": randomFunFact()}); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}
