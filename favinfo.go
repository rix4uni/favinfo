package main

import (
	"crypto/md5"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/twmb/murmur3"
	"github.com/rix4uni/favinfo/banner"
)

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

func main() {
	// Define the flags
	timeout := flag.Duration("timeout", 10*time.Second, "Set the HTTP request timeout duration")
	version := flag.Bool("version", false, "Print the version of the tool and exit.")
	silent := flag.Bool("silent", false, "Silent mode.")
	source := flag.Bool("source", false, "Enable source output for where the url coming from scraped or added /favicon.ico")
	userAgent := flag.String("H", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36", "Set the User-Agent header for HTTP requests")
	flag.Parse()

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
		DisableKeepAlives: false,
	}

	// Load the fingerprint map from fingerprint.json
	fingerprintMap, err := loadFingerprintMap("fingerprint.json")
	if err != nil {
		fmt.Println("Error loading fingerprint.json:", err)
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

			// Print the results
			fmt.Printf("%s [%d] [%s] [%s] [%s]\n", faviconURL, murmurHash, md5Hash, sha256Hash, tech)
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
