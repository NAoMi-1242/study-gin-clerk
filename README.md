# Go + Gin + Clerk 認証 & AI チャット基盤 API

本プロジェクトは、**Go (Gin)** と **Clerk** による認証基盤の上に、**GORM (PostgreSQL)** および **GoAI (マルチプロバイダ LLM: OpenRouter, OpenAI, Claude, Gemini)** を統合した「AI チャットアプリケーション」のバックエンド API サーバーです。

実務で耐えうる **「ドメイン凝集」「サービス間疎結合（DIP）」「高テスタビリティ」「コンパイル時 DI（Google Wire）」** を備えたアーキテクチャを採用しています。

---

## 目次

- [1. 技術スタック](#1-技術スタック)
- [2. システムアーキテクチャ & データフロー](#2-システムアーキテクチャ--データフロー)
- [3. ディレクトリ構成](#3-ディレクトリ構成)
- [4. これまでの設計判断と進化のプロセス（ADR）](#4-これまでの設計判断と進化のプロセスadr)
- [5. ローカル開発環境のセットアップ](#5-ローカル開発環境のセットアップ)
- [6. テスト・運用コマンド集](#6-テスト運用コマンド集)

---

## 1. 技術スタック

| カテゴリ                 | 採用技術                                                    | バージョン / 用途                             |
| :----------------------- | :---------------------------------------------------------- | :-------------------------------------------- |
| **Language**             | [Go](https://go.dev/)                                       | 1.26 (Alpine コンテナ)                        |
| **Web Framework**        | [Gin](https://github.com/gin-gonic/gin)                     | v1.12.0 (ルーティング・SSE ストリーミング)    |
| **Authentication**       | [Clerk Go SDK](https://github.com/clerk/clerk-sdk-go/v2)    | v2.7.0 (JWT 検証・セッション管理)             |
| **ORM / Database**       | [GORM](https://gorm.io/) / PostgreSQL                       | v1.31.2 / Postgres 16 (コネクションプール)    |
| **Migration**            | [golang-migrate](https://github.com/golang-migrate/migrate) | v4.18.3 (SQL マイグレーション)                |
| **AI Client**            | [GoAI](https://github.com/zendev-sh/goai)                   | v0.10.2 (マルチプロバイダ SDK)                |
| **Encryption**           | Standard crypto/cipher                                      | AES-256-GCM (ユーザー API キーの可逆暗号化)   |
| **Dependency Injection** | [Google Wire](https://github.com/google/wire)               | v0.7.0 (コンパイル時コード生成型 DI)          |
| **API Documentation**    | [swaggo/gin-swagger](https://github.com/swaggo/gin-swagger) | v1.6.1 / swag v1.16.4 (Swagger UI)            |
| **Live Reload**          | [Air](https://github.com/air-verse/air)                     | v1.63.4 (コンテナ内ホットリロード)            |
| **Container**            | Docker / Docker Compose                                     | PostgreSQL 16 + Go API + Nginx Web (テストUI) |

---

## 2. システムアーキテクチャ & データフロー

システムは自然な境界に基づく機能別（Package-by-feature）で設計されており、**「サービス間依存ゼロ（ハンドラー調停・直接引数渡し）」** を徹底しています。

```mermaid
graph TD
    Client["クライアント (Web / Swagger UI)"] -->|HTTP / SSE| Router["Router (routes.go)"]
    Router -->|Authorization Header| AuthMW["auth.RequireAuth() (middleware.go)"]

    subgraph "防腐層 (ACL) & 認証"
        AuthMW -->|JWT検証| ClerkSDK["Clerk Go SDK"]
        AuthMW -->|User ID抽出| AuthCtx["internal/auth (Context操作)"]
    end

    AuthMW -->|c.Next| Handlers["Domain Handlers (chat / user / health)"]
    Handlers -->|MustGetUserID| AuthCtx

    subgraph "ハンドラー調停 & ドメイン層"
        ChatH["chat.Handler"] -->|1. Key/Prompt取得| UserSvc["user.Service"]
        ChatH -->|2. 引数で直接渡す| ChatSvc["chat.Service"]
        UserH["user.Handler"] --> UserSvc
    end

    subgraph "データ永続化 & インフラ"
        ChatSvc --> ChatRepo["chat.Repository"]
        UserSvc --> UserRepo["user.Repository"]
        ChatSvc --> AIClient["infra/ai.Client (GoAI)"]
        UserSvc --> AESCipher["infra/crypto.AESCipher"]
        UserSvc --> AIRegistry["infra/ai.ModelRegistry"]
    end

    ChatRepo -->|SQL実行| DBClient["*gorm.DB (コネクションプール)"]
    UserRepo -->|SQL実行| DBClient
    DBClient -->|TCP接続| Database[("PostgreSQL 16")]
    AIClient -->|HTTPS| AIProviders["AI 各社 API (OpenRouter / OpenAI / Anthropic / Google)"]
```

---

## 3. ディレクトリ構成

```
study-gin-clerk/
  ├── Dockerfile                    # Go 1.26 + Air + Wire + Swag + golang-migrate
  ├── compose.yaml                  # Docker Compose 定義 (Go API + PostgreSQL 16 + Web Nginx)
  ├── .air.toml                     # ホットリロード設定
  ├── ARCHITECTURE.md               # アーキテクチャ完全解説ガイド
  ├── wire-manual.md                # Google Wire 運用マニュアル
  ├── migrations/                   # golang-migrate SQL マイグレーション
  │    ├── 000001_create_initial_tables.up.sql / down.sql
  │    └── 000002_add_system_prompt.up.sql / down.sql
  ├── cmd/
  │    └── api/
  │         ├── main.go             # エントリポイント (Graceful Shutdown & InitializeApp 実行)
  │         ├── wire.go             # Wire Injector 宣言
  │         └── wire_gen.go         # Wire 自動生成コード
  ├── docs/                         # Swagger 自動生成ドキュメント (docs.go, swagger.json, swagger.yaml)
  ├── internal/
  │    ├── auth/                    # 認証コンテキスト & Clerk 認証ミドルウェア (RequireAuth)
  │    ├── config/                  # 環境変数読み込み・バリデーション (CORS, AES鍵, DSN)
  │    ├── router/                  # ルーティング設定・CORS・Swagger エンドポイント
  │    ├── types/                   # 共通ドメイン値オブジェクト (Provider, AIModel)
  │    ├── health/                  # 【ヘルスチェック】(DB Ping 疎通確認)
  │    ├── user/                    # 【ユーザー領域】(プロファイル, システムプロンプト, 暗号化APIキー管理)
  │    ├── chat/                    # 【チャット領域】(会話・メッセージ, SSEストリーミング, モデル一覧)
  │    └── infra/                   # 【インフラ層】
  │         ├── ai/                 # GoAI クライアント, 各社モデルフェッチャー (Strategy), MemoryCache
  │         ├── crypto/             # AES-256-GCM 暗号化 / 復号化
  │         └── db/                 # GORM 接続・コネクションプール管理
  └── web/
       ├── nginx.conf               # 仮フロントエンド配信 Nginx 設定 (動的環境変数配信)
       └── index.html               # 動作検証用フロントエンド (Clerk JS 連携 & SSE チャット UI)
```

---

## 4. これまでの設計判断と進化のプロセス（ADR）

### 4.1. ドメインの再集約（5ドメイン構成への最適化）

- **背景**: 過剰な細分化（`profile`, `apikey`, `aimodel`, `chat` などへの分断）により、1ファイルしかないパッケージや無駄なプロバイダインターフェースが多発していた。
- **解決策**: 関連性の高い機能を統合し、ユーザー設定・キー管理を `internal/user`、会話とモデル取得を `internal/chat`、認証を `internal/auth` に集約。

### 4.2. サービス間ゼロ結合（アプローチA: 直接引数渡し）

- **背景**: `chat.Service` が `user.Service`（またはそのインターフェース）に依存し、サービス同士が密結合していた。
- **解決策**: `chat.Handler` が `user.Service` から API キーとシステムプロンプトを取得し、`chat.Service` の引数として直接渡す設計に変更。`chat.Service` の純粋化とテスト容易性を最大化。

### 4.3. Clerk SDK の防腐層（ACL）カプセル化

- Clerk SDK の呼び出しを `internal/auth/middleware.go` の `RequireAuth()` に完全隔離。後続のハンドラーやドメイン層は純粋な `user_id`（string）のみを扱う。

### 4.4. 依存関係逆転の原則（DIP）とコンパイル時 DI（Google Wire）

- 外部技術（DB, 暗号化, AI SDK）は各ドメインが必要とするインターフェースを満たす形で注入。コンパイル時に依存関係を安全に解決。

---

## 5. ローカル開発環境のセットアップ

### 前提条件

- Docker & Docker Compose
- [Clerk](https://clerk.com/) アカウント（API Keys）

### 1. 環境変数の準備

プロジェクト直下に `.env` を作成します：

```env
PORT=8080
APP_URL=http://localhost:3000
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:8080

CLERK_SECRET_KEY=sk_test_xxxxxxxxxxxxxxxxxxxxx
CLERK_PUBLISHABLE_KEY=pk_test_xxxxxxxxxxxxxxxxxxxxx

DATABASE_URL=postgres://postgres:password@db:5432/study_app?sslmode=disable

# AES-256-GCM 暗号化マスターキー (32バイト / 64文字の16進数文字列)
ENCRYPTION_KEY=03eb48bf7ba4f88a8bc7f269a114cd5b71c89aeb086dec25f6f58e5c09e3303f
```

### 2. コンテナの起動

```bash
docker compose up -d --build
```

### 3. 動作確認

- **動作検証用 Web UI**: `http://localhost:3000/` (Clerk ログイン・APIキー登録・SSE チャット送受信)
- **Swagger UI**: `http://localhost:8080/swagger/index.html`
- **Health Check**: `curl http://localhost:8080/health`
  ```json
  {
    "status": "ok",
    "database": "connected"
  }
  ```

---

## 6. テスト・運用コマンド集

すべてのコマンドは **`api` コンテナ内** で実行します。

### 単体テストの実行

```bash
docker compose exec api go test -v -count=1 ./...
```

### Wire コードの再生成

コンストラクタや依存関係を変更した際：

```bash
docker compose exec api wire gen ./cmd/api
docker compose exec api wire check ./cmd/api
```

### Swagger ドキュメントの再生成

API コメントを変更した際：

```bash
docker compose exec api swag init -g cmd/api/main.go -o docs
```

### DB マイグレーション操作

```bash
# マイグレーション適用
docker compose exec api migrate -path migrations -database "$DATABASE_URL" up

# ロールバック (1ステップ)
docker compose exec api migrate -path migrations -database "$DATABASE_URL" down 1
```
