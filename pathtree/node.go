package pathtree

import (
	"bytes"
	"github.com/gabstv/sandpiper/route"
)

type Node struct {
	WildNodes  *Node
	NamedNodes map[string]*Node
	FullPath   string
	EndRoute   *route.Route
}

func NewNode() *Node {
	v := &Node{}
	v.NamedNodes = make(map[string]*Node)
	return v
}

func (n *Node) Find(path []string, mq *MatchQueue) {
	if len(path) < 1 {
		// found
		mq.Add(n)
		return
	}
	if n.WildNodes != nil {
		n.WildNodes.Find(path[1:], mq)
	}
	if child := n.NamedNodes[path[0]]; child != nil {
		child.Find(path[1:], mq)
	}
}

func (n *Node) Add(spath []string, fullpath string, proute *route.Route) error {
	if len(spath) < 1 {
		n.FullPath = fullpath
		n.EndRoute = proute
		return nil
	}
	switch spath[0][0] {
	case ':':
		// :param case
		//TODO: support params
		fallthrough
	case '*':
		// wildcard
		if n.WildNodes == nil {
			n.WildNodes = NewNode()
		}
		return n.WildNodes.Add(spath[1:], fullpath, proute)
	default:
		if n.NamedNodes[spath[0]] == nil {
			n.NamedNodes[spath[0]] = NewNode()
		}
		return n.NamedNodes[spath[0]].Add(spath[1:], fullpath, proute)
	}
}

func (n *Node) debugPrint(buf *bytes.Buffer, prevs, ps string) {
	if n.WildNodes != nil {
		buf.WriteString(prevs)
		buf.WriteString(" => *\n")
	}
	for k, _ := range n.NamedNodes {
		buf.WriteString(prevs)
		buf.WriteString(" => ")
		buf.WriteString(k)
		buf.WriteString("\n")
	}
	if n.WildNodes != nil {
		news := prevs + ps + "*"
		n.WildNodes.debugPrint(buf, news, ps)
	}
	for k, v := range n.NamedNodes {
		news := prevs + ps + k
		v.debugPrint(buf, news, ps)
	}
}
