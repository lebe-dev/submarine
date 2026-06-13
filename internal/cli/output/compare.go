// Package output holds the TUI comparison view. It is a 1-to-1 port of the
// Rust `src/bin/cli/output/compare.rs`, mapping the ratatui immediate-mode UI
// onto bubbletea's Model/Update/View with lipgloss styling. The manual
// scroll/viewport math, navigation keys, jump/search input modes, side-by-side
// 50/50 panes, status/help line, error overlay, truncation, and selected-row
// styling are reproduced exactly.
package output

import (
	"fmt"
	"log/slog"
	"math/rand/v2"
	"strconv"
	"strings"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/lebe-dev/submarine/pkg/subtitle"
)

// appMode is the application mode for the state machine. Port of the Rust
// `enum AppMode`.
type appMode int

const (
	// modeNormal -> AppMode::Normal
	modeNormal appMode = iota
	// modeJumpInput -> AppMode::JumpInput (: followed by number input)
	modeJumpInput
	// modeSearchInput -> AppMode::SearchInput (/ followed by search text input)
	modeSearchInput
)

// App is the application state for the TUI comparison view, ported onto a
// bubbletea Model. Port of the Rust `struct App`.
type App struct {
	subtitles1 []subtitle.Subtitle
	subtitles2 []subtitle.Subtitle
	filename1  string
	filename2  string

	selectedIndex int // 0-based index for current selection
	scrollOffset  int // Top visible subtitle index for scrolling

	mode        appMode
	inputBuffer string // Buffer for : and / input
	// inputError holds an error message for invalid input. nil means no error
	// (was Option<String>).
	inputError *string

	searchMatches []int // Indices of subtitles matching search
	// currentMatchIndex is the current position in searchMatches. nil means no
	// active match (was Option<usize>).
	currentMatchIndex *int

	shouldQuit               bool
	shouldCenterOnNextRender bool // Flag to center view after jump
	pendingGPress            bool // Track first 'g' press for 'gg' sequence

	// width/height track the terminal size reported by bubbletea
	// (WindowSizeMsg). ratatui derives this from the backend each draw; here we
	// store the last reported size.
	width  int
	height int
}

// NewApp creates a new App instance with subtitle data. Port of `App::new`.
func NewApp(
	subtitles1 []subtitle.Subtitle,
	filename1 string,
	subtitles2 []subtitle.Subtitle,
	filename2 string,
) *App {
	return &App{
		subtitles1:               subtitles1,
		subtitles2:               subtitles2,
		filename1:                filename1,
		filename2:                filename2,
		selectedIndex:            0,
		scrollOffset:             0,
		mode:                     modeNormal,
		inputBuffer:              "",
		inputError:               nil,
		searchMatches:            nil,
		currentMatchIndex:        nil,
		shouldQuit:               false,
		shouldCenterOnNextRender: false,
		pendingGPress:            false,
	}
}

// maxIndex returns the maximum valid index (handles different file lengths).
// Port of `App::max_index`.
func (a *App) maxIndex() int {
	maxLen := maxInt(len(a.subtitles1), len(a.subtitles2))
	return saturatingSub(maxLen, 1)
}

// next moves selection down. Port of `App::next`.
func (a *App) next() {
	max := a.maxIndex()
	if a.selectedIndex < max {
		a.selectedIndex++
	}
}

// previous moves selection up. Port of `App::previous`.
func (a *App) previous() {
	if a.selectedIndex > 0 {
		a.selectedIndex--
	}
}

// openJumpInput opens jump input mode. Port of `App::open_jump_input`.
func (a *App) openJumpInput() {
	a.mode = modeJumpInput
	a.inputBuffer = ""
	a.inputError = nil
}

// openSearchInput opens search input mode. Port of `App::open_search_input`.
func (a *App) openSearchInput() {
	a.mode = modeSearchInput
	a.inputBuffer = ""
	a.inputError = nil
}

// closeInput closes input mode and returns to normal. Port of `App::close_input`.
func (a *App) closeInput() {
	a.mode = modeNormal
	a.inputBuffer = ""
	a.inputError = nil
}

// inputChar adds a character to the input buffer. Port of `App::input_char`.
func (a *App) inputChar(c rune) {
	a.inputBuffer += string(c)
	a.inputError = nil // Clear error on new input
}

