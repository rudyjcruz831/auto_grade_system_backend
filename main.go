package main

import (
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

const (
	uploadPath         = "./uploads/"
	allowedEmailDomain = "@example.edu"
)

type Response struct {
	Output string `json:"output,omitempty"`
	Error  string `json:"error,omitempty"`
}

func isAllowedEmail(email string) bool {
	re := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@example\.edu$`)
	return re.MatchString(email)
}

func saveUploadedFile(file *multipart.FileHeader, dst string) error {
	inFile, err := file.Open()
	if err != nil {
		return err
	}
	defer inFile.Close()

	outFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, inFile)
	return err
}

func processFile(filepath string) (string, error) {
	cmd := exec.Command("python3", filepath)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func uploadFileHandler(c *gin.Context) {
	email := c.PostForm("email")
	if !isAllowedEmail(email) {
		c.JSON(http.StatusBadRequest, Response{Error: "Invalid email address"})
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{Error: "No file part"})
		return
	}

	filename := fmt.Sprintf("%d_%s", time.Now().Unix(), file.Filename)
	filepath := uploadPath + filename

	if err := saveUploadedFile(file, filepath); err != nil {
		c.JSON(http.StatusInternalServerError, Response{Error: "Failed to save file"})
		return
	}

	output, err := processFile(filepath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Error: "Failed to process file"})
		return
	}

	c.JSON(http.StatusOK, Response{Output: output})
}

func main() {
	if err := os.MkdirAll(uploadPath, os.ModePerm); err != nil {
		log.Fatalf("Failed to create upload directory: %v", err)
	}

	r := gin.Default()
	r.Use(cors.Default())

	r.POST("/upload", uploadFileHandler)

	if err := r.Run(":5000"); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
