package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"image"
	"image/color"
	"io/ioutil"
	"os"

	"golang.org/x/image/bmp"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

const (
	jumpSTA uint8 = 0x1
	jumpEND uint8 = 0x2
)

func main() {
	filesPath := flag.String("path", "", "the path of files.")
	filesName := flag.String("name", "", "the name of files.")
	flag.Parse()
	if *filesPath == "" || *filesName == "" {
		fmt.Println("Wrong flag input.")
	} else {
		f, err := os.Open(*filesPath + *filesName + ".idx")
		check(err)
		f2, err := os.Open(*filesPath + *filesName + ".dat")
		check(err)
		/* Load time. */
		/* Part of IDX file READING */
		bytesOfSampleSum := make([]byte, 4) // SampleSum
		_, err = f.Read(bytesOfSampleSum)
		check(err)
		sampleSum := binary.LittleEndian.Uint32(bytesOfSampleSum)
		// fmt.Println("Sample Sum:", sampleSum)
		for i := 0; i < int(sampleSum); i++ {
			bytesOfSampleState := make([]byte, 1) // SampleState
			_, err = f.Read(bytesOfSampleState)
			check(err)
			sampleState := uint8(bytesOfSampleState[0])
			_ = sampleState
			bytesOfOswIndex := make([]byte, 4) // OswIndex
			_, err = f.Read(bytesOfOswIndex)
			check(err)
			oswIndex := binary.LittleEndian.Uint32(bytesOfOswIndex)
			_ = oswIndex
			bytesOfIdxIndex := make([]byte, 4) // IdxIndex
			_, err = f.Read(bytesOfIdxIndex)
			check(err)
			idxIndex := binary.LittleEndian.Uint32(bytesOfIdxIndex)
			_ = idxIndex
			bytesOfDatOffset := make([]byte, 4) // DatOffset
			_, err = f.Read(bytesOfDatOffset)
			check(err)
			datOffset := binary.LittleEndian.Uint32(bytesOfDatOffset)
			/* Part of DAT file READING */
			_, err = f2.Seek(int64(datOffset), os.SEEK_SET)
			check(err)
			bytesOfWordLength := make([]byte, 1) // WordLength
			_, err = f2.Read(bytesOfWordLength)
			check(err)
			wordLength := uint8(bytesOfWordLength[0])
			// fmt.Println("WordLength", wordLength)
			bytesOfWordCode := make([]byte, wordLength) // WordCode
			_, err = f2.Read(bytesOfWordCode)
			check(err)
			d, err := ioutil.ReadAll(
				transform.NewReader(bytes.NewReader(bytesOfWordCode),
					simplifiedchinese.GBK.NewDecoder()))
			check(err)
			wordCode := string(d)
			// fmt.Println("WordCode", wordCode)
			bytesOfPointNum := make([]byte, 2) // PointNum
			_, err = f2.Read(bytesOfPointNum)
			check(err)
			pointNum := binary.LittleEndian.Uint16(bytesOfPointNum)
			_ = pointNum
			// fmt.Println("PointNum", pointNum)
			bytesOfLineNum := make([]byte, 2) // LineNum
			_, err = f2.Read(bytesOfLineNum)
			check(err)
			lineNum := binary.LittleEndian.Uint16(bytesOfLineNum)
			// fmt.Println("LineNum", lineNum)
			bytesOfGetTimePointNum := make([]byte, 2) // GetTimePointNum
			_, err = f2.Read(bytesOfGetTimePointNum)
			check(err)
			getTimePointNum := binary.LittleEndian.Uint16(bytesOfGetTimePointNum)
			// fmt.Println("GetTimePointNum", getTimePointNum)
			bytesOfGetTimePointIndex := make([]byte, getTimePointNum*2) // GetTimePointIndex
			_, err = f2.Read(bytesOfGetTimePointIndex)
			check(err)
			// fmt.Println("GetTimePointIndex", bytesOfGetTimePointIndex)
			bytesOfElapsedTime := make([]byte, getTimePointNum*4) // ElapsedTime
			_, err = f2.Read(bytesOfElapsedTime)
			check(err)
			// fmt.Println("ElapsedTime", bytesOfElapsedTime)
			/*  */
			lines := make([]couchLine, lineNum)
			side := couchSide{nil, nil, nil, nil}
			for j := 0; j < int(lineNum); j++ {
				bytesOfStrokePointNum := make([]byte, 2) // StrokePointNum
				_, err = f2.Read(bytesOfStrokePointNum)
				check(err)
				strokePointNum := binary.LittleEndian.Uint16(bytesOfStrokePointNum)
				pointsOfLine := make([]couchPoint, strokePointNum)
				// fmt.Println("StrokePointNum", strokePointNum)

				for k := 0; k < int(strokePointNum); k++ {
					bytesOfPointX := make([]byte, 2) // PointX
					_, err = f2.Read(bytesOfPointX)
					check(err)
					pointX := binary.LittleEndian.Uint16(bytesOfPointX)
					bytesOfPointY := make([]byte, 2) // PointY
					_, err = f2.Read(bytesOfPointY)
					check(err)
					pointY := binary.LittleEndian.Uint16(bytesOfPointY)
					point := couchPoint{pointX, pointY}
					pointsOfLine[k] = point
					update(&side, &point)
				}
				lines[j] = couchLine{pointsOfLine}
			}
			img := couchImg{side, lines}
			wd, err := os.Getwd() // get working path
			check(err)
			wordCodeInt := int(binary.LittleEndian.Uint16(bytesOfWordCode))
			outputPath := fmt.Sprintf("%s/output/%d.bmp", wd, wordCodeInt)
			img.write(outputPath)
			fmt.Println(wordCode, wordCodeInt)
			// break // debug
		}
	}
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

type couchImg struct {
	side  couchSide
	lines []couchLine
}

func (img couchImg) write(path string) {
	fout, err := os.Create(path) // Created after judgment to prevent invalid file generation
	check(err)
	width := int((*img.side.maxX).x-(*img.side.minX).x) + 1
	height := int((*img.side.maxY).y-(*img.side.minY).y) + 1
	offsetX := int((*img.side.minX).x)
	offsetY := int((*img.side.minY).y)
	rgba := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ { // full of white color in background
		for x := 0; x < width; x++ {
			rgba.Set(x, y, color.White)
		}
	}
	for _, eachLine := range img.lines {
		if len(eachLine.points) == 1 {
			// handle one point line
			break
		}
		for z := 1; z < len(eachLine.points); z++ {
			prePointOfLine := eachLine.points[z-1]
			eachPointOfLine := eachLine.points[z]
			dy := int(eachPointOfLine.y) - int(prePointOfLine.y)
			dx := int(eachPointOfLine.x) - int(prePointOfLine.x)
			if abs(dy) > abs(dx) { // vertical
				if dx > 0 {
					drawlineP(prePointOfLine, eachPointOfLine, jumpEND,
						func(x, y int) {
							rgba.Set(x-offsetX, y-offsetY, color.Black)
						})
				} else {
					drawlineP(eachPointOfLine, prePointOfLine, jumpSTA,
						func(x, y int) {
							rgba.Set(x-offsetX, y-offsetY, color.Black)
						})
				}
			} else { // horizontal
				if dy < 0 {
					drawlineP(eachPointOfLine, prePointOfLine, jumpSTA,
						func(x, y int) {
							rgba.Set(x-offsetX, y-offsetY, color.Black)
						})
				} else {
					drawlineP(prePointOfLine, eachPointOfLine, jumpEND,
						func(x, y int) {
							rgba.Set(x-offsetX, y-offsetY, color.Black)
						})
				}
			}
		}
	}
	err = bmp.Encode(fout, rgba)
	check(err)
}

type couchLine struct {
	points []couchPoint
}

type couchPoint struct {
	x, y uint16
}

type couchSide struct {
	minX, maxX, minY, maxY *couchPoint
}

func update(side *couchSide, p *couchPoint) {
	if side.minX == nil || (*side.minX).x > (*p).x {
		side.minX = p
	}
	if side.maxX == nil || (*side.maxX).x < (*p).x {
		side.maxX = p
	}
	if side.minY == nil || (*side.minY).y > (*p).y {
		side.minY = p
	}
	if side.maxY == nil || (*side.maxY).y < (*p).y {
		side.maxY = p
	}
}

func drawlineP(p0, p1 couchPoint, jump uint8, brush func(x, y int)) {
	switch jump {
	case jumpSTA:
		drawlineNoStart(int(p0.x), int(p0.y), int(p1.x), int(p1.y), brush)
	case jumpEND:
		drawlineNoEnd(int(p0.x), int(p0.y), int(p1.x), int(p1.y), brush)
	default:
		return
	}
}

func drawlineNoStart(x0, y0, x1, y1 int, brush func(x, y int)) { // no start
	dx := abs(x1 - x0)
	dy := abs(y1 - y0)
	sx, sy := 1, 1
	if x0 >= x1 {
		sx = -1
	}
	if y0 >= y1 {
		sy = -1
	}
	err := dx - dy

	for {
		e2 := err * 2
		if e2 < dx {
			err += dx
			y0 += sy
		}
		if e2 > -dy {
			err -= dy
			x0 += sx
		}
		brush(x0, y0)
		if x0 == x1 && y0 == y1 {
			return
		}
	}
}

func drawlineNoEnd(x0, y0, x1, y1 int, brush func(x, y int)) { // no end
	dx := abs(x1 - x0)
	dy := abs(y1 - y0)
	sx, sy := 1, 1
	if x0 >= x1 {
		sx = -1
	}
	if y0 >= y1 {
		sy = -1
	}
	err := dx - dy

	for {
		if x0 == x1 && y0 == y1 {
			return
		}
		brush(x0, y0)
		e2 := err * 2
		if e2 < dx {
			err += dx
			y0 += sy
		}
		if e2 > -dy {
			err -= dy
			x0 += sx
		}
	}
}

func abs(x int) int {
	if x >= 0 {
		return x
	}
	return -x
}
