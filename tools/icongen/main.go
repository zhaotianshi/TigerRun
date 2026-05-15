package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"

	xdraw "golang.org/x/image/draw"
)

const canvas = 2048

type point struct {
	x float64
	y float64
}

func main() {
	img := image.NewRGBA(image.Rect(0, 0, canvas, canvas))
	drawBackground(img)
	drawMotion(img)
	drawTiger(img)
	drawNetworkAccent(img)

	must(os.MkdirAll(filepath.Join("build", "windows"), 0o755))
	must(os.MkdirAll(filepath.Join("chrome-extension", "icons"), 0o755))
	must(os.MkdirAll(filepath.Join("frontend", "src", "assets", "images"), 0o755))

	writePNG(img, filepath.Join("build", "appicon.png"), 1024)
	writePNG(img, filepath.Join("frontend", "src", "assets", "images", "laohukuaipao-icon.png"), 256)
	writePNG(img, filepath.Join("chrome-extension", "icons", "icon128.png"), 128)
	writePNG(img, filepath.Join("chrome-extension", "icons", "icon48.png"), 48)
	writePNG(img, filepath.Join("chrome-extension", "icons", "icon32.png"), 32)
	writePNG(img, filepath.Join("chrome-extension", "icons", "icon16.png"), 16)
	writeICO(img, filepath.Join("build", "windows", "icon.ico"), []int{256, 128, 64, 48, 32, 16})
}

func drawBackground(img *image.RGBA) {
	radius := 360.0
	cx, cy := float64(canvas)/2, float64(canvas)/2
	for y := 0; y < canvas; y++ {
		for x := 0; x < canvas; x++ {
			if !insideRoundedRect(float64(x), float64(y), radius) {
				continue
			}
			t := (float64(x) + float64(y)) / (canvas * 2)
			base := lerpColor(rgba(10, 39, 58, 255), rgba(10, 123, 102, 255), t)
			dist := math.Hypot(float64(x)-cx*0.72, float64(y)-cy*0.42) / 1150
			glow := clamp01(1 - dist)
			base = lerpColor(base, rgba(34, 181, 146, 255), glow*0.22)
			img.SetRGBA(x, y, base)
		}
	}
}

func drawMotion(img *image.RGBA) {
	fillPolygon(img, []point{{260, 760}, {920, 610}, {850, 750}, {250, 900}}, rgba(246, 184, 72, 95))
	fillPolygon(img, []point{{210, 980}, {830, 900}, {810, 1030}, {210, 1130}}, rgba(255, 255, 255, 70))
	fillPolygon(img, []point{{300, 1230}, {980, 1190}, {930, 1325}, {290, 1390}}, rgba(246, 184, 72, 105))
	strokeLine(img, 300, 1500, 780, 1450, 22, rgba(133, 232, 210, 115))
	strokeLine(img, 390, 590, 720, 535, 18, rgba(133, 232, 210, 95))
}

func drawTiger(img *image.RGBA) {
	orange := rgba(245, 132, 33, 255)
	gold := rgba(255, 190, 74, 255)
	dark := rgba(31, 36, 47, 245)
	cream := rgba(255, 244, 214, 255)

	fillEllipse(img, 900, 1080, 520, 310, orange)
	fillEllipse(img, 1060, 1000, 470, 290, gold)
	fillEllipse(img, 1305, 805, 350, 305, orange)
	fillPolygon(img, []point{{1110, 575}, {1190, 315}, {1345, 625}}, orange)
	fillPolygon(img, []point{{1420, 610}, {1605, 420}, {1540, 735}}, gold)
	fillEllipse(img, 1560, 860, 280, 180, cream)
	fillEllipse(img, 1440, 940, 170, 135, cream)

	strokeQuadratic(img, 555, 1015, 330, 840, 490, 655, 66, orange)
	strokeQuadratic(img, 520, 1050, 360, 960, 280, 1060, 28, dark)

	strokeQuadratic(img, 790, 1330, 630, 1510, 430, 1390, 84, orange)
	strokeQuadratic(img, 1120, 1315, 1340, 1490, 1530, 1360, 82, gold)
	strokeLine(img, 735, 1335, 625, 1545, 72, dark)
	strokeLine(img, 1190, 1320, 1435, 1515, 68, dark)

	fillPolygon(img, []point{{910, 795}, {1075, 745}, {1010, 890}}, dark)
	fillPolygon(img, []point{{915, 1015}, {1115, 945}, {1040, 1095}}, dark)
	fillPolygon(img, []point{{870, 1185}, {1090, 1140}, {1000, 1285}}, dark)
	fillPolygon(img, []point{{1245, 600}, {1320, 690}, {1175, 690}}, dark)
	fillPolygon(img, []point{{1390, 645}, {1500, 720}, {1330, 760}}, dark)
	fillPolygon(img, []point{{1275, 900}, {1415, 860}, {1355, 1010}}, dark)

	fillEllipse(img, 1422, 773, 34, 44, rgba(8, 21, 31, 255))
	fillEllipse(img, 1434, 760, 9, 11, rgba(255, 255, 255, 230))
	fillEllipse(img, 1665, 850, 38, 28, rgba(8, 21, 31, 255))
	strokeLine(img, 1588, 950, 1692, 1015, 15, dark)

	strokeLine(img, 705, 830, 600, 715, 18, rgba(255, 244, 214, 170))
	strokeLine(img, 760, 890, 610, 845, 15, rgba(255, 244, 214, 140))
}

