package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"lorem.video/internal/config"
)

// StartupPregeneration runs video pregeneration in the background on app startup
func StartupPregeneration() {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
		defer cancel()

		_, err := PregenerateAllVideos(ctx)
		if err != nil {
			log.Printf("❌ Failed to pregenerate videos: %v", err)
			return
		}

		_, err = PregenerateAllHLS(ctx)
		if err != nil {
			log.Printf("❌ Failed to pregenerate HLS streams: %v", err)
			return
		}
	}()
}

// PregenerateAllVideos generates all pregenerated videos for all source files
func PregenerateAllVideos(ctx context.Context) (map[string][]string, error) {
	sourceFiles, err := config.GetSourceVideoFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to get source video files: %w", err)
	}

	results := make(map[string][]string)

	for _, sourceFile := range sourceFiles {
		generatedFiles, err := PregenerateVideos(ctx, sourceFile)
		if err != nil {
			log.Printf("❌ Failed to pregenerate videos for %s: %v", filepath.Base(sourceFile), err)
			continue
		}

		results[filepath.Base(sourceFile)] = generatedFiles
	}

	return results, nil
}

func PregenerateVideos(ctx context.Context, inputPath string) ([]string, error) {
	filenameNoExt := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))
	outputDir := filepath.Join(config.AppPaths.Video, filenameNoExt)

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	var generatedFiles []string

	// Create a video service for transcoding
	videoService := NewVideoService()

	for i, spec := range config.DefaultPregenSpecs {
		spec.Name = filenameNoExt
		resultCh, errCh := videoService.Transcode(ctx, spec, inputPath, outputDir)

		// Wait for completion
		select {
		case result := <-resultCh:
			filename := filepath.Base(result)
			generatedFiles = append(generatedFiles, filename)

		case err := <-errCh:
			return nil, fmt.Errorf("failed to generate video %d (%s %dx%d): %w",
				i+1, spec.Codec, spec.Width, spec.Height, err)

		case <-ctx.Done():
			return nil, fmt.Errorf("pregeneration cancelled: %w", ctx.Err())
		}
	}

	return generatedFiles, nil
}

// GenerateDefaultSourceVideo downloads a default high-resolution video


// EnsureDefaultSourceVideo checks if default source video exists and generates it if not
func EnsureDefaultSourceVideo() error {
	defaultPath := config.AppPaths.DefaultSourceVideo
	var finalErr error

	if _, statErr := os.Stat(defaultPath); os.IsNotExist(statErr) {
		log.Printf("Default source video not found, generating/downloading: %s", defaultPath)
		finalErr = GenerateDefaultSourceVideo(defaultPath)
	} else if statErr != nil {
		return fmt.Errorf("failed to check default source video: %w", statErr)
	}

	// Always validate and extract duration
	durationSeconds, validateErr := validateVideo(defaultPath)
	if validateErr != nil {
		log.Printf("Existing default video is invalid (%v). Regenerating fallback...", validateErr)
		os.Remove(defaultPath)
		if err := generateFallbackVideo(defaultPath); err != nil {
			return fmt.Errorf("failed to generate fallback video: %w", err)
		}
		durationSeconds, _ = validateVideo(defaultPath)
	}

	// Update config.MaxDurationMinutes dynamically
	videoMinutes := max(int(math.Floor(durationSeconds / 60)), 1)
	if config.MaxDurationMinutes == 0 || config.MaxDurationMinutes > videoMinutes {
		config.MaxDurationMinutes = videoMinutes
	}

	return finalErr
}

func GenerateDefaultSourceVideo(outputPath string) error {
	url := os.Getenv("BUNNY_VIDEO_URL")
	if url == "" {
		log.Println("BUNNY_VIDEO_URL not set. Falling back to generated test video.")
		return generateFallbackVideo(outputPath)
	}

	log.Printf("Downloading default video from %s to %s", url, outputPath)
	if err := downloadLargeVideo(url, outputPath); err != nil {
		log.Printf("Failed to download video (%v). Falling back to generated test video.", err)
		os.Remove(outputPath) // clean up partial file
		return generateFallbackVideo(outputPath)
	}

	return nil
}

