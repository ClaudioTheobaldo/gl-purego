package gl

import (
	"fmt"
	"image"
	"image/draw"
	_ "image/jpeg" // register JPEG decoder
	_ "image/png"  // register PNG decoder
	"os"
)

// LoadTexture opens an image file (PNG or JPEG), uploads it to the GPU as a
// 2-D RGBA texture, and returns the texture ID with texture unit 0 still
// bound. Mipmaps are not generated; filtering is set to LINEAR.
//
// The file path is relative to the working directory of the binary.
//
// A current OpenGL context must exist before calling LoadTexture.
func LoadTexture(path string) (uint32, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, fmt.Errorf("gl.LoadTexture: %w", err)
	}
	defer f.Close()

	src, _, err := image.Decode(f)
	if err != nil {
		return 0, fmt.Errorf("gl.LoadTexture %q: %w", path, err)
	}

	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	// Convert to RGBA (handles palette, YCbCr, Gray, etc.)
	rgba := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(rgba, rgba.Bounds(), src, bounds.Min, draw.Src)

	// OpenGL expects the first row to be the bottom row of the image.
	// Most image formats (PNG, JPEG) store the top row first, so we flip.
	flipped := make([]byte, len(rgba.Pix))
	stride := w * 4
	for y := 0; y < h; y++ {
		src := rgba.Pix[(h-1-y)*stride : (h-y)*stride]
		dst := flipped[y*stride : (y+1)*stride]
		copy(dst, src)
	}

	var tex uint32
	GenTextures(1, &tex)
	BindTexture(TEXTURE_2D, tex)

	TexParameteri(TEXTURE_2D, TEXTURE_WRAP_S, int32(CLAMP_TO_EDGE))
	TexParameteri(TEXTURE_2D, TEXTURE_WRAP_T, int32(CLAMP_TO_EDGE))
	TexParameteri(TEXTURE_2D, TEXTURE_MIN_FILTER, int32(LINEAR))
	TexParameteri(TEXTURE_2D, TEXTURE_MAG_FILTER, int32(LINEAR))

	TexImage2D(
		TEXTURE_2D, 0, int32(RGBA),
		int32(w), int32(h), 0,
		RGBA, UNSIGNED_BYTE,
		Ptr(flipped),
	)

	return tex, nil
}
