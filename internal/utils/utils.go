package utils

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"math"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/image/draw"
)

func HyprctlSocket() string {
	return filepath.Join(os.Getenv("XDG_RUNTIME_DIR"), "hypr", os.Getenv("HYPRLAND_INSTANCE_SIGNATURE"), ".socket.sock")
}

func HyprQuery(cmd string) ([]byte, error) {
	conn, err := net.Dial("unix", HyprctlSocket())
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	_, err = conn.Write([]byte(cmd))
	if err != nil {
		return nil, err
	}

	return io.ReadAll(conn)
}

type Monitor struct {
	Width   int  `json:"width"`
	Height  int  `json:"height"`
	Focused bool `json:"focused"`
}

func GetCurrentMonitorDimensions() (int, int, error) {
	data, err := HyprQuery("j/monitors")
	if err != nil {
		return 0, 0, err
	}

	var monitors []Monitor
	if err := json.Unmarshal(data, &monitors); err != nil {
		return 0, 0, err
	}

	for _, m := range monitors {
		if m.Focused {
			return m.Width, m.Height, nil
		}
	}

	return 0, 0, fmt.Errorf("no focused monitor found")
}

func GetScreenSize() (width, height int, err error) {
	out, err := exec.Command("xdpyinfo").Output()
	if err != nil {
		return 0, 0, err
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "dimensions:") {
			parts := strings.Fields(line)
			dims := strings.Split(parts[1], "x")
			w, _ := strconv.Atoi(dims[0])
			h, _ := strconv.Atoi(dims[1])
			return w, h, nil
		}
	}

	return 0, 0, fmt.Errorf("cannot detect screen size")
}

func ObjectFitCover(srcPath string, targetW, targetH int, disPath string, name string) (string, error) {
	f, err := os.Open(srcPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	srcImg, format, err := image.Decode(f)
	if err != nil {
		return "", fmt.Errorf("decode image failed: %w", err)
	}

	sw := srcImg.Bounds().Dx()
	sh := srcImg.Bounds().Dy()

	scale := math.Max(
		float64(targetW)/float64(sw),
		float64(targetH)/float64(sh),
	)

	nw := int(float64(sw) * scale)
	nh := int(float64(sh) * scale)

	// scale
	scaled := image.NewRGBA(image.Rect(0, 0, nw, nh))
	draw.CatmullRom.Scale(
		scaled,
		scaled.Bounds(),
		srcImg,
		srcImg.Bounds(),
		draw.Over,
		nil,
	)

	// center crop
	x0 := (nw - targetW) / 2
	y0 := (nh - targetH) / 2

	cropped := image.NewRGBA(image.Rect(0, 0, targetW, targetH))
	draw.Draw(
		cropped,
		cropped.Bounds(),
		scaled,
		image.Point{X: x0, Y: y0},
		draw.Src,
	)

	// output format handling
	format = strings.ToLower(format)

	var (
		ext    string
		encode func(*os.File) error
	)

	switch format {
	case "png":
		ext = ".png"
		encode = func(out *os.File) error {
			return png.Encode(out, cropped)
		}

	case "jpeg", "jpg":
		ext = ".jpg"
		encode = func(out *os.File) error {
			return jpeg.Encode(out, cropped, &jpeg.Options{Quality: 92})
		}

	default:
		return "", fmt.Errorf("unsupported image format: %s", format)
	}

	// output path (cache-safe)
	dstPath := filepath.Join(
		disPath,
		name+ext,
	)

	if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		return "", err
	}

	out, err := os.Create(dstPath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	if err := encode(out); err != nil {
		return "", err
	}

	return dstPath, nil
}

func ApplyBorderRadius(
	srcPath string,
	rTL, rTR, rBR, rBL int,
	disPath string,
	name string,
) (string, error) {
	f, err := os.Open(srcPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	srcImg, format, err := image.Decode(f)
	if err != nil {
		return "", fmt.Errorf("decode image failed: %w", err)
	}

	w := srcImg.Bounds().Dx()
	h := srcImg.Bounds().Dy()

	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	mask := RoundedMask(w, h, rTL, rTR, rBR, rBL)

	draw.DrawMask(
		dst,
		dst.Bounds(),
		srcImg,
		image.Point{},
		mask,
		image.Point{},
		draw.Over,
	)

	format = strings.ToLower(format)

	var (
		ext    string
		encode func(*os.File) error
	)

	// transparent corners => PNG only
	hasRadius := rTL > 0 || rTR > 0 || rBR > 0 || rBL > 0

	switch format {
	case "jpeg", "jpg":
		if hasRadius {
			ext = ".png"
			encode = func(out *os.File) error {
				return png.Encode(out, dst)
			}
		} else {
			ext = ".jpg"
			encode = func(out *os.File) error {
				return jpeg.Encode(out, dst, &jpeg.Options{Quality: 92})
			}
		}

	case "png":
		ext = ".png"
		encode = func(out *os.File) error {
			return png.Encode(out, dst)
		}

	default:
		return "", fmt.Errorf("unsupported image format: %s", format)
	}

	dstPath := filepath.Join(disPath, name+ext)

	if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		return "", err
	}

	out, err := os.Create(dstPath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	if err := encode(out); err != nil {
		return "", err
	}

	return dstPath, nil
}

func RoundedMask(w, h, rTL, rTR, rBR, rBL int) *image.Alpha {
	mask := image.NewAlpha(image.Rect(0, 0, w, h))

	maxR := MinInt(w, h) / 2
	rTL = MinInt(rTL, maxR)
	rTR = MinInt(rTR, maxR)
	rBR = MinInt(rBR, maxR)
	rBL = MinInt(rBL, maxR)

	for y := range h {
		for x := range w {
			alpha := uint8(255)

			switch {
			// top-left
			case x < rTL && y < rTL:
				dx := x - rTL
				dy := y - rTL
				if dx*dx+dy*dy > rTL*rTL {
					alpha = 0
				}

			// top-right
			case x >= w-rTR && y < rTR:
				dx := x - (w - rTR)
				dy := y - rTR
				if dx*dx+dy*dy > rTR*rTR {
					alpha = 0
				}

			// bottom-right
			case x >= w-rBR && y >= h-rBR:
				dx := x - (w - rBR)
				dy := y - (h - rBR)
				if dx*dx+dy*dy > rBR*rBR {
					alpha = 0
				}

			// bottom-left
			case x < rBL && y >= h-rBL:
				dx := x - rBL
				dy := y - (h - rBL)
				if dx*dx+dy*dy > rBL*rBL {
					alpha = 0
				}
			}

			mask.SetAlpha(x, y, color.Alpha{A: alpha})
		}
	}

	return mask
}

func MinInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func buildFilters(inputs []string) [][]interface{} {
	var patterns [][]string
	var mimes []string

	for _, f := range inputs {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}

		// MIME type
		if strings.Contains(f, "/") {
			mimes = append(mimes, f)
			continue
		}

		// extension
		f = strings.TrimPrefix(f, ".")
		patterns = append(patterns, []string{
			"*." + strings.ToLower(f),
		})
	}

	var filters [][]interface{}

	if len(patterns) > 0 {
		filters = append(filters, []interface{}{
			"Files",
			patterns,
		})
	}

	if len(mimes) > 0 {
		filters = append(filters, []interface{}{
			"MIME",
			mimes,
		})
	}

	return filters
}
