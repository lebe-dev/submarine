# Using Submarine as a Library

Submarine is designed primarily as a command-line tool, but its core functionality is available as a library that you can integrate into your own Rust projects. This guide will walk you through the basics of using the `submarine-rs` library.

## Adding `submarine-rs` to Your Project

To use `submarine-rs` as a library, you'll first need to add it to your `Cargo.toml` file.

```toml
[dependencies]
submarine-rs = { git = "https://github.com/lebe-dev/submarine" }
```

Note: Since `submarine-rs` is not yet published on crates.io, you'll need to add it directly from its Git repository.

## Core Concepts

The library's functionality is centered around the `SubtitleService` trait, which defines the primary operations for working with subtitle files. The main implementation of this trait is `SubRipService`, which works with SubRip (.srt) files.

- **`Subtitle`**: This struct represents a single subtitle entry, including its index, start and end timestamps, and text.
- **`SubRipService`**: This is the entry point for most file-based operations. It allows you to read, write, and modify .srt files.

## Basic Usage

Here's a simple example of how to use the library to read subtitles from a file, add a new subtitle, and write the results to a new file.

First, let's look at the complete code, which can also be found in `examples/simple.rs`.

```rust
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
```

### Step-by-Step Explanation

1. **Instantiate `SubRipService`**:
   The `SubRipService` needs a base directory to work from. All filenames you provide will be relative to this path.

   ```rust
   use lib::subtitle::service::SubRipService;
   let service = SubRipService::new("examples");
   ```

2. **Reading Subtitles**:
   The `get_all` method reads a .srt file and returns a `Vec<Subtitle>` if successful.

   ```rust
   let subtitles = service.get_all("sample.srt")?;
   ```

3. **Creating a New Subtitle**:
   You can create new `Subtitle` instances programmatically. The library uses `nutype` to ensure that all data is valid (e.g., subtitle indices must be positive, text cannot be empty).

   ```rust
   use chrono::Duration;
   use lib::subtitle::model::{Subtitle, SubtitleIndex, SubtitleText, SubtitleTimestamp};

   let new_subtitle = Subtitle::new(
       SubtitleIndex::try_new(3)?,
       SubtitleTimestamp::try_new(Duration::milliseconds(7000))?,
       SubtitleTimestamp::try_new(Duration::milliseconds(8000))?,
       SubtitleText::try_new("This is a new subtitle, created in code.".to_string())?,
   )?;
   ```

4. **Writing Subtitles**:
   The `write_all` method takes a slice of `Subtitle`s and writes them to a file, overwriting it if it exists.

   ```rust
   service.write_all("output.srt", &subtitles)?;
   ```

## Further Exploration

For more advanced usage, you can explore the other methods available on the `SubtitleService` trait, such as:

- `get_by_id`: Retrieve a single subtitle by its index.
- `set`: Update an existing subtitle.
- `add`: Append a new subtitle to a file.

You can find more detailed information in the source code documentation.
