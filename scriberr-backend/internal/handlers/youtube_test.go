package handlers

import "testing"

func TestIsValidYouTubeURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{
			name:     "Standard YouTube watch URL",
			url:      "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
			expected: true,
		},
		{
			name:     "YouTube short URL",
			url:      "https://youtu.be/dQw4w9WgXcQ",
			expected: true,
		},
		{
			name:     "YouTube embed URL",
			url:      "https://www.youtube.com/embed/dQw4w9WgXcQ",
			expected: true,
		},
		{
			name:     "YouTube shorts URL",
			url:      "https://www.youtube.com/shorts/dQw4w9WgXcQ",
			expected: true,
		},
		{
			name:     "YouTube URL with additional parameters",
			url:      "https://www.youtube.com/watch?v=dQw4w9WgXcQ&t=30s",
			expected: true,
		},
		{
			name:     "Non-YouTube URL",
			url:      "https://www.google.com",
			expected: false,
		},
		{
			name:     "Empty URL",
			url:      "",
			expected: false,
		},
		{
			name:     "URL with YouTube in domain but not valid",
			url:      "https://fakeyoutube.com/watch?v=dQw4w9WgXcQ",
			expected: false,
		},
		{
			name:     "Case insensitive YouTube URL",
			url:      "HTTPS://WWW.YOUTUBE.COM/WATCH?V=DQW4W9WGXCQ",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidYouTubeURL(tt.url)
			if result != tt.expected {
				t.Errorf("isValidYouTubeURL(%q) = %v, want %v", tt.url, result, tt.expected)
			}
		})
	}
} 