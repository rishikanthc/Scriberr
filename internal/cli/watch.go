package cli

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
)

var watchCmd = &cobra.Command{
	Use:   "watch [folder]",
	Short: "Watch a folder for new audio files",
	Args:  cobra.ExactArgs(1),
	Run:   runWatch,
}

func init() {
	rootCmd.AddCommand(watchCmd)
}

func runWatch(cmd *cobra.Command, args []string) {
	folder := args[0]
	absPath, err := filepath.Abs(folder)
	if err != nil {
		log.Fatalf("Failed to get absolute path: %v", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		log.Fatalf("Folder does not exist: %s", absPath)
	}

	// Save as current watch folder
	if _, err := SaveConfig("", "", absPath); err != nil {
		fmt.Printf("Warning: Failed to save watch folder to config: %v\n", err)
	}

	watchFolder(absPath)
}

func watchFolder(path string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	// Debounce map: filename -> timer
	timers := make(map[string]*time.Timer)
	var mu sync.Mutex

	done := make(chan bool)

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				// Only care about Write and Create events
				if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
					// Check extension
					ext := strings.ToLower(filepath.Ext(event.Name))
					if !isAudioFile(ext) {
						continue
					}

					mu.Lock()
					if t, exists := timers[event.Name]; exists {
						t.Stop()
					}

					// Set new timer for 2 seconds
					timers[event.Name] = time.AfterFunc(2*time.Second, func() {
						mu.Lock()
						delete(timers, event.Name)
						mu.Unlock()

						log.Printf("Uploading %s...\n", event.Name)
						if err := UploadFile(event.Name); err != nil {
							log.Printf("Failed to upload %s: %v\n", event.Name, err)
						} else {
							log.Printf("Successfully uploaded %s\n", event.Name)
						}
					})
					mu.Unlock()
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()

	err = watcher.Add(path)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Watching %s for new audio files...\n", path)
	<-done
}

func isAudioFile(ext string) bool {
	switch ext {
	case ".mp3", ".wav", ".m4a", ".flac", ".ogg", ".aac", ".wma":
		return true
	default:
		return false
	}
}
