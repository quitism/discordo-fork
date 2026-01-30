package sixel

import (
	"bufio"
	"image"
	"io"
	"os"
	"strings"
	"time"

	"github.com/mattn/go-sixel"
	"golang.org/x/term"
)

var Enabled bool

// IsSupported checks if the terminal supports Sixel graphics.
func IsSupported() bool {
	if !term.IsTerminal(int(os.Stdout.Fd())) || !term.IsTerminal(int(os.Stdin.Fd())) {
		return false
	}

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return false
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	if _, err := os.Stdout.WriteString("\033[c"); err != nil {
		return false
	}

	ch := make(chan string)
	go func() {
		r := bufio.NewReader(os.Stdin)
		var buf []byte
		for {
			b, err := r.ReadByte()
			if err != nil {
				close(ch)
				return
			}
			buf = append(buf, b)
			if b == 'c' {
				break
			}
		}
		ch <- string(buf)
	}()

	select {
	case res, ok := <-ch:
		if !ok {
			return false
		}
		if !strings.HasPrefix(res, "\x1b[") {
			return false
		}
		res = strings.TrimPrefix(res, "\x1b[")
		res = strings.TrimSuffix(res, "c")
		res = strings.TrimPrefix(res, "?")

		parts := strings.Split(res, ";")
		for _, p := range parts {
			if p == "4" {
				return true
			}
		}
	case <-time.After(200 * time.Millisecond):
		return false
	}

	return false
}

// Encode encodes the image to Sixel format and writes to w.
func Encode(w io.Writer, img image.Image) error {
	enc := sixel.NewEncoder(w)
	return enc.Encode(img)
}
