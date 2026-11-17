package service

import (
	"bytes"
	"crypto/rand"
	"errors"
	"image"
	"image/jpeg"
	"io"
	"os"
	"path/filepath"

	"github.com/disintegration/imaging"
	_ "golang.org/x/image/webp"
)

const MaxImageBytes = 10 << 20
const maxImagePixels = 40000000

type Media struct {
	directory string
}

type SavedImage struct {
	Key          string
	Type         string
	Bytes        int64
	ThumbnailKey string
}

type Image struct {
	File *os.File
	Type string
}

// keeping media outside public static routes
func NewMedia(directory string) *Media {
	return &Media{directory: directory}
}

// validating content before writing an original and its thumbnail
func (m *Media) Save(reader io.Reader) (SavedImage, error) {
	data, err := io.ReadAll(io.LimitReader(reader, MaxImageBytes+1))
	if err != nil {
		return SavedImage{}, err
	}

	if len(data) > MaxImageBytes {
		return SavedImage{}, ErrImageTooLarge
	}

	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return SavedImage{}, invalid("image", "Не удалось прочитать изображение")
	}

	types := map[string]string{"jpeg": "image/jpeg", "png": "image/png", "webp": "image/webp"}
	contentType, supported := types[format]
	if !supported {
		return SavedImage{}, invalid("image", "Поддерживаются JPEG, PNG и WebP")
	}

	if !validImageSize(config) {
		return SavedImage{}, invalid("image", "Изображение должно содержать не больше 40 миллионов пикселей")
	}

	decoded, err := imaging.Decode(bytes.NewReader(data), imaging.AutoOrientation(true))
	if err != nil {
		return SavedImage{}, invalid("image", "Изображение повреждено")
	}

	thumbnail := imaging.Fit(decoded, 400, 400, imaging.Lanczos)
	var encoded bytes.Buffer
	if err := jpeg.Encode(&encoded, thumbnail, &jpeg.Options{Quality: 85}); err != nil {
		return SavedImage{}, err
	}

	if err := os.MkdirAll(m.directory, 0o700); err != nil {
		return SavedImage{}, err
	}

	key := rand.Text() + "." + format
	thumbnailKey := rand.Text() + ".jpg"
	if err := m.write(key, data); err != nil {
		return SavedImage{}, err
	}

	if err := m.write(thumbnailKey, encoded.Bytes()); err != nil {
		return SavedImage{}, err
	}

	return SavedImage{Key: key, Type: contentType, Bytes: int64(len(data)), ThumbnailKey: thumbnailKey}, nil
}

// never replacing an existing file
func (m *Media) write(key string, data []byte) error {
	file, err := os.OpenFile(filepath.Join(m.directory, key), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}

	_, writeError := file.Write(data)
	if writeError == nil {
		writeError = file.Sync()
	}

	return errors.Join(writeError, file.Close())
}

// checking the pixel limit without multiplying untrusted dimensions
func validImageSize(config image.Config) bool {
	if config.Width <= 0 || config.Height <= 0 {
		return false
	}

	return int64(config.Width) <= maxImagePixels/int64(config.Height)
}

// blocking directory traversal and symlinks outside media
func (m *Media) Open(key, contentType string) (Image, error) {
	if !filepath.IsLocal(key) {
		return Image{}, ErrNotFound
	}

	file, err := os.OpenInRoot(m.directory, key)
	if err != nil {
		return Image{}, ErrNotFound
	}

	info, err := file.Stat()
	if err != nil {
		file.Close()
		return Image{}, err
	}

	if !info.Mode().IsRegular() {
		file.Close()
		return Image{}, ErrNotFound
	}

	return Image{File: file, Type: contentType}, nil
}
