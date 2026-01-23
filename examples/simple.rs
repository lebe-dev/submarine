use chrono::Duration;
use lib::subtitle::model::{Subtitle, SubtitleIndex, SubtitleText, SubtitleTimestamp};
use lib::subtitle::ports::SubtitleService;
use lib::subtitle::service::SubRipService;
use std::fs;

fn main() -> Result<(), Box<dyn std::error::Error>> {
    // The library's service is file-based.
    // We'll create a service instance that operates in the 'examples' directory.
    let service = SubRipService::new("examples");
    let sample_filename = "sample.srt";
    let output_filename = "output.srt";
    let sample_filepath = format!("examples/{}", sample_filename);
    let output_filepath = format!("examples/{}", output_filename);

    // Example 1: Create a sample file and load subtitles from it
    println!("--- Loading subtitles from a file ---");
    let srt_content = "1\n00:00:03,000 --> 00:00:04,000\nThis is a sample subtitle.\n\n2\n00:00:05,000 --> 00:00:06,000\nThis is another one.\n";
    fs::write(&sample_filepath, srt_content)?;

    let mut subtitles = service.get_all(sample_filename)?;

    println!("Loaded {} subtitles from {}:", subtitles.len(), sample_filename);
    for sub in &subtitles {
        println!("{}", sub.to_string().trim());
        println!("---");
    }

    // Example 2: Create a subtitle programmatically and add it to our vector
    println!("\n--- Creating a new subtitle and adding it ---");
    let new_subtitle = Subtitle::new(
        SubtitleIndex::try_new(3)?,
        SubtitleTimestamp::try_new(Duration::milliseconds(7000))?,
        SubtitleTimestamp::try_new(Duration::milliseconds(8000))?,
        SubtitleText::try_new("This is a new subtitle, created in code.".to_string())?,
    )?;
    println!("Created new subtitle:\n{}", new_subtitle);
    subtitles.push(new_subtitle);


    // Example 3: Save the modified list of subtitles to a new file
    println!("\n--- Saving subtitles to a file ---");
    service.write_all(output_filename, &subtitles)?;
    println!("Subtitles saved to {}", output_filepath);

    // Verify the content of the output file
    let output_content = fs::read_to_string(&output_filepath)?;
    println!("\n--- Content of {} ---", output_filename);
    println!("{}", output_content.trim());
    println!("---");


    // Clean up the created files
    fs::remove_file(&sample_filepath)?;
    fs::remove_file(&output_filepath)?;
    println!("\nCleaned up temporary files.");

    Ok(())
}
