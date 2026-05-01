package tui

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"io"
	"math"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/vibium/player/internal/kitty"
	"github.com/vibium/player/internal/overlay"
	"github.com/vibium/player/internal/recording"
)

const (
	idleGapThreshold = 2000
	idleGapPad       = 500
	tickInterval     = 50 * time.Millisecond
)

type Model struct {
	rec      *recording.Recording
	out      io.Writer
	cellSize kitty.PixelSize

	selIdx    int
	frameIdx  int
	playing   bool
	speed     float64
	virtTime  int64
	lastTick  time.Time
	frameMode bool

	imgCache     map[string][]byte
	lastImageKey string
	imageDirty   bool
	paneSize     layout
	err          error

	width  int
	height int
}

type layout struct {
	imageOrigin kitty.Cell
	imageCells  kitty.Size
	imageCols   int
	imageRows   int
	actionCols  int
	topRows     int
}

type tickMsg time.Time

func New(rec *recording.Recording, out io.Writer, cellSize kitty.PixelSize) *Model {
	if out == nil {
		out = io.Discard
	}
	if cellSize.Width <= 0 {
		cellSize.Width = 8
	}
	if cellSize.Height <= 0 {
		cellSize.Height = 16
	}
	m := &Model{
		rec:        rec,
		out:        out,
		cellSize:   cellSize,
		speed:      1,
		imgCache:   map[string][]byte{},
		imageDirty: true,
		width:      100,
		height:     30,
	}
	m.syncFromAction()
	return m
}

func (m *Model) Init() tea.Cmd {
	return m.renderCmd()
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if msg.Width != m.width || msg.Height != m.height {
			m.width = msg.Width
			m.height = msg.Height
			m.imgCache = map[string][]byte{}
			m.lastImageKey = ""
			m.imageDirty = true
			cmd = m.renderCmd()
		}
	case tea.KeyMsg:
		cmd = m.handleKey(msg)
	case tickMsg:
		cmd = m.handleTick(time.Time(msg))
	}
	return m, cmd
}

func (m *Model) View() string {
	if m.width < 40 || m.height < 15 {
		return lipgloss.NewStyle().Width(max(0, m.width)).Height(max(0, m.height)).
			Align(lipgloss.Center, lipgloss.Center).
			Render("terminal too small")
	}

	l := m.computeLayout()
	m.paneSize = l

	imagePane := lipgloss.NewStyle().
		Width(l.imageCols).
		Height(l.topRows).
		Border(lipgloss.NormalBorder()).
		Render(blank(l.imageCols-2, l.topRows-2))

	actionPane := lipgloss.NewStyle().
		Width(l.actionCols).
		Height(l.topRows).
		Border(lipgloss.NormalBorder()).
		Render(m.actionList(l.actionCols-2, l.topRows-2))

	status := lipgloss.NewStyle().
		Width(m.width).
		Height(4).
		Border(lipgloss.NormalBorder()).
		Render(m.status(lipgloss.Width(imagePane) + lipgloss.Width(actionPane) - 2))

	return lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.JoinHorizontal(lipgloss.Top, imagePane, actionPane),
		status,
	)
}

