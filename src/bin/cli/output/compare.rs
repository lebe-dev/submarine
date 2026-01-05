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

/// Application state for the TUI comparison view
pub struct App {
    subtitles1: Vec<Subtitle>,
    subtitles2: Vec<Subtitle>,
    filename1: String,
    filename2: String,
    selected_index: usize, // 0-based index for current selection
    scroll_offset: usize,  // Top visible subtitle index for scrolling
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
                match key.code {
                    KeyCode::Esc | KeyCode::Char('q') => {
                        debug!("exit key pressed");
                        return Ok(());
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
                    _ => {}
                }
            }
        }
    }
}

/// Render the UI layout
fn ui(f: &mut Frame, app: &mut App) {
    // Calculate viewport
    let viewport_height = calculate_viewport_height(f.area());
    update_scroll_offset(app, viewport_height);

    let chunks = Layout::default()
        .direction(Direction::Horizontal)
        .constraints([Constraint::Percentage(50), Constraint::Percentage(50)])
        .split(f.area());

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
            .bg(Color::Yellow)
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
    ];

    ListItem::new(content).style(style)
}

/// Format a placeholder item for missing subtitles
fn format_placeholder_item(is_selected: bool) -> ListItem<'static> {
    let style = if is_selected {
        Style::default()
            .bg(Color::Yellow)
            .fg(Color::Black)
            .add_modifier(Modifier::BOLD)
    } else {
        Style::default().fg(Color::DarkGray)
    };

    ListItem::new("(no subtitle at this index)").style(style)
}

/// Calculate how many subtitles fit in the viewport
fn calculate_viewport_height(size: Rect) -> usize {
    // Account for borders (2 lines) and each subtitle takes ~2-3 lines
    // Conservative estimate: 3 lines per subtitle
    let available_height = size.height.saturating_sub(2);
    (available_height / 3).max(1) as usize
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
