package main

import (
	"crypto/md5"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/rix4uni/favinfo/banner"
	"github.com/spf13/pflag"
	"github.com/twmb/murmur3"
)

// FaviconResult represents the result structure for JSON output
type FaviconResult struct {
	InputURL   string `json:"input_url"`
	FaviconURL string `json:"favicon_url"`
	MurmurHash int32  `json:"murmur_hash"`
	MD5Hash    string `json:"md5_hash"`
	SHA256Hash string `json:"sha256_hash"`
	Technology string `json:"technology"`
}

// SearchEngineQueries represents search engine query formats
type SearchEngineQueries struct {
	Shodan  string `json:"shodan"`
	Fofa    string `json:"fofa"`
	Censys  string `json:"censys"`
	ZoomEye string `json:"zoomeye"`
	Quake   string `json:"quake"`
}

// ExtendedFaviconResult includes search engine queries
type ExtendedFaviconResult struct {
	FaviconResult
	SearchQueries SearchEngineQueries `json:"search_queries"`
}

// getFaviconUrls extracts all favicons from the given URL.
func getFaviconUrls(baseURL string, client *http.Client, source bool) ([]string, error) {
	// Send GET request to the base URL with the custom client (which includes timeout)
	resp, err := client.Get(baseURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Parse the HTML
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	// Parse the base URL to handle relative paths
	base, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	// Slice to hold favicon URLs
	var favicons []string

	// Find all <link rel="icon"> and <link rel="shortcut icon"> elements
	doc.Find("link[rel='icon'], link[rel=\"icon\"], link[rel='shortcut icon'], link[rel=\"shortcut icon\"]").Each(func(i int, s *goquery.Selection) {
		// Get the href attribute (favicon path)
		href, exists := s.Attr("href")
		if exists {
			// If the href is an absolute URL (starts with http), use it directly
			var absoluteURL string
			if strings.HasPrefix(href, "http") {
				absoluteURL = href
			} else {
				// If it's a relative URL, resolve it using the base URL
				absoluteURL = base.ResolveReference(&url.URL{Path: href}).String()
			}

			// Remove everything after .png or .ico (strip query parameters)
			if strings.Contains(absoluteURL, ".png") {
				absoluteURL = strings.Split(absoluteURL, ".png")[0] + ".png"
			} else if strings.Contains(absoluteURL, ".ico") {
				absoluteURL = strings.Split(absoluteURL, ".ico")[0] + ".ico"
			}

			// Append the cleaned URL
			favicons = append(favicons, absoluteURL)

			// If source flag is set, print the [Scraped] message
			if source {
				fmt.Printf("[Scraped]: %s\n", absoluteURL)
			}
		}
	})

	// If no favicons were found, try the /favicon.ico path
	if len(favicons) == 0 {
		// Try adding "/favicon.ico" to the base domain
		faviconURL := base.ResolveReference(&url.URL{Path: "/favicon.ico"}).String()

		// Check if this URL returns a 200 status code
		resp, err := client.Get(faviconURL)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		// If status code is 200, return the favicon URL
		if resp.StatusCode == 200 {
			favicons = append(favicons, faviconURL)

			// If source flag is set, print the [Added] message
			if source {
				fmt.Printf("[Added]: %s\n", faviconURL)
			}
		}
	}

	return favicons, nil
}

// loadFingerprintMap loads the fingerprint mapping from the fingerprint.json file.
func loadFingerprintMap(fileName string) (map[string]string, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var fingerprintMap map[string]string
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&fingerprintMap)
	if err != nil {
		return nil, err
	}

	return fingerprintMap, nil
}

// downloadFingerprintFile downloads the fingerprint.json file from GitHub and saves it to the specified path.
func downloadFingerprintFile(url string, filePath string) error {
	// Create parent directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %v", dir, err)
	}

	// Download the file
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download fingerprint.json from %s: %v", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download fingerprint.json: HTTP status %d", resp.StatusCode)
	}

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %v", err)
	}

	// Write the file
	if err := ioutil.WriteFile(filePath, body, 0644); err != nil {
		return fmt.Errorf("failed to write fingerprint.json to %s: %v", filePath, err)
	}

	return nil
}

