# ChatGPT Mastodon Proxy

ChatGPTをMastodonのメンションで利用できるようにするプロキシアプリケーションです。

[試してみる (他インスタンスからのメンションもOK)](https://building7.social/@chatgpt)

![example use case](https://raw.githubusercontent.com/kznrluk/chatgpt-mastodon/main/docs/preview.png)

## 使い方
あらかじめ環境変数の設定が必要です。

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

## 環境変数の設定
```
# OpenAPI APIキー
OPENAI_API_KEY=sk-

# Mastodon 開発->新規アプリで取得するトークン等
SERVER_URL=https://
CLIENT_KEY=aWNd
CLIENT_SECRET=fN6
ACCESS_TOKEN=XeF

# Botアカウント名
BOT_ACCOUNT_NAME=chatgpt
```

## アプリ作成時に必要なロール
```
read:notifications read:statuses write:statuses push
```

## 免責
このアプリケーションを利用したことによるトラブルの責任は一切負いかねます。特に従量課金の有料枠の利用はご注意ください。

## License
MIT