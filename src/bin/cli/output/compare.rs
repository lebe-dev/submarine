use lib::subtitle::model::Subtitle;
use log::debug;

use crossterm::{
    event::{self, DisableMouseCapture, EnableMouseCapture, Event, KeyCode},
    execute,
    terminal::{EnterAlternateScreen, LeaveAlternateScreen, disable_raw_mode, enable_raw_mode},
};

use ratatui::{
    Frame, Terminal,
    backend::CrosstermBackend,
    layout::{Constraint, Direction, Layout, Rect},
    style::{Color, Modifier, Style},
    text::{Line, Span},
    widgets::{Block, Borders, List, ListItem, Paragraph},
};

/// Application mode for state machine
enum AppMode {
    Normal,
    JumpInput,   // : followed by number input
    SearchInput, // / followed by search text input
}

/// Application state for the TUI comparison view
pub struct App {
    subtitles1: Vec<Subtitle>,
    subtitles2: Vec<Subtitle>,
    filename1: String,
    filename2: String,
    selected_index: usize, // 0-based index for current selection
    scroll_offset: usize,  // Top visible subtitle index for scrolling
    mode: AppMode,
    input_buffer: String,               // Buffer for : and / input
    input_error: Option<String>,        // Error message for invalid input
    search_matches: Vec<usize>,         // Indices of subtitles matching search
    current_match_index: Option<usize>, // Current position in search_matches
    should_quit: bool,
    should_center_on_next_render: bool, // Flag to center view after jump
    pending_g_press: bool,              // Track first 'g' press for 'gg' sequence
}

impl App {
    /// Create a new App instance with subtitle data
    pub fn new(
        subtitles1: Vec<Subtitle>,
        filename1: String,
        subtitles2: Vec<Subtitle>,
        filename2: String,
    ) -> Self {
        Self {
            subtitles1,
            subtitles2,
            filename1,
            filename2,
            selected_index: 0,
            scroll_offset: 0,
            mode: AppMode::Normal,
            input_buffer: String::new(),
            input_error: None,
            search_matches: Vec::new(),
            current_match_index: None,
            should_quit: false,
            should_center_on_next_render: false,
            pending_g_press: false,
        }
    }

    /// Get the maximum valid index (handles different file lengths)
    fn max_index(&self) -> usize {
        let max_len = self.subtitles1.len().max(self.subtitles2.len());
        max_len.saturating_sub(1)
    }

    /// Move selection down
    fn next(&mut self) {
        let max = self.max_index();
        if self.selected_index < max {
            self.selected_index += 1;
        }
    }

    /// Move selection up
    fn previous(&mut self) {
        if self.selected_index > 0 {
            self.selected_index -= 1;
        }
    }

    /// Open jump input mode
    fn open_jump_input(&mut self) {
        self.mode = AppMode::JumpInput;
        self.input_buffer.clear();
        self.input_error = None;
    }

    /// Open search input mode
    fn open_search_input(&mut self) {
        self.mode = AppMode::SearchInput;
        self.input_buffer.clear();
        self.input_error = None;
    }

    /// Close input mode and return to normal
    fn close_input(&mut self) {
        self.mode = AppMode::Normal;
        self.input_buffer.clear();
        self.input_error = None;
    }

    /// Add a character to the input buffer
    fn input_char(&mut self, c: char) {
        self.input_buffer.push(c);
        self.input_error = None; // Clear error on new input
    }

    /// Remove the last character from the input buffer
    fn input_backspace(&mut self) {
        self.input_buffer.pop();
        self.input_error = None;
    }

