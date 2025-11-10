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
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/spf13/pflag"
	"github.com/twmb/murmur3"
	"github.com/rix4uni/favinfo/banner"
)

// FaviconResult represents the result structure for JSON output
type FaviconResult struct {
	InputURL    string `json:"input_url"`
	FaviconURL  string `json:"favicon_url"`
	MurmurHash  int32  `json:"murmur_hash"`
	MD5Hash     string `json:"md5_hash"`
	SHA256Hash  string `json:"sha256_hash"`
	Technology  string `json:"technology"`
}

// SearchEngineQueries represents search engine query formats
type SearchEngineQueries struct {
	Shodan   string `json:"shodan"`
	Fofa     string `json:"fofa"`
	Censys   string `json:"censys"`
	ZoomEye  string `json:"zoomeye"`
	Quake    string `json:"quake"`
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

// printSearchExamples prints examples of how to use the hashes in various search engines
func printSearchExamples(faviconURL string, murmurHash int32, md5Hash, sha256Hash, tech string) {
	// ANSI color codes
	const (
		Reset      = "\033[0m"
		Bold       = "\033[1m"
		Red        = "\033[31m"
		Green      = "\033[32m"
		Yellow     = "\033[33m"
		Blue       = "\033[34m"
		Magenta    = "\033[35m"
		Cyan       = "\033[36m"
		White      = "\033[37m"
		BgBlue     = "\033[44m"
		BgMagenta  = "\033[45m"
		BoldYellow = "\033[1;33m"
		BoldCyan   = "\033[1;36m"
		BoldGreen  = "\033[1;32m"
	)

	fmt.Printf("\n%s%s=== Search Engine Queries for: %s ===%s\n", BgBlue, White, faviconURL, Reset)
	fmt.Printf("%sTechnology identified: %s%s%s\n\n", BoldYellow, Cyan, tech, Reset)
	
	fmt.Printf("%sSHODAN:%s\n", BoldGreen, Reset)
	fmt.Printf("  %shttp.favicon.hash:%s%d%s\n", Yellow, BoldCyan, murmurHash, Reset)
	fmt.Printf("  %shttps://www.shodan.io/search?query=http.favicon.hash%%3A%d%s\n\n", Blue, murmurHash, Reset)
	
	fmt.Printf("%sFOFA:%s\n", BoldGreen, Reset)
	fmt.Printf("  %sicon_hash=\"%s%d%s\"%s\n", Yellow, BoldCyan, murmurHash, Yellow, Reset)
	fmt.Printf("  %shttps://fofa.info/result?q=icon_hash%%3D%%22%d%%22%s\n\n", Blue, murmurHash, Reset)
	
	fmt.Printf("%sCENSYS:%s\n", BoldGreen, Reset)
	fmt.Printf("  %sservices.http.response.favicons.md5_hash = \"%s%s%s\"%s\n", Yellow, BoldCyan, md5Hash, Yellow, Reset)
	fmt.Printf("  %sservices.http.response.favicons.sha256_hash = \"%s%s%s\"%s\n", Yellow, BoldCyan, sha256Hash, Yellow, Reset)
	fmt.Printf("  %shttps://search.censys.io/search?q=services.http.response.favicons.md5_hash%%3A%s%s\n\n", Blue, md5Hash, Reset)
	
	fmt.Printf("%sHUNTER.IO:%s\n", BoldGreen, Reset)
	fmt.Printf("  %sUse domain-based search as Hunter doesn't support favicon hash directly%s\n", Yellow, Reset)
	fmt.Printf("  %shttps://hunter.io/search/ (search by domain)%s\n\n", Blue, Reset)
	
	fmt.Printf("%sZOOMEYE:%s\n", BoldGreen, Reset)
	fmt.Printf("  %siconhash:%s%d%s\n", Yellow, BoldCyan, murmurHash, Reset)
	fmt.Printf("  %shttps://www.zoomeye.org/searchResult?q=iconhash%%3A%d%s\n\n", Blue, murmurHash, Reset)
	
	fmt.Printf("%sQUAKE:%s\n", BoldGreen, Reset)
	fmt.Printf("  %sfavicon.hash:%s%d%s\n", Yellow, BoldCyan, murmurHash, Reset)
	fmt.Printf("  %shttps://quake.360.cn/quake/#/searchResult?searchVal=favicon.hash%%3A%d%s\n\n", Blue, murmurHash, Reset)
	
	fmt.Printf("%s%sSUMMARY:%s\n", BgMagenta, White, Reset)
	fmt.Printf("  %sMurmur3 Hash (most common):%s %s%d%s\n", BoldYellow, Reset, BoldCyan, murmurHash, Reset)
	fmt.Printf("  %sMD5 Hash:%s %s%s%s\n", BoldYellow, Reset, BoldCyan, md5Hash, Reset)
	fmt.Printf("  %sSHA256 Hash:%s %s%s%s\n", BoldYellow, Reset, BoldCyan, sha256Hash, Reset)
	fmt.Printf("  %sIdentified Technology:%s %s%s%s\n", BoldYellow, Reset, BoldCyan, tech, Reset)
	fmt.Printf("%s%s========================================%s\n\n", BgBlue, White, Reset)
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
		InputURL:    inputURL,
		FaviconURL:  faviconURL,
		MurmurHash:  murmurHash,
		MD5Hash:     md5Hash,
		SHA256Hash:  sha256Hash,
		Technology:  tech,
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

func main() {
	// Define the flags using pflag
	timeout := pflag.Duration("timeout", 10*time.Second, "Set the HTTP request timeout duration")
	source := pflag.Bool("source", false, "Enable source output for where the url coming from scraped or added /favicon.ico")
	userAgent := pflag.StringP("user-agent", "H", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36", "Set the User-Agent header for HTTP requests")
	fingerprintPath := pflag.String("fingerprint", "", "Path to the fingerprint.json file (default: $HOME/.config/favinfo/fingerprint.json or ./fingerprint.json)")
	jsonOutput := pflag.Bool("json", false, "Output results in JSON format")
	silent := pflag.Bool("silent", false, "Silent mode.")
	version := pflag.Bool("version", false, "Print the version of the tool and exit.")
	
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
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		DisableKeepAlives: false,
	}

	// Determine the path to fingerprint.json
	var fingerprintFilePath string
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
	    configPath := homeDir + "/.config/favinfo/fingerprint.json"
	    if _, err := os.Stat(configPath); err == nil {
	        fingerprintFilePath = configPath
	    } else if _, err := os.Stat("fingerprint.json"); err == nil {
	        // Fall back to fingerprint.json in the current directory
	        fingerprintFilePath = "fingerprint.json"
	    } else {
	        fmt.Println("Error: fingerprint.json not found in $HOME/.config/favinfo/ or current directory.")
	        return
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
		input = ensureProtocol(input, client)
		favicons, err := getFaviconUrls(input, client, *source)
		if err != nil {
			fmt.Printf("Error fetching favicons: %v\n", err)
			continue
		}

		// Process each favicon URL
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

			// Output based on format
			if *jsonOutput {
				printJSONOutput(input, faviconURL, murmurHash, md5Hash, sha256Hash, tech)
			} else {
				// Print the results in normal format
				printSearchExamples(faviconURL, murmurHash, md5Hash, sha256Hash, tech)
			}
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