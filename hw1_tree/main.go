package main

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
)

const prefix = "───"
const middle = "├"
const middleParent = "│"
const last = "└"

func main() {
	out := os.Stdout
	if !(len(os.Args) == 2 || len(os.Args) == 3) {
		panic("usage go run main.go . [-f]")
	}
	path := os.Args[1]
	printFiles := len(os.Args) == 3 && os.Args[2] == "-f"
	err := dirTree(out, path, printFiles)
	if err != nil {
		panic(err.Error())
	}
}

func dirTree(out io.Writer, path string, printFiles bool) (err error) {
	return walkDir(out, path, printFiles, "")
}

func walkDir(out io.Writer, path string, printFiles bool, spacePrefix string) (err error) {
	fileInfos, err := ioutil.ReadDir(path)
	if err != nil {
		return err
	}

	sort.SliceStable(fileInfos, func(i, j int) bool {
		return fileInfos[i].Name() < fileInfos[j].Name()
	})

	if !printFiles {
		var onlyDir []os.FileInfo
		for _, fileInfo := range fileInfos {
			if fileInfo.IsDir() {
				onlyDir = append(onlyDir, fileInfo)
			}
		}
		fileInfos = onlyDir
	}

	maxInd := len(fileInfos) - 1
	for ind, fileInfo := range fileInfos {
		var curPrefix string
		isMiddle := ind < maxInd
		if isMiddle {
			curPrefix = middle
		} else {
			curPrefix = last
		}
		_, err = out.Write([]byte(spacePrefix + curPrefix + prefix + fileInfo.Name() + fileSize(fileInfo) + "\n"))
		if err != nil {
			return err
		}
		if fileInfo.IsDir() {
			err = walkDir(out, filepath.Join(path, fileInfo.Name()), printFiles, nextSpacePrefix(spacePrefix, isMiddle))
		}
		if err != nil {
			return err
		}

	}
	return nil
}

func fileSize(fileInfo os.FileInfo) (stringSize string) {
	if !fileInfo.IsDir() {
		if fileInfo.Size() == 0 {
			stringSize = " (empty)"
		} else {
			stringSize = " (" + strconv.FormatInt(fileInfo.Size(), 10) + "b)"
		}
	}
	return stringSize
}

func nextSpacePrefix(spacePrefix string, isMiddle bool) string {
	if isMiddle {
		spacePrefix = spacePrefix + middleParent + "\t"
	} else {
		spacePrefix = spacePrefix + "" + "\t"
	}
	return spacePrefix
}
