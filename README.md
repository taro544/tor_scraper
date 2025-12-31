# Tor Scraper

A concurrent web scraper for .onion sites using Tor network.

## Features

- Scrapes .onion websites through Tor proxy
- Captures HTML content, screenshots, and extracts links
- Concurrent processing with 5 workers
- Automatic Tor connection verification

## Requirements

- Go 1.19 or higher
- Tor service running on port 9050
- Chrome/Chromium browser installed

## Installation

```bash
git clone https://github.com/taro544/tor_scraper.git
cd tor_scraper
go mod download
```

## Usage

1. Make sure Tor is running:
```bash
# macOS/Linux
tor
```

2. Edit `targets.yaml` with your target URLs:
```yaml
urls:
  - example1.onion
  - example2.onion
```

3. Build and run:
```bash
go build
./tor-scanner
```

## Output

All scraped data is saved in `scraped_data/`:
- `htmls/` - HTML content
- `images/` - Full-page screenshots
- `urls/` - Extracted links

## Configuration

Edit constants in `main.go`:
- `TorProxyServer` - Tor SOCKS5 proxy address (default: 127.0.0.1:9050)
- `WorkerCount` - Number of concurrent workers (default: 5)
- `BaseOutputDir` - Output directory (default: scraped_data)

## License

MIT
