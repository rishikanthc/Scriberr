package audio

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

// AupTrack represents a track imported in the Audacity project
type AupTrack struct {
	Filename string  `xml:"filename,attr"`
	Offset   float64 // Parsed from offset attribute (in seconds)
	Channel  int     `xml:"channel,attr"`
	Mute     int     `xml:"mute,attr"`
	Solo     int     `xml:"solo,attr"`
	Gain     float64 // Parsed from gain attribute
	Pan      float64 // Parsed from pan attribute
}

// AupWaveTrack represents a wavetrack element in the AUP file
type AupWaveTrack struct {
	Name       string     `xml:"name,attr"`
	Channel    int        `xml:"channel,attr"`
	Linked     int        `xml:"linked,attr"`
	Mute       int        `xml:"mute,attr"`
	Solo       int        `xml:"solo,attr"`
	Height     int        `xml:"height,attr"`
	Minimized  int        `xml:"minimized,attr"`
	IsSelected int        `xml:"isSelected,attr"`
	Rate       float64    `xml:"rate,attr"`
	Gain       string     `xml:"gain,attr"`
	Pan        string     `xml:"pan,attr"`
	WaveClips  []WaveClip `xml:"waveclip"`
}

// WaveClip represents a waveclip element containing the import
type WaveClip struct {
	Offset string `xml:"offset,attr"`
	Import struct {
		Filename string `xml:"filename,attr"`
		Offset   string `xml:"offset,attr"`
		Channel  int    `xml:"channel,attr"`
	} `xml:"import"`
}

// AudacityProject represents the root structure of an AUP file
type AudacityProject struct {
	XMLName    xml.Name       `xml:"project"`
	Version    string         `xml:"audacityversion,attr"`
	DataDir    string         `xml:"datadir,attr"`
	Rate       float64        `xml:"rate,attr"`
	WaveTracks []AupWaveTrack `xml:"wavetrack"`
}

// AupParser handles parsing of Audacity project files
type AupParser struct{}

// NewAupParser creates a new AUP parser instance
func NewAupParser() *AupParser {
	return &AupParser{}
}

// ParseAupFile parses an .aup file and extracts track information
func (p *AupParser) ParseAupFile(filepath string) ([]AupTrack, error) {
	// Read the AUP file
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read AUP file: %w", err)
	}

	// Parse XML
	var project AudacityProject
	if err := xml.Unmarshal(data, &project); err != nil {
		return nil, fmt.Errorf("failed to parse AUP XML: %w", err)
	}

	// Extract tracks
	var tracks []AupTrack
	for _, waveTrack := range project.WaveTracks {
		// Parse gain and pan from string to float64
		gain, _ := strconv.ParseFloat(waveTrack.Gain, 64)
		pan, _ := strconv.ParseFloat(waveTrack.Pan, 64)

		// Process each waveclip in the wavetrack
		for _, clip := range waveTrack.WaveClips {
			if clip.Import.Filename != "" {
				// Parse offset from clip
				offset, _ := strconv.ParseFloat(clip.Offset, 64)
				
				track := AupTrack{
					Filename: clip.Import.Filename,
					Offset:   offset,
					Channel:  clip.Import.Channel,
					Mute:     waveTrack.Mute,
					Solo:     waveTrack.Solo,
					Gain:     gain,
					Pan:      pan,
				}
				tracks = append(tracks, track)
			}
		}
	}

	return tracks, nil
}

// ValidateTracksExist checks if all referenced tracks exist in the given directory
func (p *AupParser) ValidateTracksExist(tracks []AupTrack, tracksDir string) error {
	for _, track := range tracks {
		trackPath := filepath.Join(tracksDir, filepath.Base(track.Filename))
		if _, err := os.Stat(trackPath); os.IsNotExist(err) {
			return fmt.Errorf("track file not found: %s", track.Filename)
		}
	}
	return nil
}