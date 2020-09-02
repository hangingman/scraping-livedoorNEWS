package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/sclevine/agouti"
)

const sleepTime = 5
const url = "https://news.livedoor.com/topics/category/dom/"

func replace(t string) string {
	r := strings.NewReplacer("\n", "", ",", "、")
	return r.Replace(t)
}

func main() {
	flag.Parse()
	arg := flag.Arg(0)

	file, err := os.OpenFile(arg, os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open file: %v", err)
		os.Exit(1)
	}
	defer file.Close()

	err = file.Truncate(0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to clear file: %v", err)
		os.Exit(1)
	}

	writer := csv.NewWriter(file)
	writer.Write([]string{"id", "title", "body", "summary1", "summary2", "summary3"})
	writer.Flush()

	driver := agouti.ChromeDriver(
		agouti.ChromeOptions("args", []string{
			"--headless",
			"--blink-settings=imagesEnabled=false", // don't load images
			"--disable-gpu",                        // ref: https://developers.google.com/web/updates/2017/04/headless-chrome#cli
			"no-sandbox",                           // ref: https://github.com/theintern/intern/issues/878
			"disable-dev-shm-usage",                // ref: https://qiita.com/yoshi10321/items/8b7e6ed2c2c15c3344c6
		}),
		agouti.Debug,
	)

	err = driver.Start()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start driver: %v", err)
		os.Exit(1)
	}
	defer driver.Stop()

	page, err := driver.NewPage(agouti.Browser("chrome"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open page: %v", err)
		os.Exit(1)
	}

	err = page.Navigate(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to navigate: %v", err)
		os.Exit(1)
	}

	curContentsDom, err := page.HTML()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get html: %v", err)
		os.Exit(1)
	}

	readerCurContents := strings.NewReader(curContentsDom)
	contentsDom, _ := goquery.NewDocumentFromReader(readerCurContents)

	var visitedIds []int
	var dataCount int

	for {
		listDom := contentsDom.Find(".articleList").Children()
		listLen := listDom.Length()

		// 記事のリストをぐるぐる
	L:
		for i := 1; i <= listLen; i++ {
			aSelector := ".articleList > li:nth-child(" + strconv.Itoa(i) + ") > a"

			// 記事のhref取得
			href, err := page.Find(aSelector).Attribute("href")
			if err != nil {
				fmt.Fprintln(os.Stderr, fmt.Sprintf("[error] 記事のhref取得 %v", err))
				continue
			}

			// hrefから記事idを取得
			id, err := strconv.Atoi(path.Base(href))
			if err != nil {
				fmt.Fprintln(os.Stderr, fmt.Sprintf("[error] hrefから記事idを取得 %v", err))
				continue
			}

			// すでに取得済みのidであれば処理を飛ばす
			for _, v := range visitedIds {
				if id == v {
					fmt.Fprintln(os.Stdout, fmt.Sprintf("訪問済です, 記事id: %s", id))
					continue L
				}
			}

			// 訪問済にする
			visitedIds = append(visitedIds, id)

			// 記事のタイトルと要約へ
			if err = page.Find(aSelector).Click(); err != nil {
				fmt.Fprintln(os.Stderr, fmt.Sprintf("[error] 記事のタイトルと要約へ %v", err))
				continue
			}
			time.Sleep(sleepTime * time.Second)

			// タイトル取得
			articleTitle, err := page.Find(".topicsTtl > a").Text()
			if err != nil {
				fmt.Fprintln(os.Stderr, fmt.Sprintf("[error] 記事のタイトル取得 %v", err))
				page.Back()
				time.Sleep(sleepTime * time.Second)
				continue
			}

			// 要約取得
			summary, err := page.FindByClass("summaryList").Text()
			if err != nil {
				fmt.Fprintln(os.Stderr, fmt.Sprintf("[error] 記事の要約取得 %v", err))
				page.Back()
				time.Sleep(sleepTime * time.Second)
				continue
			}

			// 記事の本文へ
			if err = page.Find(".articleMore > a").Click(); err != nil {
				fmt.Fprintln(os.Stderr, fmt.Sprintf("[error] 記事の本文へ %v", err))
				page.Back()
				time.Sleep(sleepTime * time.Second)
				continue
			}

			// 記事の本文取得
			articleBody, err := page.Find(".articleBody > span").Text()
			if err != nil {
				fmt.Fprintln(os.Stderr, fmt.Sprintf("[error] 記事の本文取得 %v", err))
				page.Back()
				time.Sleep(sleepTime * time.Second)
				page.Back()
				time.Sleep(sleepTime * time.Second)
				continue
			}

			// csvに書き込み
			summaryList := strings.Split(summary, "\n")
			writer.Write([]string{
				strconv.Itoa(id),
				replace(articleTitle),
				replace(articleBody),
				replace(summaryList[0]),
				replace(summaryList[1]),
				replace(summaryList[2]),
			})
			writer.Flush()
			dataCount++
			fmt.Printf("現在 %v 個の記事を取得済みです\n", dataCount)

			// 本文 -> タイトル＆要約
			page.Back()
			time.Sleep(sleepTime * time.Second)

			// タイトル＆要約 -> 記事リスト
			page.Back()
			time.Sleep(sleepTime * time.Second)
		}

		// 次の記事リストへ
		if err = page.Find(".next > a").Click(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			break
		}
		time.Sleep(sleepTime * time.Second)
	}
}
