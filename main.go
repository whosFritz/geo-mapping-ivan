package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/fsnotify/fsnotify"
	_ "github.com/go-sql-driver/mysql"
	ipdata "github.com/ipdata/go"
	"github.com/joho/godotenv"

)

func main() {
	err_env := godotenv.Load("./ivan.env")
	if err_env != nil {
		log.Fatalf("Some error occured. Err: %s", err_env)
	}

	dbUser := os.Getenv("DB_USER")
	dbPass := os.Getenv("DB_PASS")
	dbName := os.Getenv("DB_NAME")

	dsn := fmt.Sprintf("%s:%s@/%s", dbUser, dbPass, dbName)
	// Open a connection to the MariaDB database

	db, err_connect := sql.Open("mysql", dsn)

	if err_connect != nil {
		log.Fatalf("Error opening MariaDB connection: %v", err_connect)
	}
	defer db.Close()

	// Ensure the database connection is alive
	err_ping := db.Ping()
	if err_ping != nil {
		log.Fatalf("Error connecting to MariaDB: %v", err_ping)
	}
	err_createTable := createTable(db)
	if err_createTable != nil {
		panic(err_createTable.Error())
	}
	// Create the table if it doesn't exist

	token := os.Getenv("TOKEN")
	// Specify the file path you want to monitor
	filePath := os.Getenv("FILE_PATH")

	// Create a new watcher instance
	watcher, err_watcher := fsnotify.NewWatcher()
	if err_watcher != nil {
		log.Fatal(err_watcher)
	}
	defer watcher.Close()

	// Add the file to the watcher
	err_addFile := watcher.Add(filePath)
	if err_addFile != nil {
		log.Fatal(err_addFile)
	}

	log.Printf("Monitoring changes to %s...\n", filePath)

	// Start a loop to watch for events
	for {
		select {
		case event := <-watcher.Events:
			// Check if the event is a modification
			if event.Op&fsnotify.Write == fsnotify.Write {
				readLastLine(db, filePath, token)
			}
		case err := <-watcher.Errors:
			log.Println("Error:", err)
		}
	}
}