// calculateMurmurHash processes the favicon data and calculates the Murmur3 hash.
func calculateMurmurHash(faviconBytes []byte) int32 {
	// Base64 encode the favicon content
	base64Content := base64.StdEncoding.EncodeToString(faviconBytes)

	// Split the base64 string into chunks as done in the original code
	chunkSize := 76
	var chunks []string
	for i := 0; i*chunkSize+chunkSize < len(base64Content); i++ {
		chunks = append(chunks, base64Content[i*chunkSize:i*chunkSize+chunkSize])
	}

	// Add the last chunk
	lastChunk := base64Content[len(chunks)*chunkSize:]
	chunks = append(chunks, lastChunk)

	// Combine all chunks into a single string
	finalString := ""
	for _, chunk := range chunks {
		finalString = finalString + chunk + "\n"
	}

	// Calculate the Murmur3 hash of the final string
	return int32(murmur3.StringSum32(finalString))
}

// calculateMD5 calculates the MD5 hash of the favicon data.
func calculateMD5(faviconBytes []byte) string {
	hash := md5.New()
	hash.Write(faviconBytes)
	return fmt.Sprintf("%x", hash.Sum(nil))
}

// calculateSHA256 calculates the SHA256 hash of the favicon data.
func calculateSHA256(faviconBytes []byte) string {
	hash := sha256.New()
	hash.Write(faviconBytes)
	return fmt.Sprintf("%x", hash.Sum(nil))
}

func ensureProtocol(input string, client *http.Client) string {
	if !strings.HasPrefix(input, "http://") && !strings.HasPrefix(input, "https://") {
		// Try HTTPS first
		testURL := "https://" + input
		resp, err := client.Head(testURL) // Use HEAD to check availability quickly
		if err == nil && resp.StatusCode == http.StatusOK {
			return testURL
		}
		// Fallback to HTTP
		return "http://" + input
	}
	return input
}

// generateSearchQueries generates search engine queries for the given hashes
func generateSearchQueries(murmurHash int32, md5Hash string) SearchEngineQueries {
	return SearchEngineQueries{
		Shodan:  fmt.Sprintf("http.favicon.hash:%d", murmurHash),
		Fofa:    fmt.Sprintf("icon_hash=\"%d\"", murmurHash),
		Censys:  fmt.Sprintf("services.http.response.favicons.md5_hash=\"%s\"", md5Hash),
		ZoomEye: fmt.Sprintf("iconhash:%d", murmurHash),
		Quake:   fmt.Sprintf("favicon.hash:%d", murmurHash),
	}
}

// printJSONOutput prints the results in JSON format
func printJSONOutput(inputURL string, faviconURL string, murmurHash int32, md5Hash, sha256Hash, tech string) {
	result := ExtendedFaviconResult{
		FaviconResult: FaviconResult{
			InputURL:   inputURL,
			FaviconURL: faviconURL,
			MurmurHash: murmurHash,
			MD5Hash:    md5Hash,
			SHA256Hash: sha256Hash,
			Technology: tech,
		},
		SearchQueries: generateSearchQueries(murmurHash, md5Hash),
	}
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling JSON: %v\n", err)
		return
	}
	fmt.Println(string(jsonData))
}

// printSimplifiedOutput prints the results in simplified format: URL [hash1, hash2]
func printSimplifiedOutput(inputURL string, murmurHashes []int32) {
	if len(murmurHashes) == 0 {
		fmt.Printf("%s []\n", inputURL)
		return
	}

	// Convert int32 hashes to strings
	hashStrings := make([]string, len(murmurHashes))
	for i, hash := range murmurHashes {
		hashStrings[i] = fmt.Sprintf("%d", hash)
	}

	fmt.Printf("%s [%s]\n", inputURL, strings.Join(hashStrings, ", "))
}

