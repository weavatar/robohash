package robohash

import (
	"image"
	"testing"
)

func TestNewRoboHash(t *testing.T) {
	t.Run("CreateWithEmptyString", func(t *testing.T) {
		r, err := New([]byte(""), "", "")
		if err != nil {
			t.Errorf("Expected no error with empty string, got: %v", err)
		}
		if r == nil {
			t.Error("Expected RoboHash instance, got nil")
		}
	})

	t.Run("CreateWithDifferentStrings", func(t *testing.T) {
		r1, err1 := New([]byte("test1"), "", "")
		r2, err2 := New([]byte("test2"), "", "")

		if err1 != nil || err2 != nil {
			t.Errorf("Expected no errors, got: %v, %v", err1, err2)
		}

		// Different strings should create different random generators
		if r1.rd == r2.rd {
			t.Error("Expected different random generators for different strings")
		}
	})

	t.Run("CreateWithSameString", func(t *testing.T) {
		r1, _ := New([]byte("same"), "", "")
		r2, _ := New([]byte("same"), "", "")

		// Same string should produce deterministic results
		if len(r1.sets) != len(r2.sets) || len(r1.bgSets) != len(r2.bgSets) {
			t.Error("Expected same sets for same input string")
		}
	})
}

func TestAssemble(t *testing.T) {
	t.Run("DefaultAssembly", func(t *testing.T) {
		r, err := New([]byte("test"), "", "")
		if err != nil {
			t.Fatalf("Failed to create RoboHash: %v", err)
		}

		img, err := r.Assemble()
		if err != nil {
			t.Errorf("Failed to assemble robot: %v", err)
		}

		if img == nil {
			t.Error("Expected image, got nil")
		}

		bounds := img.Bounds()
		if bounds.Dx() != 1024 || bounds.Dy() != 1024 {
			t.Errorf("Expected 1024x1024 image, got %dx%d", bounds.Dx(), bounds.Dy())
		}
	})

	t.Run("AssembleWithAnySet", func(t *testing.T) {
		r, _ := New([]byte("test"), "any", "")
		img, err := r.Assemble()

		if err != nil {
			t.Errorf("Failed to assemble with 'any' set: %v", err)
		}
		if img == nil {
			t.Error("Expected image, got nil")
		}
	})

	t.Run("AssembleWithAnyBackground", func(t *testing.T) {
		r, _ := New([]byte("test"), "set1", "any")
		img, err := r.Assemble()

		if err != nil {
			t.Errorf("Failed to assemble with 'any' background: %v", err)
		}
		if img == nil {
			t.Error("Expected image, got nil")
		}
	})

	t.Run("AssembleWithInvalidSet", func(t *testing.T) {
		r, _ := New([]byte("test"), "nonexistent", "")
		img, err := r.Assemble()

		if err != nil {
			t.Errorf("Expected fallback to set1, got error: %v", err)
		}
		if img == nil {
			t.Error("Expected image despite invalid set, got nil")
		}
	})

	t.Run("AssembleWithInvalidBackground", func(t *testing.T) {
		r, _ := New([]byte("test"), "set1", "nonexistent")
		img, err := r.Assemble()

		if err != nil {
			t.Errorf("Expected no error with invalid background, got: %v", err)
		}
		if img == nil {
			t.Error("Expected image despite invalid background, got nil")
		}
	})

	t.Run("DeterministicImageGeneration", func(t *testing.T) {
		r1, _ := New([]byte("deterministic"), "set1", "")
		r2, _ := New([]byte("deterministic"), "set1", "")

		img1, _ := r1.Assemble()
		img2, _ := r2.Assemble()

		if !compareImages(img1, img2) {
			t.Error("Expected identical images for same input string")
		}
	})
}

// Helper function to compare images
func compareImages(img1, img2 image.Image) bool {
	bounds1 := img1.Bounds()
	bounds2 := img2.Bounds()

	if bounds1 != bounds2 {
		return false
	}

	for y := bounds1.Min.Y; y < bounds1.Max.Y; y++ {
		for x := bounds1.Min.X; x < bounds1.Max.X; x++ {
			if img1.At(x, y) != img2.At(x, y) {
				return false
			}
		}
	}
	return true
}
