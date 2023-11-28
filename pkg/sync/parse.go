package imagesync

import (
	"bytes"
	mdParser "imagesync/pkg/parser"
	"net/url"
	"path/filepath"
	"strings"
)

type LinkInfo struct {
	rootPath string
	path     string
	fullLink string
}

type MovedLink struct {
	to   string
	link LinkInfo
}

func GetImgsFromMD(content string) [][]string {
	p := mdParser.New()
	p.Parse([]byte(content))
	_, imgs := p.LinksAndImages()
	res := [][]string{}
	for _, img := range imgs {
		res = append(res, []string{string(img.GetContent()), string(img.Destination)})
	}
	return res
}

func filterImages(paths [][]string) [][]string {
	var result = [][]string{}

	for _, v := range paths {
		if strings.Contains(v[1], ":") { // probably an URL
			continue
		}
		result = append(result, v)
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