    /// Validate input and jump to subtitle number
    fn try_jump(&mut self) {
        if self.input_buffer.is_empty() {
            self.input_error = Some("please enter a subtitle number".to_string());
            return;
        }

        match self.input_buffer.parse::<usize>() {
            Ok(user_index) if user_index == 0 => {
                self.input_error = Some("subtitle numbers start at 1".to_string());
            }
            Ok(user_index) => {
                let zero_based_index = user_index - 1;
                if zero_based_index <= self.max_index() {
                    debug!(
                        "jumping to subtitle {} (index {})",
                        user_index, zero_based_index
                    );
                    self.selected_index = zero_based_index;
                    self.should_center_on_next_render = true;
                    self.close_input();
                } else {
                    let max_subtitle = self.max_index() + 1;
                    self.input_error = Some(format!(
                        "subtitle {} not found (max: {})",
                        user_index, max_subtitle
                    ));
                }
            }
            Err(_) => {
                self.input_error = Some("invalid number".to_string());
            }
        }
    }

    /// Perform search and jump to first match
    fn try_search(&mut self) {
        if self.input_buffer.is_empty() {
            self.input_error = Some("please enter search text".to_string());
            return;
        }

        let search_text = self.input_buffer.to_lowercase();
        self.search_matches.clear();

        let max_len = self.subtitles1.len().max(self.subtitles2.len());
        for i in 0..max_len {
            let mut found = false;

            if let Some(sub) = self.subtitles1.get(i) {
                if sub.text.as_ref().to_lowercase().contains(&search_text) {
                    found = true;
                }
            }

            if !found {
                if let Some(sub) = self.subtitles2.get(i) {
                    if sub.text.as_ref().to_lowercase().contains(&search_text) {
                        found = true;
                    }
                }
            }

            if found {
                self.search_matches.push(i);
            }
        }

        if self.search_matches.is_empty() {
            self.input_error = Some(format!("no matches found for '{}'", self.input_buffer));
            self.current_match_index = None;
        } else {
            debug!(
                "found {} matches for '{}'",
                self.search_matches.len(),
                self.input_buffer
            );
            self.current_match_index = Some(0);
            self.selected_index = self.search_matches[0];
            self.should_center_on_next_render = true;
            self.close_input();
        }
    }

    /// Jump to next search match
    fn next_match(&mut self) {
        if let Some(current) = self.current_match_index {
            if !self.search_matches.is_empty() {
                let next = (current + 1) % self.search_matches.len();
                self.current_match_index = Some(next);
                self.selected_index = self.search_matches[next];
                self.should_center_on_next_render = true;
                debug!(
                    "jumping to next match {} of {}",
                    next + 1,
                    self.search_matches.len()
                );
            }
        }
    }

    /// Jump to previous search match
    fn prev_match(&mut self) {
        if let Some(current) = self.current_match_index {
            if !self.search_matches.is_empty() {
                let prev = if current == 0 {
                    self.search_matches.len() - 1
                } else {
                    current - 1
                };
                self.current_match_index = Some(prev);
                self.selected_index = self.search_matches[prev];
                self.should_center_on_next_render = true;
                debug!(
                    "jumping to previous match {} of {}",
                    prev + 1,
                    self.search_matches.len()
                );
            }
        }
    }

    /// Jump to first subtitle (index 0)
    fn jump_to_first(&mut self) {
        debug!("jumping to first subtitle");
        self.selected_index = 0;
        self.should_center_on_next_render = true;
    }

    /// Jump to last subtitle
    fn jump_to_last(&mut self) {
        let max = self.max_index();
        debug!("jumping to last subtitle (index {})", max);
        self.selected_index = max;
        self.should_center_on_next_render = true;
    }

    /// Jump to random subtitle
    fn jump_to_random(&mut self) {
        use rand::Rng;

        let max_len = self.subtitles1.len().max(self.subtitles2.len());
        if max_len == 0 {
            return; // No subtitles to jump to
        }

        let max = self.max_index();
        let mut rng = rand::thread_rng();
        let random_index = rng.gen_range(0..=max);
        debug!("jumping to random subtitle (index {})", random_index);
        self.selected_index = random_index;
        self.should_center_on_next_render = true;
    }
}

