package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

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
				local := a.store == nil || !a.store.IsRemote()
				load := markdownImages(filepath.Dir(p.Note.Path), local)
				// HTML blocks carry a README's badges and linked
				// pictures, which the view would otherwise show as
				// source.
				images := map[string]*markdownImage{}
				opts.LoadImage = load
				opts.HTML = func(c *ui.Context, block string) {
					markdownHTMLBlock(c, block, images, load)
				}
				p.md = opts
			}
			markdown.View(c, p.Note.Text(), p.md)
		})
	})
}

// assetBody shows a picture or a PDF, or why neither can show.
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
	if v.pdfFiles == nil {
		v.pdfFiles = map[int]string{}
		v.pdfPending = map[int]bool{}
		v.pdfPageErr = map[int]string{}
	}
	path, win := p.Asset.LocalPath(), a.win
	if !v.pdfStarted {
		v.pdfStarted, v.pdfLoading = true, true
		v.pdfGen++
		gen := v.pdfGen
		go func() {
			dir, count, err := pdfOpen(path)
			update := func() {
				// The pane may have been closed or repurposed while the
				// count was read; do not publish into it, and do not leave
				// its temporary files behind.
				if p.assetView != v || v.pdfGen != gen {
					if dir != "" {
						_ = os.RemoveAll(dir)
					}
					return
				}
				v.pdfLoading, v.pdfDir, v.pdfCount = false, dir, count
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
	case v.pdfLoading:
		pdfStatus(c, k, "Reading PDF…", "The document's pages are being counted.")
	case v.pdfCount == 0:
		a.assetError(c, k, p, errors.New("this PDF has no pages"))
	default:
		page := min(max(v.pdfPage, 0), v.pdfCount-1)
		v.pdfPage = page
		errText, failed := v.pdfPageErr[page]
		if !failed && (v.pdfImage == nil || v.pdfImageAt != page) {
			if file, drawn := v.pdfFiles[page]; drawn {
				data, err := os.ReadFile(file)
				var img ui.ImageSource
				if err == nil {
					img, err = ui.DecodeImage(data)
				}
				if err != nil {
					v.pdfPageErr[page] = err.Error()
					errText, failed = v.pdfPageErr[page], true
				} else {
					v.pdfImage, v.pdfImageAt = img, page
				}
			} else {
				// Draw only the page on screen; a long document does
				// not need a PNG for every page it may never show. One
				// page is drawn at a time, so flipping through pages
				// quickly does not pile up renderer processes.
				if len(v.pdfPending) == 0 {
					v.pdfPending[page] = true
					gen, dir := v.pdfGen, v.pdfDir
					go func() {
						file, err := pdfRenderPage(path, dir, page+1)
						update := func() {
							if p.assetView != v || v.pdfGen != gen {
								return
							}
							delete(v.pdfPending, page)
							if err != nil {
								v.pdfPageErr[page] = err.Error()
							} else {
								v.pdfFiles[page] = file
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
				pdfStatus(c, k, fmt.Sprintf("Drawing page %d…", page+1),
					"Pages are drawn as you turn to them.")
				return
			}
		}
		ui.Column(c).Grow(1).MinHeight(0).Children(func() {
			if failed {
				ui.Box(c).Grow(1).MinHeight(0).Center().Padding(24).Children(func() {
					ui.Column(c).Center().Gap(6).Children(func() {
						ui.Text(c, fmt.Sprintf("Page %d of this PDF could not be drawn.", page+1)).
							FontSize(12).FontWeight(550).TextColor(k.TextMuted).
							TextAlign(ui.Center).MaxWidth(460)
						ui.Text(c, errText).FontSize(11).TextColor(k.TextFaint).
							TextAlign(ui.Center).MaxWidth(460)
					})
				})
			} else {
				ui.Box(c).Grow(1).MinHeight(0).Center().Padding(18).Children(func() {
					ui.Image(c, v.pdfImage).Fill().Fit(ui.Contain)
				})
			}
			a.pdfFooter(c, k, p)
		})
	}
}

func pdfStatus(c *ui.Context, k *colors, title, note string) {
	ui.Column(c).Grow(1).Center().Gap(8).Children(func() {
		ui.Icon(c, icons.Icon("file-text")).Size(26, 26).TextColor(k.TextMuted)
		ui.Text(c, title).FontSize(13).FontWeight(600).TextColor(k.Text)
		ui.Text(c, note).FontSize(11.5).TextColor(k.TextFaint)
	})
}

func (a *app) pdfFooter(c *ui.Context, k *colors, p *pane) {
	v := p.assetView
	n := v.pdfCount
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
