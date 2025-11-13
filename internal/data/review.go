package data

import (
    "bytes"
    "context"
    "database/sql"
    "encoding/json"
    "fmt"
    "time"

    "review-service/internal/biz"

    "github.com/go-kratos/kratos/v2/log"
    kafka "github.com/segmentio/kafka-go"
)

type reviewRepo struct {
    data *Data
    log  *log.Helper
}

func NewReviewRepo(data *Data, logger log.Logger) biz.ReviewRepo {
    return &reviewRepo{
        data: data,
        log:  log.NewHelper(logger),
    }
}

func (r *reviewRepo) Create(ctx context.Context, in *biz.Review) (uint64, error) {
    res, err := r.data.DB.ExecContext(ctx, `
        INSERT INTO reviews (user_id, subject, content, rating)
        VALUES (?, ?, ?, ?)
    `, in.UserID, in.Subject, in.Content, in.Rating)
    if err != nil {
        return 0, err
    }
    id, _ := res.LastInsertId()
    // invalidate cache
    _ = r.invalidate(ctx, uint64(id))
    // publish event
    r.publish(ctx, "create", &biz.Review{ID: uint64(id), UserID: in.UserID, Subject: in.Subject, Content: in.Content, Rating: in.Rating})
    return uint64(id), nil
}

func (r *reviewRepo) Get(ctx context.Context, id uint64) (*biz.Review, error) {
    // try cache first
    key := r.cacheKey(id)
    if r.data.RDB != nil {
        if s, err := r.data.RDB.Get(ctx, key).Result(); err == nil && len(s) > 0 {
            var out biz.Review
            if json.Unmarshal([]byte(s), &out) == nil {
                return &out, nil
            }
        }
    }

    row := r.data.DB.QueryRowContext(ctx, `
        SELECT id, user_id, subject, content, rating FROM reviews WHERE id = ?
    `, id)
    var out biz.Review
    var iid uint64
    var uid uint64
    var subject, content string
    var rating int32
    if err := row.Scan(&iid, &uid, &subject, &content, &rating); err != nil {
        if err == sql.ErrNoRows {
            return nil, biz.ErrReviewNotFound
        }
        return nil, err
    }
    out.ID = iid
    out.UserID = uid
    out.Subject = subject
    out.Content = content
    out.Rating = rating

    // set cache
    if r.data.RDB != nil {
        if b, err := json.Marshal(out); err == nil {
            _ = r.data.RDB.Set(ctx, key, string(b), 5*time.Minute).Err()
        }
    }
    return &out, nil
}

func (r *reviewRepo) Update(ctx context.Context, in *biz.Review) error {
    _, err := r.data.DB.ExecContext(ctx, `
        UPDATE reviews
        SET subject = ?, content = ?, rating = ?
        WHERE id = ?
    `, in.Subject, in.Content, in.Rating, in.ID)
    if err != nil {
        return err
    }
    // invalidate cache
    _ = r.invalidate(ctx, in.ID)
    // publish event
    r.publish(ctx, "update", in)
    return nil
}

func (r *reviewRepo) Delete(ctx context.Context, id uint64) error {
    _, err := r.data.DB.ExecContext(ctx, `DELETE FROM reviews WHERE id = ?`, id)
    if err != nil {
        return err
    }
    _ = r.invalidate(ctx, id)
    r.publish(ctx, "delete", &biz.Review{ID: id})
    return nil
}