/// Public API: Run the TUI comparison interface
///
/// Initializes the terminal, runs the event loop, and ensures proper cleanup.
pub fn run_tui(
    subtitles1: Vec<Subtitle>,
    filename1: String,
    subtitles2: Vec<Subtitle>,
    filename2: String,
) -> anyhow::Result<()> {
    enable_raw_mode()?;
    let mut stdout = std::io::stdout();
    execute!(stdout, EnterAlternateScreen, EnableMouseCapture)?;
    let backend = CrosstermBackend::new(stdout);
    let mut terminal = Terminal::new(backend)?;

    let mut app = App::new(subtitles1, filename1, subtitles2, filename2);

    let res = run_app(&mut terminal, &mut app);

    disable_raw_mode()?;
    execute!(
        terminal.backend_mut(),
        LeaveAlternateScreen,
        DisableMouseCapture
    )?;
    terminal.show_cursor()?;

    res
}

/// Event loop for handling user input and rendering
fn run_app<B: ratatui::backend::Backend>(
    terminal: &mut Terminal<B>,
    app: &mut App,
) -> anyhow::Result<()>
where
    <B as ratatui::backend::Backend>::Error: std::error::Error + Send + Sync + 'static,
{
    loop {
        terminal.draw(|f| ui(f, app))?;

        if event::poll(std::time::Duration::from_millis(100))? {
            if let Event::Key(key) = event::read()? {
                match app.mode {
                    AppMode::Normal => handle_normal_mode_input(app, key.code),
                    AppMode::JumpInput => handle_jump_input_mode(app, key.code),
                    AppMode::SearchInput => handle_search_input_mode(app, key.code),
                }
            }
        }

        if app.should_quit {
            return Ok(());
        }
    }
}

/// Handle keyboard input in normal mode
fn handle_normal_mode_input(app: &mut App, key_code: KeyCode) {
    if app.pending_g_press {
        app.pending_g_press = false;

        match key_code {
            KeyCode::Char('g') => {
                // 'gg' sequence detected - jump to first
                app.jump_to_first();
                return;
            }
            _ => {
                // Any other key cancels the pending 'g'
                // Fall through to handle current key
            }
        }
    }

    match key_code {
        KeyCode::Esc | KeyCode::Char('q') => {
            debug!("exit key pressed");
            app.should_quit = true;
        }
        KeyCode::Down | KeyCode::Char('j') => {
            debug!("navigating down to index {}", app.selected_index + 1);
            app.next();
        }
        KeyCode::Up | KeyCode::Char('k') => {
            if app.selected_index > 0 {
                debug!("navigating up to index {}", app.selected_index - 1);
            }
            app.previous();
        }
        KeyCode::Char('g') => {
            // First 'g' press - set pending state
            debug!("g pressed, waiting for second key");
            app.pending_g_press = true;
        }
        KeyCode::Char('G') => {
            // Shift+G - jump to last
            app.jump_to_last();
        }
        KeyCode::Char('r') => {
            // r - jump to random
            app.jump_to_random();
        }
        KeyCode::Char(':') => {
            // Open jump input mode
            debug!("opening jump input mode");
            app.open_jump_input();
        }
        KeyCode::Char('/') => {
            // Open search input mode
            debug!("opening search input mode");
            app.open_search_input();
        }
        KeyCode::Char('n') => {
            // Next search match
            app.next_match();
        }
        KeyCode::Char('N') => {
            // Previous search match
            app.prev_match();
        }
        _ => {}
    }
}

/// Handle keyboard input in jump input mode
fn handle_jump_input_mode(app: &mut App, key_code: KeyCode) {
    match key_code {
        KeyCode::Esc => {
            debug!("closing jump input (cancelled)");
            app.close_input();
        }
        KeyCode::Enter => {
            debug!("attempting to jump to subtitle: {}", app.input_buffer);
            app.try_jump();
        }
        KeyCode::Char(c) if c.is_ascii_digit() => {
            app.input_char(c);
        }
        KeyCode::Backspace => {
            app.input_backspace();
        }
        _ => {}
    }
}

