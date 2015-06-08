// Copyright 2014 shiena Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// +build windows

package ansicolor

import (
	"bytes"
	"io"
	"strings"
	"syscall"
	"unsafe"
)

type csiState int

const (
	outsideCsiCode csiState = iota
	firstCsiCode
	secondeCsiCode
)

type ansiColorWriter struct {
	w        io.Writer
	state    csiState
	paramBuf bytes.Buffer
}

const (
	firstCsiChar   byte = '\x1b'
	secondeCsiChar byte = '['
	separatorChar  byte = ';'
	sgrCode        byte = 'm'
)

const (
	foregroundBlue      = uint16(0x0001)
	foregroundGreen     = uint16(0x0002)
	foregroundRed       = uint16(0x0004)
	foregroundIntensity = uint16(0x0008)
	backgroundBlue      = uint16(0x0010)
	backgroundGreen     = uint16(0x0020)
	backgroundRed       = uint16(0x0040)
	backgroundIntensity = uint16(0x0080)
	underscore          = uint16(0x8000)

	foregroundMask = foregroundBlue | foregroundGreen | foregroundRed | foregroundIntensity
	backgroundMask = backgroundBlue | backgroundGreen | backgroundRed | backgroundIntensity
)

const (
	ansiReset        = "0"
	ansiIntensityOn  = "1"
	ansiIntensityOff = "21"
	ansiUnderlineOn  = "4"
	ansiUnderlineOff = "24"
	ansiBlinkOn      = "5"
	ansiBlinkOff     = "25"

	ansiForegroundBlack   = "30"
	ansiForegroundRed     = "31"
	ansiForegroundGreen   = "32"
	ansiForegroundYellow  = "33"
	ansiForegroundBlue    = "34"
	ansiForegroundMagenta = "35"
	ansiForegroundCyan    = "36"
	ansiForegroundWhite   = "37"
	ansiForegroundDefault = "39"

	ansiBackgroundBlack   = "40"
	ansiBackgroundRed     = "41"
	ansiBackgroundGreen   = "42"
	ansiBackgroundYellow  = "43"
	ansiBackgroundBlue    = "44"
	ansiBackgroundMagenta = "45"
	ansiBackgroundCyan    = "46"
	ansiBackgroundWhite   = "47"
	ansiBackgroundDefault = "49"
)

type drawType int

const (
	foreground drawType = iota
	background
)

type winColor struct {
	code     uint16
	drawType drawType
}

var colorMap = map[string]winColor{
	ansiForegroundBlack:   {0, foreground},
	ansiForegroundRed:     {foregroundRed, foreground},
	ansiForegroundGreen:   {foregroundGreen, foreground},
	ansiForegroundYellow:  {foregroundRed | foregroundGreen, foreground},
	ansiForegroundBlue:    {foregroundBlue, foreground},
	ansiForegroundMagenta: {foregroundRed | foregroundBlue, foreground},
	ansiForegroundCyan:    {foregroundGreen | foregroundBlue, foreground},
	ansiForegroundWhite:   {foregroundRed | foregroundGreen | foregroundBlue, foreground},
	ansiForegroundDefault: {foregroundRed | foregroundGreen | foregroundBlue, foreground},

	ansiBackgroundBlack:   {0, background},
	ansiBackgroundRed:     {backgroundRed, background},
	ansiBackgroundGreen:   {backgroundGreen, background},
	ansiBackgroundYellow:  {backgroundRed | backgroundGreen, background},
	ansiBackgroundBlue:    {backgroundBlue, background},
	ansiBackgroundMagenta: {backgroundRed | backgroundBlue, background},
	ansiBackgroundCyan:    {backgroundGreen | backgroundBlue, background},
	ansiBackgroundWhite:   {backgroundRed | backgroundGreen | backgroundBlue, background},
	ansiBackgroundDefault: {0, background},
}

var (
	kernel32                       = syscall.NewLazyDLL("kernel32.dll")
	procSetConsoleTextAttribute    = kernel32.NewProc("SetConsoleTextAttribute")
	procGetConsoleScreenBufferInfo = kernel32.NewProc("GetConsoleScreenBufferInfo")
	defaultAttr                    *textAttributes
)

func init() {
	screenInfo := getConsoleScreenBufferInfo(uintptr(syscall.Stdout))
	if screenInfo != nil {
		colorMap[ansiForegroundDefault] = winColor{
			screenInfo.WAttributes & (foregroundRed | foregroundGreen | foregroundBlue),
			foreground,
		}
		colorMap[ansiBackgroundDefault] = winColor{
			screenInfo.WAttributes & (backgroundRed | backgroundGreen | backgroundBlue),
			background,
		}
		defaultAttr = convertTextAttr(screenInfo.WAttributes)
	}
}

type coord struct {
	X, Y int16
}

type smallRect struct {
	Left, Top, Right, Bottom int16
}

type consoleScreenBufferInfo struct {
	DwSize              coord
	DwCursorPosition    coord
	WAttributes         uint16
	SrWindow            smallRect
	DwMaximumWindowSize coord
}

