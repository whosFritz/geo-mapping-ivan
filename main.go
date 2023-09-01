package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/fsnotify/fsnotify"
	ipdata "github.com/ipdata/go"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	failedLogins = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "failed_logins_total",
		Help: "The total number of failed login attempts",
	}, []string{
		"ip",
		"username",
		"city",
		"region",
		"region_code",
		"country",
		"country_code",
		"continent",
		"continent_code",
		"latitude",
		"longitude",
		"postal",
		"calling_code",
		"flag",
		"emoji_flag",
		"emoji_unicode",
	})
)

func init() {
	// Register the Prometheus HTTP handler
	http.Handle("/metrics", promhttp.Handler())
}

func main() {
	err := godotenv.Load("./ivan.env")
	if err != nil {
		log.Fatalf("Some error occured. Err: %s", err)
	}
	token := os.Getenv("TOKEN")
	// Specify the file path you want to monitor
	filePath := "./auth.log"

	// Create a new watcher instance
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	// Add the file to the watcher
	err = watcher.Add(filePath)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Monitoring changes to %s...\n", filePath)
	go func() {
		log.Fatal(http.ListenAndServe(":9101", nil))
	}()
	// Start a loop to watch for events
	for {
		select {
		case event := <-watcher.Events:
			// Check if the event is a modification
			if event.Op&fsnotify.Write == fsnotify.Write {
				readLastLine(filePath, token)
			}
		case err := <-watcher.Errors:
			log.Println("Error:", err)
		}
	}
}

func readLastLine(filePath string, token string) {
	file, err := os.Open(filePath)
	if err != nil {
		log.Println("Error opening file:", err)
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
		ivan := getIvan(ip, token)
		// printf statement to print username ip and Country name

		fmt.Printf("Username: %s, IP: %s, Country: %s, City: %s, Latitude: %s\n", username, ivan.IP, ivan.CountryName, ivan.City, fmt.Sprintf("%f", ivan.Latitude))
		if ivan.IP != "" || ivan.Latitude != 0 {
			recordMetrics(ivan, username)
		}
	}
	if err := scanner.Err(); err != nil {
		log.Println("Error reading file:", err)
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

	data, err := ipd.Lookup(ip)
	if err != nil {
		log.Fatal(err)
	}
	return Ivan{
		IP:            data.IP,
		IsEU:          data.IsEU,
		City:          data.City,
		Region:        data.Region,
		RegionCode:    data.RegionCode,
		CountryName:   data.CountryName,
		CountryCode:   data.CountryCode,
		ContinentName: data.ContinentName,
		ContinentCode: data.ContinentCode,
		Latitude:      data.Latitude,
		Longitude:     data.Longitude,
		Postal:        data.Postal,
		CallingCode:   data.CallingCode,
		Flag:          data.Flag,
		EmojiFlag:     data.EmojiFlag,
		EmojiUnicode:  data.EmojiUnicode,
	}
}

func recordMetrics(ivan Ivan, username string) {
	failedLogins.With(prometheus.Labels{
		"ip":             ivan.IP,
		"username":       username,
		"city":           ivan.City,
		"region":         ivan.Region,
		"region_code":    ivan.RegionCode,
		"country":        ivan.CountryName,
		"country_code":   ivan.CountryCode,
		"continent":      ivan.ContinentName,
		"continent_code": ivan.ContinentCode,
		"latitude":       fmt.Sprintf("%f", ivan.Latitude),
		"longitude":      fmt.Sprintf("%f", ivan.Longitude),
		"postal":         ivan.Postal,
		"calling_code":   ivan.CallingCode,
		"flag":           ivan.Flag,
		"emoji_flag":     ivan.EmojiFlag,
		"emoji_unicode":  ivan.EmojiUnicode,
	}).Inc()
}

type Ivan struct {
	IP            string  `json:"ip"`
	IsEU          bool    `json:"is_eu"`
	City          string  `json:"city"`
	Region        string  `json:"region"`
	RegionCode    string  `json:"region_code"`
	CountryName   string  `json:"country_name"`
	CountryCode   string  `json:"country_code"`
	ContinentName string  `json:"continent_name"`
	ContinentCode string  `json:"continent_code"`
	Latitude      float64 `json:"latitude"`
	Longitude     float64 `json:"longitude"`
	Postal        string  `json:"postal"`
	CallingCode   string  `json:"calling_code"`
	Flag          string  `json:"flag"`
	EmojiFlag     string  `json:"emoji_flag"`
	EmojiUnicode  string  `json:"emoji_unicode"`
}
