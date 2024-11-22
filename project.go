package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type imageEntry struct {
	ImagePath fyne.URI
	Tags      []string
	//
	loadedImage *ImageHighlightable
	// for jsonl to jsonl only
	mask *string
}

type projectStructure struct {
	parentdir fyne.ListableURI
	data      []imageEntry
}

type jsonlentry struct {
	Image string  `json:"image"`
	Text  string  `json:"text"`
	Mask  *string `json:"mask"`
}

func (g *gui) save(p *projectStructure, cb func(error)) {
	var d dialog.Dialog

	var errs []error
	closefunc := func() {
		cb(errors.Join(errs...))
	}

	asjsonl := widget.NewButton(".jsonl file", func() {
		jsonlfile, err := storage.Child(p.parentdir, p.parentdir.Name()+".jsonl")
		if err != nil {
			cb(err)
			return
		}

		jfw, err := storage.Writer(jsonlfile)
		if err != nil {
			cb(err)
			return
		}
		defer jfw.Close()

		for _, d := range p.data {
			entry := jsonlentry{
				Image: d.ImagePath.Path(),
				Text:  strings.Join(d.Tags, ", "),
				Mask:  d.mask,
			}

			str, err := json.Marshal(entry)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			jfw.Write(str)
			jfw.Write([]byte("\n"))
		}
		d.Hide()
	})

	asdir := widget.NewButton(".txt files", func() {
		for _, d := range p.data {
			txtfilename := strings.TrimSuffix(d.ImagePath.Name(), d.ImagePath.Extension()) + ".txt"

			uri, err := storage.Child(p.parentdir, txtfilename)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			uwc, err := storage.Writer(uri)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			_, err = io.WriteString(uwc, strings.Join(d.Tags, ", "))
			if err != nil {
				errs = append(errs, err)
				//continue
			}

			uwc.Close()
		}
		d.Hide()
	})

	d = dialog.NewCustom("Save as", "Ok", container.NewGridWithColumns(2, asjsonl, asdir), g.w)
	d.SetOnClosed(closefunc)
	d.Show()
	d.Resize(d.MinSize().Add(d.MinSize()))
}

func (g *gui) saveDialogErrorAndCallbackOnSuccess(p *projectStructure, cb func()) {
	g.save(p, func(err error) {
		if err != nil {
			dialog.ShowError(err, g.w)
		} else {
			cb()
		}
	})
}

func collecttags(in []imageEntry) (tags []string) {
	// grab all tags
	var tagsc []string
	for _, t := range in {
		tagsc = append(tagsc, t.Tags...)
	}
	// count them
	collect := make(map[string]int)
	for _, t := range tagsc {
		tag, _ := collect[t]
		tag++
		collect[t] = tag
	}
	// sort them
	type tag_tmp struct {
		tag   string
		count int
	}
	var tagged []tag_tmp
	for ttag, count := range collect {
		tagged = append(tagged, tag_tmp{tag: ttag, count: count})
	}
	slices.SortFunc(tagged, func(a, b tag_tmp) int {
		res := b.count - a.count
		if res == 0 {
			// makes the list stable
			res = strings.Compare(a.tag, b.tag)
		}
		return res
	})
	// copy them over
	for _, data := range tagged {
		tags = append(tags, data.tag)
	}
	return tags
}

func sliceAppendNoDupes(s []string, ss string) []string {
	for _, option := range s {
		if option == ss {
			// already exists, bail
			return s
		}
	}

	// was not found, append
	return append(s, ss)
}

