// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package iconvg

import (
	"errors"
)

// TODO: encode colors; opcodes for setting CREGs and NREGs.

var (
	errDrawingOpsUsedInStylingMode = errors.New("iconvg: drawing ops used in styling mode")
	errInvalidSelectorAdjustment   = errors.New("iconvg: invalid selector adjustment")
	errStylingOpsUsedInDrawingMode = errors.New("iconvg: styling ops used in drawing mode")
)

// TODO: delete the NewEncoder function, and just make the zero value usable.

// NewEncoder returns a new Encoder for the given Metadata.
func NewEncoder(m Metadata) *Encoder {
	e := &Encoder{
		buf: make(buffer, 0, 1024),
	}
	e.Reset(m)
	return e
}

// Encoder is an IconVG encoder.
type Encoder struct {
	buf      buffer
	altBuf   buffer
	metadata Metadata
	err      error

	mode     mode
	drawOp   byte
	drawArgs []float32

	cSel uint32
	nSel uint32
	lod0 float32
	lod1 float32
}

// Bytes returns the encoded form.
func (e *Encoder) Bytes() ([]byte, error) {
	if e.err != nil {
		return nil, e.err
	}
	return []byte(e.buf), nil
}

// Reset resets the Encoder for the given Metadata.
func (e *Encoder) Reset(m Metadata) {
	*e = Encoder{
		buf:      append(e.buf[:0], magic...),
		metadata: m,
		lod1:     positiveInfinity,
	}

	nMetadataChunks := 0
	mcViewBox := m.ViewBox != DefaultViewBox
	if mcViewBox {
		nMetadataChunks++
	}
	mcSuggestedPalette := m.Palette != DefaultPalette
	if mcSuggestedPalette {
		nMetadataChunks++
	}
	e.buf.encodeNatural(uint32(nMetadataChunks))

	if mcViewBox {
		e.altBuf = e.altBuf[:0]
		e.altBuf.encodeNatural(midViewBox)
		e.altBuf.encodeCoordinate(m.ViewBox.Min[0])
		e.altBuf.encodeCoordinate(m.ViewBox.Min[1])
		e.altBuf.encodeCoordinate(m.ViewBox.Max[0])
		e.altBuf.encodeCoordinate(m.ViewBox.Max[1])

		e.buf.encodeNatural(uint32(len(e.altBuf)))
		e.buf = append(e.buf, e.altBuf...)
	}

	if mcSuggestedPalette {
		panic("TODO: encode mcSuggestedPalette")
	}
}

func (e *Encoder) CSel() uint32              { return e.cSel }
func (e *Encoder) NSel() uint32              { return e.nSel }
func (e *Encoder) LOD() (lod0, lod1 float32) { return e.lod0, e.lod1 }

func (e *Encoder) SetCSel(cSel uint32) {
	if e.err != nil {
		return
	}
	if e.mode != modeStyling {
		e.err = errStylingOpsUsedInDrawingMode
		return
	}
	e.cSel = cSel
	e.buf = append(e.buf, uint8(cSel&0x3f))
}

func (e *Encoder) SetNSel(nSel uint32) {
	if e.err != nil {
		return
	}
	if e.mode != modeStyling {
		e.err = errStylingOpsUsedInDrawingMode
		return
	}
	e.nSel = nSel
	e.buf = append(e.buf, uint8((nSel&0x3f)|0x40))
}

func (e *Encoder) SetLOD(lod0, lod1 float32) {
	if e.err != nil {
		return
	}
	if e.mode != modeStyling {
		e.err = errStylingOpsUsedInDrawingMode
		return
	}
	e.lod0 = lod0
	e.lod1 = lod1
	e.buf = append(e.buf, 0xc7)
	e.buf.encodeReal(lod0)
	e.buf.encodeReal(lod1)
}

func (e *Encoder) StartPath(adj int, x, y float32) {
	if e.err != nil {
		return
	}
	if e.mode != modeStyling {
		e.err = errStylingOpsUsedInDrawingMode
		return
	}
	if adj < 0 || 6 < adj {
		e.err = errInvalidSelectorAdjustment
		return
	}
	e.buf = append(e.buf, uint8(0xc0+adj))
	e.buf.encodeCoordinate(x)
	e.buf.encodeCoordinate(y)
	e.mode = modeDrawing
}

