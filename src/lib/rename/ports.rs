use super::model::{RenameError, RenameOperation};
use std::path::PathBuf;

pub trait RenameService {
    fn find_files(&self, mask: &str) -> Result<Vec<PathBuf>, RenameError>;

    fn prepare_rename_operations(
        &self,
        files: Vec<PathBuf>,
        template: &str,
        context: &TemplateContext,
        series_mode: bool,
    ) -> Result<Vec<RenameOperation>, RenameError>;
}

#[derive(Debug, Default, Clone)]
pub struct TemplateContext {
    pub name: Option<String>,
    pub season: Option<u32>,
    pub language: Option<String>,
    pub separator: Option<String>,
}

impl TemplateContext {
    pub fn new() -> Self {
        Self::default()
    }

    pub fn with_name(mut self, name: Option<String>) -> Self {
        self.name = name;
        self
    }

    pub fn with_season(mut self, season: Option<u32>) -> Self {
        self.season = season;
        self
    }

    pub fn with_language(mut self, language: Option<String>) -> Self {
        self.language = language;
        self
    }

    pub fn with_separator(mut self, separator: Option<String>) -> Self {
        self.separator = separator;
        self
    }
}