func drawNetworkAccent(img *image.RGBA) {
	cyan := rgba(116, 237, 218, 230)
	strokeLine(img, 1420, 1240, 1590, 1170, 14, cyan)
	strokeLine(img, 1590, 1170, 1710, 1275, 14, cyan)
	strokeLine(img, 1420, 1240, 1515, 1395, 14, cyan)
	fillEllipse(img, 1420, 1240, 38, 38, rgba(221, 255, 248, 255))
	fillEllipse(img, 1590, 1170, 32, 32, cyan)
	fillEllipse(img, 1710, 1275, 32, 32, cyan)
	fillEllipse(img, 1515, 1395, 32, 32, cyan)
}

func writePNG(src image.Image, path string, size int) {
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), xdraw.Over, nil)
	file, err := os.Create(path)
	must(err)
	defer file.Close()
	must(png.Encode(file, dst))
}

func writeICO(src image.Image, path string, sizes []int) {
	type entry struct {
		size int
		data []byte
	}
	entries := make([]entry, 0, len(sizes))
	for _, size := range sizes {
		dst := image.NewRGBA(image.Rect(0, 0, size, size))
		xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), xdraw.Over, nil)
		entries = append(entries, entry{size: size, data: encodeIconDIB(dst, size)})
	}

	var out bytes.Buffer
	must(binary.Write(&out, binary.LittleEndian, uint16(0)))
	must(binary.Write(&out, binary.LittleEndian, uint16(1)))
	must(binary.Write(&out, binary.LittleEndian, uint16(len(entries))))

	offset := 6 + len(entries)*16
	for _, item := range entries {
		w := byte(item.size)
		h := byte(item.size)
		if item.size >= 256 {
			w, h = 0, 0
		}
		out.WriteByte(w)
		out.WriteByte(h)
		out.WriteByte(0)
		out.WriteByte(0)
		must(binary.Write(&out, binary.LittleEndian, uint16(1)))
		must(binary.Write(&out, binary.LittleEndian, uint16(32)))
		must(binary.Write(&out, binary.LittleEndian, uint32(len(item.data))))
		must(binary.Write(&out, binary.LittleEndian, uint32(offset)))
		offset += len(item.data)
	}
	for _, item := range entries {
		out.Write(item.data)
	}
	must(os.WriteFile(path, out.Bytes(), 0o644))
}

func encodeIconDIB(img *image.RGBA, size int) []byte {
	var out bytes.Buffer
	imageBytes := uint32(size * size * 4)

	must(binary.Write(&out, binary.LittleEndian, uint32(40)))
	must(binary.Write(&out, binary.LittleEndian, int32(size)))
	must(binary.Write(&out, binary.LittleEndian, int32(size*2)))
	must(binary.Write(&out, binary.LittleEndian, uint16(1)))
	must(binary.Write(&out, binary.LittleEndian, uint16(32)))
	must(binary.Write(&out, binary.LittleEndian, uint32(0)))
	must(binary.Write(&out, binary.LittleEndian, imageBytes))
	must(binary.Write(&out, binary.LittleEndian, int32(0)))
	must(binary.Write(&out, binary.LittleEndian, int32(0)))
	must(binary.Write(&out, binary.LittleEndian, uint32(0)))
	must(binary.Write(&out, binary.LittleEndian, uint32(0)))

	for y := size - 1; y >= 0; y-- {
		for x := 0; x < size; x++ {
			c := img.RGBAAt(x, y)
			out.WriteByte(c.B)
			out.WriteByte(c.G)
			out.WriteByte(c.R)
			out.WriteByte(c.A)
		}
	}

	maskStride := ((size + 31) / 32) * 4
	out.Write(make([]byte, maskStride*size))
	return out.Bytes()
}

