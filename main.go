package main

import (
	"errors"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"golang.org/x/net/html"
	"log"
	"net/http"
	"strings"
	"time"
)


const site = "https://o.yandex.ru/?sorting=ByPublishDateDesc&text=nintendo%20switch&region_id=213"

func getAttribute(node *html.Node, name string) (string, error) {
	for _, attr := range node.Attr {
		if attr.Key == name {
			return attr.Val, nil
		}
	}
	return "", errors.New("No such attribute")
}

func getClassNames(node *html.Node) []string {
	val, err := getAttribute(node, "class")
	if err != nil {
		return nil
	}
	return strings.Split(val, " ")
}

func hasClassName(node *html.Node, name string) bool {
	names := getClassNames(node)
	for _, n := range names {
		if n == name {
			return true
		}
	}
	return false
}

func getElementsByClassName(node *html.Node, name string) []*html.Node {
	var nodes []*html.Node
	if hasClassName(node, name) {
		nodes = append(nodes, node)
	}
	for c := node.FirstChild; c != nil; c = c.NextSibling {
		cn := getElementsByClassName(c, name)
		nodes = append(nodes, cn...)
	}
	return nodes
}

func getElementsByType(node *html.Node, name string) []*html.Node {
	var nodes []*html.Node
	if node.Data == name {
		nodes = append(nodes, node)
	}
	for c := node.FirstChild; c != nil; c = c.NextSibling {
		cn := getElementsByType(c, name)
		nodes = append(nodes, cn...)
	}
	return nodes
}

type item struct {
	Title string
	Preview *preview
	Price string
	Link string
}

type preview struct {
	Url string
}

func parsePreview(node *html.Node) *preview {
	imgs := getElementsByType(node, "img")
	src, _ := getAttribute(imgs[0], "src")
	return &preview{
		Url: src,
	}
}

func parseMainPage(document *html.Node) []item {
	blocks := getElementsByClassName(document, "ListingSnippetView__wrapper__384Rc")
	var res []item
	for _, b := range blocks {
		previewBlock := getElementsByClassName(b, "Image__image__GUPbu")
		var prev *preview
		if len(previewBlock) > 0 {
			prev = parsePreview(previewBlock[0])
		}
		headerBlock := getElementsByClassName(b, "Text__subText__qug9u")[0]
		priceBlock := getElementsByClassName(b, "Text__textBold__zEuah")[0]
		link, _ := getAttribute(getElementsByClassName(b, "ListingSnippetView__link__18Lpo")[0], "href")
		i := item{
			Title: headerBlock.FirstChild.Data,
			Price: priceBlock.FirstChild.Data,
			Preview: prev,
			Link: "https://o.yandex.ru" + link,
		}
		res = append(res, i)
	}
	return res
}

func main() {
	bot, err := tgbotapi.NewBotAPI("1625195767:AAFeRBI_PRFaLHEpuBQRViXpu_zD7qYl8rA")
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 5

	updates, err := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

		if update.Message.Text == "/start" {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Start...")
			bot.Send(msg)
			p, err := http.Get(site)
			if err != nil {
				fmt.Println("Internet connection unstable: ", err)
			} else {
				doc, _ := html.Parse(p.Body)
				items := parseMainPage(doc)
				last_link := items[0].Link
				ticker := time.NewTicker(time.Millisecond * 5000)
				go func() {
					for t := range ticker.C {
						t = t
						p1, err1 := http.Get(site)
						if err1 != nil {
							fmt.Println("Internet connection unstable: ", err)
						} else {
							doc1, _ := html.Parse(p1.Body)
							items1 := parseMainPage(doc1)
							new_link := items1[0].Link
							if last_link != new_link {
								Text := items1[0].Title + "\n" + items1[0].Price + "\n" + new_link;
								msg := tgbotapi.NewMessage(update.Message.Chat.ID, Text)
								bot.Send(msg)
								last_link = new_link
							}
						}
					}
				}()
			}
		}
	}
}