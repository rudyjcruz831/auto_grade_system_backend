package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type TestCase struct {
	N, Kr, Kc, Pr, Pc int
	Ans               string
}

type TestResult struct {
	CaseID   int
	Passed   bool
	Stdout   string
	Stderr   string
	Err      error
	ExecTime time.Duration
}

type Response struct {
	Results []TestResult `json:"results,omitempty"`
	Summary string       `json:"summary,omitempty"`
	Error   string       `json:"error,omitempty"`
}

type Users struct {
	Id        string    `json:"user_id" gorm:"column:user_id;primary_key"`
	BestScore int       `json:"best_score" gorm:"column:best_score"`
	Username  string    `json:"username" gorm:"column:username;unique;not null"`
	Email     string    `json:"email" gorm:"column:email;unique;not null"`
}

func isAllowedEmail(email string) bool {
	re := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@sjsu\.edu$`)
	return re.MatchString(email)
}

func runPythonScript(scriptContent string, testCase TestCase) (string, string, time.Duration, error) {
	// Add the test case parameters to the script content
	// formattedScript := fmt.Sprintf(scriptContent, testCase.N, testCase.Kr, testCase.Kc, testCase.Pr, testCase.Pc)

	// Create a temporary file for the Python script
	tmpFile, err := os.CreateTemp("", "*.py")
	if err != nil {
		log.Println("Failed to create temporary file")
		return "", "", 0, err
	}
	defer os.Remove(tmpFile.Name())

	// Add the __name__ == "__main__" block to the script content
	mainBlock := "if __name__ == \"__main__\":\n\timport sys\n\tinput_data = sys.stdin.read().strip()\n\tn, kr, kc, pr, pc = map(int, input_data.split())\n\tmoves = knight_attack(n, kr, kc, pr, pc)\n\tprint(moves)"
	fullScript := fmt.Sprintf("%s\n%s", scriptContent, mainBlock)

	// log.Println("Script content:", fullScript)
	// Write the full script to the temporary file
	if _, err := tmpFile.WriteString(fullScript); err != nil {
		log.Println("Failed to write to temporary file")
		return "", "", 0, err
	}
	if err := tmpFile.Close(); err != nil {
		log.Println("Failed to close temporary file")
		return "", "", 0, err
	}

	// Create a new command to run the Python interpreter
	cmd := exec.Command("python3", tmpFile.Name())

	// Provide the input for the script
	input := fmt.Sprintf("%d %d %d %d %d", testCase.N, testCase.Kr, testCase.Kc, testCase.Pr, testCase.Pc)
	cmd.Stdin = bytes.NewBufferString(input)

	// Capture the command output
	var output bytes.Buffer
	// cmd.Stdout = &output
	var stderr bytes.Buffer
	// cmd.Stderr = &stderr

	cmd.Stdout = io.MultiWriter(os.Stdout, &output)
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderr)

	// Run the command and measure execution time
	start := time.Now()
	cmd.Run()
	execTime := time.Since(start)

	// Return the command output as a string
	return strings.TrimSpace(output.String()), stderr.String(), execTime, nil
}

func uploadFileHandler(c *gin.Context) {
	email := c.PostForm("email")
	if email == "" {
		log.Println("Email address is required")
		c.JSON(http.StatusBadRequest, Response{Error: "Email address is required"})
		return
	}
	if !isAllowedEmail(email) {
		log.Println("Invalid email address")
		c.JSON(http.StatusBadRequest, Response{Error: "Invalid email address"})
		return
	}

	file, _, err := c.Request.FormFile("file")
	if err != nil {
		log.Println("No file part")
		c.JSON(http.StatusBadRequest, Response{Error: "No file part"})
		return
	}

	// Read file content into memory
	var buf bytes.Buffer
	_, err = io.Copy(&buf, file)
	if err != nil {
		log.Println("Failed to read file content")
		c.JSON(http.StatusInternalServerError, Response{Error: "Failed to read file content"})
		return
	}
	fileContent := buf.String()

	// Define test cases
	testCases := []TestCase{
		{8, 1, 1, 2, 2, "2"},
		{8, 1, 1, 2, 3, "1"},
		{8, 0, 3, 4, 2, "3"},
		{8, 0, 3, 5, 2, "4"},
		{24, 4, 7, 19, 20, "10"},
		{100, 21, 10, 0, 0, "11"},
		{3, 0, 0, 1, 2, "1"},
		{3, 0, 0, 1, 1, "None"},
	}
	db, err := initDS()
	
	// Find user by email
        var user Users
        if err := db.DB.Where("email = ?", email).First(&user).Error; err != nil {
                log.Printf("Failed to find user by email: %v", err)
                c.JSON(http.StatusInternalServerError, Response{Error: "Failed to find user by email"})
                return
        }
	// knight_attack(8, 1, 1, 2, 2) == 2
	// knight_attack(8, 1, 1, 2, 3) == 1
	// knight_attack(8, 0, 3, 4, 2) == 3
	// knight_attack(8, 0, 3, 5, 2) == 4
	// knight_attack(24, 4, 7, 19, 20) == 10
	// knight_attack(100, 21, 10, 0, 0) == 11
	// knight_attack(3, 0, 0, 1, 2) == 1
	// knight_attack(3, 0, 0, 1, 1) == none

	var results []TestResult
	passedTests := 0

	for i, tc := range testCases {
		output, stderr, execTime, err := runPythonScript(fileContent, tc)
		passed := err == nil

		// check how long it take if it takes longer then 2 seconds then it failed
		if execTime.Seconds() > 2 {
			c.JSON(http.StatusRequestTimeout, Response{Error: "Execution time exceeded 2 seconds"})
			if err := db.DB.Model(&user).Update("best_score", 0).Error; err != nil {
                        	log.Printf("Failed to update user's best score: %v", err)
                        	c.JSON(http.StatusInternalServerError, Response{Error: "Failed to update user's best score"})
                        	return
                	}
			return
		}

		if len(stderr) > 0 {
			log.Println("Went int stderr is long")
			c.JSON(http.StatusInternalServerError, Response{Error: "Failed to compile the script"})
			if err := db.DB.Model(&user).Update("best_score", 0).Error; err != nil {
                        	log.Printf("Failed to update user's best score: %v", err)
                        	c.JSON(http.StatusInternalServerError, Response{Error: "Failed to update user's best score"})
                        	return
                	}
			return
		}

		if passed {
			if output == tc.Ans {
				// passed = false
				passed = true
				passedTests++
			}
			// passedTests++
		}

		results = append(results, TestResult{
			CaseID:   i,
			Passed:   passed,
			Stdout:   output,
			Stderr:   stderr,
			Err:      err,
			ExecTime: time.Duration(execTime.Milliseconds()),
		})
	}

	score := int( float64(passedTests) / float64( len(testCases) )  * 100)


	if err != nil {
		log.Printf("Failed to connect to database: %v", err)
		c.JSON(http.StatusInternalServerError, Response{Error: "Failed to connect to database"})
		return
	}
	defer db.Close()

	// Update user's best score if the new score is higher
	// fmt.Println(type(user.BestScore)
	if score > user.BestScore {
		if err := db.DB.Model(&user).Update("best_score", score).Error; err != nil {
			log.Printf("Failed to update user's best score: %v", err)
			c.JSON(http.StatusInternalServerError, Response{Error: "Failed to update user's best score"})
			return
		}
	}

	summary := fmt.Sprintf("%d/%d tests passed", passedTests, len(testCases))

	c.JSON(http.StatusOK, Response{Results: results, Summary: summary})
}

func main() {

	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"POST, GET"},
		AllowHeaders: []string{"Content-type"},
	}))

	r.POST("/upload", uploadFileHandler)

	if err := r.Run(":50052"); err != nil {
		log.Fatalf("Failed to run server: %v", err)

	}
}

type dataSources struct {
	DB    *gorm.DB
	sqlDB *sql.DB
}

func initDS() (*dataSources, error) {
	// load env variables for postgres
	dsn := fmt.Sprintf("host=%s dbname=%s port=%s user=%s password=%s",
		os.Getenv("POSTGRES_HOST"),
		os.Getenv("POSTGRES_NAME"),
		os.Getenv("POSTGRES_PORT"),
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"))

	log.Printf("Connecting to Postgresql")
	db, err := gorm.Open(postgres.Open(dsn))
	if err != nil {
		fmt.Printf("Unable to connect to db %v \n", err)
		panic(err)
	}
	// this returns gorms own interface to use to ping in next lines of code still dont understand
	// this website can help udnerstand more if need = https://gorm.io/docs/generic_interface.html
	// Get generic database object sql.DB to use its functions
	sqlDB, err := db.DB()
	if err != nil {
		log.Printf("Failed to get sqlDB from db.DB() error : %v", err)
		panic(err)
		// return nil, err
	}

	// verify database connection is working by ping database
	if err := sqlDB.Ping(); err != nil {
		// for i := 0; i < 3; i++ {
		// 	fmt.Println("Ping")
		// }
		panic(err)
		// return nil, err
	}

	return &dataSources{DB: db, sqlDB: sqlDB}, nil
}

func (d *dataSources) Close() error {
	//close postgresDB
	if err := d.sqlDB.Close(); err != nil {
		return fmt.Errorf("error closing Postgresql: %w", err)
	}

	return nil
}
