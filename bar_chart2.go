package chart

import (
	"errors"
	"fmt"
	"io"
	"math"

	"github.com/golang/freetype/truetype"
)

// BarChart2 is a chart that draws double bars on a range.
type BarChart2 struct {
	Title      string
	TitleStyle Style

	ColorPalette ColorPalette

	Width  int
	Height int
	DPI    float64

	BarWidth int

	Background Style
	Canvas     Style

	XAxis Style
	YAxis YAxis

	BarSpacing  int
	LabelFirst  bool
	LabelTop    int
	SubLabelTop int

	Font        *truetype.Font
	defaultFont *truetype.Font

	Bars []DoubleValue
}

// GetDPI returns the dpi for the chart.
func (bc BarChart2) GetDPI() float64 {
	if bc.DPI == 0 {
		return DefaultDPI
	}
	return bc.DPI
}

// GetFont returns the text font.
func (bc BarChart2) GetFont() *truetype.Font {
	if bc.Font == nil {
		return bc.defaultFont
	}
	return bc.Font
}

// GetWidth returns the chart width or the default value.
func (bc BarChart2) GetWidth() int {
	if bc.Width == 0 {
		return DefaultChartWidth
	}
	return bc.Width
}

// GetHeight returns the chart height or the default value.
func (bc BarChart2) GetHeight() int {
	if bc.Height == 0 {
		return DefaultChartHeight
	}
	return bc.Height
}

// GetBarSpacing returns the spacing between bars.
func (bc BarChart2) GetBarSpacing() int {
	if bc.BarSpacing == 0 {
		return DefaultBarSpacing
	}
	return bc.BarSpacing
}

// GetBarWidth returns the default bar width.
func (bc BarChart2) GetBarWidth() int {
	if bc.BarWidth == 0 {
		return DefaultBarWidth
	}
	return bc.BarWidth
}

// Render renders the chart with the given renderer to the given io.Writer.
func (bc BarChart2) Render(rp RendererProvider, w io.Writer) error {
	if len(bc.Bars) == 0 {
		return errors.New("please provide at least one bar")
	}

	r, err := rp(bc.GetWidth(), bc.GetHeight())
	if err != nil {
		return err
	}

	if bc.Font == nil {
		defaultFont, err := GetDefaultFont()
		if err != nil {
			return err
		}
		bc.defaultFont = defaultFont
	}
	r.SetDPI(bc.GetDPI())

	bc.drawBackground(r)

	var canvasBox Box
	var yt []Tick
	var yr Range
	var yf ValueFormatter

	canvasBox = bc.getDefaultCanvasBox()
	yr = bc.getRanges()
	if yr.GetMax()-yr.GetMin() == 0 {
		return fmt.Errorf("invalid data range; cannot be zero")
	}
	yr = bc.setRangeDomains(canvasBox, yr)
	yf = bc.getValueFormatters()

	if bc.hasAxes() {
		yt = bc.getAxesTicks(r, yr, yf)
		canvasBox = bc.getAdjustedCanvasBox(r, canvasBox, yr, yt)
		yr = bc.setRangeDomains(canvasBox, yr)
	}
	bc.drawCanvas(r, canvasBox)
	bc.drawBars(r, canvasBox, yr)
	bc.drawXAxis(r, canvasBox)
	bc.drawYAxis(r, canvasBox, yr, yt)

	bc.drawTitle(r)

	return r.Save(w)
}

func (bc BarChart2) drawCanvas(r Renderer, canvasBox Box) {
	Draw.Box(r, canvasBox, bc.getCanvasStyle())
}

func (bc BarChart2) getRanges() Range {
	var yrange Range
	if bc.YAxis.Range != nil && !bc.YAxis.Range.IsZero() {
		yrange = bc.YAxis.Range
	} else {
		yrange = &ContinuousRange{}
	}

	if !yrange.IsZero() {
		return yrange
	}

	if len(bc.YAxis.Ticks) > 0 {
		tickMin, tickMax := math.MaxFloat64, -math.MaxFloat64
		for _, t := range bc.YAxis.Ticks {
			tickMin = math.Min(tickMin, t.Value)
			tickMax = math.Max(tickMax, t.Value)
		}
		yrange.SetMin(tickMin)
		yrange.SetMax(tickMax)
		return yrange
	}

	min, max := math.MaxFloat64, -math.MaxFloat64
	for _, b := range bc.Bars {
		min = math.Min(b.Val[0], min)
		min = math.Min(b.Val[1], min)
		max = math.Max(b.Val[0], max)
		max = math.Max(b.Val[1], max)
	}

	yrange.SetMin(min)
	yrange.SetMax(max)

	return yrange
}

