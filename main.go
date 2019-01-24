package main

import (
	"archive/zip"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	iconv "github.com/qiniu/iconv"
)

// SortFile sortfiles
type SortFile []*zip.File

var encodingCandidates = []string{"utf-8", "gbk", "big5", "shift-jis"}

var logger = log.New(os.Stderr, "", 0)

var encoding string
var targetDir string

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] zipfile ...\n\n", filepath.Base(os.Args[0]))
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		os.Exit(0)
	}
	flag.StringVar(&encoding, "c", "", "Forcing codec instead of auto detecting")
	flag.StringVar(&targetDir, "d", "", "Target directory to save unziped files")
}

func main() {
	flag.Parse()
	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(1)
	}

	for _, zipfile := range flag.Args() {
		func() {
			rzip, err := zip.OpenReader(zipfile)
			if err != nil {
				logger.Print(err)
				return
			}
			defer rzip.Close()
			sort.Sort(SortFile(rzip.File))

			var utf8name string
			var encErr error

			for _, file := range rzip.File {
			DETERMINE_ENC:
				if encoding == "" {
					for _, enc := range encodingCandidates {
						cd, encErr := iconv.Open("utf-8", enc)
						utf8name = cd.ConvString(file.Name)
						if encErr == nil || utf8name == "" {
							encoding = enc
							break
						}
					}

					if encErr != nil {
						logger.Print(encErr)
						return
					}
				} else {
					cd, encErr := iconv.Open("utf-8", encoding)
					utf8name = cd.ConvString(file.Name)
					if encErr != nil || utf8name == "" {
						encoding = ""
						goto DETERMINE_ENC
					}
				}
				fmt.Printf("filename : %s", utf8name)
				if targetDir != "" {
					utf8name = filepath.Join(targetDir, utf8name)
				}
				if strings.HasSuffix(file.Name, "/") {
					os.MkdirAll(utf8name, 0755)
				} else {
					filedir := filepath.Dir(utf8name)
					if _, err := os.Stat(filedir); err != nil {
						os.MkdirAll(filedir, 0755)
					}

					err := extractZip(utf8name, file)
					if err != nil {
						logger.Print(err)
					} else {
						fmt.Println(utf8name)
					}
				}
			}
		}()
	}
}

func extractZip(dst string, zf *zip.File) error {
	zfReader, err := zf.Open()
	if err != nil {
		return err
	}
	defer zfReader.Close()

	flag := os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	dstWriter, err := os.OpenFile(dst, flag, 0644)
	if err != nil {
		return err
	}
	defer dstWriter.Close()

	copiedSize, err := io.Copy(dstWriter, zfReader)
	if err != nil {
		return err
	}
	if uint64(copiedSize) != zf.UncompressedSize64 {
		return fmt.Errorf("Failed to extract file %s: size mismatched", dst)
	}
	return nil
}

func (f SortFile) Len() int {
	return len(f)
}

func (f SortFile) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}

func (f SortFile) Less(i, j int) bool {
	if strings.HasSuffix(f[i].Name, "/") {
		if strings.HasSuffix(f[j].Name, "/") {
			return len(f[i].Name) < len(f[j].Name)
		} else {
			return true
		}
	} else {
		if strings.HasSuffix(f[j].Name, "/") {
			return false
		} else {
			return len(f[i].Name) < len(f[j].Name)
		}
	}
}
