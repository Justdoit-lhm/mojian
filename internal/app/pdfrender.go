package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// pdfOpen makes the temporary directory a PDF's page images are drawn into
// and reads the document's page count. Poppler answers when it is
// installed; macOS asks PDFKit, which can count and draw every page
// without cgo. Quick Look draws only the first page, so a document only it
// can read counts as a single page.
func pdfOpen(path string) (dir string, count int, err error) {
	dir, err = os.MkdirTemp("", "mojian-pdf-")
	if err != nil {
		return "", 0, err
	}
	var lastErr error
	if tool := findPDFTool("pdfinfo"); tool != "" {
		n, infoErr := pdfInfoPages(tool, path)
		switch {
		case infoErr == nil && n > 0:
			return dir, n, nil
		case infoErr != nil:
			lastErr = infoErr
		}
	}
	if runtime.GOOS == "darwin" {
		n, kitErr := pdfKitPages(path)
		switch {
		case kitErr == nil && n > 0:
			return dir, n, nil
		case kitErr != nil:
			lastErr = kitErr
		}
	}
	if findPDFTool("qlmanage") != "" {
		return dir, 1, nil
	}
	if lastErr != nil {
		return dir, 0, lastErr
	}
	return dir, 0, errors.New("no PDF preview tool found")
}

// pdfRenderPage draws one page as a PNG in dir and returns its path.
// Poppler draws it when installed; otherwise macOS draws it with PDFKit,
// and Quick Look can stand in for the first page.
func pdfRenderPage(path, dir string, page int) (string, error) {
	var lastErr error
	if tool := findPDFTool("pdftoppm"); tool != "" {
		file, err := pdftoppmPage(tool, path, dir, page)
		if err == nil {
			return file, nil
		}
		lastErr = err
	}
	if runtime.GOOS == "darwin" {
		file, err := pdfKitPage(path, dir, page)
		if err == nil {
			return file, nil
		}
		lastErr = err
	}
	if page == 1 {
		if tool := findPDFTool("qlmanage"); tool != "" {
			file, err := quickLookPage(tool, path, dir)
			if err == nil {
				return file, nil
			}
			lastErr = err
		}
	}
	if lastErr == nil {
		lastErr = errors.New("no PDF preview tool found")
	}
	return "", lastErr
}

// findPDFTool looks for an executable on PATH and in the folders a GUI
// app's short PATH leaves out, so a Homebrew Poppler is still found when
// Mojian starts from Finder.
func findPDFTool(name string) string {
	if path, err := exec.LookPath(name); err == nil {
		return path
	}
	for _, dir := range []string{"/opt/homebrew/bin", "/usr/local/bin", "/opt/local/bin", "/usr/bin", "/bin"} {
		path := filepath.Join(dir, name)
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path
		}
	}
	return ""
}

var pdfInfoPagesPattern = regexp.MustCompile(`(?m)^Pages:\s+([0-9]+)`)

func pdfInfoPages(tool, path string) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	out, runErr := exec.CommandContext(ctx, tool, path).CombinedOutput()
	// pdfinfo reports harmless syntax warnings on stderr and still prints
	// the count, so read it from the combined output first.
	if m := pdfInfoPagesPattern.FindSubmatch(out); len(m) == 2 {
		return strconv.Atoi(string(m[1]))
	}
	if runErr != nil {
		return 0, fmt.Errorf("reading %s: %w", filepath.Base(path), runErr)
	}
	return 0, fmt.Errorf("reading %s: no page count", filepath.Base(path))
}

func pdfKitPages(path string) (int, error) {
	out, err := runJXA(pdfKitCountScript, path)
	if err != nil {
		return 0, err
	}
	n, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("reading %s: no page count", filepath.Base(path))
	}
	return n, nil
}

func pdfKitPage(path, dir string, page int) (string, error) {
	out := filepath.Join(dir, fmt.Sprintf("pdfkit-%d.png", page))
	if _, err := runJXA(pdfKitPageScript, path, out, strconv.Itoa(page)); err != nil {
		return "", err
	}
	if info, err := os.Stat(out); err != nil || info.Size() == 0 {
		return "", fmt.Errorf("PDFKit drew no page %d", page)
	}
	return out, nil
}

