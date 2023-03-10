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

var initProp = "Hello, you, the AI, will be asked to reply to a user's inquiry on a social networking site this time. There is a limit of 500 characters in a post, so you need to keep it under that. It is also best to omit redundant explanations, as long posts will occupy the timeline. Let's be friendly."

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

func main() {
	err := godotenv.Load()
	if err != nil {
		panic("Error loading .env file")
	}

	println("start")

	mc := mastodon.NewClient(&mastodon.Config{
		Server:       os.Getenv("SERVER_URL"),
		ClientID:     os.Getenv("CLIENT_KEY"),
		ClientSecret: os.Getenv("CLIENT_SECRET"),
		AccessToken:  os.Getenv("ACCESS_TOKEN"),
	})

	oc := openai.NewClient(os.Getenv("OPENAI_API_KEY"))
	_, err = oc.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: "Hello!",
				},
			},
		},
	)
	if err != nil {
		panic(err)
	}

	_, err = mc.GetTimelineHome(context.Background(), nil)
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
			username := t.Notification.Account.Username
			toot := textContent(t.Notification.Status.Content)

			fmt.Printf("%-16s: %s\n", username, toot)

			sc, err := mc.GetStatusContext(context.Background(), t.Notification.Status.ID)
			if err != nil {
				mc.PostStatus(context.Background(), &mastodon.Toot{
					Status:      fmt.Sprintf("%s Error!", username),
					InReplyToID: t.Notification.Status.ID,
				})
				fmt.Printf(err.Error())
				continue
			}

			var ctx []openai.ChatCompletionMessage
			ctx = append(ctx, openai.ChatCompletionMessage{
				Role:    "system",
				Content: initProp,
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

			fmt.Printf("%-16s: %v", "CONTEXT: ", ctx)

			ctx = append(ctx, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: textContent(t.Notification.Status.Content),
			})

			resp, err := oc.CreateChatCompletion(
				context.Background(),
				openai.ChatCompletionRequest{
					Model:    openai.GPT3Dot5Turbo,
					Messages: ctx,
				},
			)

			if err != nil {
				mc.PostStatus(context.Background(), &mastodon.Toot{
					Status:      fmt.Sprintf("%s Error!", username),
					InReplyToID: t.Notification.Status.ID,
				})
				fmt.Printf(err.Error())
				continue
			}

			fmt.Printf("%-16s: %s\n", "=> ASSISTANT:", strings.ReplaceAll(resp.Choices[0].Message.Content, "\n", "\\n"))
			mc.PostStatus(context.Background(), &mastodon.Toot{
				Status:      fmt.Sprintf("@%s %s", username, resp.Choices[0].Message.Content),
				InReplyToID: t.Notification.Status.ID,
			})
			continue
		case *mastodon.ErrorEvent:
			//spew.Dump(t.Error())
		}
	}
}
