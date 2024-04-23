package nvdsync

import (
	"archive/zip"
	"bufio"
	"compress/gzip"
	"io"
	"os"
)

// CompressFile compresses a file using gzip and writes it to a new file
// with the given name and removes the old file.
func CompressFile(fileName string, newFileName string) error {
	// Read old file
	file, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer file.Close()

	read := bufio.NewReader(file)
	data, err := io.ReadAll(read)
	if err != nil {
		return err
	}

	// Write new file
	newFile, err := os.Create(newFileName)
	if err != nil {
		return err
	}
	defer newFile.Close()

	writer := gzip.NewWriter(newFile)
	defer writer.Close()
	if _, err = writer.Write(data); err != nil {
		return err
	}

	// Remove old file
	if err = os.Remove(fileName); err != nil {
		return err
	}

	return nil
}

func zipFiles(sources []string, target string) error {
	// Create a new zip archive.
	zipFile, err := os.Create(target)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	for _, source := range sources {
		// Add a file to the archive.
		fileToZip, err := os.Open(source)
		if err != nil {
			return err
		}
		defer fileToZip.Close()

		// Get the file information.
		info, err := fileToZip.Stat()
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		// Using FileInfoHeader() above only uses the basename of the file. If you want
		// to preserve the folder structure (for example, if you're zipping files from
		// a directory), you would need to set header.Name to the full path.
		header.Name = source

		// Change to deflate to reduce file size but keep it compatible with unzip.
		header.Method = zip.Deflate

		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}

		if _, err = io.Copy(writer, fileToZip); err != nil {
			return err
		}
	}
	return nil
}
