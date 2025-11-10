package banner

import (
	"fmt"
)

// prints the version message
const version = "v0.0.6"

func PrintVersion() {
	fmt.Printf("Current favinfo version %s\n", version)
}

// Prints the Colorful banner
func PrintBanner() {
	banner := `
    ____               _         ____     
   / __/____ _ _   __ (_)____   / __/____ 
  / /_ / __  /| | / // // __ \ / /_ / __ \
 / __// /_/ / | |/ // // / / // __// /_/ /
/_/   \__,_/  |___//_//_/ /_//_/   \____/
`
	fmt.Printf("%s\n%50s\n\n", banner, "Current favinfo version "+version)
}
