package helper

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"io"

	"emperror.dev/errors"
	json "github.com/json-iterator/go"
)

func ZipAndBase64Encode(originalObject any) (string, error) {

	original, err := json.Marshal(originalObject)
	if err != nil {
		return "", errors.Wrap(err, "Error when convert object to byte sequence")
	}

	// Create a buffer to write our archive to.
	buf := new(bytes.Buffer)

	// Create a new zip archive.
	w := zip.NewWriter(buf)

	f, err := w.Create("original")
	if err != nil {
		return "", err
	}
	_, err = f.Write(original)
	if err != nil {
		return "", err
	}

	// Make sure to check the error on Close.
	err = w.Close()
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

func UnZipBase64Decode(original string, originalObject any) error {

	if original == "" {
		return nil
	}

	decoded, err := base64.StdEncoding.DecodeString(original)
	if err != nil {
		return errors.Wrap(err, "Error when decode original")
	}

	zipReader, err := zip.NewReader(bytes.NewReader(decoded), int64(len(decoded)))
	if err != nil {
		return errors.Wrap(err, "Error when init zip reader")
	}

	// Read the file from zip archive
	zipFile := zipReader.File[0]
	unzippedFileBytes, err := readZipFile(zipFile)
	if err != nil {
		return errors.Wrap(err, "Error when unzip object")
	}

	// Convert to object
	if err = json.Unmarshal(unzippedFileBytes, originalObject); err != nil {
		return errors.Wrap(err, "Error when convert byte sequence to object")
	}

	return nil
}

func readZipFile(zf *zip.File) ([]byte, error) {
	f, err := zf.Open()
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}
