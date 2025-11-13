package main

import (
    "bytes"
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "os"
    "os/signal"
    "syscall"
    "time"

    esv8 "github.com/elastic/go-elasticsearch/v8"
    kafka "github.com/segmentio/kafka-go"

    conf "review-service/internal/conf"
    kconfig "github.com/go-kratos/kratos/v2/config"
    kfile "github.com/go-kratos/kratos/v2/config/file"
)

type reviewEvent struct {
    Op      string        `json:"op"`
    Payload *reviewRecord `json:"payload"`
    Ts      int64         `json:"ts"`
}

type reviewRecord struct {
    ID      uint64 `json:"id"`
    UserID  uint64 `json:"user_id"`
    Subject string `json:"subject"`
    Content string `json:"content"`
    Rating  int32  `json:"rating"`
}

func main() {
    var confPath string
    flag.StringVar(&confPath, "conf", "./configs", "config path, file or directory")
    flag.Parse()

    // Load config via kratos config
    c := kconfig.New(kconfig.WithSource(kfile.NewSource(confPath)))
    defer c.Close()
    if err := c.Load(); err != nil {
        log.Fatalf("config load: %v", err)
    }
    var bc conf.Bootstrap
    if err := c.Scan(&bc); err != nil {
        log.Fatalf("config scan: %v", err)
    }

    // Setup Elasticsearch client
    es, err := esv8.NewClient(esv8.Config{Addresses: bc.Data.Elasticsearch.Addresses})
    if err != nil {
        log.Fatalf("new es client: %v", err)
    }
    indexName := bc.Data.Elasticsearch.Index
    if indexName == "" {
        indexName = "reviews"
    }

    // Setup Kafka reader
    r := kafka.NewReader(kafka.ReaderConfig{
        Brokers:  bc.Data.Kafka.Brokers,
        GroupID:  "review-task",
        Topic:    bc.Data.Kafka.Topic,
        MinBytes: 10e3, // 10KB
        MaxBytes: 10e6, // 10MB
    })
    defer r.Close()

    ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer stop()

    log.Printf("review-task started: topic=%s, es_index=%s", bc.Data.Kafka.Topic, indexName)
    for {
        m, err := r.ReadMessage(ctx)
        if err != nil {
            if ctx.Err() != nil {
                log.Printf("context done: %v", ctx.Err())
                return
            }
            log.Printf("read message error: %v", err)
            time.Sleep(time.Second)
            continue
        }
        var evt reviewEvent
        if err := json.Unmarshal(m.Value, &evt); err != nil {
            log.Printf("unmarshal event: %v", err)
            continue
        }
        if evt.Payload == nil {
            continue
        }
        // index or delete
        switch evt.Op {
        case "create", "update":
            body, _ := json.Marshal(map[string]any{
                "id":       evt.Payload.ID,
                "user_id":  evt.Payload.UserID,
                "subject":  evt.Payload.Subject,
                "content":  evt.Payload.Content,
                "rating":   evt.Payload.Rating,
                "ts":       evt.Ts,
            })
            res, err := es.Index(indexName, bytesReader(body), es.Index.WithDocumentID(idStr(evt.Payload.ID)))
            if err != nil {
                log.Printf("es index error: %v", err)
                continue
            }
            res.Body.Close()
        case "delete":
            res, err := es.Delete(indexName, idStr(evt.Payload.ID))
            if err != nil {
                log.Printf("es delete error: %v", err)
                continue
            }
            res.Body.Close()
        }
    }
}

func idStr(id uint64) string { return fmt.Sprintf("%d", id) }

func bytesReader(b []byte) *bytes.Reader { return bytes.NewReader(b) }