func readLastLine(db *sql.DB, filePath string, token string) {
	file, err_open := os.Open(filePath)
	if err_open != nil {
		log.Println("Error opening file:", err_open)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var lastLine string
	for scanner.Scan() {
		lastLine = scanner.Text()
	}

	if strings.Contains(lastLine, "Failed password for root") || strings.Contains(lastLine, "Failed password for invalid user") {
		username, ip := extractData(lastLine)
		// if ip found in table then update hitcount else insert new row

		ivan := getIvan(ip, token)
		// printf statement to print username ip and Country name

		log.Printf("Username: %s, IP: %s, Country: %s, City: %s\n", username, ivan.IP, ivan.CountryName, ivan.City)
		if ivan.IP != "" || ivan.Latitude != 0 {
			username, ip := extractData(lastLine)

			log.Printf("Extracted: username: %s, IP: %s\n", username, ip)
			// Update or insert record in database
			updateRecord(db, ip, username, token)
			log.Println("Record updated")
		}
	}
	if err_scan := scanner.Err(); err_scan != nil {
		log.Println("Error reading file:", err_scan)
		return
	}
}

func extractData(lastLine string) (string, string) {
	parts := strings.Split(lastLine, " ")
	if strings.Contains(lastLine, "invalid user") {
		return forInvalidUser(parts)
	} else {
		return forRootUserFail(parts)
	}
}
func forInvalidUser(parts []string) (string, string) {
	// find "from" in parts
	for i, v := range parts {
		if v == "invalid" {
			return parts[i+2], parts[i+4]
		}
	}
	return "", ""
}
func forRootUserFail(parts []string) (string, string) {
	// find "root" in parts
	for i, v := range parts {
		if v == "root" {
			return parts[i], parts[i+2]
		}
	}
	return "", ""
}

func getIvan(ip string, token string) Ivan {
	ipd, _ := ipdata.NewClient(token)

	data, err_lookUp := ipd.Lookup(ip)
	if err_lookUp != nil {
		log.Fatal(err_lookUp)
	}
	return Ivan{
		IP:            data.IP,
		IsEU:          data.IsEU,
		City:          data.City,
		RegionName:    data.Region,
		RegionCode:    data.RegionCode,
		CountryName:   data.CountryName,
		CountryCode:   data.CountryCode,
		ContinentName: data.ContinentName,
		ContinentCode: data.ContinentCode,
		Latitude:      data.Latitude,
		Longitude:     data.Longitude,
		Postal:        data.Postal,
		CallingCode:   data.CallingCode,
	}
}
func createTable(db *sql.DB) error {
	stmt, err := db.Prepare(`CREATE TABLE IF NOT EXISTS login_attempts (
		id INT AUTO_INCREMENT PRIMARY KEY,
		ip VARCHAR(45) NOT NULL,
		username VARCHAR(255) NOT NULL,
		city VARCHAR(255),
		region VARCHAR(255),
		region_code VARCHAR(255),
		country VARCHAR(255),
		country_code VARCHAR(255),
		continent VARCHAR(255),
		continent_code VARCHAR(255),
		latitude FLOAT,
		longitude FLOAT,
		postal VARCHAR(255),
		calling_code VARCHAR(255),
		hitcount INT DEFAULT 1
    )`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	res, err := stmt.Exec()
	if err != nil {
		return err
	}

	lastId, err := res.LastInsertId()
	if err != nil {
		return err
	}

	rowCnt, err := res.RowsAffected()
	if err != nil {
		return err
	}

	log.Printf("ID = %d, affected = %d\n", lastId, rowCnt)
	return nil
}

type Ivan struct {
	IP            string  `json:"ip"`
	IsEU          bool    `json:"is_eu"`
	City          string  `json:"city"`
	RegionName    string  `json:"region"`
	RegionCode    string  `json:"region_code"`
	CountryName   string  `json:"country_name"`
	CountryCode   string  `json:"country_code"`
	ContinentName string  `json:"continent_name"`
	ContinentCode string  `json:"continent_code"`
	Latitude      float64 `json:"latitude"`
	Longitude     float64 `json:"longitude"`
	Postal        string  `json:"postal"`
	CallingCode   string  `json:"calling_code"`
}

func updateRecord(db *sql.DB, ip string, username string, token string) {
	// Check if the IP address exists in the table
	var hitcount int
	err := db.QueryRow("SELECT hitcount FROM login_attempts WHERE ip = ? AND username = ?", ip, username).Scan(&hitcount)
	if err != nil && err != sql.ErrNoRows {
		log.Fatalf("Error querying hitcount: %v", err)
	}

	if err == sql.ErrNoRows {
		// IP address not found, insert a new record
		stmt, err := db.Prepare(`INSERT INTO login_attempts (ip, username, city, region, region_code, country, country_code, continent, continent_code, latitude, longitude, postal, calling_code) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
		if err != nil {
			log.Fatalf("Error preparing insert statement: %v", err)
		}
		defer stmt.Close()

		ivan := getIvan(ip, token)

		_, err = stmt.Exec(
			ivan.IP,
			username,
			ivan.City,
			ivan.RegionName,
			ivan.RegionCode,
			ivan.CountryName,
			ivan.CountryCode,
			ivan.ContinentName,
			ivan.ContinentCode,
			ivan.Latitude,
			ivan.Longitude,
			ivan.Postal,
			ivan.CallingCode,
		)
		if err != nil {
			log.Fatalf("Error inserting record: %v", err)
		}
	} else {
		// IP address found, update the hit count
		stmt, err := db.Prepare(`UPDATE login_attempts SET hitcount = ? WHERE ip = ?`)
		if err != nil {
			log.Fatalf("Error preparing update statement: %v", err)
		}
		defer stmt.Close()

		_, err = stmt.Exec(hitcount+1, ip)
		if err != nil {
			log.Fatalf("Error updating record: %v", err)
		}
	}
}
