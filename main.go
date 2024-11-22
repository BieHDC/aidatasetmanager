package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"strings"

	"image"
	_ "image/jpeg"
	_ "image/png"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
)

type gui struct {
	a fyne.App
	w fyne.Window
}

func main() {
	g := gui{}

	g.a = app.NewWithID("biehdc.priv.aidatasetmanager")
	g.w = g.a.NewWindow("Ai Dataset Manager")
	g.w.CenterOnScreen()
	g.w.Resize(fyne.NewSize(1128, 768))

	g.w.SetContent(g.content())
	g.w.ShowAndRun()
}

func (g *gui) content() fyne.CanvasObject {
	asdir := g.openfolder("Open Folder With Images", nil, func(lu fyne.ListableURI, u []fyne.URI) {
		entries := make(map[string]imageEntry)

		for _, fileuri := range u {
			// filter filenames
			extension := fileuri.Extension()
			switch extension {
			case ".txt", ".png", ".jpg", ".jpeg":
				// we gaming
			default:
				dialog.ShowError(fmt.Errorf("unhandled file extension: %s", extension), g.w)
				continue
			}

			key := strings.TrimRight(fileuri.Path(), extension)
			knowndata := entries[key]

			content, err := storage.Reader(fileuri)
			if err != nil {
				dialog.ShowError(fmt.Errorf("failed to read file: %w", err), g.w)
				continue
			}

			if extension == ".txt" {
				knowndata.Tags = loadtags(content)

			} else {
				nih, err := loadimage(content)
				if err != nil {
					dialog.ShowError(fmt.Errorf("failed to decode image: %w", err), g.w)
					goto itsthisorgettingannoyedbydeferinsideloop
				}
				knowndata.loadedImage = nih
				knowndata.ImagePath = fileuri
			}

			entries[key] = knowndata

		itsthisorgettingannoyedbydeferinsideloop: // cope and seethe about it
			content.Close()
		}

		project := projectStructure{parentdir: lu}
		for k, v := range entries {
			if v.ImagePath == nil {
				dialog.ShowError(fmt.Errorf("file %s.txt has no assosiacted image path, skipping", k), g.w)
				continue
			}
			if v.loadedImage == nil {
				dialog.ShowError(fmt.Errorf("file %s.txt has no loaded image, skipping", k), g.w)
				continue
			}
			project.data = append(project.data, v)
		}
		if len(project.data) < 1 {
			dialog.ShowError(fmt.Errorf("there was nothing useable in this folder"), g.w)
			return
		}
		slices.SortFunc(project.data, func(a, b imageEntry) int {
			return strings.Compare(a.ImagePath.Path(), b.ImagePath.Path())
		})

		g.w.SetContent(g.projectview(project))
	})

	asjsonl := g.openfile("Open JSONL", nil, func(uc fyne.URIReadCloser) bool {
		defer uc.Close()
		parent2, err := storage.Parent(uc.URI())
		if err != nil {
			dialog.ShowError(fmt.Errorf("could not get parent for the jsonl: %w", err), g.w)
			return false
		}
		parent, err := storage.ListerForURI(parent2)
		if err != nil {
			dialog.ShowError(fmt.Errorf("could not get lister for the jsonl parent: %w", err), g.w)
			return false
		}

		project := projectStructure{parentdir: parent}

		entries := bufio.NewScanner(uc)
		for entries.Scan() {
			var jsonlline jsonlentry
			err := json.Unmarshal(entries.Bytes(), &jsonlline)
			if err != nil {
				dialog.ShowError(fmt.Errorf("error unmarshalling line: %w", err), g.w)
				return false
			}

			ie := imageEntry{
				ImagePath: storage.NewFileURI(jsonlline.Image),
				Tags:      loadtags(strings.NewReader(jsonlline.Text)),
				mask:      jsonlline.Mask,
			}

			content, err := storage.Reader(ie.ImagePath)
			if err != nil {
				dialog.ShowError(fmt.Errorf("failed to read file: %w", err), g.w)
				continue
			}

			nih, err := loadimage(content)
			if err != nil {
				dialog.ShowError(fmt.Errorf("failed to decode image: %w", err), g.w)
				goto itsthisorgettingannoyedbydeferinsideloop2
			}
			ie.loadedImage = nih

			project.data = append(project.data, ie)

		itsthisorgettingannoyedbydeferinsideloop2: // cope and seethe about it yet again
			content.Close()
		}

		if len(project.data) < 1 {
			dialog.ShowError(fmt.Errorf("there was nothing useable in this jsonl"), g.w)
			return false
		}
		slices.SortFunc(project.data, func(a, b imageEntry) int {
			return strings.Compare(a.ImagePath.Path(), b.ImagePath.Path())
		})

		g.w.SetContent(g.projectview(project))
		return true
	})

	return container.NewCenter(container.NewGridWithColumns(2, asjsonl, asdir))
}

func loadtags(r io.Reader) []string {
	s := bufio.NewScanner(r)
	var tags []string
	for s.Scan() {
		for _, tag := range strings.Split(s.Text(), ",") {
			trimmed := strings.TrimSpace(tag)
			if trimmed == "" {
				continue
			}
			tags = append(tags, trimmed)
		}
	}

	final := make([]string, 0, len(tags))
	// remove dupes
	foundtags := make(map[string]struct{})
	for _, tag := range tags {
		_, exists := foundtags[tag]
		if !exists {
			foundtags[tag] = struct{}{}
			final = append(final, tag)
		}
	}

	return final
}

func loadimage(r io.Reader) (*ImageHighlightable, error) {
	goimg, _, err := image.Decode(r)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	img := canvas.NewImageFromImage(goimg)
	neww, newh := calculateNewResolution(goimg.Bounds().Dx(), goimg.Bounds().Dy(), 256)
	img.SetMinSize(fyne.Size{Width: float32(neww), Height: float32(newh)})
	img.FillMode = canvas.ImageFillContain
	img.ScaleMode = canvas.ImageScaleSmooth
	return NewImageHighlightable(img), nil
}

func calculateNewResolution[RR LooksLikeNumber](width, height, maxside RR) (RR, RR) {
	if width > height {
		return maxside, RR((float64(height) / float64(width)) * float64(maxside))
	} else {
		return RR((float64(width) / float64(height)) * float64(maxside)), maxside
	}
}

type LooksLikeNumber interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64
}
