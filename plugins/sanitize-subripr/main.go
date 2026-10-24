package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "open" {
		os.Exit(openPane())
	}
	os.Exit(runDeclutter())
}

func openPane() int {
	herdr := os.Getenv("HERDR_BIN_PATH")
	if herdr == "" {
		herdr = "herdr"
	}
	pluginID := os.Getenv("HERDR_PLUGIN_ID")
	if pluginID == "" {
		pluginID = "sanitize-subripr"
	}
	cmd := exec.Command(herdr, "plugin", "pane", "open", "--plugin", pluginID, "--entrypoint", "declutter")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "sanitize-subripr: open pane: %v\n", err)
		return 1
	}
	return 0
}

func runDeclutter() int {
	in := bufio.NewReader(os.Stdin)

	if _, err := exec.LookPath("sanitize"); err != nil {
		fmt.Fprintln(os.Stderr, "sanitize-subripr: 'sanitize' not found on PATH")
		fmt.Fprintln(os.Stderr, "Install with:")
		fmt.Fprintln(os.Stderr, "  go install github.com/luismascotto/subtitle-sanitizer/cmd/sanitize@latest")
		waitEnter(in)
		return 1
	}

	fmt.Print("Input .srt path: ")
	inputPath, err := readLine(in)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sanitize-subripr: read input: %v\n", err)
		return 1
	}
	inputPath = strings.TrimSpace(inputPath)
	if inputPath == "" {
		fmt.Fprintln(os.Stderr, "sanitize-subripr: input path is required")
		waitEnter(in)
		return 1
	}
	if strings.ToLower(filepath.Ext(inputPath)) != ".srt" {
		fmt.Fprintln(os.Stderr, "sanitize-subripr: input must be a .srt file")
		waitEnter(in)
		return 1
	}
	if st, err := os.Stat(inputPath); err != nil || st.IsDir() {
		fmt.Fprintf(os.Stderr, "sanitize-subripr: input not found: %s\n", inputPath)
		waitEnter(in)
		return 1
	}

	fmt.Print("Output .srt path: ")
	outputPath, err := readLine(in)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sanitize-subripr: read output: %v\n", err)
		return 1
	}
	outputPath = strings.TrimSpace(outputPath)
	if outputPath == "" {
		fmt.Fprintln(os.Stderr, "sanitize-subripr: output path is required")
		waitEnter(in)
		return 1
	}
	if strings.ToLower(filepath.Ext(outputPath)) != ".srt" {
		fmt.Fprintln(os.Stderr, "sanitize-subripr: output must be a .srt file")
		waitEnter(in)
		return 1
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "sanitize-subripr: create output dir: %v\n", err)
		waitEnter(in)
		return 1
	}

	fmt.Printf("Running: sanitize --auto --output %s %s\n", outputPath, inputPath)
	cmd := exec.Command("sanitize", "--auto", "--output", outputPath, inputPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "sanitize-subripr: sanitize failed: %v\n", err)
		waitEnter(in)
		return 1
	}

	fmt.Printf("Done. Output: %s\n", outputPath)
	waitEnter(in)
	return 0
}

func readLine(in *bufio.Reader) (string, error) {
	line, err := in.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

func waitEnter(in *bufio.Reader) {
	fmt.Print("Press Enter to close...")
	_, _ = readLine(in)
}
