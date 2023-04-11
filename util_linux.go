package main

import (
	"errors"
	"io"
	"mime/multipart"
	"os"
	fpath "path/filepath"
	"time"

	"github.com/h2non/filetype"
	cp "github.com/otiai10/copy"
)

func UploadSmallFiles(filename string, folder string, file multipart.File) (ret string, err error) {
	sub := time.Now().Format("01021504")
	workDir, _ := os.Getwd()
	filesDir := fpath.Join(workDir, folder, filename+"_"+sub)
	out, err := os.Create(filesDir)
	if err != nil {
		return "", err
	}
	defer out.Close()
	io.Copy(out, file)

	kind, _ := filetype.MatchFile(filesDir)
	if kind.Extension != "elf" || kind.Extension != "gz" {
		return "", errors.New("file is not elf or tar.gz")
	}
	return filesDir, nil
}

func CopyFile(src, dst string) (written int64, err error) {
	if stat, err := os.Stat(src); err != nil {
		return 0, err
	} else {
		// 如果是目录
		if stat.IsDir() {
			err = cp.Copy(src, dst)
			return 0, err
		}
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return
	}
	defer dstFile.Close()

	return io.Copy(dstFile, srcFile)
}