func (e *Encoder) AbsHLineTo(x float32)                   { e.draw('H', x, 0, 0, 0, 0, 0) }
func (e *Encoder) RelHLineTo(x float32)                   { e.draw('h', x, 0, 0, 0, 0, 0) }
func (e *Encoder) AbsVLineTo(y float32)                   { e.draw('V', y, 0, 0, 0, 0, 0) }
func (e *Encoder) RelVLineTo(y float32)                   { e.draw('v', y, 0, 0, 0, 0, 0) }
func (e *Encoder) AbsLineTo(x, y float32)                 { e.draw('L', x, y, 0, 0, 0, 0) }
func (e *Encoder) RelLineTo(x, y float32)                 { e.draw('l', x, y, 0, 0, 0, 0) }
func (e *Encoder) AbsSmoothQuadTo(x, y float32)           { e.draw('T', x, y, 0, 0, 0, 0) }
func (e *Encoder) RelSmoothQuadTo(x, y float32)           { e.draw('t', x, y, 0, 0, 0, 0) }
func (e *Encoder) AbsQuadTo(x1, y1, x, y float32)         { e.draw('Q', x1, y1, x, y, 0, 0) }
func (e *Encoder) RelQuadTo(x1, y1, x, y float32)         { e.draw('q', x1, y1, x, y, 0, 0) }
func (e *Encoder) AbsSmoothCubeTo(x2, y2, x, y float32)   { e.draw('S', x2, y2, x, y, 0, 0) }
func (e *Encoder) RelSmoothCubeTo(x2, y2, x, y float32)   { e.draw('s', x2, y2, x, y, 0, 0) }
func (e *Encoder) AbsCubeTo(x1, y1, x2, y2, x, y float32) { e.draw('C', x1, y1, x2, y2, x, y) }
func (e *Encoder) RelCubeTo(x1, y1, x2, y2, x, y float32) { e.draw('c', x1, y1, x2, y2, x, y) }
func (e *Encoder) ClosePathEndPath()                      { e.draw('Z', 0, 0, 0, 0, 0, 0) }
func (e *Encoder) ClosePathAbsMoveTo(x, y float32)        { e.draw('Y', x, y, 0, 0, 0, 0) }
func (e *Encoder) ClosePathRelMoveTo(x, y float32)        { e.draw('y', x, y, 0, 0, 0, 0) }

func (e *Encoder) AbsArcTo(rx, ry, xAxisRotation float32, largeArc, sweep bool, x, y float32) {
	e.arcTo('A', rx, ry, xAxisRotation, largeArc, sweep, x, y)
}

func (e *Encoder) RelArcTo(rx, ry, xAxisRotation float32, largeArc, sweep bool, x, y float32) {
	e.arcTo('a', rx, ry, xAxisRotation, largeArc, sweep, x, y)
}

func (e *Encoder) arcTo(drawOp byte, rx, ry, xAxisRotation float32, largeArc, sweep bool, x, y float32) {
	flags := uint32(0)
	if largeArc {
		flags |= 0x01
	}
	if sweep {
		flags |= 0x02
	}
	e.draw(drawOp, rx, ry, xAxisRotation, float32(flags), x, y)
}

func (e *Encoder) draw(drawOp byte, arg0, arg1, arg2, arg3, arg4, arg5 float32) {
	if e.err != nil {
		return
	}
	if e.mode != modeDrawing {
		e.err = errDrawingOpsUsedInStylingMode
		return
	}
	if e.drawOp != drawOp {
		e.flushDrawOps()
	}
	e.drawOp = drawOp
	switch drawOps[drawOp].nArgs {
	case 0:
		// No-op.
	case 1:
		e.drawArgs = append(e.drawArgs, arg0)
	case 2:
		e.drawArgs = append(e.drawArgs, arg0, arg1)
	case 4:
		e.drawArgs = append(e.drawArgs, arg0, arg1, arg2, arg3)
	case 6:
		e.drawArgs = append(e.drawArgs, arg0, arg1, arg2, arg3, arg4, arg5)
	default:
		panic("unreachable")
	}

	switch drawOp {
	case 'Z':
		e.mode = modeStyling
		fallthrough
	case 'Y', 'y':
		e.flushDrawOps()
	}
}

func (e *Encoder) flushDrawOps() {
	if e.drawOp == 0x00 {
		return
	}

	if op := drawOps[e.drawOp]; op.nArgs == 0 {
		e.buf = append(e.buf, op.opcodeBase)
	} else {
		n := len(e.drawArgs) / int(op.nArgs)
		for i := 0; n > 0; {
			m := n
			if m > int(op.maxRepCount) {
				m = int(op.maxRepCount)
			}
			e.buf = append(e.buf, op.opcodeBase+uint8(m)-1)

			switch e.drawOp {
			default:
				for j := m * int(op.nArgs); j > 0; j-- {
					e.buf.encodeCoordinate(e.drawArgs[i])
					i++
				}
			case 'A', 'a':
				for j := m; j > 0; j-- {
					e.buf.encodeCoordinate(e.drawArgs[i+0])
					e.buf.encodeCoordinate(e.drawArgs[i+1])
					e.buf.encodeZeroToOne(e.drawArgs[i+2])
					e.buf.encodeNatural(uint32(e.drawArgs[i+3]))
					e.buf.encodeCoordinate(e.drawArgs[i+4])
					e.buf.encodeCoordinate(e.drawArgs[i+5])
					i += 6
				}
			}

			n -= m
		}
	}

	e.drawOp = 0x00
	e.drawArgs = e.drawArgs[:0]
}

var drawOps = [256]struct {
	opcodeBase  byte
	maxRepCount uint8
	nArgs       uint8
}{
	'L': {0x00, 32, 2},
	'l': {0x20, 32, 2},
	'T': {0x40, 16, 2},
	't': {0x50, 16, 2},
	'Q': {0x60, 16, 4},
	'q': {0x70, 16, 4},
	'S': {0x80, 16, 4},
	's': {0x90, 16, 4},
	'C': {0xa0, 16, 6},
	'c': {0xb0, 16, 6},
	'A': {0xc0, 16, 6},
	'a': {0xd0, 16, 6},

	// Z means close path and then end path.
	'Z': {0xe1, 1, 0},
	// Y/y means close path and then open a new path (with a MoveTo/moveTo).
	'Y': {0xe2, 1, 2},
	'y': {0xe3, 1, 2},

	'H': {0xe6, 1, 1},
	'h': {0xe7, 1, 1},
	'V': {0xe8, 1, 1},
	'v': {0xe9, 1, 1},
}