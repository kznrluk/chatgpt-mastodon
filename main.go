package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/joho/godotenv"
	"github.com/mattn/go-mastodon"
	"github.com/sashabaranov/go-openai"
	"golang.org/x/net/html"
	"os"
	"os/signal"
	"strings"
)

func textContent(s string) string {
	doc, err := html.Parse(strings.NewReader(s))
	if err != nil {
		return s
	}
	var buf bytes.Buffer

	var extractText func(node *html.Node, w *bytes.Buffer)
	extractText = func(node *html.Node, w *bytes.Buffer) {
		if node.Type == html.TextNode {
			data := strings.Trim(node.Data, "\r\n")
			if data != "" {
				w.WriteString(data)
			}
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			extractText(c, w)
		}
		if node.Type == html.ElementNode {
			name := strings.ToLower(node.Data)
			if name == "br" {
				w.WriteString("\n")
			}
		}
	}
	extractText(doc, &buf)

	str := buf.String()
	spl := strings.SplitN(str, " ", 2)
	if len(spl) == 1 {
		return spl[0]
	}

	return spl[1]
}

func connect() (*mastodon.Client, *openai.Client, error) {
	fmt.Print("Connecting to mastodon server " + os.Getenv("SERVER_URL") + " ... ")
	mc := mastodon.NewClient(&mastodon.Config{
		Server:       os.Getenv("SERVER_URL"),
		ClientID:     os.Getenv("CLIENT_KEY"),
		ClientSecret: os.Getenv("CLIENT_SECRET"),
		AccessToken:  os.Getenv("ACCESS_TOKEN"),
	})

	if _, err := mc.GetTimelineHome(context.Background(), nil); err != nil {
		return &mastodon.Client{}, &openai.Client{}, err
	}

	fmt.Println("OK")
	fmt.Print("Connecting to OpenAI server ... ")

	oc := openai.NewClient(os.Getenv("OPENAI_API_KEY"))
	_, err := oc.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: "Ping",
				},
			},
		},
	)

	if err != nil {
		return &mastodon.Client{}, &openai.Client{}, err
	}

	fmt.Println("OK")
	return mc, oc, nil
}

func escapeSpecialCharacter(str string) string {
	str = strings.ReplaceAll(str, "@", "[at]")
	str = strings.ReplaceAll(str, "http://", "[http://]")
	str = strings.ReplaceAll(str, "https://", "[https://]")
	return str
}

func main() {
	err := godotenv.Load()
	if err != nil {
		panic("Error loading .env file")
	}

	if err != nil {
		panic(err)
	}

	mc, oc, err := connect()
	if err != nil {
		panic(err)
	}

	//
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var q chan mastodon.Event
	q, err = mc.StreamingUser(ctx)
	if err != nil {
		panic(err.Error())
	}

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, os.Interrupt)
	go func() {
		<-sc
		cancel()
	}()

	for e := range q {
		switch t := e.(type) {
		case *mastodon.UpdateEvent:
			// spew.Dump(&t.Status)
		case *mastodon.NotificationEvent:
			if t.Notification.Type != "mention" {
				continue
			}

			if len(t.Notification.Status.Mentions) >= 2 {
				continue
			}

			acct := t.Notification.Account.Acct
			sc, err := mc.GetStatusContext(context.Background(), t.Notification.Status.ID)
			if err != nil {
				mc.PostStatus(context.Background(), &mastodon.Toot{
					Status:      fmt.Sprintf("%s Error!", acct),
					InReplyToID: t.Notification.Status.ID,
				})
				fmt.Printf(err.Error())
				continue
			}

			var ctx []openai.ChatCompletionMessage
			ctx = append(ctx, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleSystem,
				Content: os.Getenv("SYSTEM_CONTEXT"),
			})
			for _, r := range sc.Ancestors {
				role := openai.ChatMessageRoleUser
				if r.Account.Username == os.Getenv("BOT_ACCOUNT_NAME") {
					role = openai.ChatMessageRoleAssistant
				}

				ctx = append(ctx, openai.ChatCompletionMessage{
					Role:    role,
					Content: textContent(r.Content),
				})
			}

			ctx = append(ctx, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: textContent(t.Notification.Status.Content),
			})

			fmt.Printf("%-24s: %v\n", acct, ctx[1:])

			resp, err := oc.CreateChatCompletion(
				context.Background(),
				openai.ChatCompletionRequest{
					Model:    openai.GPT3Dot5Turbo,
					Messages: ctx,
				},
			)

			if err != nil {
				mc.PostStatus(context.Background(), &mastodon.Toot{
					Status:      fmt.Sprintf("%s Error!", acct),
					InReplyToID: t.Notification.Status.ID,
				})
				fmt.Printf(err.Error())
				continue
			}

			message := escapeSpecialCharacter(resp.Choices[0].Message.Content)
			fmt.Printf("%-24s: %s\n", "=> ASSISTANT", strings.ReplaceAll(message, "\n", "\\n"))
			mc.PostStatus(context.Background(), &mastodon.Toot{
				Status:      fmt.Sprintf("@%s %s", acct, message),
				InReplyToID: t.Notification.Status.ID,
			})
			continue
		case *mastodon.ErrorEvent:
			//spew.Dump(t.Error())
		}
	}
}
