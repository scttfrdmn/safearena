package main

import (
	"fmt"
	"time"

	"github.com/scttfrdmn/safearena"
)

// Image filter example: Arena-allocated buffers for image processing pipeline
// Demonstrates working with large temporary buffers

// Image represents a simple image (grayscale for simplicity)
type Image struct {
	Width  int
	Height int
	Pixels []byte
}

// FilterPipeline holds temporary buffers for multi-pass filtering
type FilterPipeline struct {
	TempBuffer1 safearena.Slice[byte]
	TempBuffer2 safearena.Slice[byte]
	Histogram   safearena.Slice[int]
}

// applyFiltersWithArena applies a series of filters using arena for temp buffers
func applyFiltersWithArena(img *Image) *Image {
	return safearena.Scoped(func(a *safearena.Arena) *Image {
		size := img.Width * img.Height

		// Allocate pipeline buffers in arena (large allocations)
		pipeline := safearena.Alloc(a, FilterPipeline{
			TempBuffer1: safearena.AllocSlice[byte](a, size),
			TempBuffer2: safearena.AllocSlice[byte](a, size),
			Histogram:   safearena.AllocSlice[int](a, 256),
		})

		p := pipeline.Get()
		buf1 := p.TempBuffer1.Get()
		buf2 := p.TempBuffer2.Get()
		hist := p.Histogram.Get()

		// Pass 1: Blur filter (writes to buf1)
		applyBlur(img.Pixels, buf1, img.Width, img.Height)

		// Pass 2: Sharpen filter (reads buf1, writes to buf2)
		applySharpen(buf1, buf2, img.Width, img.Height)

		// Pass 3: Calculate histogram (for adjustment)
		for _, pixel := range buf2 {
			hist[pixel]++
		}

		// Pass 4: Contrast adjustment (reads buf2, writes to final result)
		result := &Image{
			Width:  img.Width,
			Height: img.Height,
			Pixels: make([]byte, size), // Heap-allocated result
		}
		applyContrast(buf2, result.Pixels, hist)

		// All temporary buffers (potentially MBs) freed here
		return result
	})
}

// applyFiltersWithoutArena uses traditional heap allocations
func applyFiltersWithoutArena(img *Image) *Image {
	size := img.Width * img.Height

	// All temp buffers allocated on heap
	buf1 := make([]byte, size)
	buf2 := make([]byte, size)
	hist := make([]int, 256)

	// Same processing pipeline
	applyBlur(img.Pixels, buf1, img.Width, img.Height)
	applySharpen(buf1, buf2, img.Width, img.Height)

	for _, pixel := range buf2 {
		hist[pixel]++
	}

	result := &Image{
		Width:  img.Width,
		Height: img.Height,
		Pixels: make([]byte, size),
	}
	applyContrast(buf2, result.Pixels, hist)

	return result
}

// Simple blur filter (box blur)
func applyBlur(src, dst []byte, width, height int) {
	for i := range src {
		// Simple averaging with neighbors
		sum := int(src[i])
		count := 1

		if i >= width { // top
			sum += int(src[i-width])
			count++
		}
		if i < len(src)-width { // bottom
			sum += int(src[i+width])
			count++
		}

		dst[i] = byte(sum / count)
	}
}

// Sharpen filter
func applySharpen(src, dst []byte, width, height int) {
	for i := range src {
		// Simple sharpening
		center := int(src[i]) * 2

		if i >= width {
			center -= int(src[i-width])
		}
		if i < len(src)-width {
			center -= int(src[i+width])
		}

		// Clamp
		if center < 0 {
			center = 0
		}
		if center > 255 {
			center = 255
		}

		dst[i] = byte(center)
	}
}

// Contrast adjustment based on histogram
func applyContrast(src, dst []byte, hist []int) {
	// Simple linear contrast stretch
	for i := range src {
		val := int(src[i])
		// Boost contrast slightly
		adjusted := (val-128)*120/100 + 128

		if adjusted < 0 {
			adjusted = 0
		}
		if adjusted > 255 {
			adjusted = 255
		}

		dst[i] = byte(adjusted)
	}
}

// Generate test image
func generateTestImage(width, height int) *Image {
	pixels := make([]byte, width*height)

	// Generate pattern
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Simple gradient pattern
			pixels[y*width+x] = byte((x + y) % 256)
		}
	}

	return &Image{
		Width:  width,
		Height: height,
		Pixels: pixels,
	}
}

func main() {
	fmt.Println("Image Filter Pipeline Example\n")

	// Test with various image sizes
	sizes := []struct {
		width  int
		height int
	}{
		{640, 480},   // VGA
		{1920, 1080}, // Full HD
		{3840, 2160}, // 4K
	}

	for _, size := range sizes {
		img := generateTestImage(size.width, size.height)
		megapixels := float64(size.width*size.height) / 1_000_000

		fmt.Printf("Processing %dx%d image (%.1f MP)...\n", size.width, size.height, megapixels)

		// Benchmark with arena
		iterations := 10
		start := time.Now()
		var result *Image
		for i := 0; i < iterations; i++ {
			result = applyFiltersWithArena(img)
		}
		arenaTime := time.Since(start)

		// Benchmark without arena
		start = time.Now()
		for i := 0; i < iterations; i++ {
			result = applyFiltersWithoutArena(img)
		}
		gcTime := time.Since(start)

		fmt.Printf("  With Arena:    %v (%v per image)\n", arenaTime, arenaTime/time.Duration(iterations))
		fmt.Printf("  Without Arena: %v (%v per image)\n", gcTime, gcTime/time.Duration(iterations))
		fmt.Printf("  Speedup: %.2fx\n", float64(gcTime)/float64(arenaTime))
		fmt.Printf("  Temp memory per image: %.1f MB\n\n", float64(size.width*size.height*2)/1_000_000)

		_ = result // Use result
	}

	fmt.Println("Key Benefits:")
	fmt.Println("- Large temp buffers (MBs) freed immediately")
	fmt.Println("- No GC pressure from intermediate results")
	fmt.Println("- Clear separation: temp buffers vs final image")
	fmt.Println("- Perfect for video processing (one arena per frame)")
}