func (bc BarChart2) drawBackground(r Renderer) {
	Draw.Box(r, Box{
		Right:  bc.GetWidth(),
		Bottom: bc.GetHeight(),
	}, bc.getBackgroundStyle())
}

func (bc BarChart2) drawBars(r Renderer, canvasBox Box, yr Range) {
	xoffset := canvasBox.Left

	width, spacing, _ := bc.calculateScaledTotalWidth(canvasBox)
	bs2 := spacing >> 1

	var barBox Box
	var bxl, bxr0, bxr1, by0, by1 int
	for index, bar := range bc.Bars {
		bxl = xoffset + bs2
		bxr0 = bxl + width/2
		bxr1 = bxl + width

		by0 = canvasBox.Bottom - yr.Translate(bar.Val[0])
		barBox = Box{
			Top:    by0,
			Left:   bxl,
			Right:  bxr0,
			Bottom: canvasBox.Bottom,
		}
		Draw.Box(r, barBox, bar.Style.InheritFrom(bc.styleDefaultsBar(index)))

		by1 = canvasBox.Bottom - yr.Translate(bar.Val[1])
		barBox = Box{
			Top:    by1,
			Left:   bxr0,
			Right:  bxr1,
			Bottom: canvasBox.Bottom,
		}
		Draw.Box(r, barBox, bar.Style.InheritFrom(bc.styleDefaultsBar(index)))

		xoffset += width + spacing
	}
}

func (bc BarChart2) drawXAxis(r Renderer, canvasBox Box) {
	if !bc.XAxis.Hidden {
		axisStyle := bc.XAxis.InheritFrom(bc.styleDefaultsAxes())
		axisStyle.WriteToRenderer(r)

		width, spacing, _ := bc.calculateScaledTotalWidth(canvasBox)

		r.MoveTo(canvasBox.Left, canvasBox.Bottom)
		r.LineTo(canvasBox.Right, canvasBox.Bottom)
		r.Stroke()

		r.MoveTo(canvasBox.Left, canvasBox.Bottom)
		r.LineTo(canvasBox.Left, canvasBox.Bottom+DefaultVerticalTickHeight)
		r.Stroke()

		cursor := canvasBox.Left
		for _, bar := range bc.Bars {
			var bottom, top [2]int
			if bc.LabelFirst {
				top[0] = canvasBox.Bottom + DefaultXAxisMargin
				bottom[0] = top[0] + bc.LabelTop
				top[1] = bottom[0]
				bottom[1] = top[1] + bc.SubLabelTop
			} else {
				top[1] = canvasBox.Bottom + DefaultXAxisMargin
				bottom[1] = top[1] + bc.SubLabelTop
				top[0] = bottom[1]
				bottom[0] = top[0] + bc.LabelTop
			}

			barLabelBox := Box{
				Top:    top[0],
				Left:   cursor,
				Right:  cursor + width + spacing,
				Bottom: bottom[0],
			}
			if len(bar.Label) > 0 {
				Draw.TextWithin(r, bar.Label, barLabelBox, axisStyle)
			}
			axisStyle.WriteToRenderer(r)

			barLabelBox0 := Box{
				Top:    top[1],
				Left:   cursor,
				Right:  cursor + width/2,
				Bottom: bottom[1],
			}
			if len(bar.Lab[0]) > 0 {
				Draw.TextWithin(r, bar.Lab[0], barLabelBox0, axisStyle)
			}
			axisStyle.WriteToRenderer(r)

			barLabelBox1 := Box{
				Top:    top[1],
				Left:   cursor + width/2,
				Right:  cursor + width + spacing,
				Bottom: bottom[1],
			}
			if len(bar.Lab[1]) > 0 {
				Draw.TextWithin(r, bar.Lab[1], barLabelBox1, axisStyle)
			}
			axisStyle.WriteToRenderer(r)

			cursor += width + spacing
		}
	}
}

