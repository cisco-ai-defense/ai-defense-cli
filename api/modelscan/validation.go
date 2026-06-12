package modelscan

import (
	"fmt"
	"path"
	"strings"
)

const fileTypeNotSupported = "file type not supported"

// supportedExtensions flattens the register-scan response into a set of
// lowercase extensions (e.g. ".bin", ".safetensors").
func supportedExtensions(types SupportedFileTypes) map[string]struct{} {
	exts := make(map[string]struct{})
	for _, category := range types.Categories {
		for _, sub := range category.Subcategories {
			for _, ext := range sub.FileExtensions {
				ext = strings.ToLower(strings.TrimSpace(ext))
				if ext == "" {
					continue
				}
				if !strings.HasPrefix(ext, ".") {
					ext = "." + ext
				}
				exts[ext] = struct{}{}
			}
		}
	}
	return exts
}

// hasSupportedExtension reports whether filename ends with one of the supported
// extensions, checking compound suffixes (e.g. ".tar.gz") like the Python CLI.
func hasSupportedExtension(filename string, exts map[string]struct{}) bool {
	if len(exts) == 0 {
		return true
	}
	name := strings.ToLower(path.Base(filename))
	suffixes := allSuffixes(name)
	for i := range suffixes {
		candidate := strings.Join(suffixes[i:], "")
		if _, ok := exts[candidate]; ok {
			return true
		}
	}
	return false
}

func allSuffixes(filename string) []string {
	var suffixes []string
	for {
		ext := path.Ext(filename)
		if ext == "" {
			break
		}
		suffixes = append([]string{ext}, suffixes...)
		filename = strings.TrimSuffix(filename, ext)
	}
	return suffixes
}

func validateSupportedFileType(filename string, types SupportedFileTypes) error {
	exts := supportedExtensions(types)
	if len(exts) == 0 {
		return nil
	}
	if hasSupportedExtension(filename, exts) {
		return nil
	}
	return fmt.Errorf("%s: %s", fileTypeNotSupported, path.Base(filename))
}
