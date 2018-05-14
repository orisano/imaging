package imaging

import (
	"image"
	"math"
)

func gaussianBlurKernel(x, sigma float64) float64 {
	return math.Exp(-(x*x)/(2*sigma*sigma)) / (sigma * math.Sqrt(2*math.Pi))
}

// Blur produces a blurred version of the image using a Gaussian function.
// Sigma parameter must be positive and indicates how much the image will be blurred.
//
// Usage example:
//
//	dstImage := imaging.Blur(srcImage, 3.5)
//
func Blur(img image.Image, sigma float64) *image.NRGBA {
	if sigma <= 0 {
		return Clone(img)
	}

	radius := int(math.Ceil(sigma * 3.0))
	kernel := make([]float64, radius+1)

	for i := 0; i <= radius; i++ {
		kernel[i] = gaussianBlurKernel(float64(i), sigma)
	}

	return blurVertical(blurHorizontal(img, kernel), kernel)
}

func blurHorizontal(img image.Image, kernel []float64) *image.NRGBA {
	src := newScanner(img)
	dst := image.NewNRGBA(image.Rect(0, 0, src.w, src.h))
	radius := len(kernel) - 1

	parallel(0, src.h, func(ys <-chan int) {
		scanLine := make([]uint8, src.w*4)
		scanLineF := make([]float64, len(scanLine))
		for y := range ys {
			src.scan(0, y, src.w, y+1, scanLine)
			for i, v := range scanLine {
				scanLineF[i] = float64(v)
			}
			for x, idx := 0, 0; x < src.w; x, idx = x+1, idx+4 {
				min := x - radius
				if min < 0 {
					min = 0
				}
				max := x + radius
				if max > src.w-1 {
					max = src.w - 1
				}

				var r, g, b, a, wsum float64
				for ix := min; ix <= max; ix++ {
					i := ix * 4
					weight := kernel[absint(x-ix)]
					wsum += weight
					wa := scanLineF[i+3] * weight
					r += scanLineF[i+0] * wa
					g += scanLineF[i+1] * wa
					b += scanLineF[i+2] * wa
					a += wa
				}
				if a != 0 {
					r /= a
					g /= a
					b /= a
				}

				scanLine[idx] = clamp(r)
				scanLine[idx+1] = clamp(g)
				scanLine[idx+2] = clamp(b)
				scanLine[idx+3] = clamp(a / wsum)
			}
			copy(dst.Pix[y*dst.Stride:], scanLine)
		}
	})

	return dst
}

func blurVertical(img image.Image, kernel []float64) *image.NRGBA {
	src := newScanner(img)
	dst := image.NewNRGBA(image.Rect(0, 0, src.w, src.h))
	radius := len(kernel) - 1

	parallel(0, src.w, func(xs <-chan int) {
		scanLine := make([]uint8, src.h*4)
		scanLineF := make([]float64, len(scanLine))
		for x := range xs {
			src.scan(x, 0, x+1, src.h, scanLine)
			for i, v := range scanLine {
				scanLineF[i] = float64(v)
			}
			for y, idx := 0, 0; y < src.h; y, idx = y+1, idx+4 {
				min := y - radius
				if min < 0 {
					min = 0
				}
				max := y + radius
				if max > src.h-1 {
					max = src.h - 1
				}

				var r, g, b, a, wsum float64
				for iy := min; iy <= max; iy++ {
					i := iy * 4
					weight := kernel[absint(y-iy)]
					wsum += weight
					v := scanLineF[i : i+4]
					wa := v[3] * weight
					r += v[0] * wa
					g += v[1] * wa
					b += v[2] * wa
					a += wa
				}
				if a != 0 {
					r /= a
					g /= a
					b /= a
				}

				j := y*dst.Stride + x*4
				dst.Pix[j+0] = clamp(r)
				dst.Pix[j+1] = clamp(g)
				dst.Pix[j+2] = clamp(b)
				dst.Pix[j+3] = clamp(a / wsum)
			}
		}
	})

	return dst
}

// Sharpen produces a sharpened version of the image.
// Sigma parameter must be positive and indicates how much the image will be sharpened.
//
// Usage example:
//
//	dstImage := imaging.Sharpen(srcImage, 3.5)
//
func Sharpen(img image.Image, sigma float64) *image.NRGBA {
	if sigma <= 0 {
		return Clone(img)
	}

	src := newScanner(img)
	dst := image.NewNRGBA(image.Rect(0, 0, src.w, src.h))
	blurred := Blur(img, sigma)

	parallel(0, src.h, func(ys <-chan int) {
		scanLine := make([]uint8, src.w*4)
		for y := range ys {
			src.scan(0, y, src.w, y+1, scanLine)
			j := y * dst.Stride
			for i := 0; i < src.w*4; i++ {
				val := int(scanLine[i])<<1 - int(blurred.Pix[j])
				if val < 0 {
					val = 0
				} else if val > 0xff {
					val = 0xff
				}
				dst.Pix[j] = uint8(val)
				j++
			}
		}
	})

	return dst
}
