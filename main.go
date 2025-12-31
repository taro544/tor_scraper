package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
	"golang.org/x/net/proxy"
)

// SETTINGS
const (
	TorProxyServer = "socks5://127.0.0.1:9050"
	TargetsFile    = "targets.yaml"
	BaseOutputDir  = "scraped_data"
	WorkerCount    = 5 // 5 threads
)

func main() {

	fmt.Println("\n[CHECK] Checking Tor connection...")
	if !checkTorConnection() {
		fmt.Println(" NOT CONNECTED TO TOR! Scan cannot start.")
		return
	}
	fmt.Println(" Tor connection successful! Starting scan...")

	folders := []string{"htmls", "images", "urls"}
	for _, f := range folders {
		path := fmt.Sprintf("%s/%s", BaseOutputDir, f)
		if err := os.MkdirAll(path, 0755); err != nil {
			log.Fatal("Failed to create folder:", err)
		}
	}

	targets, err := readTargets(TargetsFile)
	if err != nil {
		log.Fatal("Failed to read targets file:", err)
	}
	fmt.Printf("[INIT] %d targets loaded.\n", len(targets))

	jobs := make(chan string, len(targets))
	var wg sync.WaitGroup

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ProxyServer(TorProxyServer),
		chromedp.Flag("headless", true),
		chromedp.IgnoreCertErrors,
		chromedp.WindowSize(1920, 1080),
	)
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	for i := 1; i <= WorkerCount; i++ {
		wg.Add(1)
		go worker(i, allocCtx, jobs, &wg)
	}

	for _, url := range targets {
		jobs <- url
	}
	close(jobs)

	wg.Wait()
	fmt.Println("\nDone!")
}

func worker(id int, parentCtx context.Context, jobs <-chan string, wg *sync.WaitGroup) {
	defer wg.Done()

	for url := range jobs {
		err := processURL(parentCtx, url)

		if err != nil {
			fmt.Printf("ERROR: %s (%v)\n", url, err)
			appendLog(url, "FAIL", err.Error())
		} else {
			fmt.Printf("COMPLETED: %s\n", url)
			appendLog(url, "SUCCESS", "Saved HTML, IMG, URLs")
		}
	}
}

func processURL(parentCtx context.Context, targetURL string) error {
	if !strings.HasPrefix(targetURL, "http") {
		targetURL = "http://" + targetURL
	}

	ctx, cancel := chromedp.NewContext(parentCtx)
	ctx, cancel = context.WithTimeout(ctx, 40*time.Second) // 40s timeout
	defer cancel()

	var imageBuf []byte
	var htmlContent string
	var links []string

	err := chromedp.Run(ctx,
		chromedp.Navigate(targetURL),
		chromedp.WaitVisible("body", chromedp.ByQuery),
		chromedp.Evaluate(`Array.from(document.querySelectorAll('a')).map(a => a.href)`, &links),
		chromedp.OuterHTML("html", &htmlContent),
		chromedp.FullScreenshot(&imageBuf, 90),
	)

	if err != nil {
		return err
	}

	safeName := generateFilename(targetURL)

	htmlPath := fmt.Sprintf("%s/htmls/%s.html", BaseOutputDir, safeName)
	if err := os.WriteFile(htmlPath, []byte(htmlContent), 0644); err != nil {
		return err
	}

	imgPath := fmt.Sprintf("%s/images/%s.png", BaseOutputDir, safeName)
	if err := os.WriteFile(imgPath, imageBuf, 0644); err != nil {
		return err
	}

	linksPath := fmt.Sprintf("%s/urls/%s_links.txt", BaseOutputDir, safeName)
	linkContent := strings.Join(links, "\n")
	if err := os.WriteFile(linksPath, []byte(linkContent), 0644); err != nil {
		return err
	}

	return nil
}

func readTargets(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var urls []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "urls:") {
			continue
		}
		line = strings.TrimPrefix(line, "- ")
		if line != "" {
			urls = append(urls, line)
		}
	}
	return urls, scanner.Err()
}

func appendLog(url, status, msg string) {
	f, err := os.OpenFile("scan_report.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	logLine := fmt.Sprintf("[%s] %s -> %s (%s)\n", time.Now().Format("15:04:05"), url, status, msg)
	f.WriteString(logLine)
}

func generateFilename(url string) string {
	safe := strings.ReplaceAll(url, "http://", "")
	safe = strings.ReplaceAll(safe, "https://", "")
	safe = strings.ReplaceAll(safe, ".onion", "")
	safe = strings.ReplaceAll(safe, "/", "_")

	return safe
}

func checkTorConnection() bool {
	dialer, err := proxy.SOCKS5("tcp", "127.0.0.1:9050", nil, proxy.Direct)
	if err != nil {
		fmt.Printf("Proxy setup error: %v\n", err)
		return false
	}

	transport := &http.Transport{
		Dial: dialer.Dial,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   15 * time.Second,
	}

	resp, err := client.Get("https://check.torproject.org")
	if err != nil {
		fmt.Printf("Tor connection error: %v\n", err)
		return false
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false
	}

	bodyStr := string(body)
	if strings.Contains(bodyStr, "Sorry. You are not using Tor") {
		return false
	}

	return strings.Contains(bodyStr, "Congratulations") || strings.Contains(bodyStr, "successfully")
}
