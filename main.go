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
    //fileNames := getFiles(root) 
    hocrFiles := getFiles(root) 

    page, err := gohocr.Parse(filepath.Join(*root, (*hocrFiles)[40]))
    if err != nil {
        errorLog.Println(fmt.Sprintf("Error parsing hOCR file: %s", err))
        return
    }

    // Search title letter
    words := getWordsInBbox(page, 0, 300, 3200, 700)
    infoLog.Println(fmt.Sprintf("%+v", (*words)))

    page, err = gohocr.Parse(filepath.Join(*root, (*hocrFiles)[127]))
    if err != nil {
        errorLog.Println(fmt.Sprintf("Error parsing hOCR file: %s", err))
        return
    }

    // Search index words
    words = getWordsInBbox(page, 0, 150, 3200, 300)
    infoLog.Println(fmt.Sprintf("%+v", (*words)))
}


func extension(fileName string) string {
    return filepath.Ext(fileName);
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


func parseBbox(title string) (int, int, int, int) {
    re, _ := regexp.Compile(`bbox \d+ \d+ \d+ \d+;`)

    bbox := re.FindString(title)

    if bbox == "" {
        return -1, -1, -1, -1
    }
    
    splitted := strings.Split(bbox[:len(bbox)-1], " ")

    x0, _ := strconv.Atoi(splitted[1])
    y0, _ := strconv.Atoi(splitted[2])
    x1, _ := strconv.Atoi(splitted[3])
    y1, _ := strconv.Atoi(splitted[4])

    return x0, y0, x1, y1
}


func contained(xIn0, yIn0, xIn1, yIn1, xOut0, yOut0, xOut1, yOut1 int) bool {
    return xIn0 >= xOut0 && xIn1 <= xOut1 && yIn0 >= yOut0 && yIn1 <= yOut1
}


func getWordsInBbox(page gohocr.Page, X0, Y0, X1, Y1 int) *[]gohocr.Word {
    words := []gohocr.Word{}

    for _, word := range page.Words {
        x0, y0, x1, y1 := parseBbox(word.Title)

        if contained(x0, y0, x1, y1, X0, Y0, X1, Y1) {
            words = append(words, word)
        }
    }

    return &words
}