func (m *Model) handleKey(msg tea.KeyMsg) tea.Cmd {
	if len(m.rec.Actions) == 0 && len(m.rec.Frames) == 0 {
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return tea.Batch(clearCmd(m.out), tea.Quit)
		}
		return nil
	}

	switch msg.String() {
	case "q", "esc", "ctrl+c":
		return tea.Batch(clearCmd(m.out), tea.Quit)
	case " ":
		m.playing = !m.playing
		m.lastTick = time.Now()
		if m.playing {
			return tickCmd()
		}
	case "left":
		m.playing = false
		m.frameMode = false
		if m.selIdx > 0 {
			m.selIdx--
		}
		m.syncFromAction()
		return m.renderCmd()
	case "right":
		m.playing = false
		m.frameMode = false
		if m.selIdx < len(m.rec.Actions)-1 {
			m.selIdx++
		}
		m.syncFromAction()
		return m.renderCmd()
	case "j":
		m.playing = false
		if m.frameIdx > 0 {
			m.frameIdx--
		}
		m.syncFromFrame()
		return m.renderCmd()
	case "k":
		m.playing = false
		if m.frameIdx < len(m.rec.Frames)-1 {
			m.frameIdx++
		}
		m.syncFromFrame()
		return m.renderCmd()
	case "g":
		m.playing = false
		m.frameMode = false
		m.selIdx = 0
		m.syncFromAction()
		return m.renderCmd()
	case "G":
		m.playing = false
		m.frameMode = false
		if len(m.rec.Actions) > 0 {
			m.selIdx = len(m.rec.Actions) - 1
		}
		m.syncFromAction()
		return m.renderCmd()
	case "+":
		m.speed = nextSpeed(m.speed, 1)
	case "-":
		m.speed = nextSpeed(m.speed, -1)
	case "r":
		m.virtTime = 0
		m.playing = true
		m.lastTick = time.Now()
		m.frameMode = true
		m.syncFromTime(m.rec.StartMs)
		return tea.Batch(m.renderCmd(), tickCmd())
	}
	return nil
}

func (m *Model) handleTick(now time.Time) tea.Cmd {
	if !m.playing {
		return nil
	}
	if m.lastTick.IsZero() {
		m.lastTick = now
	}
	elapsed := now.Sub(m.lastTick)
	m.lastTick = now

	m.virtTime += int64(float64(elapsed.Milliseconds()) * m.speed)
	if m.virtTime < 0 {
		m.virtTime = 0
	}

	absTime := m.rec.StartMs + m.virtTime
	if next, ok := m.nextEventTime(absTime); ok && next-absTime > idleGapThreshold {
		absTime = max64(m.rec.StartMs, next-idleGapPad)
		m.virtTime = absTime - m.rec.StartMs
	}

	if absTime >= m.rec.EndMs {
		absTime = m.rec.EndMs
		m.virtTime = max64(0, m.rec.EndMs-m.rec.StartMs)
		m.playing = false
	}

	m.frameMode = true
	m.syncFromTime(absTime)
	cmds := []tea.Cmd{m.renderCmd()}
	if m.playing {
		cmds = append(cmds, tickCmd())
	}
	return tea.Batch(cmds...)
}

func (m *Model) syncFromAction() {
	if len(m.rec.Actions) > 0 {
		action := m.rec.Actions[m.selIdx]
		m.virtTime = max64(0, action.StartTime-m.rec.StartMs)
		if idx := m.frameIndexForAction(action); idx >= 0 {
			m.frameIdx = idx
		}
	} else if len(m.rec.Frames) > 0 {
		m.virtTime = max64(0, m.rec.Frames[m.frameIdx].Timestamp-m.rec.StartMs)
	}
	m.imageDirty = true
}

func (m *Model) syncFromFrame() {
	if len(m.rec.Frames) == 0 {
		return
	}
	m.frameMode = true
	frame := m.rec.Frames[m.frameIdx]
	m.virtTime = max64(0, frame.Timestamp-m.rec.StartMs)
	m.selIdx = m.actionIndexForFrame(frame.Timestamp)
	m.imageDirty = true
}

func (m *Model) syncFromTime(absTime int64) {
	if len(m.rec.Actions) > 0 {
		m.selIdx = m.actionIndexForFrame(absTime)
	}
	if len(m.rec.Frames) > 0 {
		if f, ok := m.rec.FrameAt(absTime); ok {
			m.frameIdx = sort.Search(len(m.rec.Frames), func(i int) bool {
				return m.rec.Frames[i].Timestamp >= f.Timestamp
			})
		} else {
			m.frameIdx = 0
		}
	}
	m.imageDirty = true
}

