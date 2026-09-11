# Google Wire 運用マニュアル

本書は、当プロジェクトにおける **Google Wire（コンパイル時 依存性注入 / DI ツール）** の基本概念、現在の構成、今後の拡張手順、および注意点をまとめたマニュアルです。

---

## 目次

1. [Google Wire の基本概念と仕組み](#1-google-wire-の基本概念と仕組み)
2. [当プロジェクトでのセットアップ構成](#2-当プロジェクトでのセットアップ構成)
3. [新しいコンポーネントを追加する手順（実践ガイド）](#3-新しいコンポーネントを追加する手順実践ガイド)
4. [今後の拡張例（GORM + Supabase + GoAI）](#4-今後の拡張例gorm--supabase--goai)
5. [知っておくべき注意点・ベストプラクティス](#5-知っておくべき注意点ベストプラクティス)
6. [よく使うコマンド集](#6-よく使うコマンド集)

---

## 1. Google Wire の基本概念と仕組み

### なぜ Wire を使うのか？

Go にはリフレクションを用いた実行時 DI（例: Uber Dig や Spring のようなもの）もありますが、Google Wire は **「コンパイル前にコード生成によって依存関係を解決する」** アプローチを採用しています。

- **完全な型安全性**: 依存の渡し忘れや循環参照があれば、ビルド前（コード生成時）にエラーで検知できます。
- **ゼロ・オーバーヘッド**: 生成されるのは純粋な Go コード（`wire_gen.go`）のため、リフレクション不要で最速で動作します。
- **可読性**: 生成コードを直接確認・デバッグでき、ブラックボックスがありません。

### 3 つの基本要素

```
┌─────────────────┐       ┌─────────────────┐       ┌─────────────────┐
│    Provider     │  ──>  │   ProviderSet   │  ──>  │    Injector     │
│  (部品を作る関数) │       │   (部品のまとめ)  │       │  (完成品を作る関数) │
└─────────────────┘       └─────────────────┘       └─────────────────┘
```

1. **Provider（プロバイダ）**:
   通常のコンストラクタ関数（例: `NewUserHandler() *UserHandler`）。特別なライブラリは不要で、Go の標準的な書き方をするだけです。
2. **ProviderSet（プロバイダセット）**:
   `wire.NewSet(...)` で関連する Provider 群を 1 つに束ねたもの（例: `handler.Set`, `router.Set`）。
3. **Injector（インジェクタ）**:
   `cmd/api/wire.go` に記述する「最終的に何が欲しいか」を宣言する関数（例: `InitializeApp() (*gin.Engine, error)`）。`wire.Build(...)` を使って ProviderSet を渡します。

---

## 2. 当プロジェクトでのセットアップ構成

### ディレクトリとファイル構成

```
study-gin-clerk/
  ├── Dockerfile                    # wire CLI がインストールされている
  ├── cmd/
  │    └── api/
  │         ├── main.go             # InitializeApp() を呼ぶだけ
  │         ├── wire.go             # 【手動編集】Wire の設計図 (Injector)
  │         └── wire_gen.go         # 【自動生成】Wire が生成した初期化コード
  └── internal/
       ├── handler/
       │    ├── provider.go         # Handler 全体の ProviderSet (handler.Set)
       │    ├── health.go
       │    └── user.go
       └── router/
            └── routes.go           # Router の ProviderSet (router.Set) と Dependencies
```

### 各ファイルの役割

#### ① `cmd/api/wire.go`（設計図）

```go
//go:build wireinject
// +build wireinject

package main

import (
	"github.com/gin-gonic/gin"
	"github.com/google/wire"

	"study-gin-clerk/internal/handler"
	"study-gin-clerk/internal/router"
)

func InitializeApp() (*gin.Engine, error) {
	wire.Build(
		handler.Set,
		router.Set,
	)
	return nil, nil
}
```

> **POINT**: 1行目の `//go:build wireinject` により、このファイルは通常の `go build` から除外されます。

#### ② `cmd/api/wire_gen.go`（自動生成コード）

`wire gen ./cmd/api` を叩くと自動生成されます。手動で編集してはいけません。

```go
func InitializeApp() (*gin.Engine, error) {
	healthHandler := handler.NewHealthHandler()
	userHandler := handler.NewUserHandler()
	dependencies := router.Dependencies{
		HealthHandler: healthHandler,
		UserHandler:   userHandler,
	}
	engine := router.New(dependencies)
	return engine, nil
}
```

#### ③ `cmd/api/main.go`（エントリポイント）

`main.go` は自動生成された `InitializeApp()` を呼ぶだけです。

```go
engine, err := InitializeApp()
if err != nil {
    log.Fatal(err)
}
```

---

## 3. 新しいコンポーネントを追加する手順（実践ガイド）

新しい Handler や Service を追加する際の基本サイクルは以下の **4ステップ** です。

### ステップ 1: 通常通りコンストラクタ（Provider）を実装する

例として、新しい `ChatHandler` を作成する場合：

```go
// internal/handler/chat.go
package handler

import "github.com/gin-gonic/gin"

type ChatHandler struct {
    // 将来 Service 層などがここに入ります
}

func NewChatHandler() *ChatHandler {
    return &ChatHandler{}
}

func (h *ChatHandler) SendMessage(c *gin.Context) {
    // ...
}
```

### ステップ 2: パッケージの `ProviderSet` に登録する

`internal/handler/provider.go` の `wire.NewSet` に新しいコンストラクタを追加します。

```go
// internal/handler/provider.go
var Set = wire.NewSet(
    NewHealthHandler,
    NewUserHandler,
    NewChatHandler, // ← 追加
)
```

### ステップ 3: 受け取り側の構造体（依存先）に追加する

ルーターが受け取れるように `internal/router/routes.go` に登録します。

```go
// internal/router/routes.go
type Dependencies struct {
    HealthHandler *handler.HealthHandler
    UserHandler   *handler.UserHandler
    ChatHandler   *handler.ChatHandler // ← 追加
}

func New(deps Dependencies) *gin.Engine {
    // ...
    api.POST("/chat/messages", deps.ChatHandler.SendMessage) // ← ルート紐付け
    return engine
}
```

### ステップ 4: Wire コマンドでコードを再生成する

ターミナルから以下のコマンドを実行します。

```bash
docker compose exec api wire gen ./cmd/api
```

これだけで `cmd/api/wire_gen.go` が自動更新され、`main.go` に一切手を触れることなく依存解決が完了します。

---

## 4. 今後の拡張例（GORM + Supabase + GoAI）

本番の「AIチャットアプリ」向けに、GORM・GoAI クライアント・Repository・Service 層を追加する際の設計イメージです。

### 構成イメージ

```
[Database (GORM)] ──> [ChatRepository] ──┐
                                         ├──> [ChatService] ──> [ChatHandler] ──> [Router]
[AI Client (GoAI)] ──────────────────────┘
```

### 各層の Provider 定義例

#### ① DB / AI クライアント層（`internal/infra/` など）

```go
// internal/infra/db.go
func NewGormDB(cfg config.Config) (*gorm.DB, error) {
    return gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{})
}

// internal/infra/ai.go
func NewAIClient(ctx context.Context, cfg config.Config) (*genai.Client, error) {
    return genai.NewClient(ctx, option.WithAPIKey(cfg.GeminiAPIKey))
}

var Set = wire.NewSet(NewGormDB, NewAIClient)
```

#### ② Repository 層（`internal/repository/`）

```go
// internal/repository/chat.go
type ChatRepository struct { db *gorm.DB }
func NewChatRepository(db *gorm.DB) *ChatRepository {
    return &ChatRepository{db: db}
}

var Set = wire.NewSet(NewChatRepository)
```

#### ③ Service 層（`internal/service/`）

```go
// internal/service/chat.go
type ChatService struct {
    repo *repository.ChatRepository
    ai   *genai.Client
}
func NewChatService(repo *repository.ChatRepository, ai *genai.Client) *ChatService {
    return &ChatService{repo: repo, ai: ai}
}

var Set = wire.NewSet(NewChatService)
```

#### ④ `cmd/api/wire.go` に登録

```go
func InitializeApp(ctx context.Context, cfg config.Config) (*gin.Engine, func(), error) {
    wire.Build(
        infra.Set,
        repository.Set,
        service.Set,
        handler.Set,
        router.Set,
    )
    return nil, nil, nil
}
```

> **POINT**: 各層の依存関係（誰が誰を必要としているか）を Wire が自動で逆算し、正しい順序で初期化コードを出力してくれます。

---

## 5. 知っておくべき注意点・ベストプラクティス

### 1. `wire_gen.go` を直接手動編集しない

- `wire_gen.go` は Wire コマンドで上書きされます。変更したい場合は元の Provider 関数や `wire.go` を修正し、`wire gen` を実行してください。
- `wire_gen.go` は **Git のコミット対象** に含めます（デプロイ先で Wire CLI を叩く必要をなくすため）。

### 2. 「同じ型」が複数ある場合の衝突回避

Wire は **型名（Type）** をキーにして依存関係をマッチングします。

- 例えば `string` 型の引数を2つ（`dbURL string`, `apiKey string`）そのまま Provider に渡すと、Wire は「どちらの string を渡せばいいのか」分からずエラーになります。
- **対策**: `config.Config` 構造体ごと渡すか、`type DBURL string` のように専用の型（エイリアス）を定義して区別します。

### 3. インターフェースを使う場合（`wire.Bind`）

Handler が具体的な構造体ではなく `interface` に依存している場合、Wire に「どの構造体がそのインターフェースを満たしているか」を教える必要があります。

```go
// 例: ChatService が IChatRepository インターフェースに依存する場合
var Set = wire.NewSet(
    NewChatRepository,
    wire.Bind(new(repository.IChatRepository), new(*repository.ChatRepository)),
)
```

### 4. クリーンアップ関数（切断処理）のサポート

DB 接続やコネクションプールなど、アプリ終了時に Close したい処理がある場合、Provider は第2戻り値に `func()` を返すことができます。

```go
func NewDB(...) (*gorm.DB, func(), error) {
    db := ...
    cleanup := func() {
        sqlDB, _ := db.DB()
        sqlDB.Close()
    }
    return db, cleanup, nil
}
```

Wire の Injector（`InitializeApp`）のシグネチャを `(*gin.Engine, func(), error)` にしておくと、Wire がすべてのクリーンアップ処理をまとめた関数を返してくれます。

---

## 6. よく使うコマンド集

| 操作             | コマンド                                       | 説明                                                   |
| :--------------- | :--------------------------------------------- | :----------------------------------------------------- |
| **コード生成**   | `docker compose exec api wire gen ./cmd/api`   | `wire_gen.go` を生成・更新します。                     |
| **依存チェック** | `docker compose exec api wire check ./cmd/api` | 生成を行わず、依存エラーがないか検査します。           |
| **diff 確認**    | `docker compose exec api wire diff ./cmd/api`  | 現在の `wire_gen.go` との差分を表示します。            |
| **一括生成**     | `docker compose exec api go generate ./...`    | `go:generate` ディレクティブ経由で Wire を走らせます。 |
