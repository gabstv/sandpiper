package pathtree

import (
	"testing"

	"github.com/gabstv/sandpiper/internal/pkg/route"
)

func TestAddition(t *testing.T) {

	rr := &route.Route{}
	rr.SetupWsCfgDefaults()

	trie := NewTrie(".")
	err := trie.Add("www.newgrounds.com", rr)
	if err != nil {
		t.Fatal(err)
	}
	trie.Add("dev.newgrounds.net", rr)
	trie.Add("*.newgrounds.net", rr)
	trie.Add("www.newgrounds.com.br", rr)
	t.Log(trie)
	t.Log(trie.Find("dev.newgrounds.net"))
	t.Log("\n", trie.debugPrint())
}

func Benchmark10(b *testing.B) {

	rr := &route.Route{}
	rr.SetupWsCfgDefaults()

	trie := NewTrie(".")
	trie.Add("www.google.com", rr)
	trie.Add("www.google.net", rr)
	trie.Add("www.yahoo.com", rr)
	trie.Add("www.gmail.com", rr)
	trie.Add("www.reddit.com", rr)
	trie.Add("www.reddit.tv", rr)
	trie.Add("www.reddit.z", rr)
	trie.Add("m.reddit.com", rr)
	trie.Add("m.yahoo.com", rr)
	trie.Add("*.newgrounds.com", rr)
	trie.Add("newgrounds.com", rr)
	trie.Add("google.com", rr)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		trie.Find("www.reddit.com")
	}
}