func (r *reviewRepo) List(ctx context.Context, in *biz.ReviewQuery) ([]*biz.Review, int64, error) {
    // Prefer Elasticsearch when available
    if r.data.ES != nil && r.data.ESIndex != "" {
        // Build ES query
        must := make([]map[string]any, 0)
        filter := make([]map[string]any, 0)
        if in.Q != "" {
            must = append(must, map[string]any{
                "multi_match": map[string]any{
                    "query":    in.Q,
                    "fields":   []string{"subject^2", "content"},
                    "operator": "and",
                },
            })
        }
        if in.UserID != 0 {
            filter = append(filter, map[string]any{"term": map[string]any{"user_id": in.UserID}})
        }
        if in.RatingMin != 0 || in.RatingMax != 0 {
            rangeBody := map[string]any{}
            if in.RatingMin != 0 { rangeBody["gte"] = in.RatingMin }
            if in.RatingMax != 0 { rangeBody["lte"] = in.RatingMax }
            filter = append(filter, map[string]any{"range": map[string]any{"rating": rangeBody}})
        }
        body := map[string]any{
            "track_total_hits": true,
            "from":             int((in.Page-1)*in.PageSize),
            "size":             int(in.PageSize),
            "query": map[string]any{"bool": map[string]any{
                "must":   must,
                "filter": filter,
            }},
        }
        // sorting
        switch in.Sort {
        case "rating":
            body["sort"] = []map[string]any{{"rating": map[string]any{"order": in.Order}}}
        case "ts":
            body["sort"] = []map[string]any{{"ts": map[string]any{"order": in.Order}}}
        default:
            // relevance: do not set sort
        }
        // execute search
        b, _ := json.Marshal(body)
        res, err := r.data.ES.Search(r.data.ES.Search.WithIndex(r.data.ESIndex), r.data.ES.Search.WithBody(bytes.NewReader(b)))
        if err != nil {
            r.log.WithContext(ctx).Errorf("es search error: %v", err)
        } else {
            defer res.Body.Close()
            var parsed struct {
                Hits struct {
                    Total struct{ Value int64 `json:"value"` } `json:"total"`
                    Hits []struct {
                        ID     string         `json:"_id"`
                        Source map[string]any `json:"_source"`
                    } `json:"hits"`
                } `json:"hits"`
            }
            if err := json.NewDecoder(res.Body).Decode(&parsed); err == nil {
                out := make([]*biz.Review, 0, len(parsed.Hits.Hits))
                for _, h := range parsed.Hits.Hits {
                    src := h.Source
                    var item biz.Review
                    // id may be numeric or string in _source; prefer _id
                    if v, ok := src["user_id"].(float64); ok { item.UserID = uint64(v) }
                    if v, ok := src["subject"].(string); ok { item.Subject = v }
                    if v, ok := src["content"].(string); ok { item.Content = v }
                    if v, ok := src["rating"].(float64); ok { item.Rating = int32(v) }
                    // parse id from _id
                    var iid uint64
                    if _, err := fmt.Sscanf(h.ID, "%d", &iid); err == nil { item.ID = iid }
                    out = append(out, &item)
                }
                return out, parsed.Hits.Total.Value, nil
            }
        }
        // fall through to DB if ES errors
    }

    // Fallback: MySQL pagination
    offset := (in.Page - 1) * in.PageSize
    var total int64
    if err := r.data.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM reviews`).Scan(&total); err != nil {
        return nil, 0, err
    }
    rows, err := r.data.DB.QueryContext(ctx, `
        SELECT id, user_id, subject, content, rating
        FROM reviews
        ORDER BY id DESC
        LIMIT ? OFFSET ?
    `, in.PageSize, offset)
    if err != nil { return nil, 0, err }
    defer rows.Close()
    var list []*biz.Review
    for rows.Next() {
        var out biz.Review
        var iid, uid uint64
        var subject, content string
        var rating int32
        if err := rows.Scan(&iid, &uid, &subject, &content, &rating); err != nil { return nil, 0, err }
        out.ID = iid; out.UserID = uid; out.Subject = subject; out.Content = content; out.Rating = rating
        list = append(list, &out)
    }
    if err := rows.Err(); err != nil { return nil, 0, err }
    return list, total, nil
}

func (r *reviewRepo) cacheKey(id uint64) string {
    return fmt.Sprintf("review:%d", id)
}

func (r *reviewRepo) invalidate(ctx context.Context, id uint64) error {
    if r.data.RDB == nil {
        return nil
    }
    return r.data.RDB.Del(ctx, r.cacheKey(id)).Err()
}

type reviewEvent struct {
    Op       string      `json:"op"`
    Payload  *biz.Review `json:"payload"`
    Ts       int64       `json:"ts"`
}

func (r *reviewRepo) publish(ctx context.Context, op string, rev *biz.Review) {
    if r.data.Kafka == nil {
        return
    }
    evt := reviewEvent{Op: op, Payload: rev, Ts: time.Now().Unix()}
    b, err := json.Marshal(evt)
    if err != nil {
        r.log.WithContext(ctx).Errorf("marshal event: %v", err)
        return
    }
    _ = r.data.Kafka.WriteMessages(ctx, kafka.Message{Value: b})
}