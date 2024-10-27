package main

import (
	"bufio"
	"fmt"
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
	opennewproject := g.openfolder("Open Folder With Images", nil, func(lu fyne.ListableURI, u []fyne.URI) {
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
				// load tags
				s := bufio.NewScanner(content)
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

				// remove dupes
				foundtags := make(map[string]struct{})
				for _, tag := range tags {
					_, exists := foundtags[tag]
					if !exists {
						foundtags[tag] = struct{}{}
						knowndata.Tags = append(knowndata.Tags, tag)
					}
				}

			} else {
				// load image
				goimg, _, err := image.Decode(content)
				if err != nil {
					dialog.ShowError(fmt.Errorf("failed to decode image: %w", err), g.w)
					goto itsthisorgettingannoyedbydeferinsideloop
				}

				knowndata.ImagePath = fileuri
				img := canvas.NewImageFromImage(goimg)
				neww, newh := calculateNewResolution(goimg.Bounds().Dx(), goimg.Bounds().Dy(), 256)
				img.SetMinSize(fyne.Size{Width: float32(neww), Height: float32(newh)})
				img.FillMode = canvas.ImageFillContain
				img.ScaleMode = canvas.ImageScaleSmooth
				knowndata.loadedImage = NewImageHighlightable(img)
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
	opennewproject.Tapped(&fyne.PointEvent{})

	return container.NewCenter(opennewproject)
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
