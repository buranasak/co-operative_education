package main

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {

	//ที่อยู่ไฟล์ต้นทาง
	sourcePath := "/Users/buranasak/Downloads/file1"

	//ที่อยู่ไฟล์ปลายทาง
	destinationPath := "/Users/buranasak/targetfile1"

	//เปลี่ยนชื่อไฟล์ -> zip -> copy zip จากต้นทางไปไฟล์ปลายทาง
	if err := modifiedFile(sourcePath, destinationPath); err != nil {
		fmt.Println("Error:", err)
	}

}

// เปลี่ยนชื่อไฟล์ -> zip -> copy zip จากต้นทางไปไฟล์ปลายทาง
func modifiedFile(sourcePath, destinationPath string) error {
	err := filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			newFolderName := strings.ToUpper(filepath.Join(filepath.Dir(path), info.Name()))
			if newFolderName != path {
				err := os.Rename(path, newFolderName)
				if err != nil {
					return err
				}
			}
			return nil
		}

		// Rename CSV files
		if filepath.Ext(path) == ".csv" {
			fileName := strings.TrimSuffix(info.Name(), filepath.Ext(info.Name()))
			newName := strings.ToUpper(fileName) + ".csv"
			if strings.HasSuffix(strings.ToUpper(fileName), "WATERWORK") {
				newName = strings.ToUpper(fileName) + "S.csv"
			}

			newPath := filepath.Join(filepath.Dir(path), newName)

			if newPath != path {
				if err := os.Rename(path, newPath); err != nil {
					return err
				}
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	//zip csv file
	err = filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() && !strings.HasSuffix(info.Name(), "000") {
			zipCsvFile(path)
			return nil
		}

		return nil
	})

	if err != nil {
		return err
	}

	copyZipFileToDestination(sourcePath, destinationPath)

	return nil
}

// zip csv file function
func zipCsvFile(folderPath string) error {
	date := time.Now().Format("20060102")
	zipFileName := fmt.Sprintf("%s_%s_ATTRIBUTE.zip", date, strings.ToUpper(filepath.Base(folderPath)))
	csvFilesExist := false

	files, err := os.ReadDir(folderPath)
	if err != nil {
		return err
	}

	// check if have .csv file or not
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".csv") {
			csvFilesExist = true
			break
		}
	}

	if !csvFilesExist {
		return nil
	}

	// //สร้าง zip file
	// zipFilePath := filepath.Join(folderPath, zipFileName)
	zipFile, err := os.Create(filepath.Join(folderPath, zipFileName))
	if err != nil {
		return err
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".csv") {
			filePath := filepath.Join(folderPath, file.Name())

			zipEntry, err := zipWriter.Create(file.Name())
			if err != nil {
				return err
			}

			sourceFile, err := os.Open(filePath)
			if err != nil {
				return err
			}
			defer sourceFile.Close()

			_, err = io.Copy(zipEntry, sourceFile)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// copy zip file to another folder
func copyZipFileToDestination(sourcePath, destinationPath string) error {

	_, err := os.Stat(destinationPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("Destination folder '%s' does not exist.\n", destinationPath)
			return err
		}
		fmt.Printf("Error accessing destination folder '%s': %v\n", destinationPath, err)
		return err
	}

	err = filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(sourcePath, path)
		if err != nil {
			return err
		}

		destinationPath := filepath.Join(destinationPath, relPath)

		if !info.IsDir() && filepath.Ext(path) == ".zip" {
			if err := copyFile(path, destinationPath); err != nil {
				return err
			}
		}

		return nil

	})
	if err != nil {
		return err
	}

	fmt.Println("Zip file had been copy to destination folder")
	return nil
}

// copy zip file from source to destination folder
func copyFile(sourcePath, destinationPath string) error {
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destinationFile, err := os.Create(destinationPath)
	if err != nil {
		return err
	}
	defer destinationFile.Close()

	zipReader, err := zip.OpenReader(sourcePath)
	if err != nil {
		return err
	}
	defer zipReader.Close()

	zipWriter := zip.NewWriter(destinationFile)
	defer zipWriter.Close()

	for _, file := range zipReader.File {
		fileReader, err := file.Open()
		if err != nil {
			return err
		}
		defer fileReader.Close()

		destinationFile, err := zipWriter.Create(file.Name)
		if err != nil {
			return err
		}

		_, err = io.Copy(destinationFile, fileReader)
		if err != nil {
			return err
		}
	}

	return nil
}