// inputBackspace removes the last character from the input buffer. Port of
// `App::input_backspace`.
func (a *App) inputBackspace() {
	// Rust's String::pop removes the last char (rune), not the last byte.
	if a.inputBuffer != "" {
		r := []rune(a.inputBuffer)
		a.inputBuffer = string(r[:len(r)-1])
	}
	a.inputError = nil
}

// tryJump validates input and jumps to subtitle number. Port of `App::try_jump`.
func (a *App) tryJump() {
	if a.inputBuffer == "" {
		a.inputError = strPtr("please enter a subtitle number")
		return
	}

	userIndex, err := strconv.ParseUint(a.inputBuffer, 10, 64)
	if err != nil {
		a.inputError = strPtr("invalid number")
		return
	}

	if userIndex == 0 {
		a.inputError = strPtr("subtitle numbers start at 1")
		return
	}

	zeroBasedIndex := int(userIndex) - 1
	if zeroBasedIndex <= a.maxIndex() {
		slog.Debug(fmt.Sprintf("jumping to subtitle %d (index %d)", userIndex, zeroBasedIndex))
		a.selectedIndex = zeroBasedIndex
		a.shouldCenterOnNextRender = true
		a.closeInput()
		return
	}

	maxSubtitle := a.maxIndex() + 1
	a.inputError = strPtr(fmt.Sprintf("subtitle %d not found (max: %d)", userIndex, maxSubtitle))
}

// trySearch performs search and jumps to first match. Port of `App::try_search`.
func (a *App) trySearch() {
	if a.inputBuffer == "" {
		a.inputError = strPtr("please enter search text")
		return
	}

	searchText := strings.ToLower(a.inputBuffer)
	a.searchMatches = nil

	maxLen := maxInt(len(a.subtitles1), len(a.subtitles2))
	for i := 0; i < maxLen; i++ {
		found := false

		if i < len(a.subtitles1) &&
			strings.Contains(strings.ToLower(a.subtitles1[i].Text.Value()), searchText) {
			found = true
		}

		if !found && i < len(a.subtitles2) &&
			strings.Contains(strings.ToLower(a.subtitles2[i].Text.Value()), searchText) {
			found = true
		}

		if found {
			a.searchMatches = append(a.searchMatches, i)
		}
	}

	if len(a.searchMatches) == 0 {
		a.inputError = strPtr(fmt.Sprintf("no matches found for '%s'", a.inputBuffer))
		a.currentMatchIndex = nil
		return
	}

	slog.Debug(fmt.Sprintf("found %d matches for '%s'", len(a.searchMatches), a.inputBuffer))
	a.currentMatchIndex = intPtr(0)
	a.selectedIndex = a.searchMatches[0]
	a.shouldCenterOnNextRender = true
	a.closeInput()
}

// nextMatch jumps to the next search match. Port of `App::next_match`.
func (a *App) nextMatch() {
	if a.currentMatchIndex == nil || len(a.searchMatches) == 0 {
		return
	}
	current := *a.currentMatchIndex
	next := (current + 1) % len(a.searchMatches)
	a.currentMatchIndex = intPtr(next)
	a.selectedIndex = a.searchMatches[next]
	a.shouldCenterOnNextRender = true
	slog.Debug(fmt.Sprintf("jumping to next match %d of %d", next+1, len(a.searchMatches)))
}

// prevMatch jumps to the previous search match. Port of `App::prev_match`.
func (a *App) prevMatch() {
	if a.currentMatchIndex == nil || len(a.searchMatches) == 0 {
		return
	}
	current := *a.currentMatchIndex
	var prev int
	if current == 0 {
		prev = len(a.searchMatches) - 1
	} else {
		prev = current - 1
	}
	a.currentMatchIndex = intPtr(prev)
	a.selectedIndex = a.searchMatches[prev]
	a.shouldCenterOnNextRender = true
	slog.Debug(fmt.Sprintf("jumping to previous match %d of %d", prev+1, len(a.searchMatches)))
}

// jumpToFirst jumps to the first subtitle (index 0). Port of `App::jump_to_first`.
func (a *App) jumpToFirst() {
	slog.Debug("jumping to first subtitle")
	a.selectedIndex = 0
	a.shouldCenterOnNextRender = true
}