func (m *Model) actionIndexForFrame(t int64) int {
	if len(m.rec.Actions) == 0 {
		return 0
	}
	best := 0
	foundContaining := false
	for i, action := range m.rec.Actions {
		end := action.EndTime
		if end == 0 {
			end = action.StartTime
		}
		if t >= action.StartTime && t <= end {
			if !foundContaining || action.StartTime >= m.rec.Actions[best].StartTime {
				best = i
				foundContaining = true
			}
			continue
		}
		if !foundContaining && action.StartTime <= t {
			best = i
		}
	}
	return best
}

func (m *Model) frameIndexForAction(action recording.Action) int {
	if len(m.rec.Frames) == 0 {
		return -1
	}
	end := action.EndTime
	if end == 0 {
		end = action.StartTime
	}
	best := -1
	for i, frame := range m.rec.Frames {
		if frame.Timestamp >= action.StartTime && frame.Timestamp <= end {
			best = i
		}
	}
	if best >= 0 {
		return best
	}
	for i, frame := range m.rec.Frames {
		if frame.Timestamp > action.StartTime {
			return i
		}
	}
	for i := len(m.rec.Frames) - 1; i >= 0; i-- {
		if m.rec.Frames[i].Timestamp <= action.StartTime {
			return i
		}
	}
	return 0
}

func (m *Model) selectedImage() (imageSelection, bool) {
	if m.frameMode && len(m.rec.Frames) > 0 {
		frame := m.rec.Frames[m.frameIdx]
		var action recording.Action
		if len(m.rec.Actions) > 0 {
			action = m.rec.Actions[m.selIdx]
		}
		return imageSelection{sha: frame.SHA1, action: action, frame: frame, frameSize: image.Pt(frame.Width, frame.Height)}, true
	}
	if len(m.rec.Actions) > 0 {
		action := m.rec.Actions[m.selIdx]
		if _, hasBox := m.rec.Boxes[action.CallID]; hasBox && action.BeforeImageSHA != "" {
			return imageSelection{sha: action.BeforeImageSHA, action: action}, true
		}
		if action.AfterImageSHA != "" {
			return imageSelection{sha: action.AfterImageSHA, action: action}, true
		}
		if idx := m.frameIndexForAction(action); idx >= 0 {
			frame := m.rec.Frames[idx]
			return imageSelection{sha: frame.SHA1, action: action, frame: frame, frameSize: image.Pt(frame.Width, frame.Height)}, true
		}
	}
	if len(m.rec.Frames) > 0 {
		frame := m.rec.Frames[m.frameIdx]
		return imageSelection{sha: frame.SHA1, frame: frame, frameSize: image.Pt(frame.Width, frame.Height)}, true
	}
	return imageSelection{}, false
}

type imageSelection struct {
	sha       string
	action    recording.Action
	frame     recording.Frame
	frameSize image.Point
}

func (m *Model) renderCmd() tea.Cmd {
	if m.width < 40 || m.height < 15 {
		return clearCmd(m.out)
	}
	l := m.computeLayout()
	m.paneSize = l

	selection, ok := m.selectedImage()
	if !ok || selection.sha == "" {
		return nil
	}

	overlayID := "none"
	box, hasBox := m.rec.Boxes[selection.action.CallID]
	if hasBox {
		overlayID = selection.action.CallID
	}

	cacheKey := selection.sha + ":" + overlayID
	pngBytes, ok := m.imgCache[cacheKey]
	if !ok {
		data, contentType, err := m.rec.Resource(selection.sha)
		if err != nil {
			m.err = err
			return nil
		}
		if hasBox {
			pngBytes, err = overlay.DrawBox(data, contentType, box, selection.frameSize)
		} else {
			pngBytes, err = overlay.EncodePNG(data, contentType)
		}
		if err != nil {
			m.err = err
			return nil
		}
		m.imgCache[cacheKey] = pngBytes
	}

	cfg, err := png.DecodeConfig(bytes.NewReader(pngBytes))
	if err != nil {
		m.err = err
		return nil
	}
	displayOrigin, displayCells := m.fitImage(l, cfg.Width, cfg.Height)
	imageKey := fmt.Sprintf("%s:%d,%d:%dx%d", cacheKey, displayOrigin.Row, displayOrigin.Col, displayCells.Cols, displayCells.Rows)
	if imageKey == m.lastImageKey {
		m.err = nil
		m.imageDirty = false
		return nil
	}

	m.err = nil
	m.imageDirty = false
	m.lastImageKey = imageKey
	return func() tea.Msg {
		_ = kitty.Clear(m.out)
		_ = kitty.Display(m.out, pngBytes, displayOrigin, displayCells)
		return nil
	}
}

