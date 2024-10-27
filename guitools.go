package main

import (
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
)

// guitools
func (g *gui) openfolder(title string, location *fyne.ListableURI, cb func(fyne.ListableURI, []fyne.URI)) *widget.Button {
	return widget.NewButton(title, func() {
		d := dialog.NewFolderOpen(func(lu fyne.ListableURI, err error) {
			if err != nil || lu == nil {
				return
			}

			cb(lu, filesinfolder(lu))
		}, g.w)

		if location != nil && *location != nil {
			d.SetLocation(*location)
		} else {
			d.SetLocation(currentPathAsURI("."))
		}

		d.Show()
		d.Resize(d.MinSize().Add(d.MinSize()))
	})
}

func filesinfolder(lu fyne.ListableURI) []fyne.URI {
	items, _ := lu.List()
	files := make([]fyne.URI, 0, len(items))
	for _, uri := range items {
		fileinfo, err := os.Lstat(uri.Path())
		if err != nil {
			continue
		}
		mode := fileinfo.Mode()

		if mode.IsRegular() {
			files = append(files, uri)
		}
	}
	return files
}

/*
// dont forget to defer uc.Close()
func (g *gui) openfile(title string, location *fyne.ListableURI, cb func(uc fyne.URIReadCloser) bool) *widget.Button {
	var buttonopen *widget.Button

	buttonopen = widget.NewButton(title, func() {
		d := dialog.NewFileOpen(func(uc fyne.URIReadCloser, err error) {
			if uc == nil || err != nil {
				return
			}

			if cb(uc) {
				buttonopen.Importance = widget.SuccessImportance
			} else {
				buttonopen.Importance = widget.WarningImportance
			}
			buttonopen.Refresh()
		}, g.w)

		if location != nil && *location != nil {
			d.SetLocation(*location)
		} else {
			d.SetLocation(currentPathAsURI("."))
		}

		d.Show()
		d.Resize(d.MinSize().Add(d.MinSize()))
	})
	buttonopen.Importance = widget.WarningImportance

	return buttonopen
}
*/

func currentPathAsURI(basepath string) fyne.ListableURI {
	// if we can open the current dir, do so
	rootdir, err := filepath.Abs(basepath)
	if err != nil {
		rootdir, err = filepath.Abs(".")
		if err != nil {
			return nil
		}
	}
	fileuri := storage.NewFileURI(rootdir)
	diruri, err := storage.ListerForURI(fileuri)
	if err != nil {
		return nil
	}

	return diruri
}

/*
type secondaryTapper struct {
	widget.BaseWidget
	child fyne.CanvasObject
	cb    func(*fyne.PointEvent)
}

var _ fyne.SecondaryTappable = (*secondaryTapper)(nil)

func (st *secondaryTapper) TappedSecondary(pe *fyne.PointEvent) {
	if st.cb == nil {
		panic("no callback set, pointless use of widget")
	}
	st.cb(pe)
}

func NewSecondaryTapperLayer(child fyne.CanvasObject, cb func(*fyne.PointEvent)) *secondaryTapper {
	st := &secondaryTapper{child: child, cb: cb}
	st.ExtendBaseWidget(st)
	return st
}

func (st *secondaryTapper) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(st.child)
}
*/
