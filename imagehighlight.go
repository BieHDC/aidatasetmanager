package main

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type ImageHighlightable struct {
	widget.BaseWidget
	image *canvas.Image
	rect  canvas.Rectangle
	col   color.NRGBA
}

var _ fyne.Widget = (*ImageHighlightable)(nil)

func NewImageHighlightable(image *canvas.Image) *ImageHighlightable {
	cc := color.NRGBA{R: 255, G: 0, B: 0, A: 0}
	ih := &ImageHighlightable{
		image: image,
		rect: canvas.Rectangle{
			StrokeColor: cc,
			StrokeWidth: theme.Padding() / 2,
		},
		col: cc,
	}
	ih.ExtendBaseWidget(ih)
	return ih
}

func (ih *ImageHighlightable) SetHighlight(highlight bool) {
	if highlight {
		ih.col.A = 255
	} else {
		ih.col.A = 0
	}
	ih.rect.StrokeColor = ih.col
}

// MinSize returns the size that this widget should not shrink below
func (ih *ImageHighlightable) MinSize() fyne.Size {
	return ih.image.MinSize().AddWidthHeight(theme.Padding(), theme.Padding())
}

func (ih *ImageHighlightable) GetImage() *canvas.Image {
	return ih.image
}

type imageHighlightableRenderer struct {
	ih *ImageHighlightable
}

var _ fyne.WidgetRenderer = (*imageHighlightableRenderer)(nil)

// CreateRenderer is a private method to Fyne which links this widget to its renderer
func (ih *ImageHighlightable) CreateRenderer() fyne.WidgetRenderer {
	ih.ExtendBaseWidget(ih)

	return &imageHighlightableRenderer{ih: ih}
}

func (c *imageHighlightableRenderer) Destroy() {}

func (c *imageHighlightableRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{c.ih.image, &c.ih.rect}
}

// Layout the components of the card container.
func (c *imageHighlightableRenderer) Layout(size fyne.Size) {
	if c.ih.image != nil {
		c.ih.image.Move(fyne.NewPos(c.ih.rect.StrokeWidth, c.ih.rect.StrokeWidth))
		c.ih.image.Resize(size.Subtract(fyne.NewSquareSize(c.ih.rect.StrokeWidth * 2)))
	}

	c.ih.rect.Move(fyne.NewPos(0, 0))
	c.ih.rect.Resize(size)
}

// MinSize calculates the minimum size of a card.
// This is based on the filename text, image.
func (c *imageHighlightableRenderer) MinSize() fyne.Size {
	return c.ih.MinSize()
}

func (c *imageHighlightableRenderer) Refresh() {
	c.Layout(c.ih.BaseWidget.Size())
	if c.ih.image != nil {
		c.ih.image.Refresh()
	}
	c.ih.rect.Refresh()
}
