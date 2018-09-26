package pathtree

import (
	"bytes"
	"errors"
	"strings"

	"github.com/gabstv/sandpiper/internal/pkg/route"
)

type Trie struct {
	Root          *Node
	PathSeparator string
	RawRoutes     map[string]*route.Route
}

func NewTrie(separator string) *Trie {
	v := &Trie{}
	v.Root = NewNode()
	v.PathSeparator = separator
	v.RawRoutes = make(map[string]*route.Route)
	return v
}

func (t *Trie) Find(path string) *Node {
	pl := strings.Split(path, t.PathSeparator)
	mq := NewMatchQueue()
	t.Root.Find(pl, mq)
	if mq.First == nil {
		return nil
	}
	return mq.First.Val
}

func (t *Trie) Add(path string, proute *route.Route) error {
	if _, ok := t.RawRoutes[path]; ok {
		return errors.New("duplicate path")
	}
	spath := strings.Split(path, t.PathSeparator)
	t.RawRoutes[path] = proute
	return t.Root.Add(spath, path, proute)
}

func (t *Trie) debugPrint() string {
	buf := new(bytes.Buffer)
	t.Root.debugPrint(buf, "", t.PathSeparator)
	return buf.String()
}
