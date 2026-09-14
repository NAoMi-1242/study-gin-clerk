# Google Wire 運用マニュアル

本書は、当プロジェクトにおける **Google Wire（コンパイル時 依存性注入 / DI ツール）** の基本概念、現在の構成、拡張手順、およびベストプラクティスをまとめたマニュアルです。

---

## 目次

1. [Google Wire の基本概念と仕組み](#1-google-wire-の基本概念と仕組み)
2. [当プロジェクトでのセットアップ構成（ドメイン駆動）](#2-当プロジェクトでのセットアップ構成ドメイン駆動)
3. [新しいコンポーネントを追加する手順（実践ガイド）](#3-新しいコンポーネントを追加する手順実践ガイド)
4. [知っておくべき注意点・ベストプラクティス](#4-知っておくべき注意点ベストプラクティス)
5. [よく使うコマンド集](#5-よく使うコマンド集)

---

## 1. Google Wire の基本概念と仕組み

### なぜ Wire を使うのか？

Go にはリフレクションを用いた実行時 DI（例: Uber Dig や Fx）もありますが、Google Wire は **「コンパイル前にコード生成によって依存関係を解決する」** アプローチを採用しています。

- **完全な型安全性**: 依存の渡し忘れや型の不一致があれば、ビルド前（コード生成時）にエラーで検知できます。
- **ゼロ・オーバーヘッド**: 生成されるのは純粋な Go コード（`wire_gen.go`）のため、リフレクション不要で最速で動作します。
- **可読性**: 自動生成された初期化コード（`InitializeApp`）を直接目で確認・デバッグできます。

### 3つの基本要素

```
┌─────────────────┐       ┌─────────────────┐       ┌─────────────────┐
│    Provider     │  ──>  │   ProviderSet   │  ──>  │    Injector     │
│  (部品を作る関数) │       │   (部品のまとめ)  │       │  (完成品を作る関数) │
└─────────────────┘       └─────────────────┘       └─────────────────┘
```

1. **Provider（プロバイダ）**:
   通常のコンストラクタ関数（例: `NewRepository`, `ProvideService`, `NewHandler`）。
2. **ProviderSet（プロバイダセット）**:
   `wire.NewSet(...)` でドメインごとに関連する Provider 群を 1 つに束ねたもの（例: `chat.Set`, `apikey.Set`）。
3. **Injector（インジェクタ）**:
   `cmd/api/wire.go` に記述する「最終的に何が欲しいか」を宣言する関数（例: `InitializeApp(cfg config.Config) (*gin.Engine, func(), error)`）。

---

## 2. 当プロジェクトでのセットアップ構成（ドメイン駆動）

### ディレクトリとファイル構成

```
study-gin-clerk/
  ├── cmd/
  │    └── api/
  │         ├── main.go             # InitializeApp() の実行と Graceful Shutdown
  │         ├── wire.go             # 【手動編集】Wire 設計図 (Injector 宣言)
  │         └── wire_gen.go         # 【自動生成】Wire が生成した初期化コード
  └── internal/
       ├── apikey/wire.go           # apikey.Set (Repository, ProvideService, Handler)
       ├── chat/wire.go             # chat.Set (Repository, ProvideService, Handler)
       ├── profile/wire.go          # profile.Set (Repository, Service, Handler)
       ├── aimodel/wire.go          # aimodel.Set (ProvideService, Handler)
       ├── health/wire.go           # health.Set (Handler)
       ├── router/wire.go           # router.Set (Router 構築)
       └── infra/
            ├── ai/wire.go          # ai.Set (Cache, ModelRegistry, Client + wire.Bind)
            ├── crypto/wire.go      # crypto.Set (ProvideCipher + wire.Bind)
            └── db/wire.go          # db.Set (NewDB コネクションプール & クリーンアップ)
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

	"study-gin-clerk/internal/aimodel"
	"study-gin-clerk/internal/apikey"
	"study-gin-clerk/internal/chat"
	"study-gin-clerk/internal/config"
	"study-gin-clerk/internal/health"
	"study-gin-clerk/internal/infra/ai"
	"study-gin-clerk/internal/infra/crypto"
	"study-gin-clerk/internal/infra/db"
	"study-gin-clerk/internal/profile"
	"study-gin-clerk/internal/router"
)

func InitializeApp(cfg config.Config) (*gin.Engine, func(), error) {
	wire.Build(
		db.Set,
		crypto.Set,
		ai.Set,
		health.Set,
		profile.Set,
		apikey.Set,
		aimodel.Set,
		chat.Set,
		router.Set,
	)
	return nil, nil, nil
}
```

> **POINT**: 1行目の `//go:build wireinject` により、このファイルは通常ビルドから除外されます。

#### ② `cmd/api/wire_gen.go`（自動生成コード）

`docker compose exec api wire gen ./cmd/api` で自動生成されます。手動で編集してはいけません。

```go
func InitializeApp(cfg config.Config) (*gin.Engine, func(), error) {
	gormDB, cleanup, err := db.NewDB(cfg)
	if err != nil {
		return nil, nil, err
	}
	handler := health.NewHandler(gormDB)
	repository := profile.NewRepository(gormDB)
	service := profile.NewService(repository)
	profileHandler := profile.NewHandler(service)
	apikeyRepository := apikey.NewRepository(gormDB)
	modelRegistry := ai.NewModelRegistry()
	memoryCache, cleanup2 := ai.NewMemoryCacheDefault()
	aesCipher, err := crypto.ProvideCipher(cfg)
	if err != nil {
		cleanup2()
		cleanup()
		return nil, nil, err
	}
	apikeyService := apikey.ProvideService(apikeyRepository, modelRegistry, memoryCache, aesCipher)
	apikeyHandler := apikey.NewHandler(apikeyService)
	// ... 依存関係が正しい順序で初期化される
```

---

## 3. 新しいコンポーネントを追加する手順（実践ガイド）

新しいドメイン機能（例: `billing`）を追加する際の基本サイクルは以下の **4ステップ** です。

### ステップ 1: ドメイン配下にコンストラクタを実装する

```go
// internal/billing/service.go
package billing

type Service struct { ... }
func NewService(...) *Service { return &Service{...} }
```

### ステップ 2: ドメイン内の `wire.go` に ProviderSet を定義する

```go
// internal/billing/wire.go
package billing

import "github.com/google/wire"

var Set = wire.NewSet(
    NewRepository,
    NewService,
    NewHandler,
)
```

### ステップ 3: `router` および `cmd/api/wire.go` に追加する

1. `internal/router/routes.go` の `Dependencies` 構造体とルーティングに新しいハンドラーを追加。
2. `cmd/api/wire.go` の `wire.Build(...)` に `billing.Set` を追加。

### ステップ 4: Wire コマンドでコードを再生成する

```bash
docker compose exec api wire gen ./cmd/api
```

---

## 4. 知っておくべき注意点・ベストプラクティス

### 1. `wire_gen.go` を直接手動編集しない

- `wire_gen.go` は Wire コマンドで上書きされます。必ず Provider 関数や `wire.go` を修正してから `wire gen` を実行してください。
- `wire_gen.go` は **Git のコミット対象** に含めます（デプロイ先で Wire CLI を叩く必要をなくすため）。

### 2. インターフェースを使う場合（`wire.Bind` またはアダプター関数）

構造体がインターフェースを満たしていることを Wire に伝えるには、以下の2つの手法があります：

1. **`wire.Bind` を使う場合**:
   ```go
   var Set = wire.NewSet(
       ProvideCipher,
       wire.Bind(new(Cipher), new(*AESCipher)),
   )
   ```
2. **`ProvideService` アダプターを使う場合**:
   サービス側が複数のインターフェースを受け取る場合、アダプター関数を用意して具象型を受け取って `NewService` に渡すのが最も明示的でエラーが起きにくいです。
   ```go
   func ProvideService(repo *Repository, client *ai.Client) *Service {
       return NewService(repo, client)
   }
   ```

### 3. クリーンアップ関数（リソース解放）のサポート

DB 接続プールやキャッシュのクリーンアップゴルーチンなど、終了時に停止したいリソースがある場合、コンストラクタの第2戻り値に `func()` を返します。

```go
func NewMemoryCacheDefault() (*MemoryCache, func()) {
    c := NewMemoryCache(15 * time.Minute)
    return c, func() { c.Close() }
}
```

Wire はこれらを自動で集約し、`InitializeApp` のクリーンアップ関数（`cleanup()`）としてまとめて返却してくれます。

---

## 5. よく使うコマンド集

| 操作             | コマンド                                       | 説明                                        |
| :--------------- | :--------------------------------------------- | :------------------------------------------ |
| **コード生成**   | `docker compose exec api wire gen ./cmd/api`   | `wire_gen.go` を生成・更新します。          |
| **依存チェック** | `docker compose exec api wire check ./cmd/api` | 依存関係が解決可能か検査します。            |
| **差分確認**     | `docker compose exec api wire diff ./cmd/api`  | 現在の `wire_gen.go` との差分を表示します。 |