func main() {
	// Define the flags using pflag
	timeout := pflag.Duration("timeout", 30*time.Second, "Set the HTTP request timeout duration")
	source := pflag.Bool("source", false, "Enable source output for where the url coming from scraped or added /favicon.ico")
	userAgent := pflag.StringP("user-agent", "H", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36", "Set the User-Agent header for HTTP requests")
	fingerprintPath := pflag.String("fingerprint", "", "Path to the fingerprint.json file (default: $HOME/.config/favinfo/fingerprint.json or ./fingerprint.json)")
	jsonOutput := pflag.Bool("json", false, "Output results in JSON format")
	silent := pflag.Bool("silent", false, "Silent mode.")
	version := pflag.Bool("version", false, "Print the version of the tool and exit.")
	verbose := pflag.Bool("verbose", false, "Verbose mode. Show verbose output.")

	// Parse the flags
	pflag.Parse()

	if *version {
		banner.PrintBanner()
		banner.PrintVersion()
		return
	}

	if !*silent {
		banner.PrintBanner()
	}

	// Create a custom HTTP client with the specified timeout
	client := &http.Client{
		Timeout: *timeout,
	}

	// Add the User-Agent header to the client request
	client.Transport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		// Modify the default transport to include the User-Agent header
		TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
		DisableKeepAlives: false,
	}

	// Determine the path to fingerprint.json
	var fingerprintFilePath string
	const fingerprintURL = "https://raw.githubusercontent.com/rix4uni/favinfo/refs/heads/main/fingerprint.json"

	if *fingerprintPath != "" {
		// Use the custom path provided via the flag
		fingerprintFilePath = *fingerprintPath
	} else {
		// Get the user's home directory
		homeDir, err := os.UserHomeDir()
		if err != nil {
			fmt.Println("Error getting home directory:", err)
			return
		}

		// Check for fingerprint.json in $HOME/.config/favinfo/
		configPath := filepath.Join(homeDir, ".config", "favinfo", "fingerprint.json")
		if _, err := os.Stat(configPath); err == nil {
			fingerprintFilePath = configPath
		} else if _, err := os.Stat("fingerprint.json"); err == nil {
			// Fall back to fingerprint.json in the current directory
			fingerprintFilePath = "fingerprint.json"
		} else {
			// File not found, attempt to download it
			if *verbose {
				fmt.Println("fingerprint.json not found. Downloading from GitHub...")
			}
			if err := downloadFingerprintFile(fingerprintURL, configPath); err != nil {
				fmt.Printf("Error downloading fingerprint.json: %v\n", err)
				return
			}
			fingerprintFilePath = configPath
		}
	}

	// Load the fingerprint map
	fingerprintMap, err := loadFingerprintMap(fingerprintFilePath)
	if err != nil {
		fmt.Printf("Error loading fingerprint.json from %s: %v\n", fingerprintFilePath, err)
		return
	}

	// Read URL(s) from stdin
	var input string
	for {
		_, err := fmt.Scanln(&input)
		if err != nil {
			break
		}

		// Set User-Agent header
		req, err := http.NewRequest("GET", input, nil)
		if err != nil {
			fmt.Printf("Error creating request: %v\n", err)
			continue
		}
		req.Header.Set("User-Agent", *userAgent)

		// Fetch the favicons
		processedInput := ensureProtocol(input, client)
		favicons, err := getFaviconUrls(processedInput, client, *source)
		if err != nil {
			fmt.Printf("Error fetching favicons: %v\n", err)
			continue
		}

		// Output based on format
		if *jsonOutput {
			// JSON output: process each favicon separately
			for _, faviconURL := range favicons {
				// Fetch the favicon content
				faviconBytes, err := fetchFavicon(faviconURL, client)
				if err != nil {
					fmt.Printf("Error fetching favicon content: %v\n", err)
					continue
				}

				// Calculate the hashes
				murmurHash := calculateMurmurHash(faviconBytes)
				md5Hash := calculateMD5(faviconBytes)
				sha256Hash := calculateSHA256(faviconBytes)

				// Find the technology based on the Murmur3 hash
				tech := fingerprintMap[fmt.Sprintf("%d", murmurHash)]
				if tech == "" {
					tech = "unknown"
				}

				printJSONOutput(processedInput, faviconURL, murmurHash, md5Hash, sha256Hash, tech)
			}
		} else {
			// Simplified output: collect murmur hashes from all favicons
			var murmurHashes []int32
			for _, faviconURL := range favicons {
				// Fetch the favicon content
				faviconBytes, err := fetchFavicon(faviconURL, client)
				if err != nil {
					// Skip this favicon if fetch fails, but continue with others
					continue
				}

				// Calculate only the murmur hash
				murmurHash := calculateMurmurHash(faviconBytes)
				murmurHashes = append(murmurHashes, murmurHash)
			}

			// Print simplified output
			printSimplifiedOutput(processedInput, murmurHashes)
		}
	}
}

// fetchFavicon fetches the favicon from the given URL.
func fetchFavicon(url string, client *http.Client) ([]byte, error) {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	response, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	return ioutil.ReadAll(response.Body)
}