/// Handle keyboard input in search input mode
fn handle_search_input_mode(app: &mut App, key_code: KeyCode) {
    match key_code {
        KeyCode::Esc => {
            debug!("closing search input (cancelled)");
            app.close_input();
        }
        KeyCode::Enter => {
            debug!("attempting to search for: {}", app.input_buffer);
            app.try_search();
        }
        KeyCode::Char(c) => {
            app.input_char(c);
        }
        KeyCode::Backspace => {
            app.input_backspace();
        }
        _ => {}
    }
}

/// Render the UI layout
fn ui(f: &mut Frame, app: &mut App) {
    let main_chunks = Layout::default()
        .direction(Direction::Vertical)
        .constraints([
            Constraint::Min(0),    // Main comparison area
            Constraint::Length(1), // Help text (1 line)
        ])
        .split(f.area());

    let viewport_height = calculate_viewport_height(main_chunks[0]);

    if app.should_center_on_next_render {
        center_selected_item(app, viewport_height);
        app.should_center_on_next_render = false;
    } else {
        update_scroll_offset(app, viewport_height);
    }

    let chunks = Layout::default()
        .direction(Direction::Horizontal)
        .constraints([Constraint::Percentage(50), Constraint::Percentage(50)])
        .split(main_chunks[0]);

    render_subtitle_pane(
        f,
        chunks[0],
        &app.subtitles1,
        &app.filename1,
        app.selected_index,
        app.scroll_offset,
        viewport_height,
    );

    render_subtitle_pane(
        f,
        chunks[1],
        &app.subtitles2,
        &app.filename2,
        app.selected_index,
        app.scroll_offset,
        viewport_height,
    );

    // Render status/input line at bottom
    let status_text = match app.mode {
        AppMode::Normal if app.pending_g_press => {
            " Waiting for second key... (g: first | other: cancel)".to_string()
        }
        AppMode::Normal => {
            if let Some(match_idx) = app.current_match_index {
                if !app.search_matches.is_empty() {
                    format!(
                        " j/k: move | /: search | n/N: next/prev match ({}/{}) | :: jump | gg: first | G: last | r: random | q: quit",
                        match_idx + 1,
                        app.search_matches.len()
                    )
                } else {
                    " j/k: move | /: search | :: jump | gg: first | G: last | r: random | q: quit"
                        .to_string()
                }
            } else {
                " j/k: move | /: search | :: jump | gg: first | G: last | r: random | q: quit"
                    .to_string()
            }
        }
        AppMode::JumpInput => {
            let input_display = if app.input_buffer.is_empty() {
                "_".to_string()
            } else {
                app.input_buffer.clone()
            };
            format!(":{}", input_display)
        }
        AppMode::SearchInput => {
            let input_display = if app.input_buffer.is_empty() {
                "_".to_string()
            } else {
                app.input_buffer.clone()
            };
            format!("/{}", input_display)
        }
    };

    let status_style = if matches!(app.mode, AppMode::JumpInput | AppMode::SearchInput) {
        Style::default().fg(Color::Yellow)
    } else {
        Style::default().fg(Color::DarkGray)
    };

    let status_line = Paragraph::new(status_text).style(status_style);
    f.render_widget(status_line, main_chunks[1]);

    // Show error message if present
    if let Some(error) = &app.input_error {
        let error_line =
            Paragraph::new(format!(" Error: {}", error)).style(Style::default().fg(Color::Red));
        // Overlay error on top of status line
        f.render_widget(error_line, main_chunks[1]);
    }
}

