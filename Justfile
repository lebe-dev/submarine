init:
  cargo install cargo-llvm-cov

run-examples:
  cargo run --example subtitle_usage

build:
  cargo build --bin sm

test:
  cargo test --bin sm
  cargo test --lib

test-all:
  cargo test --bin sm
  cargo test --lib
  # Integration tests
  cargo test --test '*'

# Run tests with coverage report (HTML output)
coverage:
  cargo llvm-cov --all-features --workspace --html

# Run tests with coverage report (terminal output)
coverage-text:
  cargo llvm-cov --all-features --workspace

# Run tests with coverage and open HTML report in browser
coverage-open:
  cargo llvm-cov --all-features --workspace --open

# Clean coverage artifacts
coverage-clean:
  cargo llvm-cov clean --workspace