func (bc BarChart2) drawYAxis(r Renderer, canvasBox Box, yr Range, ticks []Tick) {
	if !bc.YAxis.Style.Hidden {
		axisStyle := bc.YAxis.Style.InheritFrom(bc.styleDefaultsAxes())
		axisStyle.WriteToRenderer(r)

		r.MoveTo(canvasBox.Right, canvasBox.Top)
		r.LineTo(canvasBox.Right, canvasBox.Bottom)
		r.Stroke()

		r.MoveTo(canvasBox.Right, canvasBox.Bottom)
		r.LineTo(canvasBox.Right+DefaultHorizontalTickWidth, canvasBox.Bottom)
		r.Stroke()

		var ty int
		var tb Box
		for _, t := range ticks {
			ty = canvasBox.Bottom - yr.Translate(t.Value)

			axisStyle.GetStrokeOptions().WriteToRenderer(r)
			r.MoveTo(canvasBox.Right, ty)
			r.LineTo(canvasBox.Right+DefaultHorizontalTickWidth, ty)
			r.Stroke()

			axisStyle.GetTextOptions().WriteToRenderer(r)
			tb = r.MeasureText(t.Label)
			Draw.Text(r, t.Label, canvasBox.Right+DefaultYAxisMargin+5, ty+(tb.Height()>>1), axisStyle)
		}

	}
}

func (bc BarChart2) drawTitle(r Renderer) {
	if len(bc.Title) > 0 && !bc.TitleStyle.Hidden {
		r.SetFont(bc.TitleStyle.GetFont(bc.GetFont()))
		r.SetFontColor(bc.TitleStyle.GetFontColor(bc.GetColorPalette().TextColor()))
		titleFontSize := bc.TitleStyle.GetFontSize(bc.getTitleFontSize())
		r.SetFontSize(titleFontSize)

		textBox := r.MeasureText(bc.Title)

		textWidth := textBox.Width()
		textHeight := textBox.Height()

		titleX := (bc.GetWidth() >> 1) - (textWidth >> 1)
		titleY := bc.TitleStyle.Padding.GetTop(DefaultTitleTop) + textHeight

		r.Text(bc.Title, titleX, titleY)
	}
}

func (bc BarChart2) getCanvasStyle() Style {
	return bc.Canvas.InheritFrom(bc.styleDefaultsCanvas())
}

func (bc BarChart2) styleDefaultsCanvas() Style {
	return Style{
		FillColor:   bc.GetColorPalette().CanvasColor(),
		StrokeColor: bc.GetColorPalette().CanvasStrokeColor(),
		StrokeWidth: DefaultCanvasStrokeWidth,
	}
}

func (bc BarChart2) hasAxes() bool {
	return !bc.YAxis.Style.Hidden
}

func (bc BarChart2) setRangeDomains(canvasBox Box, yr Range) Range {
	yr.SetDomain(canvasBox.Height())
	return yr
}

func (bc BarChart2) getDefaultCanvasBox() Box {
	return bc.box()
}

func (bc BarChart2) getValueFormatters() ValueFormatter {
	if bc.YAxis.ValueFormatter != nil {
		return bc.YAxis.ValueFormatter
	}
	return FloatValueFormatter
}

func (bc BarChart2) getAxesTicks(r Renderer, yr Range, yf ValueFormatter) (yticks []Tick) {
	if !bc.YAxis.Style.Hidden {
		yticks = bc.YAxis.GetTicks(r, yr, bc.styleDefaultsAxes(), yf)
	}
	return
}

func (bc BarChart2) calculateEffectiveBarSpacing(canvasBox Box) int {
	totalWithBaseSpacing := bc.calculateTotalBarWidth(bc.GetBarWidth(), bc.GetBarSpacing())
	if totalWithBaseSpacing > canvasBox.Width() {
		lessBarWidths := canvasBox.Width() - (len(bc.Bars) * bc.GetBarWidth())
		if lessBarWidths > 0 {
			return int(math.Ceil(float64(lessBarWidths) / float64(len(bc.Bars))))
		}
		return 0
	}
	return bc.GetBarSpacing()
}

func (bc BarChart2) calculateEffectiveBarWidth(canvasBox Box, spacing int) int {
	totalWithBaseWidth := bc.calculateTotalBarWidth(bc.GetBarWidth(), spacing)
	if totalWithBaseWidth > canvasBox.Width() {
		totalLessBarSpacings := canvasBox.Width() - (len(bc.Bars) * spacing)
		if totalLessBarSpacings > 0 {
			return int(math.Ceil(float64(totalLessBarSpacings) / float64(len(bc.Bars))))
		}
		return 0
	}
	return bc.GetBarWidth()
}

func (bc BarChart2) calculateTotalBarWidth(barWidth, spacing int) int {
	return len(bc.Bars) * (barWidth + spacing)
}