func (m *Model) fitImage(l layout, imageWidth, imageHeight int) (kitty.Cell, kitty.Size) {
	if imageWidth <= 0 || imageHeight <= 0 {
		return l.imageOrigin, l.imageCells
	}

	cols := l.imageCells.Cols
	rows := int(math.Round(float64(cols*m.cellSize.Width*imageHeight) / float64(imageWidth*m.cellSize.Height)))
	if rows < 1 {
		rows = 1
	}
	if rows > l.imageCells.Rows {
		rows = l.imageCells.Rows
		cols = int(math.Round(float64(rows*m.cellSize.Height*imageWidth) / float64(imageHeight*m.cellSize.Width)))
		if cols < 1 {
			cols = 1
		}
	}
	if cols > l.imageCells.Cols {
		cols = l.imageCells.Cols
	}

	origin := kitty.Cell{
		Row: l.imageOrigin.Row + max(0, (l.imageCells.Rows-rows)/2),
		Col: l.imageOrigin.Col + max(0, (l.imageCells.Cols-cols)/2),
	}
	return origin, kitty.Size{Cols: cols, Rows: rows}
}

func (m *Model) computeLayout() layout {
	topRows := max(8, m.height-6)
	imageCols := int(math.Round(float64(m.width) * 0.70))
	imageCols = clamp(imageCols, 20, m.width-12)
	actionCols := max(12, m.width-imageCols)
	imageRows := max(1, topRows-2)
	imageContentCols := max(1, imageCols-2)
	return layout{
		imageOrigin: kitty.Cell{Row: 2, Col: 2},
		imageCells:  kitty.Size{Cols: imageContentCols, Rows: imageRows},
		imageCols:   imageCols,
		imageRows:   imageRows,
		actionCols:  actionCols,
		topRows:     topRows,
	}
}

func (m *Model) actionList(width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	lines := []string{"Actions"}
	if len(m.rec.Actions) == 0 {
		lines = append(lines, "No actions")
		return strings.Join(padLines(lines, height, width), "\n")
	}
	visible := max(1, height-1)
	start := clamp(m.selIdx-visible/2, 0, max(0, len(m.rec.Actions)-visible))
	for i := start; i < len(m.rec.Actions) && len(lines) < height; i++ {
		action := m.rec.Actions[i]
		prefix := strings.Repeat("  ", action.Depth)
		if i == m.selIdx {
			prefix += "> "
		} else {
			prefix += "  "
		}
		line := trimRunes(prefix+action.Title, width)
		if i == m.selIdx {
			line = lipgloss.NewStyle().Reverse(true).Render(padRight(line, width))
		}
		lines = append(lines, line)
	}
	return strings.Join(padLines(lines, height, width), "\n")
}

