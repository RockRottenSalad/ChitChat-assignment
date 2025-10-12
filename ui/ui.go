package ui

import (
	"fmt"
	"strings"
)

type UI struct {
	fg Color
	bg Color
	row uint
	column uint

	buffer *strings.Builder
}

type Color byte
const (
	Default Color = iota
	Reset
	Black
	Red
	Green
	Yellow
	Blue
	Magenta
	Cyan
	White
)

type ColorType bool
const (
	Foreground ColorType = false
	Background ColorType = true
)

type Style byte
const (
	Normal Style = iota
	Bold
	Italic
	Underlined
	Blinking
	Reversed
	Invisible
	Striketrhough
)

// \u001B[
const escape = "\033["

func NewUI() UI {
	ui := UI {fg: Default, bg: Default, row: 0, column: 0, buffer: &strings.Builder{}}
	ui.buffer.Reset()
	ui.SetCursor(0, 0)
	ui.clear()

	return ui
}

func rawColorEscapeCode(color Color) uint {
	switch color {
		case Reset:
			return 0
		case Black:
			return 30
		case Red:
			return 31
		case Green:
			return 32
		case Yellow:
			return 33
		case Blue:
			return 34
		case Magenta:
			return 35
		case Cyan:
			return 36
		case White:
			return 37
		case Default:
			return 39
	}

	return 0
}

func styleEscapeCode(style Style) uint {
	switch style {
		case Normal:
			return 0 // Don't acutally use this, it resets all colors + styles
		case Bold:
			return 1
		case Italic:
			return 3
		case Underlined:
			return 4
		case Blinking:
			return 5
		case Reversed:
			return 7
		case Invisible:
			return 8
		case Striketrhough:
			return 9
	}

	return 0
}

func colorEscapeCode(color Color, colorType ColorType) uint {
	code := rawColorEscapeCode(color)
	if colorType == Foreground {
		return code
	} else {
		// +10 offset indicates the background version of the same color
		return code + 10
	}
}

func (ui *UI) writeText(str string) {
	ui.buffer.WriteString(str)
//	for _, c := range str {
//		if c == '\n' || c == '\t' { c = ' ' }
//		ui.buffer.WriteRune(c)
//	}
	ui.column += uint(len(str))
}

func (ui *UI) writeStyledText(str string, style Style) {
	code := styleEscapeCode(style)

	ui.writeEscape(fmt.Sprintf("%dm", code))
	ui.writeText(str)

	// +20 offset indicates the end marker for the styling
	ui.writeEscape(fmt.Sprintf("%dm", code+20))
}

func (ui *UI) writeEscape(str string) {
	ui.buffer.WriteString(escape)
	ui.writeText(str)
}

func (ui *UI) writeColor(color Color, colorType ColorType) {
	ui.writeEscape(fmt.Sprintf("%dm", colorEscapeCode(color, colorType)))
}

func (ui *UI) clear() {
	ui.writeEscape("2J")
}

func (ui *UI) SetCursor(row uint, column uint) {
	ui.row = row
	ui.column = column
	ui.writeEscape(fmt.Sprintf("%d;%df", ui.row+1, ui.column+1))
//	ui.writeEscape("0m")
}

func (ui *UI) Write(text string, fgColor Color, bgColor Color, style Style) {
	if fgColor != ui.fg {
		ui.writeColor(fgColor, Foreground)
	}

	if bgColor != ui.bg {
		ui.writeColor(bgColor, Background)
	}

	if style == Normal {
		ui.writeText(text)
	} else {
		ui.writeStyledText(text, style)
	}

	ui.writeEscape("0m")
}

func (ui *UI) Render() {
	fmt.Print(ui.buffer.String())
	ui.buffer.Reset()
	ui.clear()
}

