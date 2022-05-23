package imagesync

import (
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	mdImage    = `!\[.+?\]\(\s?(.+?)\s?(?:".+?")?\)`                                         // ![alternate text](imgpath "title")
	mdImageRef = `\[.+?\]:\s?(\S+)`                                                          // [image_id]: imgpath "title"
	htmlImage  = "<img" + "(?:.|\n)+?" + `src\s?=\s?(?:"(.+?)"|(\S*))` + "(?:.|\n)+?" + "/>" // <img .. src="imgpath" ... />
)

var mdRegexp = regexp.MustCompile(mdImage + "|" + mdImageRef + "|" + htmlImage)
var htmlRegexp = regexp.MustCompile(htmlImage)
var imageExtensions = regexp.MustCompile("(?i)(?:.png|.jpg|.jpeg|.webp|.svg|.tiff|.tff|.gif)$")

// return flat slice of non-empty capturing groups
func extractSubmatches(groups [][]string) []string {
	var result = []string{}

	for _, v := range groups {
		for _, group := range v[1:] {
			if group != "" {
				result = append(result, group)
			}
		}
	}
	return result
}

func GetImgsFromMD(content string) []string {
	return extractSubmatches(mdRegexp.FindAllStringSubmatch(content, -1))
}

func GetImgsFromHTML(content string) []string {
	return extractSubmatches(htmlRegexp.FindAllStringSubmatch(content, -1))
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

	var imgPaths []string
	result := []string{}

	switch strings.ToLower(filepath.Ext(path)) {
	case ".md":
		imgPaths = GetImgsFromMD(string(file))
	case ".html":
		imgPaths = GetImgsFromHTML(string(file))
	}

	imgPaths = filterImages(imgPaths)

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
