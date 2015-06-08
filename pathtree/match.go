package pathtree

type Match struct {
	Next *Match
	Val  *Node
}

type MatchQueue struct {
	First *Match
	Last  *Match
}

func NewMatchQueue() *MatchQueue {
	return &MatchQueue{}
}

func (q *MatchQueue) Add(v *Node) {
	m := &Match{}
	m.Val = v
	if q.First == nil {
		q.First = m
		q.Last = m
		return
	}
	q.Last.Next = m
	q.Last = m
}