func insideRoundedRect(x, y, r float64) bool {
	max := float64(canvas)
	if x >= r && x <= max-r || y >= r && y <= max-r {
		return true
	}
	cx := r
	if x > max-r {
		cx = max - r
	}
	cy := r
	if y > max-r {
		cy = max - r
	}
	return math.Hypot(x-cx, y-cy) <= r
}

func fillEllipse(img *image.RGBA, cx, cy, rx, ry float64, c color.RGBA) {
	minX := maxInt(0, int(cx-rx))
	maxX := minInt(canvas-1, int(cx+rx))
	minY := maxInt(0, int(cy-ry))
	maxY := minInt(canvas-1, int(cy+ry))
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			dx := (float64(x) - cx) / rx
			dy := (float64(y) - cy) / ry
			if dx*dx+dy*dy <= 1 {
				blend(img, x, y, c)
			}
		}
	}
}

func fillPolygon(img *image.RGBA, pts []point, c color.RGBA) {
	minX, minY := canvas, canvas
	maxX, maxY := 0, 0
	for _, p := range pts {
		minX = minInt(minX, int(p.x))
		minY = minInt(minY, int(p.y))
		maxX = maxInt(maxX, int(p.x))
		maxY = maxInt(maxY, int(p.y))
	}
	minX, minY = maxInt(0, minX), maxInt(0, minY)
	maxX, maxY = minInt(canvas-1, maxX), minInt(canvas-1, maxY)
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			if pointInPolygon(float64(x)+0.5, float64(y)+0.5, pts) {
				blend(img, x, y, c)
			}
		}
	}
}

func strokeLine(img *image.RGBA, x1, y1, x2, y2, width float64, c color.RGBA) {
	steps := int(math.Hypot(x2-x1, y2-y1)/6) + 1
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		fillEllipse(img, x1+(x2-x1)*t, y1+(y2-y1)*t, width/2, width/2, c)
	}
}

func strokeQuadratic(img *image.RGBA, x1, y1, cx, cy, x2, y2, width float64, c color.RGBA) {
	steps := 150
	prevX, prevY := x1, y1
	for i := 1; i <= steps; i++ {
		t := float64(i) / float64(steps)
		x := (1-t)*(1-t)*x1 + 2*(1-t)*t*cx + t*t*x2
		y := (1-t)*(1-t)*y1 + 2*(1-t)*t*cy + t*t*y2
		strokeLine(img, prevX, prevY, x, y, width, c)
		prevX, prevY = x, y
	}
}

func pointInPolygon(x, y float64, pts []point) bool {
	inside := false
	j := len(pts) - 1
	for i := range pts {
		xi, yi := pts[i].x, pts[i].y
		xj, yj := pts[j].x, pts[j].y
		if (yi > y) != (yj > y) && x < (xj-xi)*(y-yi)/(yj-yi)+xi {
			inside = !inside
		}
		j = i
	}
	return inside
}

func blend(img *image.RGBA, x, y int, src color.RGBA) {
	if src.A == 255 {
		img.SetRGBA(x, y, src)
		return
	}
	dst := img.RGBAAt(x, y)
	a := float64(src.A) / 255
	inv := 1 - a
	img.SetRGBA(x, y, color.RGBA{
		R: uint8(float64(src.R)*a + float64(dst.R)*inv),
		G: uint8(float64(src.G)*a + float64(dst.G)*inv),
		B: uint8(float64(src.B)*a + float64(dst.B)*inv),
		A: uint8(float64(src.A) + float64(dst.A)*inv),
	})
}

func rgba(r, g, b, a uint8) color.RGBA {
	return color.RGBA{R: r, G: g, B: b, A: a}
}

func lerpColor(a, b color.RGBA, t float64) color.RGBA {
	t = clamp01(t)
	return color.RGBA{
		R: uint8(float64(a.R)*(1-t) + float64(b.R)*t),
		G: uint8(float64(a.G)*(1-t) + float64(b.G)*t),
		B: uint8(float64(a.B)*(1-t) + float64(b.B)*t),
		A: uint8(float64(a.A)*(1-t) + float64(b.A)*t),
	}
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
