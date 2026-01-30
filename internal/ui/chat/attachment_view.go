package chat

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ayn2op/discordo/internal/consts"
	"github.com/ayn2op/tview"
	"github.com/gdamore/tcell/v3"
	"github.com/mattn/go-sixel"
	"golang.org/x/image/draw"
)

type AttachmentView struct {
	*tview.Box
	app      *tview.Application
	url      string
	filename string

	img     image.Image
	loading bool
	loaded  bool
	err     error
	mu      sync.Mutex

	// Caching
	cachedSixel  []byte
	lastW, lastH int
}

func NewAttachmentView(app *tview.Application, url, filename string) *AttachmentView {
	return &AttachmentView{
		Box:      tview.NewBox(),
		app:      app,
		url:      url,
		filename: filename,
	}
}

func (a *AttachmentView) Draw(screen tcell.Screen) {
	// Draw background and borders if any
	a.Box.DrawForSubclass(screen, a)

	x, y, w, h := a.GetInnerRect()
	if w <= 0 || h <= 0 {
		return
	}

	a.mu.Lock()

	if a.err != nil {
		tview.Print(screen, fmt.Sprintf("Error: %v", a.err), x, y, w, tview.AlignmentLeft, tcell.ColorRed)
		a.mu.Unlock()
		return
	}

	if !a.loaded {
		if !a.loading {
			a.loading = true
			go a.loadImage()
		}
		tview.Print(screen, "Loading...", x, y, w, tview.AlignmentLeft, tcell.ColorYellow)
		a.mu.Unlock()
		return
	}

	if a.img == nil {
		a.mu.Unlock()
		return
	}

	// Calculate target pixel size
	// Assuming 10x20 pixels per cell as a safe default for terminals
	cellW, cellH := 10, 20
	targetW := uint(w * cellW)
	targetH := uint(h * cellH)

	// Check cache
	if a.lastW == int(targetW) && a.lastH == int(targetH) && len(a.cachedSixel) > 0 {
		data := a.cachedSixel
		a.mu.Unlock()
		a.renderSixelAsync(x, y, data)
		return
	}

	img := a.img
	a.mu.Unlock() // Unlock for expensive operation

	// Resize
	dst := image.NewRGBA(image.Rect(0, 0, int(targetW), int(targetH)))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, img.Bounds(), draw.Over, nil)

	// Encode
	var buf bytes.Buffer
	enc := sixel.NewEncoder(&buf)
	if err := enc.Encode(dst); err != nil {
		slog.Error("failed to encode sixel", "err", err)
		return
	}
	data := buf.Bytes()

	a.mu.Lock()
	a.cachedSixel = data
	a.lastW = int(targetW)
	a.lastH = int(targetH)
	a.mu.Unlock()

	a.renderSixelAsync(x, y, data)
}

func (a *AttachmentView) renderSixelAsync(x, y int, data []byte) {
	go func() {
		// Small delay to allow tcell to flush its buffer and clear the screen area.
		// This mitigates the race condition where tcell overwrites the Sixel image.
		time.Sleep(10 * time.Millisecond)

		// Save cursor, Move cursor, Print Sixel, Restore cursor
		fmt.Print("\0337")
		fmt.Printf("\033[%d;%dH", y+1, x+1)
		os.Stdout.Write(data)
		fmt.Print("\0338")
	}()
}

func (a *AttachmentView) loadImage() {
	err := a.downloadAndDecode()

	a.mu.Lock()
	a.err = err
	a.loaded = true
	a.loading = false
	a.mu.Unlock()

	a.app.QueueUpdateDraw(func() {
		// Just trigger a redraw
	})
}

func (a *AttachmentView) downloadAndDecode() error {
	// Check cache
	cacheDir := filepath.Join(consts.CacheDir(), "attachments")
	// Use hash of URL for uniqueness
	hash := sha256.Sum256([]byte(a.url))
	hashStr := hex.EncodeToString(hash[:])
	ext := filepath.Ext(a.filename)
	if ext == "" {
		ext = ".png" // Default
	}
	cachePath := filepath.Join(cacheDir, hashStr+ext)

	if err := os.MkdirAll(cacheDir, os.ModePerm); err != nil {
		return err
	}

	// Try reading from cache
	f, err := os.Open(cachePath)
	if err == nil {
		defer f.Close()
		img, _, err := image.Decode(f)
		if err == nil {
			a.mu.Lock()
			a.img = img
			a.mu.Unlock()
			return nil
		}
	}

	// Download
	resp, err := http.Get(a.url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// Save to cache
	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		slog.Warn("failed to write cache", "err", err)
	}

	// Decode
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return err
	}

	a.mu.Lock()
	a.img = img
	a.mu.Unlock()
	return nil
}
