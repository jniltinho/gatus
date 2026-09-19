package cmd

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"gatus/v5/config"
	"github.com/spf13/cobra"
)

const (
	urlFlagName = "url"

	// healthcheckTimeout is the total time of the check, connection and answer included
	healthcheckTimeout = 5 * time.Second
)

var errUnhealthy = errors.New("unhealthy")

// healthcheckCmd exists because the image has no shell, no curl and no wget: the binary checks itself. It reads only
// the web section of the configuration, and nothing of what the server starts: no delay, no storage and no OIDC.
var healthcheckCmd = &cobra.Command{
	Use:   "healthcheck",
	Short: "Check the /health of the local server, for the HEALTHCHECK of the image",
	Long: `Requests /health and exits with 0 when the answer is 200 and with 1 otherwise, in at most 5 seconds.

Without --url, the address, the port and the scheme come from the web section of the configuration: a wildcard address
becomes the loopback, and with web.tls the check speaks HTTPS to the loopback without verifying the certificate, which
may be self-signed. With --url the configuration is not read.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		target, _ := cmd.Flags().GetString(urlFlagName)
		if len(target) == 0 {
			configPath, err := resolveConfigPath(cmd)
			if err != nil {
				return err
			}
			webConfig, err := config.LoadWebConfiguration(configPath)
			if err != nil {
				return err
			}
			target = healthcheckURL(webConfig.Address, webConfig.Port, webConfig.HasTLS())
		}
		if err := checkHealth(target); err != nil {
			return fmt.Errorf("%s: %w", target, err)
		}
		fmt.Fprintln(cmd.OutOrStdout(), "healthy")
		return nil
	},
}

func init() {
	healthcheckCmd.Flags().String(urlFlagName, "", "URL to check instead of the /health of the configuration")
	rootCmd.AddCommand(healthcheckCmd)
}

// healthcheckURL returns the URL of /health for the address the server binds to. A wildcard address is where the
// server listens, not where it can be reached: it becomes the loopback.
func healthcheckURL(address string, port int, hasTLS bool) string {
	host := address
	if ip := net.ParseIP(address); len(address) == 0 || (ip != nil && ip.IsUnspecified()) {
		host = "127.0.0.1"
	}
	scheme := "http"
	if hasTLS {
		scheme = "https"
	}
	return scheme + "://" + net.JoinHostPort(host, strconv.Itoa(port)) + "/health"
}

func checkHealth(target string) error {
	client := &http.Client{
		Timeout: healthcheckTimeout,
		Transport: &http.Transport{
			// The proxy of the environment is for the checks of Gatus, never for the server checking itself
			Proxy: nil,
			// The certificate of the server may be self-signed or issued for another name than the loopback
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // loopback check of the server itself
		},
		// A redirect is not a healthy answer
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	response, err := client.Get(target)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: status %d", errUnhealthy, response.StatusCode)
	}
	return nil
}
