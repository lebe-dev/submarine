use super::model::{RenameError, RenameOperation};
use super::ports::{RenameService, TemplateContext};
use log::{debug, info};
use std::fs;
use std::path::PathBuf;
use tera::Tera;

pub struct FileRenameService {
    base_dir: PathBuf,
}

impl FileRenameService {
    pub fn new(base_dir: PathBuf) -> Self {
        Self { base_dir }
    }
}

impl RenameService for FileRenameService {
    fn find_files(&self, mask: &str) -> Result<Vec<PathBuf>, RenameError> {
        info!("searching for files with mask: {}", mask);

        let mask_lower = mask.to_lowercase();
        let mut found_files = Vec::new();

        let entries = fs::read_dir(&self.base_dir).map_err(|e| {
            RenameError::IoError(std::io::Error::new(
                e.kind(),
                format!("failed to read directory: {}", e),
            ))
        })?;

        for entry in entries {
            let entry = entry?;
            let path = entry.path();

            if !path.is_file() {
                continue;
            }

            if let Some(ext) = path.extension() {
                if ext.to_str() != Some("srt") {
                    continue;
                }
            } else {
                continue;
            }

            if let Some(filename) = path.file_name().and_then(|n| n.to_str())
                && filename.to_lowercase().contains(&mask_lower)
            {
                debug!("found file: {:?}", path);
                found_files.push(path);
            }
        }

        if found_files.is_empty() {
            return Err(RenameError::NoFilesFound(mask.to_string()));
        }

        // Sort alphabetically
        found_files.sort();

        info!("found {} files", found_files.len());
        Ok(found_files)
    }

    fn prepare_rename_operations(
        &self,
        files: Vec<PathBuf>,
        template: &str,
        context: &TemplateContext,
        series_mode: bool,
    ) -> Result<Vec<RenameOperation>, RenameError> {
        info!("preparing rename operations");
        debug!("series_mode: {}", series_mode);
        debug!("template: {}", template);

        let mut tera = Tera::default();
        tera.add_raw_template("filename", template)
            .map_err(|e| RenameError::InvalidTemplate(e.to_string()))?;

        let mut operations = Vec::new();

        for (index, file_path) in files.iter().enumerate() {
            let mut tera_context = tera::Context::new();

            if let Some(name) = &context.name {
                tera_context.insert("name", name);
            }

            if let Some(season) = context.season {
                tera_context.insert("season", &format!("{:02}", season));
            }

            if let Some(language) = &context.language {
                tera_context.insert("language", language);
            }

            if let Some(separator) = &context.separator {
                tera_context.insert("separator", separator);
            }

            if series_mode {
                let episode_number = index + 1;
                tera_context.insert("episode", &format!("{:02}", episode_number));
                debug!("series mode: file {} -> episode {}", index, episode_number);
            }

            let new_name = tera
                .render("filename", &tera_context)
                .map_err(|e| RenameError::TemplateError(e.to_string()))?;

            debug!("new name for {:?}: {}", file_path, new_name);

            let new_path = self.base_dir.join(&new_name);
            let collision = new_path.exists() && new_path != *file_path;

            if collision {
                debug!("collision detected for: {}", new_name);
            }

            operations.push(RenameOperation::new(file_path.clone(), new_name, collision));
        }

        info!("prepared {} rename operations", operations.len());
        Ok(operations)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::fs::File;
    use tempfile::TempDir;

    #[test]
    fn test_find_files_case_insensitive() {
        let temp_dir = TempDir::new().unwrap();
        let base_path = temp_dir.path();

        // Create test files
        File::create(base_path.join("Resident.Alien.S01E01.srt")).unwrap();
        File::create(base_path.join("resident.alien.s01e02.srt")).unwrap();
        File::create(base_path.join("Other.Show.S01E01.srt")).unwrap();

        let service = FileRenameService::new(base_path.to_path_buf());
        let files = service.find_files("resident").unwrap();

        assert_eq!(files.len(), 2);
    }

    #[test]
    fn test_find_files_only_srt() {
        let temp_dir = TempDir::new().unwrap();
        let base_path = temp_dir.path();

        File::create(base_path.join("test.srt")).unwrap();
        File::create(base_path.join("test.txt")).unwrap();
        File::create(base_path.join("test.mp4")).unwrap();

        let service = FileRenameService::new(base_path.to_path_buf());
        let files = service.find_files("test").unwrap();

        assert_eq!(files.len(), 1);
        assert!(files[0].to_str().unwrap().ends_with("test.srt"));
    }

    #[test]
    fn test_find_files_sorted() {
        let temp_dir = TempDir::new().unwrap();
        let base_path = temp_dir.path();

        File::create(base_path.join("c.srt")).unwrap();
        File::create(base_path.join("a.srt")).unwrap();
        File::create(base_path.join("b.srt")).unwrap();

        let service = FileRenameService::new(base_path.to_path_buf());
        let files = service.find_files(".srt").unwrap();

        assert_eq!(files.len(), 3);
        assert!(files[0].file_name().unwrap().to_str().unwrap() == "a.srt");
        assert!(files[1].file_name().unwrap().to_str().unwrap() == "b.srt");
        assert!(files[2].file_name().unwrap().to_str().unwrap() == "c.srt");
    }

    #[test]
    fn test_prepare_rename_operations_series_mode() {
        let temp_dir = TempDir::new().unwrap();
        let base_path = temp_dir.path();

        let file1 = base_path.join("old1.srt");
        let file2 = base_path.join("old2.srt");
        File::create(&file1).unwrap();
        File::create(&file2).unwrap();

        let service = FileRenameService::new(base_path.to_path_buf());
        let files = vec![file1.clone(), file2.clone()];

        let context = TemplateContext::new()
            .with_name(Some("Test Show".to_string()))
            .with_season(Some(1))
            .with_separator(Some(".".to_string()));

        let template = "{{ name }}.S{{ season }}E{{ episode }}.srt";
        let operations = service
            .prepare_rename_operations(files, template, &context, true)
            .unwrap();

        assert_eq!(operations.len(), 2);
        assert_eq!(operations[0].new_name, "Test Show.S01E01.srt");
        assert_eq!(operations[1].new_name, "Test Show.S01E02.srt");
    }

    #[test]
    fn test_prepare_rename_operations_collision_detection() {
        let temp_dir = TempDir::new().unwrap();
        let base_path = temp_dir.path();

        let file1 = base_path.join("old.srt");
        let existing = base_path.join("new.srt");
        File::create(&file1).unwrap();
        File::create(&existing).unwrap();

        let service = FileRenameService::new(base_path.to_path_buf());
        let files = vec![file1.clone()];

        let context = TemplateContext::new();
        let template = "new.srt";

        let operations = service
            .prepare_rename_operations(files, template, &context, false)
            .unwrap();

        assert_eq!(operations.len(), 1);
        assert!(operations[0].collision);
    }

    #[test]
    fn test_template_with_optional_variables() {
        let temp_dir = TempDir::new().unwrap();
        let base_path = temp_dir.path();

        let file1 = base_path.join("old.srt");
        File::create(&file1).unwrap();

        let service = FileRenameService::new(base_path.to_path_buf());
        let files = vec![file1.clone()];

        let context = TemplateContext::new()
            .with_name(Some("Show".to_string()))
            .with_separator(Some(".".to_string()));

        let template = "{{ name }}{{ separator }}subtitle.srt";
        let operations = service
            .prepare_rename_operations(files, template, &context, false)
            .unwrap();

        assert_eq!(operations.len(), 1);
        assert_eq!(operations[0].new_name, "Show.subtitle.srt");
    }
}
