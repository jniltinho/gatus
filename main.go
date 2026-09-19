// Gatus is a health dashboard that monitors services, evaluates conditions on their results, sends alerts and serves
// public status pages. The command line lives in the cmd package.
package main

import "gatus/v5/cmd"

// main is the entry point of the binary: it delegates everything to the cmd package
func main() {
	cmd.Execute()
}
