package chat

import (
	"github.com/ayn2op/discordo/internal/sixel"
	"github.com/ayn2op/tview"
	"github.com/gdamore/tcell/v3"
)

type Rect struct {
	X, Y, W, H int
}

type MessageItem struct {
	*tview.TextView
	SixelData []byte
	ImageRows int
	ClipRect  Rect
}

func NewMessageItem(tv *tview.TextView, sixelData []byte, imageRows int, clipRect Rect) *MessageItem {
	return &MessageItem{
		TextView:  tv,
		SixelData: sixelData,
		ImageRows: imageRows,
		ClipRect:  clipRect,
	}
}

func (m *MessageItem) Draw(screen tcell.Screen) {
	m.TextView.Draw(screen)

	if len(m.SixelData) > 0 {
		x, y, _, h := m.GetRect()
		// The image is at the bottom.
		imgY := y + h - m.ImageRows

		// Clipping logic:
		// We only draw the image if it fully fits within the ClipRect vertically to avoid bleeding.
		// Sixel cannot be easily clipped per-row without re-encoding.
		// So we ensure the image top is >= ClipRect.Y
		// And image bottom is <= ClipRect.Y + ClipRect.H

		imgBottom := imgY + m.ImageRows
		clipBottom := m.ClipRect.Y + m.ClipRect.H

		if imgY >= m.ClipRect.Y && imgBottom <= clipBottom {
			// Also check screen bounds
			w, screenH := screen.Size()
			if imgY >= 0 && imgBottom <= screenH && x >= 0 && x < w {
				if s, ok := screen.(*sixel.Screen); ok {
					s.RegisterImage(x, imgY, m.SixelData)
				}
			}
		}
	}
}
