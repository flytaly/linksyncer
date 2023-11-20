package imagesync

import (
	"bytes"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	mdImage    = `!\[.*?\]\(\s?(.+?)\s?(?:".+?")?\)`                                                 // ![alternate text](imgpath "title")
	mdImageRef = `\[.*?\]:\s?(\S+)`                                                                  // [image_id]: imgpath "title"
	htmlImage  = "<img" + "(?:.|\n)+?" + `src\s?=\s?(?:"(.+?)"|'(.+?)'|(\S*))` + "(?:.|\n)+?" + "/>" // <img .. src="imgpath" ... />
)

var mdRegexp = regexp.MustCompile(mdImage + "|" + mdImageRef + "|" + htmlImage)
var htmlRegexp = regexp.MustCompile(htmlImage)
var imageExtensions = regexp.MustCompile("(?i)(?:" + ImgExtensions + ")$")

type LinkInfo struct {
	rootPath string
	path     string
	fullLink string
}

type MovedLink struct {
	to   string
	link LinkInfo
}

// return slice of non-empty capturing groups
func extractSubmatches(groups [][]string) [][]string {
	var result = [][]string{}

	for _, v := range groups {
		matches := []string{v[0]}
		for _, group := range v[1:] {
			if group != "" {
				matches = append(matches, group)
			}
		}
		result = append(result, matches)
	}

	return result
}

func GetImgsFromMD(content string) [][]string {
	return extractSubmatches(mdRegexp.FindAllStringSubmatch(content, -1))
}

func GetImgsFromHTML(content string) [][]string {
	return extractSubmatches(htmlRegexp.FindAllStringSubmatch(content, -1))
}

func filterImages(paths [][]string) [][]string {
	var result = [][]string{}

	for _, v := range paths {
		if strings.Contains(v[1], ":") { // probably an URL
			continue
		}
		if imageExtensions.MatchString(v[1]) {
			result = append(result, v)
		}
	}

	return result
}

func decodePath(path string) string {
	decoded, err := url.PathUnescape(path)
	if err != nil {
		decoded = path
	}
	return decoded
}

// Extracts images from a file's content. filePath argument should be absolute.
func GetImagesFromFile(filePath string, content string) []LinkInfo {
	var links [][]string
	result := []LinkInfo{}

	switch strings.ToLower(filepath.Ext(filePath)) {
	case ".md":
		links = GetImgsFromMD(content)
	case ".html":
		links = GetImgsFromHTML(content)
	default:
		return result
	}

	links = filterImages(links)

	for _, l := range links {
		link, path := l[0], l[1]
		decoded := decodePath(path)

		if filepath.IsAbs(path) {
			result = append(result, LinkInfo{fullLink: link, path: path, rootPath: decoded})
			continue
		}
		dir := filepath.Dir(filePath)
		// save as path with slash for consistency on Windows
		info := LinkInfo{fullLink: link, path: path, rootPath: filepath.ToSlash(filepath.Join(dir, decoded))}
		result = append(result, info)
	}

	return result
}

func ReplaceImageLinks(fPath string, fileContent []byte, imgs []MovedLink) []byte {
	result := fileContent

	for _, img := range imgs {
		targpath := ""
		if !filepath.IsAbs(img.link.path) {
			targpath, _ = filepath.Rel(filepath.Dir(fPath), img.to)
		}
		if targpath == "" {
			targpath = img.to
		}
		// Replace path in the link and then replace link in the file
		newLink := strings.Replace(img.link.fullLink, img.link.path, targpath, 1)
		result = bytes.ReplaceAll(result, []byte(img.link.fullLink), []byte(newLink))
	}

	return result
}
