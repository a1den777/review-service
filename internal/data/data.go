package data

import (
    "context"
    "database/sql"
    "time"

    "review-service/internal/conf"

    "github.com/go-kratos/kratos/v2/log"
    "github.com/google/wire"
    _ "github.com/go-sql-driver/mysql"
    redis "github.com/redis/go-redis/v9"
    kafka "github.com/segmentio/kafka-go"
    esv8 "github.com/elastic/go-elasticsearch/v8"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewData, NewGreeterRepo, NewReviewRepo)

// Data holds shared clients.
type Data struct {
    DB  *sql.DB
    RDB *redis.Client
    Kafka *kafka.Writer
    ES *esv8.Client
    ESIndex string
}

// NewData initializes database and redis clients from configuration.
func NewData(c *conf.Data, logger log.Logger) (*Data, func(), error) {
    helper := log.NewHelper(logger)

    // Setup SQL DB (MySQL)
    db, err := sql.Open(c.Database.Driver, c.Database.Source)
    if err != nil {
        return nil, nil, err
    }
    db.SetMaxOpenConns(20)
    db.SetMaxIdleConns(10)
    db.SetConnMaxLifetime(60 * time.Minute)
    if err := db.Ping(); err != nil {
        return nil, nil, err
    }

    // Setup Redis
    var rdb *redis.Client
    if c.Redis != nil && c.Redis.Addr != "" {
        rdb = redis.NewClient(&redis.Options{
            Network: c.Redis.Network,
            Addr:    c.Redis.Addr,
            // Convert timeouts if provided
            ReadTimeout:  durationOrZero(c.Redis.ReadTimeout),
            WriteTimeout: durationOrZero(c.Redis.WriteTimeout),
        })
        // simple ping
        if err := rdb.Ping(context.Background()).Err(); err != nil {
            helper.Warnf("redis ping failed: %v", err)
        }
    }

    // Setup Kafka writer
    var kw *kafka.Writer
    if c.Kafka != nil && len(c.Kafka.Brokers) > 0 && c.Kafka.Topic != "" {
        kw = &kafka.Writer{
            Addr:     kafka.TCP(c.Kafka.Brokers...),
            Topic:    c.Kafka.Topic,
            Balancer: &kafka.LeastBytes{},
        }
    }

    // Setup Elasticsearch client
    var es *esv8.Client
    var esIndex string
    if c.Elasticsearch != nil && len(c.Elasticsearch.Addresses) > 0 {
        esIndex = c.Elasticsearch.Index
        if esIndex == "" { esIndex = "reviews" }
        esClient, err := esv8.NewClient(esv8.Config{
            Addresses: c.Elasticsearch.Addresses,
            Username:  c.Elasticsearch.Username,
            Password:  c.Elasticsearch.Password,
        })
        if err != nil {
            helper.Warnf("es client init failed: %v", err)
        } else {
            es = esClient
        }
    }

    cleanup := func() {
        helper.Info("closing the data resources")
        if rdb != nil {
            _ = rdb.Close()
        }
        if db != nil {
            _ = db.Close()
        }
        if kw != nil { _ = kw.Close() }
    }
    return &Data{DB: db, RDB: rdb, Kafka: kw, ES: es, ESIndex: esIndex}, cleanup, nil
}

func durationOrZero(dur interface{ AsDuration() time.Duration }) time.Duration {
    if dur == nil {
        return 0
    }
    return dur.AsDuration()
}
