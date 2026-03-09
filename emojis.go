package main

import (
	"encoding/json"
	"net/http"
)

type emojiOption struct {
	Emoji string `json:"emoji"`
	Name  string `json:"name"`
	Tags  string `json:"tags"`
}

var emojiOptions = []emojiOption{
	{Emoji: "😀", Name: "grinning", Tags: "happy smile"},
	{Emoji: "😄", Name: "grinning with eyes", Tags: "happy smile laugh"},
	{Emoji: "😁", Name: "beaming", Tags: "grin happy"},
	{Emoji: "😂", Name: "joy", Tags: "laugh tears"},
	{Emoji: "🤣", Name: "rofl", Tags: "laugh rolling"},
	{Emoji: "😊", Name: "blush", Tags: "smile warm"},
	{Emoji: "😉", Name: "wink", Tags: "playful"},
	{Emoji: "😍", Name: "heart eyes", Tags: "love"},
	{Emoji: "😘", Name: "kiss", Tags: "love"},
	{Emoji: "😎", Name: "sunglasses", Tags: "cool"},
	{Emoji: "🤩", Name: "star struck", Tags: "wow excited"},
	{Emoji: "🥳", Name: "party", Tags: "celebrate"},
	{Emoji: "🤯", Name: "mind blown", Tags: "shock wow"},
	{Emoji: "😭", Name: "loudly crying", Tags: "sad tears"},
	{Emoji: "😅", Name: "sweat smile", Tags: "relief"},
	{Emoji: "😤", Name: "huff", Tags: "frustrated"},
	{Emoji: "🤔", Name: "thinking", Tags: "hmm"},
	{Emoji: "🫡", Name: "salute", Tags: "respect"},
	{Emoji: "😴", Name: "sleeping", Tags: "tired"},
	{Emoji: "🤗", Name: "hug", Tags: "support"},
	{Emoji: "👍", Name: "thumbs up", Tags: "yes ok"},
	{Emoji: "👎", Name: "thumbs down", Tags: "no"},
	{Emoji: "👏", Name: "clap", Tags: "applause"},
	{Emoji: "🙌", Name: "raised hands", Tags: "praise"},
	{Emoji: "🙏", Name: "pray", Tags: "thanks"},
	{Emoji: "🤝", Name: "handshake", Tags: "deal"},
	{Emoji: "👀", Name: "eyes", Tags: "watch"},
	{Emoji: "🔥", Name: "fire", Tags: "hot lit"},
	{Emoji: "💯", Name: "hundred", Tags: "perfect"},
	{Emoji: "✨", Name: "sparkles", Tags: "magic"},
	{Emoji: "⭐", Name: "star", Tags: "favorite"},
	{Emoji: "⚡", Name: "lightning", Tags: "energy fast"},
	{Emoji: "💥", Name: "boom", Tags: "impact"},
	{Emoji: "🎉", Name: "tada", Tags: "party celebrate"},
	{Emoji: "🎊", Name: "confetti", Tags: "party celebrate"},
	{Emoji: "🎁", Name: "gift", Tags: "present"},
	{Emoji: "🏆", Name: "trophy", Tags: "winner"},
	{Emoji: "🥇", Name: "gold medal", Tags: "winner first"},
	{Emoji: "🚀", Name: "rocket", Tags: "launch"},
	{Emoji: "💡", Name: "idea", Tags: "think"},
	{Emoji: "🧠", Name: "brain", Tags: "smart"},
	{Emoji: "🎮", Name: "gamepad", Tags: "gaming"},
	{Emoji: "🕹️", Name: "joystick", Tags: "gaming"},
	{Emoji: "⌛", Name: "hourglass", Tags: "time"},
	{Emoji: "✅", Name: "check", Tags: "yes done"},
	{Emoji: "❌", Name: "cross", Tags: "no wrong"},
	{Emoji: "⚠️", Name: "warning", Tags: "alert"},
	{Emoji: "❤️", Name: "red heart", Tags: "love"},
	{Emoji: "💜", Name: "purple heart", Tags: "love"},
	{Emoji: "🖤", Name: "black heart", Tags: "love"},
	{Emoji: "💙", Name: "blue heart", Tags: "love"},
	{Emoji: "💚", Name: "green heart", Tags: "love"},
	{Emoji: "🧡", Name: "orange heart", Tags: "love"},
	{Emoji: "🤍", Name: "white heart", Tags: "love"},
	{Emoji: "🤎", Name: "brown heart", Tags: "love"},
	{Emoji: "🎵", Name: "music note", Tags: "song"},
	{Emoji: "📣", Name: "megaphone", Tags: "announce"},
	{Emoji: "👋", Name: "wave", Tags: "hello"},
	{Emoji: "🤟", Name: "love you gesture", Tags: "hand"},
	{Emoji: "🤖", Name: "robot", Tags: "bot"},
	{Emoji: "🧩", Name: "puzzle", Tags: "game"},
	{Emoji: "🎯", Name: "target", Tags: "goal"},
	{Emoji: "🌍", Name: "globe", Tags: "world"},
	{Emoji: "☕", Name: "coffee", Tags: "break"},
}

func handleEmojiOptions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string][]emojiOption{"emojis": emojiOptions}); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}
