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
    layout::{Alignment, Constraint, Direction, Flex, Layout, Rect},
    style::{Color, Modifier, Style},
    text::{Line, Span},
    widgets::{Block, Borders, Clear, List, ListItem, Paragraph},
};

/// Application mode for state machine
enum AppMode {
    Normal,
    JumpDialog,
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
    jump_input: String,
    jump_error: Option<String>,
    should_quit: bool,
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
            jump_input: String::new(),
            jump_error: None,
            should_quit: false,
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

    /// Open the jump dialog
    fn open_jump_dialog(&mut self) {
        self.mode = AppMode::JumpDialog;
        self.jump_input.clear();
        self.jump_error = None;
    }

    /// Close the jump dialog without jumping
    fn close_jump_dialog(&mut self) {
        self.mode = AppMode::Normal;
        self.jump_input.clear();
        self.jump_error = None;
    }

    /// Add a character to the jump input
    fn input_char(&mut self, c: char) {
        if c.is_ascii_digit() {
            self.jump_input.push(c);
            self.jump_error = None; // Clear error on new input
        }
    }

    /// Remove the last character from the jump input
    fn input_backspace(&mut self) {
        self.jump_input.pop();
        self.jump_error = None;
    }

    /// Validate input and jump to subtitle if valid
    fn try_jump(&mut self) {
        if self.jump_input.is_empty() {
            self.jump_error = Some("please enter a subtitle number".to_string());
            return;
        }

        match self.jump_input.parse::<usize>() {
            Ok(user_index) if user_index == 0 => {
                self.jump_error = Some("subtitle numbers start at 1".to_string());
            }
            Ok(user_index) => {
                let zero_based_index = user_index - 1;
                if zero_based_index <= self.max_index() {
                    debug!(
                        "jumping to subtitle {} (index {})",
                        user_index, zero_based_index
                    );
                    self.selected_index = zero_based_index;
                    self.close_jump_dialog();
                } else {
                    let max_subtitle = self.max_index() + 1;
                    self.jump_error = Some(format!(
                        "subtitle {} not found (max: {})",
                        user_index, max_subtitle
                    ));
                }
            }
            Err(_) => {
                self.jump_error = Some("invalid number".to_string());
            }
        }
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
                    AppMode::JumpDialog => handle_dialog_mode_input(app, key.code),
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
            debug!("opening jump dialog");
            app.open_jump_dialog();
        }
        _ => {}
    }
}

/// Handle keyboard input in jump dialog mode
fn handle_dialog_mode_input(app: &mut App, key_code: KeyCode) {
    match key_code {
        KeyCode::Esc => {
            debug!("closing jump dialog (cancelled)");
            app.close_jump_dialog();
        }
        KeyCode::Enter => {
            debug!("attempting to jump to subtitle: {}", app.jump_input);
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

/// Calculate a centered rectangle for the popup dialog
fn popup_area(area: Rect, percent_x: u16, percent_y: u16) -> Rect {
    let vertical = Layout::vertical([Constraint::Percentage(percent_y)]).flex(Flex::Center);
    let horizontal = Layout::horizontal([Constraint::Percentage(percent_x)]).flex(Flex::Center);
    let [area] = vertical.areas(area);
    let [area] = horizontal.areas(area);
    area
}

/// Render the jump dialog overlay
fn render_jump_dialog(f: &mut Frame, app: &App) {
    let area = popup_area(f.area(), 40, 25);

    f.render_widget(Clear, area);

    let chunks = Layout::default()
        .direction(Direction::Vertical)
        .margin(1)
        .constraints([
            Constraint::Length(1), // Title
            Constraint::Length(1), // Spacing
            Constraint::Length(1), // Input label
            Constraint::Length(3), // Input box
            Constraint::Length(1), // Error message
            Constraint::Length(1), // Spacing
            Constraint::Length(1), // Help text
        ])
        .split(area);

    let block = Block::default()
        .borders(Borders::ALL)
        .border_style(Style::default().fg(Color::White))
        .title(" Jump to Subtitle ");
    f.render_widget(block, area);

    let title = Paragraph::new("Enter subtitle number:");
    f.render_widget(title, chunks[2]);

    let input_text = if app.jump_input.is_empty() {
        "_".to_string()
    } else {
        app.jump_input.clone()
    };

    let input = Paragraph::new(input_text)
        .block(Block::default().borders(Borders::ALL))
        .style(Style::default().fg(Color::Yellow));
    f.render_widget(input, chunks[3]);

    if let Some(error) = &app.jump_error {
        let error_msg = Paragraph::new(error.as_str()).style(Style::default().fg(Color::Red));
        f.render_widget(error_msg, chunks[4]);
    }

    let help = Paragraph::new("Enter: jump | Esc: cancel")
        .style(Style::default().fg(Color::DarkGray))
        .alignment(Alignment::Center);
    f.render_widget(help, chunks[6]);
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
    update_scroll_offset(app, viewport_height);

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

    let help_text_str = match app.mode {
        AppMode::Normal => " ↑/k: up | ↓/j: down | g: jump | Esc/q: exit",
        AppMode::JumpDialog => "", // Help shown in dialog
    };
    let help_text = Paragraph::new(help_text_str).style(Style::default().fg(Color::DarkGray));
    f.render_widget(help_text, main_chunks[1]);

    if matches!(app.mode, AppMode::JumpDialog) {
        render_jump_dialog(f, app);
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