// jumpToLast jumps to the last subtitle. Port of `App::jump_to_last`.
func (a *App) jumpToLast() {
	max := a.maxIndex()
	slog.Debug(fmt.Sprintf("jumping to last subtitle (index %d)", max))
	a.selectedIndex = max
	a.shouldCenterOnNextRender = true
}

// jumpToRandom jumps to a random subtitle. Port of `App::jump_to_random`.
func (a *App) jumpToRandom() {
	maxLen := maxInt(len(a.subtitles1), len(a.subtitles2))
	if maxLen == 0 {
		return // No subtitles to jump to
	}

	max := a.maxIndex()
	// Rust: rng.gen_range(0..=max) — inclusive range [0, max].
	randomIndex := rand.IntN(max + 1)
	slog.Debug(fmt.Sprintf("jumping to random subtitle (index %d)", randomIndex))
	a.selectedIndex = randomIndex
	a.shouldCenterOnNextRender = true
}

// --- bubbletea Model implementation ---

// Init is the bubbletea initialization hook. No initial command.
func (a *App) Init() tea.Cmd {
	return nil
}

// Update handles incoming messages. It mirrors the Rust `run_app` event loop:
// dispatch key events by mode, and quit when should_quit is set. Port of the
// per-mode input handlers.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = m.Width
		a.height = m.Height
		return a, nil
	case tea.KeyMsg:
		switch a.mode {
		case modeNormal:
			a.handleNormalModeInput(m)
		case modeJumpInput:
			a.handleJumpInputMode(m)
		case modeSearchInput:
			a.handleSearchInputMode(m)
		}
		if a.shouldQuit {
			return a, tea.Quit
		}
		return a, nil
	}
	return a, nil
}

// handleNormalModeInput handles keyboard input in normal mode. Port of
// `handle_normal_mode_input`.
func (a *App) handleNormalModeInput(key tea.KeyMsg) {
	if a.pendingGPress {
		a.pendingGPress = false

		if isCharKey(key, 'g') {
			// 'gg' sequence detected - jump to first
			a.jumpToFirst()
			return
		}
		// Any other key cancels the pending 'g'
		// Fall through to handle current key
	}

	switch {
	case key.Type == tea.KeyEsc || isCharKey(key, 'q'):
		slog.Debug("exit key pressed")
		a.shouldQuit = true
	case key.Type == tea.KeyDown || isCharKey(key, 'j'):
		slog.Debug(fmt.Sprintf("navigating down to index %d", a.selectedIndex+1))
		a.next()
	case key.Type == tea.KeyUp || isCharKey(key, 'k'):
		if a.selectedIndex > 0 {
			slog.Debug(fmt.Sprintf("navigating up to index %d", a.selectedIndex-1))
		}
		a.previous()
	case isCharKey(key, 'g'):
		// First 'g' press - set pending state
		slog.Debug("g pressed, waiting for second key")
		a.pendingGPress = true
	case isCharKey(key, 'G'):
		// Shift+G - jump to last
		a.jumpToLast()
	case isCharKey(key, 'r'):
		// r - jump to random
		a.jumpToRandom()
	case isCharKey(key, ':'):
		// Open jump input mode
		slog.Debug("opening jump input mode")
		a.openJumpInput()
	case isCharKey(key, '/'):
		// Open search input mode
		slog.Debug("opening search input mode")
		a.openSearchInput()
	case isCharKey(key, 'n'):
		// Next search match
		a.nextMatch()
	case isCharKey(key, 'N'):
		// Previous search match
		a.prevMatch()
	}
}

// handleJumpInputMode handles keyboard input in jump input mode. Port of
// `handle_jump_input_mode`.
func (a *App) handleJumpInputMode(key tea.KeyMsg) {
	switch {
	case key.Type == tea.KeyEsc:
		slog.Debug("closing jump input (cancelled)")
		a.closeInput()
	case key.Type == tea.KeyEnter:
		slog.Debug(fmt.Sprintf("attempting to jump to subtitle: %s", a.inputBuffer))
		a.tryJump()
	case key.Type == tea.KeyBackspace:
		a.inputBackspace()
	case key.Type == tea.KeyRunes && len(key.Runes) == 1 && isASCIIDigit(key.Runes[0]):
		a.inputChar(key.Runes[0])
	}
}

