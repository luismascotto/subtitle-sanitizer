package mkv

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type subtitleTrack struct {
	Index         int    `json:"index"`
	Codec         string `json:"codec_name"`
	CodecTag      string `json:"codec_tag_string"`
	CodecLongName string `json:"codec_long_name"`
	Tags          Tag    `json:"tags"`
}
type Tag struct {
	Language string `json:"language"`
	Title    string `json:"title"`
}

type ffprobeOutput struct {
	Streams []subtitleTrack `json:"streams"`
}

func trackLanguage(t subtitleTrack) string {
	return strings.ToLower(strings.TrimSpace(t.Tags.Language))
}

func trackTitle(t subtitleTrack) string {
	return t.Tags.Title
}

func titleContainsSDH(title string) bool {
	return strings.Contains(strings.ToLower(title), "sdh")
}

// subtitleTrackOrders reports whether a should sort before b: English (eng) first, then titles without "sdh", then stream index.
func subtitleTrackOrders(a, b subtitleTrack) bool {
	engA, engB := trackLanguage(a) == "eng", trackLanguage(b) == "eng"
	if engA != engB {
		return engA
	}
	sdhA, sdhB := titleContainsSDH(trackTitle(a)), titleContainsSDH(trackTitle(b))
	if sdhA != sdhB {
		return !sdhA
	}
	return a.Index < b.Index
}

func VerifyDependencies() error {
	missing := []string{}
	for _, binary := range []string{"ffprobe", "ffmpeg"} {
		if _, err := execLookPath(binary); err != nil {
			missing = append(missing, binary)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf(
		"missing required tools: %s. Install FFmpeg (which includes ffmpeg + ffprobe) and ensure both are in PATH. Note: MediaInfo is useful for inspection, but it cannot extract subtitle streams",
		strings.Join(missing, ", "),
	)
}

// Extract keeps backward compatibility by extracting only the first subtitle
// and returning its file name and content.
func Extract(inputPath string) (string, []byte, error) {
	paths, err := ExtractAll(inputPath, 1)
	if err != nil {
		return "", nil, err
	}
	if len(paths) == 0 {
		return "", nil, errors.New("no subtitle tracks extracted")
	}
	data, err := osReadFile(paths[0])
	if err != nil {
		return "", nil, fmt.Errorf("read extracted subtitle: %w", err)
	}
	return paths[0], data, nil
}

func ExtractAll(inputPath string, maxTracks int) ([]string, error) {
	if strings.ToLower(filepath.Ext(inputPath)) != ".mkv" {
		return nil, fmt.Errorf("unsupported extension: %s (only .mkv)", filepath.Ext(inputPath))
	}
	tracks, err := probeSubtitleTracks(inputPath)
	if err != nil {
		return nil, err
	}
	if len(tracks) == 0 {
		return nil, errors.New("no subtitle tracks found in mkv")
	}
	sort.Slice(tracks, func(i, j int) bool {
		return subtitleTrackOrders(tracks[i], tracks[j])
	})
	if maxTracks > 0 && maxTracks < len(tracks) {
		tracks = tracks[:maxTracks]
	}

	extCount := map[string]int{}
	outPaths := make([]string, 0, len(tracks))
	baseNoExt := strings.TrimSuffix(inputPath, filepath.Ext(inputPath))

	for _, track := range tracks {
		ext := subtitleExtension(track)
		suffix := extCount[ext]
		outPath := baseNoExt + ext
		if suffix > 0 {
			outPath = fmt.Sprintf("%s_%d%s", baseNoExt, suffix, ext)
		}
		extCount[ext]++

		if err := runFFmpegExtractTrack(inputPath, track.Index, outPath); err != nil {
			return nil, err
		}
		outPaths = append(outPaths, outPath)
	}
	return outPaths, nil
}

func probeSubtitleTracks(inputPath string) ([]subtitleTrack, error) {
	cmd := exec.Command(
		"ffprobe",
		"-v", "error",
		"-select_streams", "s",
		"-show_entries", "stream=index,codec_name,codec_tag_string,codec_long_name:stream_tags=language,title",
		"-of", "json",
		inputPath,
	)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe failed: %w", err)
	}
	var parsed ffprobeOutput
	if err := json.Unmarshal(out, &parsed); err != nil {
		return nil, fmt.Errorf("parse ffprobe output: %w", err)
	}
	return parsed.Streams, nil
}

func runFFmpegExtractTrack(inputPath string, streamIndex int, outPath string) error {
	cmd := exec.Command(
		"ffmpeg",
		"-v", "error",
		"-y",
		"-i", inputPath,
		"-map", fmt.Sprintf("0:%d", streamIndex),
		"-c:s", "copy",
		outPath,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg extract stream %d failed: %v (%s)", streamIndex, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func subtitleExtension(track subtitleTrack) string {
	codec := strings.ToLower(track.Codec)
	tag := strings.ToLower(track.CodecTag)

	switch {
	case codec == "subrip" || codec == "srt":
		return ".srt"
	case codec == "ass" || codec == "ssa":
		return ".ass"
	case codec == "webvtt":
		return ".vtt"
	case codec == "hdmv_pgs_subtitle":
		return ".sup"
	case strings.Contains(codec, "dvd") || strings.Contains(tag, "dvd"):
		return ".sub"
	default:
		return ".sub"
	}
}

func osReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

var execLookPath = exec.LookPath