func (bc BarChart2) calculateScaledTotalWidth(canvasBox Box) (width, spacing, total int) {
	spacing = bc.calculateEffectiveBarSpacing(canvasBox)
	width = bc.calculateEffectiveBarWidth(canvasBox, spacing)
	total = bc.calculateTotalBarWidth(width, spacing)
	return
}

func (bc BarChart2) getAdjustedCanvasBox(r Renderer, canvasBox Box, yrange Range, yticks []Tick) Box {
	axesOuterBox := canvasBox.Clone()

	_, _, totalWidth := bc.calculateScaledTotalWidth(canvasBox)

	if !bc.XAxis.Hidden {
		xaxisHeight := DefaultVerticalTickHeight

		axisStyle := bc.XAxis.InheritFrom(bc.styleDefaultsAxes())
		axisStyle.WriteToRenderer(r)

		cursor := canvasBox.Left
		for _, bar := range bc.Bars {
			if len(bar.Label) > 0 {
				barLabelBox := Box{
					Top:    canvasBox.Bottom + DefaultXAxisMargin,
					Left:   cursor,
					Right:  cursor + bc.GetBarWidth() + bc.GetBarSpacing(),
					Bottom: bc.GetHeight(),
				}
				lines := Text.WrapFit(r, bar.Label, barLabelBox.Width(), axisStyle)
				linesBox := Text.MeasureLines(r, lines, axisStyle)

				xaxisHeight = MinInt(linesBox.Height()+(2*DefaultXAxisMargin), xaxisHeight)
			}
		}

		xbox := Box{
			Top:    canvasBox.Top,
			Left:   canvasBox.Left,
			Right:  canvasBox.Left + totalWidth,
			Bottom: bc.GetHeight() - xaxisHeight,
		}

		axesOuterBox = axesOuterBox.Grow(xbox)
	}

	if !bc.YAxis.Style.Hidden {
		axesBounds := bc.YAxis.Measure(r, canvasBox, yrange, bc.styleDefaultsAxes(), yticks)
		axesOuterBox = axesOuterBox.Grow(axesBounds)
	}

	return canvasBox.OuterConstrain(bc.box(), axesOuterBox)
}

// box returns the chart bounds as a box.
func (bc BarChart2) box() Box {
	dpr := bc.Background.Padding.GetRight(10)
	dpb := bc.Background.Padding.GetBottom(50)

	return Box{
		Top:    bc.Background.Padding.GetTop(20),
		Left:   bc.Background.Padding.GetLeft(20),
		Right:  bc.GetWidth() - dpr,
		Bottom: bc.GetHeight() - dpb,
	}
}

func (bc BarChart2) getBackgroundStyle() Style {
	return bc.Background.InheritFrom(bc.styleDefaultsBackground())
}

func (bc BarChart2) styleDefaultsBackground() Style {
	return Style{
		FillColor:   bc.GetColorPalette().BackgroundColor(),
		StrokeColor: bc.GetColorPalette().BackgroundStrokeColor(),
		StrokeWidth: DefaultStrokeWidth,
	}
}

func (bc BarChart2) styleDefaultsBar(index int) Style {
	return Style{
		StrokeColor: bc.GetColorPalette().GetSeriesColor(index),
		StrokeWidth: 3.0,
		FillColor:   bc.GetColorPalette().GetSeriesColor(index),
	}
}

func (bc BarChart2) getTitleFontSize() float64 {
	effectiveDimension := MinInt(bc.GetWidth(), bc.GetHeight())
	if effectiveDimension >= 2048 {
		return 48
	} else if effectiveDimension >= 1024 {
		return 24
	} else if effectiveDimension >= 512 {
		return 18
	} else if effectiveDimension >= 256 {
		return 12
	}
	return 10
}

func (bc BarChart2) styleDefaultsAxes() Style {
	return Style{
		StrokeColor:         bc.GetColorPalette().AxisStrokeColor(),
		Font:                bc.GetFont(),
		FontSize:            DefaultAxisFontSize,
		FontColor:           bc.GetColorPalette().TextColor(),
		TextHorizontalAlign: TextHorizontalAlignCenter,
		TextVerticalAlign:   TextVerticalAlignTop,
		TextWrap:            TextWrapWord,
	}
}

// GetColorPalette returns the color palette for the chart.
func (bc BarChart2) GetColorPalette() ColorPalette {
	if bc.ColorPalette != nil {
		return bc.ColorPalette
	}
	return AlternateColorPalette
}
