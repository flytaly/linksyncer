package syncer

import (
	"bytes"
	"net/url"
	"path/filepath"
	"strings"

	mdParser "github.com/flytaly/linksyncer/pkg/parser"
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

type ContentLink struct {
	content string
	dest    string
}

func GetLinksFromMD(content string) (links []ContentLink, images []ContentLink) {
	p := mdParser.New()
	p.Parse([]byte(content))
	links_, imgs_ := p.LinksAndImages()
	for _, link := range links_ {
		links = append(links, ContentLink{string(link.GetContent()), string(link.Destination)})
	}
	for _, img := range imgs_ {
		images = append(images, ContentLink{string(img.GetContent()), string(img.Destination)})
	}
	return links, images
}

func filterLinks(paths []ContentLink) []ContentLink {
	var result = []ContentLink{}

	for _, v := range paths {
		if strings.Contains(v.dest, ":") { // probably an URL
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

func processLinks(filePath string, links []ContentLink) []LinkInfo {
	links = filterLinks(links)
	result := []LinkInfo{}

	for _, l := range links {
		link, path := l.content, l.dest
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

// Extracts links from a file's content. filePath argument should be absolute.
func GetLinksFromFile(filePath string, content string) (links []LinkInfo, images []LinkInfo) {
	var imgList, linkList []ContentLink

	switch strings.ToLower(filepath.Ext(filePath)) {
	case ".md":
		linkList, imgList = GetLinksFromMD(content)
	}

	links = processLinks(filePath, linkList)
	images = processLinks(filePath, imgList)
	return links, images
}

// ReplaceLinks updates links in the file
func ReplaceLinks(fPath string, fileContent []byte, moves []MovedLink) []byte {
	result := fileContent

	for _, move := range moves {
		targpath := ""
		if !filepath.IsAbs(move.link.path) {
			targpath, _ = filepath.Rel(filepath.Dir(fPath), move.to)
		}
		if targpath == "" {
			targpath = move.to
		}

		// encode spaces
		targpath = strings.Replace(targpath, " ", "%20", -1)

		// Replace path in the link and then replace link in the file
		newLink := strings.Replace(move.link.fullLink, move.link.path, targpath, 1)
		result = bytes.ReplaceAll(result, []byte(move.link.fullLink), []byte(newLink))
	}

	return result
}
