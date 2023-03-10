# ChatGPT Mastodon Proxy

[日本語バージョンはこちら](./README-JA.md)

Proxy application to make ChatGPT available in Mastodon's Mention

![example use case](https://raw.githubusercontent.com/kznrluk/chatgpt-mastodon/main/docs/preview.png)

## Hou to use
Environment variables must be set in advance.

**Docker Compose**
```
> docker compose up
```

**Docker Image**
```
> docker run -it kznrluk/chatgpt-mastodon
```

**Go Run**
```
> go run ./main.go
```

## Environment variables
```
# OpenAPI API key
OPENAI_API_KEY=sk-

# Mastodon application tokens
SERVER_URL=https://
CLIENT_KEY=aWNd
CLIENT_SECRET=fN6
ACCESS_TOKEN=XeF

# Bot account name
BOT_ACCOUNT_NAME=chatgpt
```

## Permissions required when creating an application
```
read:notifications read:statuses write:statuses push
```

## Notice
We are not responsible for any problems caused by the use of this application. Please be especially careful when using pay-as-you-go paid slots.

## License
MIT