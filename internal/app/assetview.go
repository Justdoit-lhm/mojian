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
	"sort"
	"strconv"
	"strings"
	"time"

	"mojian/internal/icons"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
)

// markdownBody shows a note as rendered Markdown, with the same buffer
// the editor writes.
func (a *app) markdownBody(c *ui.Context, k *colors, p *pane) {
	if p.Note == nil {
		return
	}
	if a.focusReq == p {
		a.focusReq = nil
	}
	scroll := ui.Scroll(c).Grow(1).MinHeight(0).TrackScroll(&p.previewScroll)
	scroll.Children(func() {
		ui.Column(c).MaxWidth(780).AlignSelf(ui.Center).
			Padding(26, 34, 44, 34).Children(func() {
			if p.md == nil {
				opts := &markdown.Options{}
				if a.store == nil || !a.store.IsRemote() {
					opts.LoadImage = markdown.FileImages(filepath.Dir(p.Note.Path))
				}
				p.md = opts
			}
			markdown.View(c, p.Note.Text(), p.md)
		})
	})
}

// assetBody shows a picture, a PDF's first page, or why neither can show.
func (a *app) assetBody(c *ui.Context, k *colors, p *pane) {
	if p.Asset == nil {
		return
	}
	if !a.loadAsset(p.Asset) {
		return
	}
	if a.focusReq == p {
		a.focusReq = nil
	}
	if p.assetView == nil {
		p.assetView = &assetView{}
	}
	if isPDFPath(p.Asset.Path) {
		a.pdfBody(c, k, p)
		return
	}
	img, err := p.Asset.Image()
	if err != nil {
		a.assetError(c, k, p, err)
		return
	}
	ui.Column(c).Grow(1).MinHeight(0).Children(func() {
		ui.Box(c).Grow(1).MinHeight(0).Center().Padding(18).Children(func() {
			ui.Image(c, img).Fill().Fit(ui.Contain)
		})
		a.assetFooter(c, k, p, "Image")
	})
}

func (a *app) pdfBody(c *ui.Context, k *colors, p *pane) {
	v := p.assetView
	if !v.pdfStarted {
		v.pdfStarted, v.pdfLoading = true, true
		v.pdfGen++
		gen := v.pdfGen
		path, win := p.Asset.LocalPath(), a.win
		go func() {
			pages, dir, err := pdfPreview(path)
			update := func() {
				// The pane may have been closed or repurposed while the
				// preview was made; do not publish into it, and do not
				// leave its temporary pages behind.
				if p.assetView != v || v.pdfGen != gen {
					if dir != "" {
						_ = os.RemoveAll(dir)
					}
					return
				}
				v.pdfLoading, v.pdfPages, v.pdfDir, v.pdfErr = false, pages, dir, ""
				if err != nil {
					v.pdfErr = err.Error()
				}
				a.invalidate()
			}
			if win != nil {
				win.Update(update)
			} else {
				update()
			}
		}()
	}
	switch {
	case v.pdfErr != "":
		a.assetError(c, k, p, errors.New(v.pdfErr))
	case v.pdfLoading || len(v.pdfPages) == 0:
		ui.Column(c).Grow(1).Center().Gap(8).Children(func() {
			ui.Icon(c, icons.Icon("file-text")).Size(26, 26).TextColor(k.TextMuted)
			ui.Text(c, "Reading PDF…").FontSize(13).FontWeight(600).TextColor(k.Text)
			ui.Text(c, "The page previews are being prepared.").
				FontSize(11.5).TextColor(k.TextFaint)
		})
	default:
		page := min(max(v.pdfPage, 0), len(v.pdfPages)-1)
		if v.pdfImage == nil || v.pdfImageAt != page {
			data, err := os.ReadFile(v.pdfPages[page])
			if err != nil {
				a.assetError(c, k, p, err)
				return
			}
			img, err := ui.DecodeImage(data)
			if err != nil {
				a.assetError(c, k, p, err)
				return
			}
			v.pdfImage, v.pdfImageAt = img, page
		}
		ui.Column(c).Grow(1).MinHeight(0).Children(func() {
			ui.Box(c).Grow(1).MinHeight(0).Center().Padding(18).Children(func() {
				ui.Image(c, v.pdfImage).Fill().Fit(ui.Contain)
			})
			a.pdfFooter(c, k, p)
		})
	}
}

func (a *app) pdfFooter(c *ui.Context, k *colors, p *pane) {
	v := p.assetView
	n := len(v.pdfPages)
	ui.Row(c).Height(31).Padding(0, 12, 0, 12).Gap(8).AlignItems(ui.Center).
		Children(func() {
			ui.Text(c, "PDF").FontSize(10.5).FontWeight(600).TextColor(k.TextFaint)
			ui.Text(c, "·").FontSize(10).TextColor(k.TextFaint)
			if n > 1 {
				if iconButton(c, k, "chevron-left", "Previous Page", 22, 12).
					Disabled(v.pdfPage <= 0).Clicked() {
					v.pdfPage--
					v.pdfImage = nil
					a.invalidate()
				}
				ui.Text(c, fmt.Sprintf("Page %d of %d", v.pdfPage+1, n)).
					FontSize(10.5).TextColor(k.TextMuted)
				if iconButton(c, k, "chevron-right", "Next Page", 22, 12).
					Disabled(v.pdfPage >= n-1).Clicked() {
					v.pdfPage++
					v.pdfImage = nil
					a.invalidate()
				}
			} else {
				ui.Text(c, "First page").FontSize(10.5).TextColor(k.TextMuted)
			}
			ui.Spacer(c)
			ui.Text(c, fileSize(p.Asset.Size)).FontSize(10.5).TextColor(k.TextFaint)
			a.openButton(c, k, p)
		})
}

