package app

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
	"golang.org/x/net/html"
)

// markdownImage is a picture of a Markdown preview's HTML block, loading
// or loaded.
type markdownImage struct {
	mu  sync.Mutex
	src ui.ImageSource
}

// markdownImages loads the pictures of a Markdown preview: files beside
// the note, and pictures on the web, as a README's badges are. Relative
// files are read only for a local project.
func markdownImages(dir string, local bool) func(string) (ui.ImageSource, error) {
	files := markdown.FileImages(dir)
	return func(raw string) (ui.ImageSource, error) {
		if isWebImage(raw) {
			return fetchWebImage(raw)
		}
		if !local {
			return nil, os.ErrNotExist
		}
		return files(localImagePath(raw))
	}
}

// localImagePath turns the address of a picture beside a note into its
// path: the cache-busting query of a README's screenshot is not part of
// the file name.
func localImagePath(raw string) string {
	if i := strings.IndexAny(raw, "?#"); i >= 0 {
		raw = raw[:i]
	}
	if unescaped, err := url.PathUnescape(raw); err == nil {
		raw = unescaped
	}
	return raw
}

func isWebImage(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return strings.EqualFold(u.Scheme, "http") || strings.EqualFold(u.Scheme, "https")
}

// webImageClient fetches the pictures a Markdown preview shows. The
// preview loads them off the UI thread; the timeout keeps one slow site
// from holding a placeholder for long.
var webImageClient = &http.Client{Timeout: 20 * time.Second}

const maxWebImageBytes = 16 << 20

func fetchWebImage(raw string) (ui.ImageSource, error) {
	req, err := http.NewRequest(http.MethodGet, shieldPNG(raw), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mojian")
	resp, err := webImageClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("image %s: %s", raw, resp.Status)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxWebImageBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxWebImageBytes {
		return nil, errors.New("image is too large")
	}
	return ui.DecodeImage(data)
}

// shieldPNG asks shields.io for the PNG of a badge. Its SVG draws the
// words as <text>, which the SVG renderer does not draw yet, while the
// same badge as a PNG reads complete.
func shieldPNG(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || !strings.EqualFold(u.Host, "img.shields.io") ||
		strings.HasSuffix(u.Path, ".png") || strings.HasSuffix(u.Path, ".json") {
		return raw
	}
	u.Path += ".png"
	return u.String()
}

// markdownHTMLBlock builds an HTML block of a Markdown preview from the
// little HTML documents carry: paragraphs, links, pictures and text,
// which the Markdown view would otherwise show as source. The pictures
// load on goroutines of their own.
func markdownHTMLBlock(c *ui.Context, src string, images map[string]*markdownImage, load func(string) (ui.ImageSource, error)) {
	doc, err := html.Parse(strings.NewReader(src))
	if err != nil {
		ui.Text(c, src).FontSize(14).TextColor(colorsOf(c).TextMuted)
		return
	}
	body := htmlBody(doc)
	if body == nil {
		return
	}
	htmlChildren(c, body, images, load)
}

func htmlBody(doc *html.Node) *html.Node {
	var body *html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if body != nil {
			return
		}
		if n.Type == html.ElementNode && n.Data == "body" {
			body = n
			return
		}
		for child := n.FirstChild; child != nil && body == nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(doc)
	return body
}

// htmlChildren builds a parent's children, a row for each run of inline
// content between the blocks.
func htmlChildren(c *ui.Context, parent *html.Node, images map[string]*markdownImage, load func(string) (ui.ImageSource, error)) {
	var run []*html.Node
	flush := func() {
		if len(run) == 0 {
			return
		}
		ui.Row(c).Wrap().Gap(6).AlignItems(ui.Center).Children(func() {
			for _, n := range run {
				htmlInlineNode(c, n, images, load)
			}
		})
		run = nil
	}
	for n := parent.FirstChild; n != nil; n = n.NextSibling {
		if htmlSkipped(n) {
			continue
		}
		if htmlBlockElement(n) {
			flush()
			htmlBlock(c, n, images, load)
			continue
		}
		run = append(run, n)
	}
	flush()
}

func htmlBlock(c *ui.Context, n *html.Node, images map[string]*markdownImage, load func(string) (ui.ImageSource, error)) {
	k := colorsOf(c)
	switch n.Data {
	case "hr":
		ui.Box(c).Height(1).AlignSelf(ui.Stretch).Background(k.Hover).Margin(6, 0)
	case "ul", "ol":
		ui.Column(c).Gap(4).AlignSelf(ui.Stretch).Children(func() {
			htmlChildren(c, n, images, load)
		})
	case "li":
		ui.Row(c).AlignItems(ui.Start).Gap(6).Children(func() {
			ui.Text(c, "•").FontSize(14).TextColor(k.TextFaint)
			ui.Column(c).Grow(1).Children(func() {
				htmlChildren(c, n, images, load)
			})
		})
	case "blockquote":
		ui.Row(c).Gap(10).AlignSelf(ui.Stretch).Children(func() {
			ui.Box(c).Width(3).AlignSelf(ui.Stretch).Background(k.Hover)
			ui.Column(c).Grow(1).Children(func() {
				htmlChildren(c, n, images, load)
			})
		})
	default:
		ui.Column(c).AlignSelf(ui.Stretch).Children(func() {
			htmlChildren(c, n, images, load)
		})
	}
}

func htmlInlineNode(c *ui.Context, n *html.Node, images map[string]*markdownImage, load func(string) (ui.ImageSource, error)) {
	k := colorsOf(c)
	switch n.Type {
	case html.TextNode:
		if text := strings.Join(strings.Fields(n.Data), " "); text != "" {
			ui.Text(c, text).FontSize(14).TextColor(k.Text)
		}
	case html.ElementNode:
		if htmlSkipped(n) {
			return
		}
		switch n.Data {
		case "img":
			htmlImage(c, k, n, images, load)
		case "a":
			htmlLink(c, k, n, images, load)
		case "br":
			// The row's gap separates what a break divides.
		default:
			for child := n.FirstChild; child != nil; child = child.NextSibling {
				htmlInlineNode(c, child, images, load)
			}
		}
	}
}

func htmlLink(c *ui.Context, k *colors, n *html.Node, images map[string]*markdownImage, load func(string) (ui.ImageSource, error)) {
	href := htmlAttr(n, "href")
	link := ui.Row(c).AlignItems(ui.Center)
	if ui.SafeToOpen(href) {
		link.Cursor(ui.CursorPointer).Role(ui.RoleLink).Label(href).
			OnClick(func() { _ = mygo.Shell.OpenExternal(href) })
	}
	link.Children(func() {
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			htmlInlineNode(c, child, images, load)
		}
	})
}