// handleSearchInputMode handles keyboard input in search input mode. Port of
// `handle_search_input_mode`.
func (a *App) handleSearchInputMode(key tea.KeyMsg) {
	switch {
	case key.Type == tea.KeyEsc:
		slog.Debug("closing search input (cancelled)")
		a.closeInput()
	case key.Type == tea.KeyEnter:
		slog.Debug(fmt.Sprintf("attempting to search for: %s", a.inputBuffer))
		a.trySearch()
	case key.Type == tea.KeyBackspace:
		a.inputBackspace()
	case key.Type == tea.KeySpace:
		// crossterm reports space as KeyCode::Char(' '); bubbletea reports a
		// dedicated KeySpace. Treat it as a regular character.
		a.inputChar(' ')
	case key.Type == tea.KeyRunes:
		for _, c := range key.Runes {
			a.inputChar(c)
		}
	}
}

// View renders the UI. Port of the ratatui `ui` function, reproducing the
// vertical split (main area + 1-line status), the centering/scrolling logic,
// the horizontal 50/50 panes, and the status/error line.
func (a *App) View() string {
	// Layout: vertical split into a main comparison area (Min(0)) and a 1-line
	// help/status row. The total area is the terminal size reported by
	// bubbletea.
	totalHeight := a.height
	totalWidth := a.width
	if totalHeight < 1 {
		totalHeight = 1
	}
	if totalWidth < 1 {
		totalWidth = 1
	}

	// main area gets all rows except the final status line.
	mainHeight := totalHeight - 1
	if mainHeight < 0 {
		mainHeight = 0
	}

	viewportHeight := calculateViewportHeight(mainHeight)

	if a.shouldCenterOnNextRender {
		a.centerSelectedItem(viewportHeight)
		a.shouldCenterOnNextRender = false
	} else {
		a.updateScrollOffset(viewportHeight)
	}

	// Horizontal split: 50% / 50%. ratatui's Percentage(50)/Percentage(50)
	// splits the available width; left pane gets ceil, right gets the rest.
	leftWidth := (totalWidth + 1) / 2
	rightWidth := totalWidth - leftWidth

	leftPane := renderSubtitlePane(
		leftWidth, mainHeight,
		a.subtitles1, a.filename1,
		a.selectedIndex, a.scrollOffset, viewportHeight,
	)
	rightPane := renderSubtitlePane(
		rightWidth, mainHeight,
		a.subtitles2, a.filename2,
		a.selectedIndex, a.scrollOffset, viewportHeight,
	)

	mainArea := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)

	// Render status/input line at bottom.
	statusText := a.statusText()

	statusStyle := lipgloss.NewStyle()
	if a.mode == modeJumpInput || a.mode == modeSearchInput {
		statusStyle = statusStyle.Foreground(colorYellow)
	} else {
		statusStyle = statusStyle.Foreground(colorDarkGray)
	}

	statusLine := statusStyle.Render(statusText)

	// Show error message if present (overlays on top of status line).
	if a.inputError != nil {
		errorStyle := lipgloss.NewStyle().Foreground(colorRed)
		statusLine = errorStyle.Render(fmt.Sprintf(" Error: %s", *a.inputError))
	}

	return lipgloss.JoinVertical(lipgloss.Left, mainArea, statusLine)
}

// statusText builds the status/help line text. Port of the `match app.mode`
// block in the Rust `ui` function.
func (a *App) statusText() string {
	switch {
	case a.mode == modeNormal && a.pendingGPress:
		return " Waiting for second key... (g: first | other: cancel)"
	case a.mode == modeNormal:
		if a.currentMatchIndex != nil {
			if len(a.searchMatches) != 0 {
				return fmt.Sprintf(
					" j/k: move | /: search | n/N: next/prev match (%d/%d) | :: jump | gg: first | G: last | r: random | q: quit",
					*a.currentMatchIndex+1,
					len(a.searchMatches),
				)
			}
			return " j/k: move | /: search | :: jump | gg: first | G: last | r: random | q: quit"
		}
		return " j/k: move | /: search | :: jump | gg: first | G: last | r: random | q: quit"
	case a.mode == modeJumpInput:
		inputDisplay := a.inputBuffer
		if inputDisplay == "" {
			inputDisplay = "_"
		}
		return fmt.Sprintf(":%s", inputDisplay)
	case a.mode == modeSearchInput:
		inputDisplay := a.inputBuffer
		if inputDisplay == "" {
			inputDisplay = "_"
		}
		return fmt.Sprintf("/%s", inputDisplay)
	}
	return ""
}

