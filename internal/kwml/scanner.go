package kwml

import "unicode/utf8"

type scanner struct {
	raw                  []byte
	index                int
	prev, curr           rune
	prevWidth, currWidth int
}

func (s *scanner) advance() (char rune, width int) {
	if s.isAtEnd() {
		return
	}

	s.prev = s.curr
	s.prevWidth = s.currWidth
	s.curr, s.currWidth = utf8.DecodeRune(s.raw[s.index:])
	s.index += s.currWidth
	return s.curr, s.currWidth
}

func (s *scanner) peek() (char rune, width int) {
	if s.isAtEnd() {
		return
	}

	return utf8.DecodeRune(s.raw[s.index:])
}

func (s *scanner) match(char rune) bool {
	if s.isAtEnd() {
		return false
	}

	next, _ := s.peek()
	return next == char
}

func (s *scanner) cursor() []byte {
	return s.raw[s.index:s.index]
}

func (s *scanner) current() (char rune, width int) {
	return s.curr, s.currWidth
}

func (s *scanner) currentBytes() []byte {
	i := s.index
	h := i - s.currWidth
	return s.raw[h:i]
}

func (s *scanner) previous() (char rune, width int) {
	return s.prev, s.prevWidth
}

func (s *scanner) isAtEnd() bool {
	return s.index >= len(s.raw)
}

func (s *scanner) isEmpty() bool {
	return len(s.raw) == 0
}

func (s *scanner) clone() *scanner {
	return &scanner{
		raw:       s.raw,
		index:     s.index,
		prev:      s.prev,
		prevWidth: s.prevWidth,
		curr:      s.curr,
		currWidth: s.currWidth,
	}
}

func newScanner(raw []byte) *scanner {
	return &scanner{raw: raw}
}
