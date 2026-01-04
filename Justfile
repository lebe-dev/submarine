version := `cat Cargo.toml | grep version | head -1 | cut -d " " -f 3 | tr -d "\""`

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

release-macos: test-all
  cargo build --release --bin sm
  cp target/release/sm sm
  zip -9 -r sm-{{version}}-macos-arm64.zip sm
  rm -f sm

release-linux:
  rm -f sm
  rm -rf out
  docker build --progress=plain --platform=linux/amd64 -t submarine .
  docker run -it --rm -v $(pwd)/out:/out submarine
  cp out/sm .
  chmod +x sm
  zip -9 -r sm-{{version}}-linux-amd64.zip sm
  rm -f sm
