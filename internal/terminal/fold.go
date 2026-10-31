package terminal

// A block is the output of one command, from the prompt line it was typed on
// up to the next prompt. Clicking a block's prompt line folds its output away.

type lineRange struct {
	lo, hi int
}

func inRanges(ranges []lineRange, abs int) bool {
	for _, r := range ranges {
		if abs >= r.lo && abs <= r.hi {
			return true
		}
	}
	return false
}

// blockEnd is the absolute line after block i's output, or -1 while its
// command is still running.
func (s *Screen) blockEnd(i int) int {
	blocks := s.buf.blocks
	if i+1 < len(blocks) {
		return blocks[i+1].header
	}
	if s.status.busy != "" {
		return -1
	}
	return s.buf.dropped + len(s.buf.lines) - 1
}

func (s *Screen) foldedRanges() []lineRange {
	var ranges []lineRange
	for i, b := range s.buf.blocks {
		if !b.folded {
			continue
		}
		if end := s.blockEnd(i); end > b.header+1 {
			ranges = append(ranges, lineRange{lo: b.header + 1, hi: end - 1})
		}
	}
	return ranges
}

func (s *Screen) foldedCount(header int) int {
	for i, b := range s.buf.blocks {
		if b.header == header && b.folded {
			if end := s.blockEnd(i); end > header+1 {
				return end - header - 1
			}
		}
	}
	return 0
}

func (s *Screen) toggleFold(abs int) bool {
	for i := range s.buf.blocks {
		b := &s.buf.blocks[i]
		if b.header != abs {
			continue
		}
		if end := s.blockEnd(i); end <= b.header+1 {
			return false
		}
		b.folded = !b.folded
		return true
	}
	return false
}

func (s *Screen) unfoldAround(abs int) {
	for i := range s.buf.blocks {
		b := &s.buf.blocks[i]
		if b.folded && abs > b.header && abs < s.blockEnd(i) {
			b.folded = false
		}
	}
}