func downloadLargeVideo(url, destPath string) error {
	client := &http.Client{
		Timeout: 30 * time.Minute, // ample time for large file
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func validateVideo(path string) (float64, error) {
	cmd := exec.Command(
		"ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		path,
	)

	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("ffprobe failed (is it a valid video?): %w", err)
	}

	durationStr := strings.TrimSpace(string(output))
	duration, err := strconv.ParseFloat(durationStr, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse duration '%s': %w", durationStr, err)
	}

	return duration, nil
}

func generateFallbackVideo(outputPath string) error {
	fallbackMinutes := 10 // default
	if valStr := os.Getenv("MAX_LOREM_VIDEO_MINUTE"); valStr != "" {
		if v, err := strconv.Atoi(valStr); err == nil && v > 0 {
			fallbackMinutes = v
		}
	}
	durationSeconds := fallbackMinutes * 60

	log.Printf("Generating fallback 'beep beep' test video to %s with duration %ds", outputPath, durationSeconds)
	// Create a test video
	cmd := exec.Command(
		"ffmpeg", "-y",
		"-f", "lavfi", "-i", fmt.Sprintf("testsrc=duration=%d:size=1280x720:rate=30", durationSeconds),
		"-f", "lavfi", "-i", fmt.Sprintf("sine=frequency=1000:duration=%d", durationSeconds),
		"-c:v", "libx264", "-c:a", "aac",
		"-pix_fmt", "yuv420p",
		outputPath,
	)

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg failed: %w\n%s", err, string(output))
	}
	return nil
}


// PregenerateAllHLS generates HLS streams for all source video files
func PregenerateAllHLS(ctx context.Context) (map[string][]string, error) {
	sourceFiles, err := config.GetSourceVideoFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to get source video files: %w", err)
	}

	results := make(map[string][]string)

	for _, sourceFile := range sourceFiles {
		generatedStreams, err := PregenerateHLS(ctx, sourceFile)
		if err != nil {
			log.Printf("❌ Failed to pregenerate HLS streams for %s: %v", filepath.Base(sourceFile), err)
			continue
		}

		results[filepath.Base(sourceFile)] = generatedStreams
	}

	return results, nil
}

// PregenerateHLS generates HLS streams for a specific source video file
func PregenerateHLS(ctx context.Context, inputPath string) ([]string, error) {
	filenameNoExt := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))
	outputDir := filepath.Join(config.AppPaths.Stream, filenameNoExt)

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Check if input video is vertical (portrait orientation)
	isVertical, err := isVideoVertical(inputPath)
	if err != nil {
		log.Printf("⚠️  Failed to detect video orientation for %s, using default resolutions: %v", filenameNoExt, err)
		isVertical = false
	}

	hlsResolutions := map[string]config.Resolution{
		"480p":  config.Resolutions["480p"],
		"720p":  config.Resolutions["720p"],
		"1080p": config.Resolutions["1080p"],
	}

	// If video is vertical, swap width/height for HLS transcoding
	if isVertical {
		for key, res := range hlsResolutions {
			hlsResolutions[key] = config.Resolution{
				Width:  res.Height,
				Height: res.Width,
			}
		}
		// log.Printf("Detected vertical video %s, using portrait resolutions for HLS", filenameNoExt)
	}

	var generatedStreams []string
	videoService := NewVideoService()

	for resName, resolution := range hlsResolutions {
		hlsDir := filepath.Join(outputDir, resName)
		playlistPath := filepath.Join(hlsDir, config.HLSMediaPlaylist)

		if _, err := os.Stat(playlistPath); err == nil {
			// HLS stream already exists, skip generation
			generatedStreams = append(generatedStreams, resName+": "+filepath.Base(playlistPath)+" (existing)")
			continue
		}

		// Create directory before transcoding
		if err := os.MkdirAll(hlsDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create HLS directory %s: %w", hlsDir, err)
		}

		resultCh, errCh := videoService.TranscodeHLS(ctx, resolution, inputPath, hlsDir)

		select {
		case result := <-resultCh:
			generatedStreams = append(generatedStreams, resName+": "+filepath.Base(result))
			log.Printf("✅ Generated HLS stream %s for %s: %s", resName, filenameNoExt, filepath.Base(result))

		case err := <-errCh:
			return nil, fmt.Errorf("failed to generate HLS stream %s (%dx%d): %w",
				resName, resolution.Width, resolution.Height, err)

		case <-ctx.Done():
			return nil, fmt.Errorf("HLS pregeneration cancelled: %w", ctx.Err())
		}
	}

	masterPlaylistPath := filepath.Join(outputDir, config.HLSMasterPlaylist)
	if _, err := os.Stat(masterPlaylistPath); err == nil {
		// Master playlist already exists, skip generation
		generatedStreams = append(generatedStreams, "master: "+filepath.Base(masterPlaylistPath)+" (existing)")
	} else {
		if err := generateMasterPlaylist(masterPlaylistPath, hlsResolutions, filenameNoExt); err != nil {
			return nil, fmt.Errorf("failed to generate master playlist: %w", err)
		}

		generatedStreams = append(generatedStreams, "master: "+filepath.Base(masterPlaylistPath))
		log.Printf("✅ Generated master playlist for %s: %s", filenameNoExt, filepath.Base(masterPlaylistPath))
	}

	return generatedStreams, nil
}

