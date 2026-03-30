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

var execLookPath = exec.LookPath

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

// ExtractSingleSubtitle keeps backward compatibility by extracting only the first subtitle
// and returning its file name and content.
func ExtractSingleSubtitle(inputPath string) (string, []byte, error) {
	path, _, err := ExtractMultipleSubtitles(inputPath, 1)
	if err != nil {
		return "", nil, err
	}
	if path == nil || *path == "" {
		return "", nil, errors.New("no subtitle tracks extracted")
	}
	data, err := os.ReadFile(*path)
	if err != nil {
		return "", nil, fmt.Errorf("read extracted subtitle: %w", err)
	}
	return *path, data, nil
}

func ExtractMultipleSubtitles(inputPath string, maxTracks int) (first *string, list []string, err error) {

	if strings.ToLower(filepath.Ext(inputPath)) != ".mkv" {
		return nil, nil, fmt.Errorf("unsupported extension: %s (only .mkv)", filepath.Ext(inputPath))
	}
	tracks, err := probeSubtitleTracks(inputPath)
	if err != nil {
		return nil, nil, err
	}
	if len(tracks) == 0 {
		return nil, nil, errors.New("no subtitle tracks found in mkv")
	}
	sort.Slice(tracks, func(i, j int) bool {
		return subtitleTrackOrders(tracks[i], tracks[j])
	})
	if maxTracks > 0 && maxTracks < len(tracks) {
		tracks = tracks[:maxTracks]
	}
	//OPTIMIZE: if only one subtitle is being extracted, we can just extract it and return the path
	if len(tracks) == 1 {
		out := fmt.Sprintf("%s%s", inputPath[:len(inputPath)-4], subtitleExtension(tracks[0]))
		if err := runFFmpegExtractTrack(inputPath, tracks[0].Index, out); err != nil {
			return nil, nil, err
		}
		return &out, nil, nil
	}

	collisionCount := map[string]int{}
	subtitleOutPaths := make([]string, 0, len(tracks))
	// remove extension (E:\folder\Mediafolder\file.mkv -> E:\folder\Mediafolder\file)
	baseFilename := inputPath[:len(inputPath)-4]
	sbOutPath := strings.Builder{}
	enoughSpace := max(0, (len(inputPath)+4+4+4)-sbOutPath.Cap()) // 4 for count, 4 for language, 4 extra
	sbOutPath.Grow(enoughSpace)

	for _, track := range tracks {
		// get extension alwithout the dot to match the base filename
		ext := subtitleExtension(track)
		mapKey := fmt.Sprintf("%s%s", track.Tags.Language, ext)

		countSuffix := collisionCount[mapKey]

		sbOutPath.Reset()
		// ->  E:\folder\Mediafolder\file
		sbOutPath.WriteString(baseFilename)

		if countSuffix > 0 {
			// ->  E:\folder\Mediafolder\file_01
			fmt.Fprintf(&sbOutPath, "_%02d", countSuffix)
		}

		collisionCount[mapKey]++

		if track.Tags.Language != "" {
			// ->  E:\folder\Mediafolder\file_01.eng
			fmt.Fprintf(&sbOutPath, ".%s", track.Tags.Language)
		}
		// ->  E:\folder\Mediafolder\file_01.eng.srt
		fmt.Fprintf(&sbOutPath, "%s", ext)

		outString := sbOutPath.String()
		//fmt.Printf("runFFmpegExtractTrack: %s \n", outString)

		if err := runFFmpegExtractTrack(inputPath, track.Index, outString); err != nil {
			return nil, nil, err
		}
		subtitleOutPaths = append(subtitleOutPaths, outString)
	}
	return &subtitleOutPaths[0], subtitleOutPaths, nil
}

func BatchExtractSubtitles(inputPath string) (count int, err error) {
	firstPath, paths, err := ExtractMultipleSubtitles(inputPath, 0)
	if err != nil {
		return 0, err
	}
	if firstPath != nil {
		count = 1
	}
	if paths != nil {
		count = len(paths)
	}
	//return len(paths), nil
	return count, nil
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
