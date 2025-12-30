package cli

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with the Scriberr server",
	Run:   runLogin,
}

var serverURL string

func init() {
	rootCmd.AddCommand(loginCmd)
	loginCmd.Flags().StringVarP(&serverURL, "server", "s", "http://localhost:8080", "Scriberr server URL")
}

func runLogin(cmd *cobra.Command, args []string) {
	// 1. Start local server
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		fmt.Printf("Failed to start local server: %v\n", err)
		os.Exit(1)
	}
	port := listener.Addr().(*net.TCPAddr).Port

	tokenChan := make(chan string)
	errChan := make(chan error)

	server := &http.Server{}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		username := r.URL.Query().Get("username")

		if token != "" {
			fmt.Fprintf(w, "Login successful! You can close this window now.")
			// Save config
			if _, err := SaveConfig(serverURL, token, ""); err != nil {
				fmt.Printf("Error saving config: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("\nLogged in as %s\n", username)
			tokenChan <- token
		} else {
			fmt.Fprintf(w, "Login failed: No token received.")
			errChan <- fmt.Errorf("no token received")
		}
	})

	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	// 2. Open browser
	callbackURL := fmt.Sprintf("http://localhost:%d", port)
	authURL := fmt.Sprintf("%s/auth/cli/authorize?callback_url=%s&device_name=%s",
		serverURL,
		url.QueryEscape(callbackURL),
		url.QueryEscape("Scriberr CLI"),
	)

	fmt.Printf("Opening browser to authorize: %s\n", authURL)
	if err := openBrowser(authURL); err != nil {
		fmt.Printf("Failed to open browser: %v\n", err)
		fmt.Println("Please open the URL above manually.")
	}

	// 3. Wait for callback
	select {
	case <-tokenChan:
		fmt.Println("Configuration saved.")
	case err := <-errChan:
		fmt.Printf("Login failed: %v\n", err)
		os.Exit(1)
	case <-time.After(5 * time.Minute):
		fmt.Println("Login timed out.")
		os.Exit(1)
	}

	// Shutdown server
	_ = server.Shutdown(context.Background())
}

func openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}
	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}
