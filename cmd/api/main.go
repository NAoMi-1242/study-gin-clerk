package main

import (
    "log"
    "net/http"
    "time"

    "github.com/clerk/clerk-sdk-go/v2"

    "study-gin-clerk/internal/config"
)

func main() {
    cfg, err := config.Load()
    if err != nil {
        log.Fatal(err)
    }

    clerk.SetKey(cfg.ClerkSecretKey)

    engine, err := InitializeApp()
    if err != nil {
        log.Fatal(err)
    }

    server := &http.Server{
        Addr:              ":" + cfg.Port,
        Handler:           engine,
        ReadHeaderTimeout: 5 * time.Second,
    }

    log.Printf("server listening on :%s", cfg.Port)

    if err := server.ListenAndServe(); err != nil &&
        err != http.ErrServerClosed {
        log.Fatal(err)
    }
}