func generateMasterPlaylist(masterPlaylistPath string, hlsResolutions map[string]config.Resolution, videoName string) error {
	// Define approximate bandwidth for each resolution (these are rough estimates)
	bandwidths := map[string]int{
		"480p":  800000,  // 800 kbps
		"720p":  2000000, // 2 Mbps
		"1080p": 5000000, // 5 Mbps
	}

	var content strings.Builder
	content.WriteString("#EXTM3U\n")
	content.WriteString("#EXT-X-VERSION:6\n\n")

	resolutionOrder := []string{"480p", "720p", "1080p"}
	baseURL := config.GetBaseURL()

	for _, resKey := range resolutionOrder {
		if resolution, exists := hlsResolutions[resKey]; exists {
			bandwidth := bandwidths[resKey]
			resName := config.ResolutionsName[resKey]

			content.WriteString(fmt.Sprintf("#EXT-X-STREAM-INF:BANDWIDTH=%d,NAME=%s,RESOLUTION=%dx%d\n",
				bandwidth, resName, resolution.Width, resolution.Height))
			content.WriteString(fmt.Sprintf("%s/hls/%s/%s/%s\n\n", baseURL, videoName, resKey, config.HLSMediaPlaylist))
		}
	}

	return os.WriteFile(masterPlaylistPath, []byte(content.String()), 0644)
}

func isVideoVertical(inputPath string) (bool, error) {
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=width,height:stream_side_data=rotation",
		"-of", "json",
		inputPath,
	)

	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("ffprobe failed: %w", err)
	}

	// Parse JSON output
	var result struct {
		Streams []struct {
			Width        int `json:"width"`
			Height       int `json:"height"`
			SideDataList []struct {
				Rotation int `json:"rotation"`
			} `json:"side_data_list"`
		} `json:"streams"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return false, fmt.Errorf("failed to parse ffprobe JSON: %w", err)
	}

	if len(result.Streams) == 0 {
		return false, fmt.Errorf("no video streams found")
	}

	stream := result.Streams[0]
	width, height := stream.Width, stream.Height

	// Check for rotation metadata
	rotation := 0
	if len(stream.SideDataList) > 0 {
		rotation = stream.SideDataList[0].Rotation
	}

	// Video is considered vertical if:
	// 1. Natural portrait orientation (height > width), OR
	// 2. Rotated 90 or 270 degrees (±90)
	isNaturalPortrait := height > width
	isRotatedPortrait := math.Abs(float64(rotation)) == 90

	return isNaturalPortrait || isRotatedPortrait, nil
}
