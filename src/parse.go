package imagesync

import (
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	mdImage    = `!\[.+?\]\(\s?(.+?)\s?(?:".+?")?\)` // ![alternate text](imgpath "title")
	mdImageRef = `\[.+?\]:\s?(\S+)`                  // [image_id]: imgpath "title"
)

var mdRegexp = regexp.MustCompile(mdImage + "|" + mdImageRef)
var imageExtensions = regexp.MustCompile("(?i)(?:.png|.jpg|.jpeg|.webp|.svg|.tiff|.tff|.gif)$")

func GetImgsFromMD(content string) []string {
	var result = []string{}
	groups := mdRegexp.FindAllStringSubmatch(content, -1)

	for _, v := range groups {
		for _, group := range v[1:] {
			if group != "" {
				result = append(result, group)
			}
		}
	}
	return result
}

func GetImgsFromHTML(content string) []string {
	return []string{}
}

func filterImages(paths []string) []string {
	var result = []string{}

	for _, v := range paths {
		if strings.Contains(v, ":") { // probably an URL
			continue
		}
		if imageExtensions.MatchString(v) {
			result = append(result, v)
		}
	}

	return result
}

func GetImagesFromFile(fileSystem fs.FS, path string) ([]string, error) {
	file, err := fs.ReadFile(fileSystem, path)

	if err != nil {
		return []string{}, err
	}

	switch strings.ToLower(filepath.Ext(path)) {
	case ".md":
		imgPaths := filterImages(GetImgsFromMD(string(file)))
		result := []string{}
		for _, v := range imgPaths {
			if !filepath.IsAbs(v) {
				dir := filepath.Dir(path)
				result = append(result, filepath.Join(dir, v))
			} else {
				result = append(result, v)
			}
		}

		return result, nil
	}

	return []string{}, nil
}