/// Render a single subtitle pane (left or right)
fn render_subtitle_pane(
    f: &mut Frame,
    area: Rect,
    subtitles: &[Subtitle],
    filename: &str,
    selected_index: usize,
    scroll_offset: usize,
    viewport_height: usize,
) {
    let block = Block::default()
        .borders(Borders::ALL)
        .title(format!(" {} ", filename))
        .border_style(Style::default().fg(Color::White));

    if subtitles.is_empty() {
        let text = Paragraph::new("no subtitles found in this file")
            .block(block)
            .style(Style::default().fg(Color::DarkGray));
        f.render_widget(text, area);
        return;
    }

    let end_index = (scroll_offset + viewport_height).min(subtitles.len().max(selected_index + 1));

    let items: Vec<ListItem> = (scroll_offset..end_index)
        .map(|i| {
            let is_selected = i == selected_index;

            if let Some(subtitle) = subtitles.get(i) {
                format_subtitle_item(subtitle, is_selected)
            } else {
                format_placeholder_item(is_selected)
            }
        })
        .collect();

    let list = List::new(items).block(block);
    f.render_widget(list, area);
}

/// Format a subtitle as a ListItem
fn format_subtitle_item(subtitle: &Subtitle, is_selected: bool) -> ListItem<'static> {
    let style = if is_selected {
        Style::default()
            .bg(Color::Blue)
            .fg(Color::Black)
            .add_modifier(Modifier::BOLD)
    } else {
        Style::default()
    };

    let header = format!(
        "{}. {} --> {}",
        subtitle.index,
        Subtitle::format_timestamp(subtitle.start_time.as_ref()),
        Subtitle::format_timestamp(subtitle.end_time.as_ref())
    );

    let text = subtitle.text.as_ref();
    let display_text = if text.chars().count() > 60 {
        let truncated: String = text.chars().take(57).collect();
        format!("{}...", truncated)
    } else {
        text.to_string()
    };

    let display_text = display_text.replace('\n', " ");

    let content = vec![
        Line::from(Span::styled(header.clone(), style)),
        Line::from(Span::styled(format!("  {}", display_text), style)),
        Line::from(""), // Vertical spacing between subtitles
    ];

    ListItem::new(content).style(style)
}

/// Format a placeholder item for missing subtitles
fn format_placeholder_item(is_selected: bool) -> ListItem<'static> {
    let style = if is_selected {
        Style::default()
            .bg(Color::Blue)
            .fg(Color::Black)
            .add_modifier(Modifier::BOLD)
    } else {
        Style::default().fg(Color::DarkGray)
    };

    ListItem::new("(no subtitle at this index)").style(style)
}

/// Calculate how many subtitles fit in the viewport
fn calculate_viewport_height(size: Rect) -> usize {
    // Account for borders (2 lines) and each subtitle takes ~3-4 lines
    // Conservative estimate: 4 lines per subtitle (header + text + spacing)
    let available_height = size.height.saturating_sub(2);
    (available_height / 4).max(1) as usize
}

/// Update scroll offset to keep selected item visible
fn update_scroll_offset(app: &mut App, viewport_height: usize) {
    // If selected item is above viewport, scroll up
    if app.selected_index < app.scroll_offset {
        app.scroll_offset = app.selected_index;
    }
    // If selected item is below viewport, scroll down
    else if app.selected_index >= app.scroll_offset + viewport_height {
        app.scroll_offset = app.selected_index.saturating_sub(viewport_height - 1);
    }
}

/// Center the selected item in viewport (used after jump)
fn center_selected_item(app: &mut App, viewport_height: usize) {
    let max_len = app.subtitles1.len().max(app.subtitles2.len());

    let half_viewport = viewport_height / 2;

    if app.selected_index >= half_viewport {
        let desired_offset = app.selected_index.saturating_sub(half_viewport);

        let max_offset = max_len.saturating_sub(viewport_height);
        app.scroll_offset = desired_offset.min(max_offset);
    } else {
        app.scroll_offset = 0;
    }
}
