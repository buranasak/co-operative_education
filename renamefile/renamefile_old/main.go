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
	destinationPath := "/Users/buranasak/targetfile5"

	//เปลี่ยนชื่อไฟล์ -> zip -> copy zip จากต้นทางไปไฟล์ปลายทาง
	if err := modifiedAndCopyFile(sourcePath, destinationPath); err != nil {
		fmt.Println("Error:", err)
	}

}

// เปลี่ยนชื่อไฟล์ -> zip -> copy zip จากต้นทางไปไฟล์ปลายทาง
func modifiedAndCopyFile(sourcePath, destinationPath string) error {
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

		}

		if filepath.Ext(path) == ".csv" {
			fileName := strings.TrimSuffix(info.Name(), filepath.Ext(info.Name()))
			if strings.HasSuffix(strings.ToUpper(fileName), "WATERWORK") {
				newName := strings.ToUpper(fileName) + "S.csv"
				newPath := filepath.Join(filepath.Dir(path), newName)

				err := os.Rename(path, newPath)
				if err != nil {
					fmt.Println("Error:", err)
					return err
				}

			} else {

				newName := strings.ToUpper(fileName) + ".csv"
				newPath := filepath.Join(filepath.Dir(path), newName)

				err := os.Rename(path, newPath)
				if err != nil {
					fmt.Println("Error:", err)
					return err
				}

			}

		}

		//zip csv file
		if !strings.HasSuffix(info.Name(), "000") && filepath.Ext(path) != ".csv" {
			format := fmt.Sprintf("%s/", path)
			zipCsvFile(format)

		}

		return nil
	})

	if err != nil {
		return err
	}

	copyZipFileToDestination(sourcePath, destinationPath)

	return nil
}

// zip csv file
func zipCsvFile(folderPath string) error {
	date := time.Now().Format("20060102")
	zipFileName := fmt.Sprintf("%s_%s_ATTRIBUTE.zip", date, strings.ToUpper(filepath.Base(folderPath)))
	csvFilesExist := false

	files, err := os.ReadDir(folderPath)
	if err != nil {
		return err
	}

	//check if have .csv file or not
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".csv") {
			csvFilesExist = true
			break
		}
	}

	if !csvFilesExist {
		return nil
	}

	//สร้าง zip file
	zipFile, err := os.Create(folderPath + zipFileName)
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

			file, err := os.Open(filePath)
			if err != nil {
				return err
			}
			defer file.Close()

			_, err = io.Copy(zipEntry, file)
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

	fmt.Println("Copying zip files to destination completed successfully.")
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
