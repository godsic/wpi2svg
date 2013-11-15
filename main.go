// Copyright (c) 2013, Mykola Dvornik <mykola.dvornik@gmail.com>
// All rights reserved.

// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are met:
//     * Redistributions of source code must retain the above copyright
//       notice, this list of conditions and the following disclaimer.
//     * Redistributions in binary form must reproduce the above copyright
//       notice, this list of conditions and the following disclaimer in the
//       documentation and/or other materials provided with the distribution.
//     * Neither the name of the Mykola Dvornik nor the
//       names of its contributors may be used to endorse or promote products
//       derived from this software without specific prior written permission.

// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL <COPYRIGHT HOLDER> BE LIABLE FOR ANY
// DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/ajstarks/svgo"
	"io"
	"log"
	"math"
	"os"
	"path"
	"strconv"
)

const BUFLEN = 1024 * 1024 * 10
const HEADERLEN = 2059
const MAX_P_TO_WIDTH = 6.0
const MAX_P_TO_COLOUR = 254.0
const MAX_P_LEVELS = 1024.0

const (
	BLK_STROKE       uint8 = 241
	BLK_PEN_XY       uint8 = 97
	BLK_PEN_PRESSURE uint8 = 100
	BLK_PEN_TILT     uint8 = 101
	BLK_UNKNOWN_1    uint8 = 197
	BLK_UNKNOWN_2    uint8 = 194
	BLK_UNKNOWN_3    uint8 = 199
	ID_LAYER         uint8 = 128
	ID_STROKE_BEGIN  uint8 = 1
	ID_STROKE_END    uint8 = 0
)

var layersCount int = 0
var f0, f1 *os.File

type Stroke struct {
	X, Y, P, TiltX, TiltY []int
	SegmentsCount         uint64
}

func (s *Stroke) AddCoords(x, y int16) {
	s.X = append(s.X, int(x))
	s.Y = append(s.Y, int(y))
	s.SegmentsCount++
}

func (s *Stroke) AddPressure(p int16) {
	s.P = append(s.P, int(p))
}

func (s *Stroke) AddTilt(tx, ty byte) {
	s.TiltX = append(s.TiltX, int(tx))
	s.TiltY = append(s.TiltY, int(ty))
}

type Layer struct {
	Name         string
	Strokes      []Stroke
	StrokesCount uint64
}

func (l *Layer) AddStroke() {
	l.Strokes = append(l.Strokes, Stroke{})
	l.StrokesCount++
}

type Canvas struct {
	Layers      []Layer
	LayersCount uint64
}

func (c *Canvas) AddLayer() {
	c.Layers = append(c.Layers, Layer{})
	c.LayersCount++
}

func ReadLayers(r *bufio.Reader) (Canvas, error) {
	e := error(nil)
	var p, x, y int16
	var id byte
	head := make([]byte, 2)
	data := make([]byte, 4)

	/*WPI is rather inconsistent format*/
	/*It might not contain any 'layer' IDs*/
	/*It has no 'end-of-the-layer' marker*/
	/*In mixes up Big Endiness with Little Endiness*/

	c := Canvas{}
	c.AddLayer()
	l := &c.Layers[c.LayersCount-1]
	l.Name = "l" + strconv.FormatUint(c.LayersCount, 10)
	for {
		_, e = r.Read(head)
		if e == io.EOF {
			break
		}
		switch head[0] {

		case BLK_STROKE:
			id, e = r.ReadByte()
			switch id {
			case ID_LAYER:
				c.AddLayer()
				l = &c.Layers[c.LayersCount-1]
			case ID_STROKE_BEGIN:
				l.AddStroke()
			case ID_STROKE_END:
				/*Do sanity check*/
			}

		case BLK_PEN_XY:
			_, e = r.Read(data)

			x = (int16(data[0]) << 8) | int16(data[1])
			y = (int16(data[2]) << 8) | int16(data[3])

			x = (x+5)/8 + 1414
			y = (2*y + 5) / 8

			l.Strokes[l.StrokesCount-1].AddCoords(x, y)

		case BLK_PEN_PRESSURE:
			_, e = r.Read(data)
			p = (int16(data[2]) << 8) | int16(data[3])
			l.Strokes[l.StrokesCount-1].AddPressure(p)

		case BLK_PEN_TILT:
			_, e = r.Read(data)
			l.Strokes[l.StrokesCount-1].AddTilt(data[0], data[1])

		case BLK_UNKNOWN_1, BLK_UNKNOWN_2, BLK_UNKNOWN_3:
			tmp := make([]byte, head[1]-2)
			_, e = r.Read(tmp)
		}
	}
	layersCount++
	return c, e
}

func PressureToColourFunc(p float64) float64 {
	return (1.0 - math.Sqrt(p)) * MAX_P_TO_COLOUR
}

func PressureToWidthFunc(p float64) float64 {
	return p * MAX_P_TO_WIDTH
}

func AddLayersToSVG(s *svg.SVG, c Canvas) error {
	for _, l := range c.Layers {
		s.Gid(l.Name)
		for i, stroke := range l.Strokes {
			strokeID := "s" + strconv.Itoa(i)
			s.Gid(strokeID)
			for i := uint64(0); i < stroke.SegmentsCount-1; i++ {
				p := 0.5 * float64(stroke.P[i]+stroke.P[i+1]) / MAX_P_LEVELS
				width := PressureToWidthFunc(p)
				colour := PressureToColourFunc(p)
				icolour := int(colour)
				swidth := fmt.Sprintf("%.2f", width)
				scolour := s.RGB(icolour, icolour, icolour)[5:]
				s.Line(stroke.X[i], stroke.Y[i], stroke.X[i+1], stroke.Y[i+1], "stroke-linejoin:round;stroke-linecap:round;fill:none;stroke:"+scolour+";stroke-width:"+swidth)
			}
			s.Gend()
		}
		s.Gend()
	}
	return nil
}

func main() {
	var e error

	flag.Parse()

	wpiName := flag.Arg(0)
	if len(wpiName) == 0 {
		os.Exit(1)
	}

	f0, e = os.Open(wpiName)
	defer func() {
		if e != nil {
			log.Print(e)
		}
		f0.Close()
	}()
	f0.Seek(HEADERLEN, 0)

	svgName := wpiName[0:len(wpiName)-len(path.Ext(wpiName))] + ".svg"
	f1, e = os.OpenFile(svgName, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0750)
	defer func() {
		if e != nil {
			log.Print(e)
		}
		f1.Close()
	}()

	wpiReader := bufio.NewReaderSize(f0, BUFLEN)

	canvas := svg.New(f1)
	canvas.Start(2828, 4000) // WTF? NO DPI, NO PHYSICAL DIMENSONS?

	c, _ := ReadLayers(wpiReader)
	AddLayersToSVG(canvas, c)

	canvas.End()
	f1.Sync()
}