func runJXA(script string, args ...string) ([]byte, error) {
	tool := findPDFTool("osascript")
	if tool == "" {
		return nil, errors.New("osascript not found")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	argv := append([]string{"-l", "JavaScript", "-e", script}, args...)
	out, err := exec.CommandContext(ctx, tool, argv...).Output()
	if err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			if msg := strings.TrimSpace(string(exit.Stderr)); msg != "" {
				return nil, cleanJXAError(msg)
			}
		}
		return nil, err
	}
	return out, nil
}

// cleanJXAError turns osascript's "execution error: Error: Error: cannot
// open … (-2700)" into the message alone.
func cleanJXAError(msg string) error {
	msg = strings.TrimPrefix(msg, "execution error: ")
	for strings.HasPrefix(msg, "Error: ") {
		msg = strings.TrimPrefix(msg, "Error: ")
	}
	msg = jxaCodePattern.ReplaceAllString(msg, "")
	if msg = strings.TrimSpace(msg); msg == "" {
		msg = "PDFKit failed"
	}
	return errors.New(msg)
}

var jxaCodePattern = regexp.MustCompile(` \(-?[0-9]+\)$`)

func pdftoppmPage(tool, path, dir string, page int) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	base := filepath.Join(dir, "page")
	number := strconv.Itoa(page)
	out, err := exec.CommandContext(ctx, tool, "-png", "-r", "110",
		"-f", number, "-l", number, path, base).CombinedOutput()
	if err != nil {
		if msg := strings.TrimSpace(string(out)); msg != "" {
			return "", errors.New(msg)
		}
		return "", err
	}
	matches, _ := filepath.Glob(base + "-*.png")
	for _, match := range matches {
		if pageNumber(match) == page {
			return match, nil
		}
	}
	return "", fmt.Errorf("pdftoppm drew no page %d", page)
}

var pageNumberPattern = regexp.MustCompile(`-([0-9]+)\.png$`)

func pageNumber(path string) int {
	if m := pageNumberPattern.FindStringSubmatch(path); len(m) == 2 {
		n, _ := strconv.Atoi(m[1])
		return n
	}
	return 0
}

func quickLookPage(tool, path, dir string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, tool, "-t", "-s", "1600", "-o", dir, path).CombinedOutput()
	if err != nil {
		if msg := strings.TrimSpace(string(out)); msg != "" {
			return "", errors.New(msg)
		}
		return "", err
	}
	file := filepath.Join(dir, filepath.Base(path)+".png")
	if info, err := os.Stat(file); err == nil && info.Size() > 0 {
		return file, nil
	}
	if matches, _ := filepath.Glob(filepath.Join(dir, "*.pdf.png")); len(matches) == 1 {
		return matches[0], nil
	}
	return "", errors.New("Quick Look drew no page")
}

// pdfKitPageScript draws a page with PDFKit, which macOS ships, so the
// viewer needs no extra program; osascript runs JavaScript against the
// system frameworks without cgo. argv: path, output PNG, 1-based page.
const pdfKitPageScript = `ObjC.import('PDFKit')
ObjC.import('Foundation')
ObjC.import('AppKit')
function run(argv) {
	var doc = $.PDFDocument.alloc.initWithURL($.NSURL.fileURLWithPath(argv[0]))
	if (doc.isNil()) throw new Error('cannot open ' + argv[0])
	var page = doc.pageAtIndex(Number(argv[2]) - 1)
	if (page.isNil()) throw new Error('no page ' + argv[2])
	var box = page.boundsForBox($.kPDFDisplayBoxCropBox)
	var width = box.size.width, height = box.size.height
	var scale = 1600 / Math.max(1, width, height)
	var image = page.thumbnailOfSizeForBox(
		$.NSMakeSize(Math.max(1, Math.round(width * scale)), Math.max(1, Math.round(height * scale))),
		$.kPDFDisplayBoxCropBox)
	var rep = $.NSBitmapImageRep.imageRepWithData(image.TIFFRepresentation)
	var data = rep.representationUsingTypeProperties($.NSBitmapImageFileTypePNG, $({}))
	if (!data.writeToFileAtomically(argv[1], true)) throw new Error('cannot write ' + argv[1])
	return argv[2]
}`

const pdfKitCountScript = `ObjC.import('PDFKit')
ObjC.import('Foundation')
function run(argv) {
	var doc = $.PDFDocument.alloc.initWithURL($.NSURL.fileURLWithPath(argv[0]))
	if (doc.isNil()) throw new Error('cannot open ' + argv[0])
	return String(doc.pageCount)
}`
