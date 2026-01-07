use anyhow::{Context, Result};
use log::debug;
use std::fs;
use tera::{Context as TeraContext, Tera};

/// Default hardcoded template
const DEFAULT_TEMPLATE: &str = r#"Please, translate subtitle to {{ language }} language.

Rules:
- Save format: [INDEX] text
- Translate ALL lines, don't skip anything
- Don't add comments, only translation

"#;

/// Render template with language substitution
/// If template_file is None, uses hardcoded default template
/// If template_file is Some, loads template from file
pub fn render_template(template_file: Option<&str>, language: &str) -> Result<String> {
    let template_content = match template_file {
        Some(path) => {
            debug!("loading template from file: {}", path);
            fs::read_to_string(path)
                .with_context(|| format!("failed to read template file: {}", path))?
        }
        None => {
            debug!("using hardcoded default template");
            DEFAULT_TEMPLATE.to_string()
        }
    };

    let mut tera = Tera::default();
    tera.add_raw_template("prompt", &template_content)
        .context("failed to parse template")?;

    let mut context = TeraContext::new();
    context.insert("language", language);

    tera.render("prompt", &context)
        .context("failed to render template")
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::fs;
    use tempfile::TempDir;

    #[test]
    fn test_render_default_template() {
        let result = render_template(None, "Russian").unwrap();
        assert!(result.contains("Please, translate subtitle to Russian language."));
        assert!(result.contains("Save format: [INDEX] text"));
        assert!(result.contains("Translate ALL lines"));
    }

    #[test]
    fn test_render_custom_template_from_file() {
        let temp_dir = TempDir::new().unwrap();
        let template_path = temp_dir.path().join("test.md");
        fs::write(&template_path, "Translate to {{ language }} language.").unwrap();

        let result = render_template(Some(template_path.to_str().unwrap()), "Russian").unwrap();
        assert_eq!(result, "Translate to Russian language.");
    }

    #[test]
    fn test_render_template_multiple_placeholders() {
        let temp_dir = TempDir::new().unwrap();
        let template_path = temp_dir.path().join("test.md");
        fs::write(
            &template_path,
            "{{ language }} to {{ language }} translation",
        )
        .unwrap();

        let result = render_template(Some(template_path.to_str().unwrap()), "english").unwrap();
        assert_eq!(result, "english to english translation");
    }

    #[test]
    fn test_render_template_no_placeholder() {
        let temp_dir = TempDir::new().unwrap();
        let template_path = temp_dir.path().join("test.md");
        fs::write(&template_path, "No placeholder here").unwrap();

        let result = render_template(Some(template_path.to_str().unwrap()), "russian").unwrap();
        assert_eq!(result, "No placeholder here");
    }
}
