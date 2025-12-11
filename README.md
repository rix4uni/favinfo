## FavInfo

A powerful reconnaissance tool for extracting and analyzing favicons from websites. FavInfo extracts favicons, calculates multiple hash formats (Murmur3, MD5, SHA256), identifies technologies, and generates search engine queries for threat intelligence and asset discovery.

## Features

- 🔍 **Automatic Favicon Extraction**: Scrapes favicons from HTML or falls back to `/favicon.ico`
- 🎯 **Multiple Hash Algorithms**: Calculates Murmur3, MD5, and SHA256 hashes (MD5/SHA256 in JSON mode only)
- 🛠️ **Technology Identification**: Matches favicon hashes against fingerprint database
- 🔎 **Search Engine Integration**: Generates queries for Shodan, FOFA, Censys, ZoomEye, and Quake (JSON mode)
- 📊 **Flexible Output**: Simplified format for batch processing or detailed JSON format
- ⚡ **High Performance**: Concurrent processing with configurable workers (default 50) for scanning thousands of URLs efficiently
- 🔒 **Security Focused**: Supports custom User-Agents and TLS configuration
- 📥 **Auto-Download**: Automatically downloads fingerprint database from GitHub if not found

## Installation
```
go install github.com/rix4uni/favinfo@latest
```

## Download prebuilt binaries
```
wget https://github.com/rix4uni/favinfo/releases/download/v0.0.7/favinfo-linux-amd64-0.0.7.tgz
tar -xvzf favinfo-linux-amd64-0.0.7.tgz
rm -rf favinfo-linux-amd64-0.0.7.tgz
mv favinfo ~/go/bin/favinfo
```
Or download [binary release](https://github.com/rix4uni/favinfo/releases) for your platform.

## Compile from source
```
git clone --depth 1 https://github.com/rix4uni/favinfo.git
cd favinfo; go install
```

## Usage
```
Usage of favinfo:
      --concurrent int       Number of URLs to process concurrently (default 50)
      --fingerprint string   Path to the fingerprint.json file (default: $HOME/.config/favinfo/fingerprint.json or ./fingerprint.json)
      --json                 Output results in JSON format
      --silent               Silent mode.
      --source               Enable source output for where the url coming from scraped or added /favicon.ico
      --timeout duration     Set the HTTP request timeout duration (default 30s)
  -H, --user-agent string    Set the User-Agent header for HTTP requests (default "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36")
      --version              Print the version of the tool and exit.
```

## Usage Examples
### Basic Usage
```yaml
echo "https://www.google.com" | favinfo
```

### Multiple URLs
```yaml
cat subs.txt | favinfo
```

### With Custom Flags
```yaml
echo "example.com" | favinfo --timeout 30s --user-agent "Custom Agent" --source
```

### JSON Output
```yaml
echo "https://www.google.com" | favinfo --json
```

## Command Line Options

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--concurrent` | | Number of URLs to process concurrently | `50` |
| `--timeout` | | HTTP request timeout | `30s` |
| `--source` | | Show source of favicon URLs | `false` |
| `--user-agent` | `-H` | Set custom User-Agent | Mozilla/5.0... |
| `--fingerprint` | | Path to fingerprint.json | Auto-detected |
| `--json` | | Output in JSON format (full details) | `false` |
| `--silent` | | Silent mode (no banner) | `false` |
| `--version` | | Print version and exit | `false` |

**Note**: Default output is simplified format (`URL [hash1, hash2]`). Use `--json` flag for detailed output with all hashes and search engine queries.

## Fingerprint Database

FavInfo uses a fingerprint database to identify technologies based on favicon hashes. The tool automatically looks for the database in:

1. `$HOME/.config/favinfo/fingerprint.json` (recommended)
2. `./fingerprint.json` (current directory)

If the database is not found in either location, FavInfo will automatically:
- Create the `~/.config/favinfo/` directory if it doesn't exist
- Download `fingerprint.json` from the GitHub repository
- Save it to `~/.config/favinfo/fingerprint.json`

The download message is shown unless `--silent` flag is used.

### Custom Fingerprint Location
```yaml
echo "example.com" | favinfo --fingerprint /path/to/custom/fingerprint.json
```

## Output Examples

### Simplified Output (Default)
The default output format is optimized for batch processing thousands of URLs. It shows only the URL and Murmur3 hash(es):

```yaml
https://www.google.com [708578229]
https://www.bing.com [-583637433]
https://example.com [123456789, 987654321]
```

This format:
- Only calculates Murmur3 hash (fastest processing)
- Skips MD5/SHA256 calculations and fingerprint lookups
- Perfect for scanning large lists of subdomains

### JSON Output
```json
{
  "input_url": "https://www.google.com",
  "favicon_url": "https://www.google.com/favicon.ico",
  "murmur_hash": 708578229,
  "md5_hash": "f3418a443e7d841097c714d69ec4bcb8",
  "sha256_hash": "6da5620880159634213e197fafca1dde0272153be3e4590818533fab8d040770",
  "technology": "google",
  "search_queries": {
    "shodan": "http.favicon.hash:708578229",
    "fofa": "icon_hash=\"708578229\"",
    "censys": "services.http.response.favicons.md5_hash=\"f3418a443e7d841097c714d69ec4bcb8\"",
    "zoomeye": "iconhash:708578229",
    "quake": "favicon.hash:708578229"
  }
}
```

## Supported Search Engines

- **Shodan**: `http.favicon.hash:123456789`
- **FOFA**: `icon_hash="123456789"`
- **Censys**: MD5 and SHA256 hash searches
- **ZoomEye**: `iconhash:123456789`
- **Quake**: `favicon.hash:123456789`

## Use Cases

### Threat Intelligence
- Identify infrastructure belonging to specific organizations
- Track threat actor infrastructure across different services
- Discover related assets through favicon correlation

### Asset Discovery
- Find all instances of a particular technology stack
- Map organizational digital footprint
- Identify shadow IT resources

### Red Team Operations
- Fingerprint target technologies for vulnerability analysis
- Discover related subdomains and infrastructure
- Enumerate external attack surface

## Advanced Usage

### Batch Processing
```yaml
# Fast batch processing with simplified output (default, 50 concurrent workers)
cat urls.txt | favinfo --timeout 15s --silent > hashes.txt

# High-performance batch processing with custom concurrency
cat urls.txt | favinfo --timeout 15s --concurrent 100 --silent > hashes.txt

# Detailed batch processing with JSON output
cat urls.txt | favinfo --timeout 15s --json > results.json
```

### Integration with Other Tools
```yaml
# Extract domains from subfinder and get favicon hashes (simplified output)
subfinder -d example.com | favinfo --silent

# Get detailed JSON output for specific analysis
echo "example.com" | favinfo --json | jq '.murmur_hash'

# Process thousands of URLs efficiently with simplified output
cat urls.txt | favinfo --silent > hashes.txt
```

### Custom Fingerprint Database
Create your own fingerprint database by adding entries to `fingerprint.json`:
```json
{
  "123456789": "Custom Technology",
  "987654321": "Internal Application v1.0"
}
```

## Configuration

### Default Configuration Locations
- **Linux/Mac**: `$HOME/.config/favinfo/fingerprint.json`
- **Windows**: `%USERPROFILE%\.config\favinfo\fingerprint.json`
