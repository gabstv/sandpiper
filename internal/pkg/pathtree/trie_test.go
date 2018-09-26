package pathtree

import (
	"testing"
)

func TestAddition(t *testing.T) {
	trie := NewTrie(".")
	err := trie.Add("www.newgrounds.com")
	if err != nil {
		t.Fatal(err)
	}
	trie.Add("dev.newgrounds.net")
	trie.Add("*.newgrounds.net")
	trie.Add("www.newgrounds.com.br")
	t.Log(trie)
	t.Log(trie.Find("dev.newgrounds.net"))
	t.Log("\n", trie.debugPrint())
}

func Benchmark10(b *testing.B) {
	trie := NewTrie(".")
	trie.Add("www.google.com")
	trie.Add("www.google.net")
	trie.Add("www.yahoo.com")
	trie.Add("www.gmail.com")
	trie.Add("www.reddit.com")
	trie.Add("www.reddit.tv")
	trie.Add("www.reddit.z")
	trie.Add("m.reddit.com")
	trie.Add("m.yahoo.com")
	trie.Add("*.newgrounds.com")
	trie.Add("newgrounds.com")
	trie.Add("google.com")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		trie.Find("www.reddit.com")
	}
}