func (a *app) assetError(c *ui.Context, k *colors, p *pane, err error) {
	ui.Column(c).Grow(1).Center().Gap(10).Padding(24).Children(func() {
		ui.Box(c).Size(48, 48).Radius(radius).Background(k.CardFocused).
			Border(1, k.CardBorderFocused).Center().Children(func() {
			ui.Icon(c, icons.Icon("file-text")).Size(24, 24).TextColor(k.TextMuted)
		})
		ui.Text(c, p.Asset.Name).FontSize(14).FontWeight(600).TextColor(k.Text).
			TextAlign(ui.Center).MaxWidth(460)
		ui.Text(c, err.Error()).FontSize(11.5).TextColor(k.TextFaint).
			TextAlign(ui.Center).MaxWidth(460)
		a.openButton(c, k, p)
	})
}

func (a *app) assetFooter(c *ui.Context, k *colors, p *pane, kind string) {
	ui.Row(c).Height(31).Padding(0, 12, 0, 12).Gap(8).AlignItems(ui.Center).
		Children(func() {
			ui.Text(c, kind).FontSize(10.5).FontWeight(600).TextColor(k.TextFaint)
			ui.Text(c, "·").FontSize(10).TextColor(k.TextFaint)
			ui.Text(c, fileSize(p.Asset.Size)).FontSize(10.5).TextColor(k.TextFaint)
			ui.Spacer(c)
			a.openButton(c, k, p)
		})
}

func (a *app) openButton(c *ui.Context, k *colors, p *pane) {
	if runtime.GOOS == "js" {
		return
	}
	b := ui.Row(c).Padding(4, 9).Radius(radius).Cursor(ui.CursorPointer).
		Role(ui.RoleButton).Label("Open in Default App")
	if b.Hovered() {
		b.Background(k.Hover)
	}
	if b.Clicked() {
		mygo.Shell.OpenPath(p.Asset.LocalPath())
	}
	b.Children(func() {
		ui.Text(c, "Open").FontSize(11.5).FontWeight(550).TextColor(k.Text)
	})
}

func fileSize(n int64) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.0f KB", float64(n)/(1<<10))
	default:
		return fmt.Sprintf("%d B", n)
	}
}

func isPDFPath(path string) bool {
	return strings.EqualFold(filepath.Ext(path), ".pdf")
}

// pdfPreview renders a PDF's pages to PNGs in a temporary directory:
// pdftoppm when it is installed, else Quick Look on macOS for the first
// page. The pages are capped so a long document does not decode without
// bound.
func pdfPreview(path string) (pages []string, dir string, err error) {
	const maxPages = 60
	dir, err = os.MkdirTemp("", "mojian-pdf-")
	if err != nil {
		return nil, "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	if tool, lookErr := exec.LookPath("pdftoppm"); lookErr == nil {
		base := filepath.Join(dir, "page")
		cmd := exec.CommandContext(ctx, tool, "-png", "-r", "110",
			"-f", "1", "-l", strconv.Itoa(maxPages), path, base)
		if _, runErr := cmd.CombinedOutput(); runErr == nil {
			matches, _ := filepath.Glob(base + "-*.png")
			sort.SliceStable(matches, func(i, j int) bool {
				return pageNumber(matches[i]) < pageNumber(matches[j])
			})
			if len(matches) > 0 {
				return matches, dir, nil
			}
		}
	}
	if tool, lookErr := exec.LookPath("qlmanage"); lookErr == nil {
		cmd := exec.CommandContext(ctx, tool, "-t", "-s", "1600", "-o", dir, path)
		if out, runErr := cmd.CombinedOutput(); runErr == nil {
			if matches, _ := filepath.Glob(filepath.Join(dir, "*.png")); len(matches) > 0 {
				sort.Strings(matches)
				return matches[:1], dir, nil
			}
		} else if len(out) > 0 {
			err = errors.New(strings.TrimSpace(string(out)))
		}
	}
	if err == nil {
		err = errors.New("no PDF preview tool found")
	}
	return nil, dir, err
}

var pageNumberPattern = regexp.MustCompile(`-([0-9]+)\.png$`)

func pageNumber(path string) int {
	if m := pageNumberPattern.FindStringSubmatch(path); len(m) == 2 {
		n, _ := strconv.Atoi(m[1])
		return n
	}
	return 0
}

// releaseAsset drops a pane's PDF preview files before it shows another
// file or closes.
func releaseAsset(p *pane) {
	if p == nil || p.assetView == nil {
		return
	}
	p.assetView.pdfGen++
	if p.assetView.pdfDir == "" {
		return
	}
	_ = os.RemoveAll(p.assetView.pdfDir)
	p.assetView.pdfDir = ""
}
