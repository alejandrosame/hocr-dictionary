package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/alejandrosame/gohocr"
)

// Define useful datatypes
type Bbox struct {
	X0 int
	Y0 int
	X1 int
	Y1 int
}

func parseBbox(title string) Bbox {
	bbox := Bbox{}
	re, _ := regexp.Compile(`bbox \d+ \d+ \d+ \d+;`)

	bboxStr := re.FindString(title)

	if bboxStr == "" {
		bbox.X0 = -1
		bbox.Y0 = -1
		bbox.X1 = -1
		bbox.Y1 = -1
		return bbox
	}

	splitted := strings.Split(bboxStr[:len(bboxStr)-1], " ")

	bbox.X0, _ = strconv.Atoi(splitted[1])
	bbox.Y0, _ = strconv.Atoi(splitted[2])
	bbox.X1, _ = strconv.Atoi(splitted[3])
	bbox.Y1, _ = strconv.Atoi(splitted[4])

	return bbox
}

func (b Bbox) String() string {
	return fmt.Sprintf("BBOX %d %d %d %d", b.X0, b.Y0, b.X1, b.Y1)
}

func (in Bbox) contained(out Bbox) bool {
	return in.X0 >= out.X0 && in.X1 <= out.X1 && in.Y0 >= out.Y0 && in.Y1 <= out.Y1
}

// Start main code

func main() {
	infoLog := log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	errorLog := log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)

	// Flag declaration
	root := flag.String("input", "", "Input folder with hOCR files to process")
	startPage := flag.Int("start-page", 0, "Page where dictionary content starts (index starts with 0)")
	required := []string{"input"}
	flag.Parse()

	// Check mandatory flags were explicitly stated
	seen := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) { seen[f.Name] = true })
	for _, req := range required {
		if !seen[req] {
			errorLog.Println(fmt.Sprintf("missing required argument, '-%s' \n", req))
			os.Exit(2)
		}
	}

	infoLog.Println(fmt.Sprintf("%s - %d", *root, *startPage))
	hocrFiles := getFiles(root)

	page, err := gohocr.Parse(filepath.Join(*root, (*hocrFiles)[40]))
	if err != nil {
		errorLog.Println(fmt.Sprintf("Error parsing hOCR file: %s", err))
		return
	}

	// Define BBOX to search for tile letter in the dictionary
	titleBbox := Bbox{X0: 0, Y0: 300, X1: 3200, Y1: 700}
	// Define BBOX to search for index letter in dictionary page
	indexBbox := Bbox{X0: 0, Y0: 150, X1: 3200, Y1: 300}

	// Search title letter
	words := getWordsInBbox(page, titleBbox)
	infoLog.Println(fmt.Sprintf("%+v", (*words)))

	page, err = gohocr.Parse(filepath.Join(*root, (*hocrFiles)[127]))
	if err != nil {
		errorLog.Println(fmt.Sprintf("Error parsing hOCR file: %s", err))
		return
	}

	// Search index words
	words = getWordsInBbox(page, indexBbox)
	infoLog.Println(fmt.Sprintf("%+v", (*words)))
}

func extension(fileName string) string {
	return filepath.Ext(fileName)
}

func sortName(filename string) string {
	ext := filepath.Ext(filename)
	name := filename[:len(filename)-len(ext)]
	// split numeric suffix
	i := len(name) - 1
	for ; i >= 0; i-- {
		if '0' > name[i] || name[i] > '9' {
			break
		}
	}
	i++
	// string numeric suffix to uint64 bytes
	// empty string is zero, so integers are plus one
	b64 := make([]byte, 64/8)
	s64 := name[i:]
	if len(s64) > 0 {
		u64, err := strconv.ParseUint(s64, 10, 64)
		if err == nil {
			binary.BigEndian.PutUint64(b64, u64+1)
		}
	}
	// prefix + numeric-suffix + ext
	return name[:i] + string(b64) + ext
}

func getFiles(root *string) *[]string {
	files, err := ioutil.ReadDir(*root)
	if err != nil {
		log.Fatal(err)
	}

	fileNames := []string{}
	for _, file := range files {
		if extension(file.Name()) == ".hocr" {
			fileNames = append(fileNames, file.Name())
		}
	}

	// Sorting solution credited to https://stackoverflow.com/a/51363401
	sort.Slice(
		fileNames,
		func(i, j int) bool {
			return sortName(fileNames[i]) < sortName(fileNames[j])
		},
	)

	return &fileNames
}

func getWordsInBbox(page gohocr.Page, out Bbox) *[]gohocr.Word {
	words := []gohocr.Word{}

	for _, word := range page.Words {
		wordBbox := parseBbox(word.Title)

		if wordBbox.contained(out) {
			words = append(words, word)
		}
	}

	return &words
}
