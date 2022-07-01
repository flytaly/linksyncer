package imagesync

import (
	"bytes"
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
	absPath      string
	originalLink string
}

type RenamedImage struct {
	prevPath string
	newPath  string
	link     string
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

// Extracts images from a file's content. filePath argument should be absolute.
func GetImagesFromFile(filePath string, content string) []ImageInfo {
	var imgPaths []string
	result := []ImageInfo{}

	switch strings.ToLower(filepath.Ext(filePath)) {
	case ".md":
		imgPaths = GetImgsFromMD(content)
	case ".html":
		imgPaths = GetImgsFromHTML(content)
	default:
		return result
	}

	imgPaths = filterImages(imgPaths)

	for _, p := range imgPaths {
		if !filepath.IsAbs(p) {
			dir := filepath.Dir(filePath)
			info := ImageInfo{originalLink: p, absPath: filepath.Join(dir, p)}
			result = append(result, info)
		} else {
			result = append(result, ImageInfo{originalLink: p, absPath: p})
		}
	}

	return result
}

func ReplaceImageLinks(filePath string, fileContent []byte, imgs []RenamedImage) []byte {
	result := fileContent

	for _, img := range imgs {
		targpath := ""
		if !filepath.IsAbs(img.link) {
			targpath, _ = filepath.Rel(filepath.Dir(filePath), img.newPath)
		}
		if targpath == "" {
			targpath = img.newPath
		}

		result = bytes.ReplaceAll(result, []byte(img.link), []byte(targpath))
	}

	return result
}
