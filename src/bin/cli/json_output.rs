use crate::cli::OutputFormat;
use serde::Serialize;

#[derive(Serialize)]
struct SuccessEnvelope<'a, T: Serialize> {
    ok: bool,
    data: &'a T,
}

#[derive(Serialize)]
struct ErrorEnvelope<'a> {
    ok: bool,
    error: ErrorDetail<'a>,
}

#[derive(Serialize)]
struct ErrorDetail<'a> {
    code: &'a str,
    message: &'a str,
    #[serde(skip_serializing_if = "Option::is_none")]
    hint: Option<&'a str>,
}

/// Output a successful result in the requested format.
///
/// In text mode, `text_fn` is called to produce human-readable output.
/// In JSON mode, `data` is serialized into a `{"ok": true, "data": ...}` envelope.
pub fn output_success<T: Serialize>(format: &OutputFormat, data: &T, text_fn: impl FnOnce()) {
    match format {
        OutputFormat::Text => text_fn(),
        OutputFormat::Json => {
            let envelope = SuccessEnvelope { ok: true, data };
            println!(
                "{}",
                serde_json::to_string(&envelope).expect("failed to serialize JSON")
            );
        }
    }
}

/// Output an error in the requested format.
///
/// In text mode, error goes to stderr. In JSON mode, error goes to stdout as
/// `{"ok": false, "error": {"code": ..., "message": ..., "hint": ...}}`.
pub fn output_error(format: &OutputFormat, code: &str, message: &str, hint: Option<&str>) {
    match format {
        OutputFormat::Text => {
            eprintln!("error: {}", message);
            if let Some(h) = hint {
                eprintln!("hint: {}", h);
            }
        }
        OutputFormat::Json => {
            let envelope = ErrorEnvelope {
                ok: false,
                error: ErrorDetail {
                    code,
                    message,
                    hint,
                },
            };
            println!(
                "{}",
                serde_json::to_string(&envelope).expect("failed to serialize JSON")
            );
        }
    }
}
