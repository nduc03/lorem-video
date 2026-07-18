package config

import (
	"fmt"
)

// ValidateResolution checks if width and height are within allowed bounds and aspect ratio.
func ValidateResolution(width, height int) error {
	if width < MinDimension || width > MaxDimension || height < MinDimension || height > MaxDimension {
		return fmt.Errorf("resolution out of bounds: %dx%d (min %d, max %d)", width, height, MinDimension, MaxDimension)
	}

	if float64(width)/float64(height) > 10 || float64(height)/float64(width) > 10 {
		return fmt.Errorf("aspect ratio out of bounds: %dx%d (max 10:1 or 1:10)", width, height)
	}

	return nil
}

// ValidateDuration checks if the duration is within the allowed limit.
func ValidateDuration(durationSeconds, maxDurationMinutes int) error {
    maxSeconds := maxDurationMinutes * 60
	if durationSeconds > maxSeconds {
		return fmt.Errorf("duration out of bounds: %ds exceeds maximum allowed %ds (%d minutes)", durationSeconds, maxSeconds, maxDurationMinutes)
	}
	return nil
}

// ValidateFramerate checks if framerate is within predefined options
func ValidateFramerate(fps int) error {
	valid := []int{24, 30, 60}
	for _, v := range valid {
		if fps == v {
			return nil
		}
	}
	return fmt.Errorf("invalid framerate: %d (must be one of %v)", fps, valid)
}


// ValidateVideoBitrate checks if video bitrate is within predefined options
func ValidateVideoBitrate(bitrate string) error {
	valid := []string{"18crf", "23crf", "28crf", "1000cbr", "2500cbr", "5000cbr", "1000vbr", "2500vbr", "5000vbr"}
	for _, v := range valid {
		if bitrate == v {
			return nil
		}
	}
	return fmt.Errorf("invalid video bitrate: %s (must be one of %v)", bitrate, valid)
}

// ValidateAudioBitrate checks if audio bitrate is within predefined options
func ValidateAudioBitrate(bitrate int, acodec string) error {
    if acodec == "noaudio" {
        return nil
    }
	valid := []int{96, 128, 192}
	for _, v := range valid {
		if bitrate == v {
			return nil
		}
	}
	return fmt.Errorf("invalid audio bitrate: %d (must be one of %v)", bitrate, valid)
}

// ValidateAudioCodec checks if audio codec is within predefined options
func ValidateAudioCodec(codec string) error {
	valid := []string{"aac", "opus", "noaudio"}
	for _, v := range valid {
		if codec == v {
			return nil
		}
	}
	return fmt.Errorf("invalid audio codec: %s (must be one of %v)", codec, valid)
}
