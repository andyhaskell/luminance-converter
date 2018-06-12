package main

import (
	"image"
	"image/color"
	"image/jpeg"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "luminance-converter"
	app.Usage = "Convert an image's pixels by their luminance"
	app.Action = run

	app.Flags = []cli.Flag {
		cli.StringFlag{
			Name: "in",
			Usage: "filename of image to convert",
		},
		cli.StringFlag{
			Name: "out",
			Usage: "filename to output converted image",
		},
		cli.StringFlag{
			Name: "t",
			Usage: "comma-separated thresholds for each percent tier of luminance",
			Value: "0,50",
		},
		cli.StringFlag{
			Name: "c",
			Usage: "comma-separated hex values for the color each luminance tier is converted to",
			Value: "000000,FFFFFF",
		},
	}
	app.Run(os.Args)
}

func luminance(c color.Color) float64 {
	r, g, b, _ := c.RGBA()
	rf := float64(r)/65536 * 100
	gf := float64(g)/65536 * 100
	bf := float64(b)/65536 * 100
	return rf*.2126+gf*.7152+bf*.0722
}

func hexToColor(hex uint32) color.Color {
	r := uint8(hex&0xFF0000>>16)
	g := uint8(hex&0x00FF00>>8)
	b := uint8(hex&0x0000FF)

	return color.RGBA{R: r, G: g, B: b}
}

func convertLuminance(l float64, thresholds []float64, converted []color.Color) color.Color {
	for i := 0; i < len(thresholds)-1; i++ {
		if l > thresholds[i] && l <= thresholds[i+1] || l == 0 {
			return converted[i]
		}
	}
	log.Fatalf("luminance is over all thresholds, l=%v, thresholds=%v", l, thresholds)
	return color.RGBA{}
}

func convertImage(img image.Image, thresholds []float64, colors []color.Color) image.Image {
	b := img.Bounds()
	converted := image.NewRGBA(b)

	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			l := luminance(img.At(x, y))
			converted.Set(x, y, convertLuminance(l, thresholds, colors))
		}
	}
	return converted
}

func run(c *cli.Context) error{
	thresholdStrings := strings.Split(c.String("t"), ",")
	colorStrings := strings.Split(c.String("c"), ",")

	var thresholds []float64
	for _, t := range thresholdStrings {
		th, err := strconv.ParseFloat(t, 64)
		if err != nil || th < 0.0 || th > 100.0 {
			log.Fatalf(`invalid luminance threshold "%s"`, t)
		}
		thresholds = append(thresholds, th)
	}

	var convertedColors []color.Color
	for _, c := range colorStrings {
		h, err := strconv.ParseUint(c, 16, 32)
		if err != nil || h < 0 || h > 0xFFFFFF {
			log.Fatalf(`invalid converted color "%s"`, c)
		}

		convertedColors = append(convertedColors, hexToColor(uint32(h)))
	}

	thresholds = append(thresholds, 100.0)
	convertedColors = append(convertedColors, convertedColors[len(convertedColors)-1])

	f, err := os.Open(c.String("in"))
	if err != nil {
		log.Fatalf(`error reading file "%s": %v`, c.String("in"), err)
	}

	img, _, err := image.Decode(f)
	if err != nil {
		log.Fatalf(`error converting file to Image: %v`, err)
	}

	converted := convertImage(img, thresholds, convertedColors)
	out, err := os.Create(c.String("out"))
	if err != nil {
		log.Fatalf(`error creating file "%s": %v`, c.String("out"), err)
	}

	jpeg.Encode(out, converted, &jpeg.Options{Quality: 100})
	return nil
}