// renderSubtitlePane renders a single subtitle pane (left or right). Port of
// `render_subtitle_pane`. The ratatui Block with all borders and a centered
// title is reproduced with a lipgloss bordered box; the visible subtitle items
// (from scroll_offset to end_index) are rendered inside.
func renderSubtitlePane(
	width, height int,
	subtitles []subtitle.Subtitle,
	filename string,
	selectedIndex, scrollOffset, viewportHeight int,
) string {
	// Block::default().borders(ALL).title(" {filename} ") with a white border.
	block := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(colorWhite).
		Width(saturatingSub(width, 2)).
		Height(saturatingSub(height, 2))

	if len(subtitles) == 0 {
		content := lipgloss.NewStyle().Foreground(colorDarkGray).Render("no subtitles found in this file")
		return titledBox(block, filename, content)
	}

	// end_index = (scroll_offset + viewport_height)
	//                 .min(subtitles.len().max(selected_index + 1))
	endIndex := minInt(scrollOffset+viewportHeight, maxInt(len(subtitles), selectedIndex+1))

	var lines []string
	for i := scrollOffset; i < endIndex; i++ {
		isSelected := i == selectedIndex

		if i < len(subtitles) {
			lines = append(lines, formatSubtitleItem(subtitles[i], isSelected))
		} else {
			lines = append(lines, formatPlaceholderItem(isSelected))
		}
	}

	content := strings.Join(lines, "\n")
	return titledBox(block, filename, content)
}

// titledBox renders a lipgloss bordered box with the filename centered in the
// top border, mirroring ratatui's Block title (" {filename} "). lipgloss does
// not support titled borders directly, so we render the box and splice the
// title into the top border row.
func titledBox(block lipgloss.Style, filename, content string) string {
	rendered := block.Render(content)
	title := fmt.Sprintf(" %s ", filename)

	lines := strings.Split(rendered, "\n")
	if len(lines) == 0 {
		return rendered
	}

	top := lines[0]
	topRunes := []rune(top)
	titleRunes := []rune(title)
	// Splice the title starting just after the left corner (index 1), matching
	// ratatui's default left-aligned title position.
	if len(topRunes) >= 1 {
		for i := 0; i < len(titleRunes) && 1+i < len(topRunes); i++ {
			topRunes[1+i] = titleRunes[i]
		}
		lines[0] = string(topRunes)
	}

	return strings.Join(lines, "\n")
}

// formatSubtitleItem formats a subtitle as a list item. Port of
// `format_subtitle_item`. Produces three lines: header, indented text, and a
// blank spacer line (matching the ratatui Line vec).
func formatSubtitleItem(sub subtitle.Subtitle, isSelected bool) string {
	style := lipgloss.NewStyle()
	if isSelected {
		style = style.Background(colorBlue).Foreground(colorBlack).Bold(true)
	}

	header := fmt.Sprintf(
		"%s. %s --> %s",
		sub.Index.String(),
		subtitle.FormatTimestamp(sub.StartTime.Value()),
		subtitle.FormatTimestamp(sub.EndTime.Value()),
	)

	text := sub.Text.Value()
	var displayText string
	if utf8.RuneCountInString(text) > 60 {
		truncated := takeRunes(text, 57)
		displayText = fmt.Sprintf("%s...", truncated)
	} else {
		displayText = text
	}

	displayText = strings.ReplaceAll(displayText, "\n", " ")

	headerLine := style.Render(header)
	textLine := style.Render(fmt.Sprintf("  %s", displayText))
	spacer := style.Render("")

	return strings.Join([]string{headerLine, textLine, spacer}, "\n")
}

