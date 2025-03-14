package robohash

import (
	"embed"
	"fmt"
	"hash/fnv"
	"image"
	"image/png"
	"math/rand/v2"
	"path"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/image/draw"
)

//go:embed all:parts/*
var parts embed.FS

type RoboHash struct {
	rd                   *rand.Rand
	roboSet, bgSet       string
	sets, bgSets, colors []string
}

func New(hash []byte, roboSet, bgSet string) (*RoboHash, error) {
	h := fnv.New64a()
	if _, err := h.Write(hash); err != nil {
		return nil, err
	}
	r := &RoboHash{
		rd:      rand.New(rand.NewPCG(h.Sum64(), (h.Sum64()>>1)|1)),
		roboSet: roboSet,
		bgSet:   bgSet,
	}

	var err error
	r.sets, err = r.listDirs("parts/sets")
	if err != nil {
		return nil, err
	}
	r.bgSets, err = r.listDirs("parts/backgrounds")
	if err != nil {
		return nil, err
	}
	r.colors, err = r.listDirs("parts/sets/set1") // only set1 has colors
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (r *RoboHash) Assemble() (image.Image, error) {
	if r.roboSet == "any" {
		r.roboSet = r.sets[r.rd.IntN(len(r.sets))]
	} else {
		found := false
		for _, s := range r.sets {
			if s == r.roboSet {
				found = true
				break
			}
		}
		if !found {
			r.roboSet = r.sets[0]
		}
	}

	// only set1 has colors
	if r.roboSet == "set1" {
		color := r.colors[r.rd.IntN(len(r.colors))]
		r.roboSet = "set1/" + color
	}

	var bgFound bool
	if r.bgSet != "" {
		for _, b := range r.bgSets {
			if b == r.bgSet {
				bgFound = true
				break
			}
		}
	}

	if !bgFound && r.bgSet == "any" {
		r.bgSet = r.bgSets[r.rd.IntN(len(r.bgSets))]
		bgFound = true
	}

	robotSetPath := path.Join("parts", "sets", r.roboSet)
	roboParts, err := r.getListOfFiles(robotSetPath)
	if err != nil {
		return nil, err
	}

	var background string
	if bgFound {
		bgDirPath := path.Join("parts", "backgrounds", r.bgSet)
		bgFiles, err := r.getFilesFromDir(bgDirPath)
		if err != nil {
			return nil, err
		}

		if len(bgFiles) > 0 {
			background = bgFiles[r.rd.IntN(len(bgFiles))]
		}
	}

	if len(roboParts) == 0 {
		return nil, fmt.Errorf("no robot parts found in %s", robotSetPath)
	}

	roboImg := image.NewRGBA(image.Rect(0, 0, 1024, 1024))
	for _, part := range roboParts {
		partFile, err := parts.Open(part)
		if err != nil {
			return nil, err
		}

		partImg, err := png.Decode(partFile)
		partFile.Close()
		if err != nil {
			return nil, err
		}

		resizedPart := resizeImage(partImg, 1024, 1024)
		draw.Draw(roboImg, roboImg.Bounds(), resizedPart, image.Point{}, draw.Over)
	}

	// Handle background if specified
	if background != "" {
		bgFile, err := parts.Open(background)
		if err != nil {
			return nil, err
		}
		defer bgFile.Close()

		bgImg, err := png.Decode(bgFile)
		if err != nil {
			return nil, err
		}

		resizedBg := resizeImage(bgImg, 1024, 1024)
		finalImg := image.NewRGBA(image.Rect(0, 0, 1024, 1024))
		draw.Draw(finalImg, finalImg.Bounds(), resizedBg, image.Point{}, draw.Src)
		draw.Draw(finalImg, finalImg.Bounds(), roboImg, image.Point{}, draw.Over)
		roboImg = finalImg
	}

	return roboImg, nil
}

func (r *RoboHash) listDirs(dirPath string) ([]string, error) {
	var dirs []string

	entries, err := parts.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			dirs = append(dirs, entry.Name())
		}
	}

	sort.Strings(dirs)
	return dirs, nil
}

func (r *RoboHash) getFilesFromDir(dirPath string) ([]string, error) {
	var files []string

	entries, err := parts.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			files = append(files, path.Join(dirPath, entry.Name()))
		}
	}

	sort.Strings(files)
	return files, nil
}

func (r *RoboHash) getAllSubdirectories(basePath string) ([]string, error) {
	var directories []string
	var walkFn func(string) error

	walkFn = func(dir string) error {
		entries, err := parts.ReadDir(dir)
		if err != nil {
			return err
		}

		for _, entry := range entries {
			if entry.IsDir() {
				sub := path.Join(dir, entry.Name())
				if !strings.HasPrefix(path.Base(sub), ".") {
					directories = append(directories, sub)
					if err := walkFn(sub); err != nil {
						return err
					}
				}
			}
		}
		return nil
	}

	if err := walkFn(basePath); err != nil {
		return nil, err
	}

	sort.Strings(directories)
	return directories, nil
}

func (r *RoboHash) getRandomFile(dirPath string) (string, error) {
	files, err := r.getFilesFromDir(dirPath)
	if err != nil {
		return "", err
	}

	if len(files) == 0 {
		return "", fmt.Errorf("no files found in %s", dirPath)
	}

	index := r.rd.IntN(len(files))
	return files[index], nil
}

func (r *RoboHash) getListOfFiles(basePath string) ([]string, error) {
	directories, err := r.getAllSubdirectories(basePath)
	if err != nil {
		return nil, err
	}

	base, err := r.getFilesFromDir(basePath)
	if err == nil && len(base) > 0 {
		directories = append([]string{basePath}, directories...)
	}

	var chosen []string
	for _, dir := range directories {
		file, err := r.getRandomFile(dir)
		if err != nil {
			return nil, err
		}
		chosen = append(chosen, file)
	}

	sort.Slice(chosen, func(i, j int) bool {
		iParts := strings.Split(path.Base(chosen[i]), "#")
		jParts := strings.Split(path.Base(chosen[j]), "#")

		if len(iParts) > 1 && len(jParts) > 1 {
			iNumStr := strings.Split(iParts[1], ".")[0]
			jNumStr := strings.Split(jParts[1], ".")[0]
			iNum, iErr := strconv.Atoi(iNumStr)
			jNum, jErr := strconv.Atoi(jNumStr)
			if iErr == nil && jErr == nil {
				return iNum < jNum
			}
		}
		return chosen[i] < chosen[j]
	})

	return chosen, nil
}

func resizeImage(img image.Image, width, height int) image.Image {
	bounds := img.Bounds()
	if bounds.Dx() == width && bounds.Dy() == height {
		return img
	}

	resized := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.ApproxBiLinear.Scale(resized, resized.Bounds(), img, bounds, draw.Over, nil)

	return resized
}
