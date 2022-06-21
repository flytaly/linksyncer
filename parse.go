package imagesync

import (
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	mdImage    = `!\[.*?\]\(\s?(.+?)\s?(?:".+?")?\)`                                         // ![alternate text](imgpath "title")
	mdImageRef = `\[.*?\]:\s?(\S+)`                                                          // [image_id]: imgpath "title"
	htmlImage  = "<img" + "(?:.|\n)+?" + `src\s?=\s?(?:"(.+?)"|(\S*))` + "(?:.|\n)+?" + "/>" // <img .. src="imgpath" ... />
)

var mdRegexp = regexp.MustCompile(mdImage + "|" + mdImageRef + "|" + htmlImage)
var htmlRegexp = regexp.MustCompile(htmlImage)
var imageExtensions = regexp.MustCompile("(?i)(?:" + ImgExtensions + ")$")

type ImageInfo struct {
	absPath  string
	original string
}

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

func GetImagesFromFile(fileSystem fs.FS, filePath string, root string) ([]ImageInfo, error) {
	file, err := fs.ReadFile(fileSystem, filePath)

	if err != nil {
		return []ImageInfo{}, err
	}

	var imgPaths []string
	result := []ImageInfo{}

	switch strings.ToLower(filepath.Ext(filePath)) {
	case ".md":
		imgPaths = GetImgsFromMD(string(file))
	case ".html":
		imgPaths = GetImgsFromHTML(string(file))
	}

	imgPaths = filterImages(imgPaths)

	for _, p := range imgPaths {
		if !filepath.IsAbs(p) {
			dir := filepath.Dir(filePath)
			info := ImageInfo{original: p, absPath: filepath.Join(root, dir, p)}
			result = append(result, info)
		} else {
			result = append(result, ImageInfo{original: p, absPath: p})
		}
	}

	return result, nil
}
