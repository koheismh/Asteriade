package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/koheismh/Asteriade/internal/httpapi"
	"github.com/koheismh/Asteriade/internal/privatefs"
	"github.com/koheismh/Asteriade/internal/runner"
	"github.com/koheismh/Asteriade/internal/store"
)

func main() {
	if err := run(); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}

func run() error {
	defaultDir, err := os.UserCacheDir()
	if err != nil {
		defaultDir = ".local"
	} else {
		defaultDir = filepath.Join(defaultDir, "Asteriade")
	}
	dataDir := flag.String("data-dir", defaultDir, "Asteriade 専用の保存ディレクトリ")
	port := flag.Int("port", 0, "127.0.0.1 のポート（0: 自動選択）")
	noBrowser := flag.Bool("no-browser", false, "既定ブラウザを自動で開かない")
	backup := flag.String("backup", "", "新しいファイルに一貫したDBバックアップを作成して終了")
	diagnose := flag.Bool("diagnose", false, "Runner の検証状態をJSONで表示して終了")
	flag.Parse()
	if *diagnose {
		return json.NewEncoder(os.Stdout).Encode(map[string]any{"stage": "0", "execution_enabled": false, "runner": runner.Describe()})
	}
	if *port < 0 || *port > 65535 {
		return errors.New("port must be between 0 and 65535")
	}
	dir, err := filepath.Abs(*dataDir)
	if err != nil {
		return err
	}
	if err = privatefs.EnsureDirectory(dir); err != nil {
		return fmt.Errorf("create data directory: %w", err)
	}
	st, err := store.Open(filepath.Join(dir, "asteriade.db"))
	if err != nil {
		return fmt.Errorf("open database (original data preserved): %w", err)
	}
	defer st.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	if err = st.Health(ctx); err != nil {
		cancel()
		return fmt.Errorf("database integrity: %w", err)
	}
	if *backup != "" {
		err = st.Backup(ctx, *backup)
		cancel()
		if err != nil {
			return fmt.Errorf("backup: %w", err)
		}
		log.Print("バックアップと整合性確認が完了しました。")
		return nil
	}
	count, err := st.Recover(ctx)
	cancel()
	if err != nil {
		return fmt.Errorf("recovery: %w", err)
	}
	if count > 0 {
		log.Printf("%d 件を結果確認待ちにしました。自動再実行はしません。", count)
	}
	listener, err := net.Listen("tcp4", fmt.Sprintf("127.0.0.1:%d", *port))
	if err != nil {
		return err
	}
	defer listener.Close()
	api, err := httpapi.New(st, "http://"+listener.Addr().String())
	if err != nil {
		return err
	}
	launch, err := os.CreateTemp(dir, "launch-*.html")
	if err != nil {
		return err
	}
	launchName := launch.Name()
	defer os.Remove(launchName)
	if err = launch.Chmod(0600); err != nil {
		launch.Close()
		return err
	}
	_, err = fmt.Fprintf(launch, "<!doctype html><meta charset=\"utf-8\"><meta name=\"referrer\" content=\"no-referrer\"><title>Asteriade</title><p><a href=\"%s\">Asteriade 管理画面を開く</a></p><p>この起動リンクは10分・1回限り有効です。</p>", html.EscapeString(api.LaunchURL()))
	closeErr := launch.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if err = privatefs.CheckFile(launchName); err != nil {
		return fmt.Errorf("private browser handoff: %w", err)
	}
	srv := &http.Server{Handler: api.Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 << 10}
	stopCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	result := make(chan error, 1)
	go func() { result <- srv.Serve(listener) }()
	log.Print("Asteriade Stage 0 を起動しました。自律実行は未検証のため無効です。")
	log.Printf("管理画面のローカル起動ファイル: %s", launchName)
	if !*noBrowser && runtime.GOOS == "windows" {
		if err = openBrowser(api.LaunchURL()); err != nil {
			log.Print("ブラウザを開けませんでした。起動ファイルを開いてください。")
		}
	}
	select {
	case err = <-result:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	case <-stopCtx.Done():
		shutdownCtx, done := context.WithTimeout(context.Background(), 5*time.Second)
		defer done()
		if err = srv.Shutdown(shutdownCtx); err != nil {
			srv.Close()
			return err
		}
		err = <-result
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}
}
