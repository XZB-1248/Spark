package file

import (
	"Spark/client/config"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"strconv"

	"github.com/imroc/req/v3"
)

type file struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
	Time int64  `json:"time"`
	Type int    `json:"type"` //0: file, 1: folder, 2: volume
}

// listFiles returns files and directories find in path.
func listFiles(path string) ([]file, error) {
	result := make([]file, 0)
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(files); i++ {
		itemType := 0
		if files[i].IsDir() {
			itemType = 1
		}
		result = append(result, file{
			Name: files[i].Name(),
			Size: files[i].Size(),
			Time: files[i].ModTime().Unix(),
			Type: itemType,
		})
	}
	return result, nil
}

func UploadFile(path, trigger string, start, end int64) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	reader, writer := io.Pipe()
	defer file.Close()
	uploadReq := req.R()
	stat, err := file.Stat()
	if err != nil {
		return err
	}
	size := stat.Size()
	headers := map[string]string{
		`Trigger`:  trigger,
		`FileName`: stat.Name(),
		`FileSize`: strconv.FormatInt(size, 10),
	}
	if size < end {
		return errors.New(`Invalid file size.`)
	}
	if end == 0 {
		uploadReq.RawRequest.ContentLength = size - start
	} else {
		uploadReq.RawRequest.ContentLength = end - start
	}
	shouldRead := uploadReq.RawRequest.ContentLength
	file.Seek(start, 0)
	go func() {
		for {
			bufSize := int64(2 << 14)
			if shouldRead < bufSize {
				bufSize = shouldRead
			}
			buffer := make([]byte, bufSize) // 32768
			n, err := file.Read(buffer)
			buffer = buffer[:n]
			shouldRead -= int64(n)
			writer.Write(buffer)
			if n == 0 || shouldRead == 0 || err != nil {
				break
			}
		}
		writer.Close()
	}()
	url := config.GetBaseURL(false) + `/api/device/file/put`
	_, err = uploadReq.SetBody(reader).SetHeaders(headers).Send(`PUT`, url)
	reader.Close()
	return err
}