func htmlImage(c *ui.Context, k *colors, n *html.Node, images map[string]*markdownImage, load func(string) (ui.ImageSource, error)) {
	alt, src := htmlAttr(n, "alt"), htmlAttr(n, "src")
	img := previewImage(c, images, load, src)
	if img == nil {
		label := alt
		if label == "" {
			label = "Image"
		}
		box := ui.Row(c).Padding(4, 8).Radius(radius).Background(k.Hover)
		if alt != "" {
			box.Role(ui.RoleImage).Label(alt)
		}
		box.Children(func() {
			ui.Text(c, label).FontSize(11).TextColor(k.TextFaint)
		})
		return
	}
	el := ui.Image(c, img).MaxWidthPercent(100).Radius(radius)
	if alt != "" {
		el.Label(alt)
	}
	if title := htmlAttr(n, "title"); title != "" {
		el.Tooltip(title)
	}
	width, height := ui.ImageSize(img)
	aw, ah := htmlPixels(htmlAttr(n, "width")), htmlPixels(htmlAttr(n, "height"))
	switch {
	case aw > 0 && ah > 0:
		el.Width(aw).Height(ah)
	case ah > 0:
		el.Height(ah)
		if width > 0 && height > 0 {
			el.AspectRatio(width / height)
		}
	case aw > 0:
		el.Width(aw)
		if width > 0 && height > 0 {
			el.AspectRatio(width / height)
		}
	case width > 0 && height > 0:
		el.AspectRatio(width / height)
	}
}

// previewImage returns the picture at url once it has loaded, starting
// the load on a goroutine the first time the block asks.
func previewImage(c *ui.Context, cache map[string]*markdownImage, load func(string) (ui.ImageSource, error), url string) ui.ImageSource {
	if url == "" || load == nil {
		return nil
	}
	e, loaded := cache[url]
	if !loaded {
		e = &markdownImage{}
		cache[url] = e
		services := c.Services()
		go func() {
			src, err := load(url)
			if err != nil {
				src = nil
			}
			e.mu.Lock()
			e.src = src
			e.mu.Unlock()
			services.Invalidate()
		}()
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.src
}

func htmlAttr(n *html.Node, name string) string {
	for _, a := range n.Attr {
		if a.Key == name {
			return a.Val
		}
	}
	return ""
}

func htmlPixels(v string) float32 {
	v = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(v), "px"))
	f, err := strconv.ParseFloat(v, 32)
	if err != nil || f <= 0 {
		return 0
	}
	return float32(f)
}

func htmlSkipped(n *html.Node) bool {
	if n.Type == html.CommentNode {
		return true
	}
	if n.Type != html.ElementNode {
		return false
	}
	switch n.Data {
	case "script", "style", "template":
		return true
	}
	return false
}

func htmlBlockElement(n *html.Node) bool {
	return n.Type == html.ElementNode && htmlBlockTags[n.Data]
}

var htmlBlockTags = map[string]bool{
	"address": true, "article": true, "aside": true, "blockquote": true,
	"center": true, "details": true, "div": true, "dl": true, "dd": true,
	"dt": true, "fieldset": true, "figcaption": true, "figure": true,
	"footer": true, "form": true, "h1": true, "h2": true, "h3": true,
	"h4": true, "h5": true, "h6": true, "header": true, "hr": true,
	"li": true, "main": true, "nav": true, "ol": true, "p": true,
	"pre": true, "section": true, "summary": true, "table": true,
	"tbody": true, "td": true, "tfoot": true, "th": true, "thead": true,
	"tr": true, "ul": true,
}