func (g *gui) projectview(p projectStructure) fyne.CanvasObject {
	g.w.SetCloseIntercept(func() {
		dialog.ShowConfirm("Save Changes", "Do you want to save your changes?", func(b bool) {
			if b {
				g.saveDialogErrorAndCallbackOnSuccess(&p, func() { g.w.Close() })
			} else {
				g.w.Close()
			}
		}, g.w)
	})

	var imagelist *widget.List
	currentselectedimageid := -1
	alltagslist := widget.NewCheckGroup(collecttags(p.data), func(s []string) {
		// assign tags to current selected image
		if currentselectedimageid >= 0 {
			p.data[currentselectedimageid].Tags = s
			imagelist.RefreshItem(currentselectedimageid)
		}
	})
	defaultcolumns := g.a.Preferences().IntWithFallback("numcolums", 2)
	alltagslist.SetColumns(defaultcolumns)

	addtoall := widget.NewCheck("Add to All", nil)
	addtag := widget.NewEntry()
	addtag.TextStyle = fyne.TextStyle{Monospace: true}
	addtag.ActionItem = widget.NewButtonWithIcon("", theme.ConfirmIcon(), func() { addtag.OnSubmitted(addtag.Text) })
	addtag.OnSubmitted = func(s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			return // we dont need an empty tag
		}
		// add tag to tag list and apply it to current selected image if any, or all of them
		alltagslist.Append(s)

		if addtoall.Checked {
			for i := range p.data {
				p.data[i].Tags = sliceAppendNoDupes(p.data[i].Tags, s)
			}
		} else if currentselectedimageid >= 0 {
			p.data[currentselectedimageid].Tags = sliceAppendNoDupes(p.data[currentselectedimageid].Tags, s)
		}
		addtag.TypedShortcut(&fyne.ShortcutSelectAll{})

		if currentselectedimageid >= 0 {
			alltagslist.SetSelected(p.data[currentselectedimageid].Tags)
		}
	}

	griditemsize := float32(g.a.Preferences().IntWithFallback("griditemsize", 256))
	imageviewer := container.NewGridWrap(fyne.NewSquareSize(griditemsize))
	imageviewercontainer := container.NewVScroll(imageviewer)

	selectedindexes := make(map[widget.ListItemID]struct{})
	imagelist = widget.NewList(
		// length
		func() int { return len(p.data) },
		// create
		func() fyne.CanvasObject {
			return widget.NewLabel("averagefilename.len")
		},
		// update
		func(lii widget.ListItemID, co fyne.CanvasObject) {
			label, ok := co.(*widget.Label)
			if !ok {
				return
			}

			_, isSelected := selectedindexes[lii]
			if lii == currentselectedimageid {
				label.Importance = widget.DangerImportance
			} else if isSelected {
				label.Importance = widget.SuccessImportance
			} else {
				label.Importance = widget.MediumImportance
			}

			label.SetText(fmt.Sprintf("%s (%d)", p.data[lii].ImagePath.Name(), len(p.data[lii].Tags)))
		},
	)

	swapselected := func(id widget.ListItemID) {
		if currentselectedimageid >= 0 {
			p.data[currentselectedimageid].loadedImage.SetHighlight(false)
		}
		p.data[id].loadedImage.SetHighlight(true)
		currentselectedimageid = id
	}

	imagelist.OnSelected = func(id widget.ListItemID) {
		_, wasSelected := selectedindexes[id]
		if wasSelected {
			if id != currentselectedimageid {
				// promote previously selected to current selected
				swapselected(id)
				alltagslist.SetSelected(p.data[id].Tags)
			} else {
				// demote to unselected
				delete(selectedindexes, id)
				imageviewer.Remove(p.data[id].loadedImage)
				currentselectedimageid = -1
				alltagslist.SetSelected(nil) // clear
			}
		} else {
			// promote to current selected
			selectedindexes[id] = struct{}{}
			swapselected(id)
			imageviewer.Add(p.data[id].loadedImage)
			imageviewercontainer.ScrollToBottom()
			alltagslist.SetSelected(p.data[id].Tags)
		}
		imagelist.Unselect(id)
		imagelist.RefreshItem(id)
		imageviewercontainer.Refresh()
	}

	settings := widget.NewButtonWithIcon("", theme.SettingsIcon(), func() {
		colslabel := widget.NewLabel("")
		cols := widget.NewSlider(1, 12)
		cols.Step = 1
		cols.OnChanged = func(f float64) {
			colslabel.SetText(fmt.Sprintf("%0.0f", f))
		}
		cols.SetValue(float64(defaultcolumns))
		//
		imgsizelabel := widget.NewLabel("")
		imgsize := widget.NewSlider(64, 2048)
		imgsize.Step = 32
		imgsize.OnChanged = func(f float64) {
			imgsizelabel.SetText(fmt.Sprintf("%0.0f", f))
		}
		imgsize.SetValue(float64(griditemsize))

		//
		content := container.NewBorder(nil, nil, nil,
			colslabel,
			widget.NewForm(
				widget.NewFormItem("Columns", cols),
			),
		)
		content2 := container.NewBorder(nil, nil, nil,
			imgsizelabel,
			widget.NewForm(
				widget.NewFormItem("Size", imgsize),
			),
		)
		//

		cb := func(b bool) {
			if b {
				defaultcolumns = int(cols.Value)
				alltagslist.SetColumns(defaultcolumns)
				g.a.Preferences().SetInt("numcolums", defaultcolumns)
				//
				griditemsize = float32(imgsize.Value)
				g.a.Preferences().SetInt("griditemsize", int(imgsize.Value))
				imageviewer = container.NewGridWrap(fyne.NewSquareSize(griditemsize), imageviewer.Objects...)
				imageviewercontainer.Content = imageviewer
				imageviewercontainer.Refresh()
			}
		}

		// manual save
		savenow := widget.NewButtonWithIcon("Save", theme.DocumentSaveIcon(), func() {
			g.saveDialogErrorAndCallbackOnSuccess(&p, func() { dialog.ShowInformation("Success", "Data Saved", g.w) })
		})

		d := dialog.NewCustomConfirm("Settings", "Ok", "Cancel", container.NewVBox(savenow, content, content2), cb, g.w)

		d.Show()
		d.Resize(d.MinSize().AddWidthHeight(d.MinSize().Width*3, 0))
	})

	imgvcont := container.NewVSplit(
		imageviewercontainer,
		imagelist,
	)
	imgvcont.SetOffset(0.6)
	splitter := container.NewHSplit(
		imgvcont,
		container.NewBorder(container.NewBorder(nil, nil, nil, container.NewHBox(addtoall, settings), addtag), nil, nil, nil, container.NewVScroll(alltagslist)),
	)
	splitter.SetOffset(0.6)

	return splitter
}