// formatPlaceholderItem formats a placeholder item for missing subtitles. Port
// of `format_placeholder_item`.
func formatPlaceholderItem(isSelected bool) string {
	style := lipgloss.NewStyle()
	if isSelected {
		style = style.Background(colorBlue).Foreground(colorBlack).Bold(true)
	} else {
		style = style.Foreground(colorDarkGray)
	}
	return style.Render("(no subtitle at this index)")
}

// calculateViewportHeight calculates how many subtitles fit in the viewport.
// Port of `calculate_viewport_height`. `height` is the height of the main
// comparison area (ratatui's main_chunks[0].height).
func calculateViewportHeight(height int) int {
	// Account for borders (2 lines) and ~4 lines per subtitle.
	availableHeight := saturatingSub(height, 2)
	return maxInt(availableHeight/4, 1)
}

// updateScrollOffset updates the scroll offset to keep the selected item
// visible. Port of `update_scroll_offset`.
func (a *App) updateScrollOffset(viewportHeight int) {
	// If selected item is above viewport, scroll up.
	if a.selectedIndex < a.scrollOffset {
		a.scrollOffset = a.selectedIndex
		return
	}
	// If selected item is below viewport, scroll down.
	if a.selectedIndex >= a.scrollOffset+viewportHeight {
		a.scrollOffset = saturatingSub(a.selectedIndex, viewportHeight-1)
	}
}

// centerSelectedItem centers the selected item in the viewport (used after a
// jump). Port of `center_selected_item`.
func (a *App) centerSelectedItem(viewportHeight int) {
	maxLen := maxInt(len(a.subtitles1), len(a.subtitles2))

	halfViewport := viewportHeight / 2

	if a.selectedIndex >= halfViewport {
		desiredOffset := saturatingSub(a.selectedIndex, halfViewport)
		maxOffset := saturatingSub(maxLen, viewportHeight)
		a.scrollOffset = minInt(desiredOffset, maxOffset)
		return
	}
	a.scrollOffset = 0
}

// RunTUI is the public API: run the TUI comparison interface. It initializes
// the terminal (alt screen + mouse capture, matching the crossterm setup),
// runs the bubbletea event loop, and ensures cleanup. Port of `run_tui`.
func RunTUI(
	subtitles1 []subtitle.Subtitle,
	filename1 string,
	subtitles2 []subtitle.Subtitle,
	filename2 string,
) error {
	app := NewApp(subtitles1, filename1, subtitles2, filename2)

	program := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseCellMotion())

	_, err := program.Run()
	return err
}

// --- helpers ---

// ratatui/crossterm color → lipgloss ANSI-16 palette index. ratatui emits
// standard SGR codes: Black=30, Red=31, Yellow=33, Blue=34, DarkGray=90,
// White=97. Mapped to the corresponding ANSI palette indices.
var (
	colorBlack    = lipgloss.Color("0")
	colorRed      = lipgloss.Color("1")
	colorYellow   = lipgloss.Color("3")
	colorBlue     = lipgloss.Color("4")
	colorDarkGray = lipgloss.Color("8")  // bright black (SGR 90)
	colorWhite    = lipgloss.Color("15") // bright white (SGR 97)
)

// strPtr returns a pointer to s.
func strPtr(s string) *string { return &s }

// intPtr returns a pointer to v.
func intPtr(v int) *int { return &v }

// maxInt returns the larger of a and b (Rust usize::max).
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// minInt returns the smaller of a and b (Rust usize::min).
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// saturatingSub mirrors Rust's usize::saturating_sub: clamps at 0.
func saturatingSub(a, b int) int {
	if a < b {
		return 0
	}
	return a - b
}

// takeRunes returns the first n runes of s (Rust chars().take(n).collect()).
func takeRunes(s string, n int) string {
	count := 0
	for i := range s {
		if count == n {
			return s[:i]
		}
		count++
	}
	return s
}

// isCharKey reports whether the key is a single rune matching c (was matching
// KeyCode::Char(c)).
func isCharKey(key tea.KeyMsg, c rune) bool {
	return key.Type == tea.KeyRunes && len(key.Runes) == 1 && key.Runes[0] == c
}

// isASCIIDigit mirrors Rust's char::is_ascii_digit.
func isASCIIDigit(r rune) bool {
	return r >= '0' && r <= '9'
}
