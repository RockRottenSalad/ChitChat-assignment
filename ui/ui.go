package ui

import (
	"fmt"
	"strings"
	"os"
	"log"
	term "golang.org/x/term"
)

type UI struct {
	fg Color
	bg Color
	row uint
	column uint

	height uint
	width uint

	prevState *term.State

	buffer *strings.Builder
	callback func(Key)
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

func (ui *UI) charReader() {
//	reader := bufio.NewReader(os.Stdin)
	buf := make([]byte, 3)
	for {
//		char, _, _ := reader.ReadRune()
		n, _ := os.Stdin.Read(buf[:])

		log.Printf("KEY PRESS: %v - %c", buf, rune(buf[0]))

//		fmt.Printf("PRESS: %v\n", buf)

		if n == 1 {
			switch {
			case buf[0] == 3:
				ui.callback(Key{isSpecial: true, special: CtrlC })
			case buf[0] == 127:
				ui.callback(Key{isSpecial: true, special: Backspace })
			case buf[0] == 27:
				ui.callback(Key{isSpecial: true, special: Esc })
			case buf[0] == '\n' || buf[0] == '\r':
				ui.callback(Key{isSpecial: true, special: Return })
			case buf[0] >= 'A' && buf[0] <= 'Z':
				ui.callback(Key{isSpecial: false, letter: rune(buf[0]) })
			case buf[0] >= 'a' && buf[0] <= 'z':
				ui.callback(Key{isSpecial: false, letter: rune(buf[0]) })
			case buf[0] == ' ':
				ui.callback(Key{isSpecial: false, letter: rune(buf[0]) })
			}
		}else if n == 3 {
			if buf[0] == 27 && buf[1] == 91 {
				switch(buf[2]) {
				case 65:
					ui.callback(Key{isSpecial: true, special: ArrowUp })
				case 66:
					ui.callback(Key{isSpecial: true, special: ArrowDown })
				case 67:
					ui.callback(Key{isSpecial: true, special: ArrowRight })
				case 68:
					ui.callback(Key{isSpecial: true, special: ArrowLeft })
				}
			}
		}

	}
}

func NewUI() *UI {
	width, height, _ := term.GetSize(int(os.Stdout.Fd()))
	prevState, _ := term.MakeRaw(int(os.Stdout.Fd()))

	ui := new(UI) 
	*ui = UI {
		fg: Default, bg: Default,
		row: 0, column: 0,
		height: uint(height), width: uint(width),
		buffer: &strings.Builder{},
		prevState: prevState,
		callback: func(ch Key) {fmt.Printf("Unhandled callback: '%q'\n", ch)}}
	ui.buffer.Reset()
	ui.SetCursor(0, 0)
	ui.clear()

	go ui.charReader()

	return ui
}

func (ui *UI) TerminateUI() {
	term.Restore(int(os.Stdout.Fd()), ui.prevState)
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

func (ui *UI) updateTerminalDimensions() {
	width, height, _ := term.GetSize(int(os.Stdout.Fd()))
	ui.width = uint(width)
	ui.height = uint(height)
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

	ui.row += (ui.column + uint(len(str))) / ui.width
	ui.column = (ui.column + uint(len(str))) % ui.width
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

func (ui *UI) WriteCentered(text string, fgColor Color, bgColor Color, style Style) {
	halfLen := uint(len(text) / 2)
	ui.SetCursor(ui.row, ui.width/2 - halfLen)
	ui.Write(text, fgColor, bgColor, style)
}

func (ui *UI) GetUIHeight() uint {
	return ui.height
}

func (ui *UI) GetUIWidth() uint {
	return ui.width
}

func (ui *UI) GetCursor() (uint, uint) {
	return ui.row, ui.column
}

func (ui *UI) Render() {
	fmt.Print(ui.buffer.String())
	ui.buffer.Reset()
	ui.clear()
	ui.updateTerminalDimensions()
}

func (ui *UI) SetCallback(cb func(Key)) {
	ui.callback = cb
}


type Key struct {
	special SpecialKey
	letter rune
	isSpecial bool
}

func (key *Key) IsSpecial() bool {
	return key.isSpecial
}

func (key *Key) GetSpecial() SpecialKey {
	if !key.IsSpecial() {
		log.Fatalln("Attempted to get special on non-special key")
	}
	return key.special
}

func (key *Key) GetLetter() rune {
	if key.IsSpecial() {
		log.Fatalln("Attempted to get letter on special key")
	}
	return key.letter
}

type SpecialKey uint8
const (
	CtrlC SpecialKey = iota
	Esc
	Return
	Backspace

	ArrowUp 
	ArrowDown
	ArrowRight
	ArrowLeft
)
