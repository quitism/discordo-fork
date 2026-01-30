package sixel

import (
	"fmt"
	"os"
	"sync"

	"github.com/gdamore/tcell/v3"
)

type Image struct {
	X, Y int
	Data []byte
}

type Screen struct {
	tcell.Screen
	mu     sync.Mutex
	images []Image
}

func NewScreen(s tcell.Screen) *Screen {
	return &Screen{Screen: s}
}

func (s *Screen) RegisterImage(x, y int, data []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.images = append(s.images, Image{X: x, Y: y, Data: data})
}

func (s *Screen) Show() {
	s.Screen.Show()

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, img := range s.images {
		// Move cursor to x, y (ANSI uses 1-based coordinates)
		fmt.Fprintf(os.Stdout, "\033[%d;%dH", img.Y+1, img.X+1)
		// Write Sixel data
		os.Stdout.Write(img.Data)
	}
	s.images = s.images[:0]
}