func (m *Model) status(width int) string {
	if width < 1 {
		width = m.width - 2
	}
	total := max64(0, m.rec.EndMs-m.rec.StartMs)
	cur := clamp64(m.virtTime, 0, total)
	play := " "
	if m.playing {
		play = ">"
	}
	action := ""
	if len(m.rec.Actions) > 0 {
		action = m.rec.Actions[m.selIdx].Title
		if params := summarizeParams(m.rec.Actions[m.selIdx].Params); params != "" {
			action += "  " + params
		}
	}
	if m.err != nil {
		action = "render error: " + m.err.Error()
	}

	lines := []string{
		fmt.Sprintf("%s %s / %s  %.1fx", play, formatDuration(cur), formatDuration(total), m.speed),
		trimRunes(action, width),
		progressBar(cur, total, width),
		"space=play/pause  left/right=step  j/k=frame  +/-=speed  q=quit",
	}
	for i := range lines {
		lines[i] = trimRunes(lines[i], width)
	}
	return strings.Join(lines, "\n")
}

func (m *Model) nextEventTime(absTime int64) (int64, bool) {
	var next int64
	ok := false
	for _, action := range m.rec.Actions {
		if action.StartTime > absTime && (!ok || action.StartTime < next) {
			next = action.StartTime
			ok = true
		}
	}
	currentSHA := ""
	if len(m.rec.Frames) > 0 {
		currentSHA = m.rec.Frames[m.frameIdx].SHA1
	}
	for _, frame := range m.rec.Frames {
		if frame.Timestamp > absTime && frame.SHA1 != currentSHA && (!ok || frame.Timestamp < next) {
			next = frame.Timestamp
			ok = true
		}
	}
	return next, ok
}

func tickCmd() tea.Cmd {
	return tea.Tick(tickInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func clearCmd(out io.Writer) tea.Cmd {
	return func() tea.Msg {
		_ = kitty.Clear(out)
		return nil
	}
}

func nextSpeed(current float64, direction int) float64 {
	speeds := []float64{0.5, 1, 2, 4}
	idx := 1
	for i, speed := range speeds {
		if speed == current {
			idx = i
			break
		}
	}
	idx = (idx + direction + len(speeds)) % len(speeds)
	return speeds[idx]
}

func summarizeParams(params map[string]any) string {
	if len(params) == 0 {
		return ""
	}
	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, key := range keys {
		if b.Len() > 0 {
			b.WriteString(" ")
		}
		fmt.Fprintf(&b, "%s=%q", key, params[key])
		if b.Len() > 120 {
			break
		}
	}
	return b.String()
}

func progressBar(cur, total int64, width int) string {
	if width < 1 {
		return ""
	}
	if width > 80 {
		width = 80
	}
	if total <= 0 {
		return strings.Repeat("░", width)
	}
	filled := int(float64(cur) / float64(total) * float64(width))
	filled = clamp(filled, 0, width)
	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}

func formatDuration(ms int64) string {
	if ms < 0 {
		ms = 0
	}
	min := ms / 60000
	sec := (ms % 60000) / 1000
	rem := ms % 1000
	return fmt.Sprintf("%02d:%02d.%03d", min, sec, rem)
}

func blank(width, height int) string {
	lines := make([]string, max(0, height))
	line := strings.Repeat(" ", max(0, width))
	for i := range lines {
		lines[i] = line
	}
	return strings.Join(lines, "\n")
}

func padLines(lines []string, height, width int) []string {
	out := make([]string, height)
	for i := 0; i < height; i++ {
		if i < len(lines) {
			out[i] = padRight(trimRunes(lines[i], width), width)
		} else {
			out[i] = strings.Repeat(" ", width)
		}
	}
	return out
}

func padRight(s string, width int) string {
	if lipgloss.Width(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-lipgloss.Width(s))
}

func trimRunes(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= width {
		return s
	}
	var b bytes.Buffer
	for _, r := range s {
		if lipgloss.Width(b.String()+string(r)) > width-1 {
			break
		}
		b.WriteRune(r)
	}
	if width > 1 {
		b.WriteString("…")
	}
	return b.String()
}

func clamp(v, lo, hi int) int {
	if hi < lo {
		return lo
	}
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func clamp64(v, lo, hi int64) int64 {
	if hi < lo {
		return lo
	}
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