func getConsoleScreenBufferInfo(hConsoleOutput uintptr) *consoleScreenBufferInfo {
	var csbi consoleScreenBufferInfo
	ret, _, _ := procGetConsoleScreenBufferInfo.Call(
		hConsoleOutput,
		uintptr(unsafe.Pointer(&csbi)))
	if ret == 0 {
		return nil
	}
	return &csbi
}

func setConsoleTextAttribute(hConsoleOutput uintptr, wAttributes uint16) bool {
	ret, _, _ := procSetConsoleTextAttribute.Call(
		hConsoleOutput,
		uintptr(wAttributes))
	return ret != 0
}

type textAttributes struct {
	foregroundColor     uint16
	backgroundColor     uint16
	foregroundIntensity uint16
	backgroundIntensity uint16
	underscore          uint16
	otherAttributes     uint16
}

func convertTextAttr(winAttr uint16) *textAttributes {
	fgColor := winAttr & (foregroundRed | foregroundGreen | foregroundBlue)
	bgColor := winAttr & (backgroundRed | backgroundGreen | backgroundBlue)
	fgIntensity := winAttr & foregroundIntensity
	bgIntensity := winAttr & backgroundIntensity
	underline := winAttr & underscore
	otherAttributes := winAttr &^ (foregroundMask | backgroundMask | underscore)
	return &textAttributes{fgColor, bgColor, fgIntensity, bgIntensity, underline, otherAttributes}
}

func convertWinAttr(textAttr *textAttributes) uint16 {
	var winAttr uint16 = 0
	winAttr |= textAttr.foregroundColor
	winAttr |= textAttr.backgroundColor
	winAttr |= textAttr.foregroundIntensity
	winAttr |= textAttr.backgroundIntensity
	winAttr |= textAttr.underscore
	winAttr |= textAttr.otherAttributes
	return winAttr
}

func changeColor(param []byte) {
	if defaultAttr == nil {
		return
	}

	screenInfo := getConsoleScreenBufferInfo(uintptr(syscall.Stdout))
	if screenInfo == nil {
		return
	}

	winAttr := convertTextAttr(screenInfo.WAttributes)
	strParam := string(param)
	if len(strParam) <= 0 {
		strParam = "0"
	}
	csiParam := strings.Split(strParam, string(separatorChar))
	for _, p := range csiParam {
		c, ok := colorMap[p]
		switch {
		case !ok:
			switch p {
			case ansiReset:
				winAttr.foregroundColor = defaultAttr.foregroundColor
				winAttr.backgroundColor = defaultAttr.backgroundColor
				winAttr.foregroundIntensity = defaultAttr.foregroundIntensity
				winAttr.backgroundIntensity = defaultAttr.backgroundIntensity
				winAttr.underscore = 0
				winAttr.otherAttributes = 0
			case ansiIntensityOn:
				winAttr.foregroundIntensity = foregroundIntensity
			case ansiIntensityOff:
				winAttr.foregroundIntensity = 0
			case ansiUnderlineOn:
				winAttr.underscore = underscore
			case ansiUnderlineOff:
				winAttr.underscore = 0
			case ansiBlinkOn:
				winAttr.backgroundIntensity = backgroundIntensity
			case ansiBlinkOff:
				winAttr.backgroundIntensity = 0
			default:
				// unknown code
			}
		case c.drawType == foreground:
			winAttr.foregroundColor = c.code
		case c.drawType == background:
			winAttr.backgroundColor = c.code
		}
	}
	winTextAttribute := convertWinAttr(winAttr)
	setConsoleTextAttribute(uintptr(syscall.Stdout), winTextAttribute)
}

func parseEscapeSequence(command byte, param []byte) {
	switch command {
	case sgrCode:
		changeColor(param)
	}
}

func isParameterChar(b byte) bool {
	return ('0' <= b && b <= '9') || b == separatorChar
}

func (cw *ansiColorWriter) Write(p []byte) (int, error) {
	r, nw, nc, first, last := 0, 0, 0, 0, 0
	var err error
	for i, ch := range p {
		switch cw.state {
		case outsideCsiCode:
			if ch == firstCsiChar {
				nc++
				cw.state = firstCsiCode
			}
		case firstCsiCode:
			switch ch {
			case firstCsiChar:
				nc++
				break
			case secondeCsiChar:
				nc++
				cw.state = secondeCsiCode
				last = i - 1
			default:
				cw.state = outsideCsiCode
			}
		case secondeCsiCode:
			nc++
			if isParameterChar(ch) {
				cw.paramBuf.WriteByte(ch)
			} else {
				nw, err = cw.w.Write(p[first:last])
				r += nw
				if err != nil {
					return r, err
				}
				first = i + 1
				param := cw.paramBuf.Bytes()
				cw.paramBuf.Reset()
				parseEscapeSequence(ch, param)
				cw.state = outsideCsiCode
			}
		default:
			cw.state = outsideCsiCode
		}
	}

	if cw.state == outsideCsiCode {
		nw, err = cw.w.Write(p[first:len(p)])
	}

	return r + nw + nc, err
}
