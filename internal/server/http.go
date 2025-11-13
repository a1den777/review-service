package server

import (
    "encoding/json"
    "net/http"
    "strconv"

    hv1 "review-service/api/helloworld/v1"
    rv1 "review-service/api/review/v1"
    "review-service/internal/biz"
    "review-service/internal/conf"
    "review-service/internal/service"

    "github.com/go-kratos/kratos/v2/log"
    "github.com/go-kratos/kratos/v2/middleware/recovery"
    khttp "github.com/go-kratos/kratos/v2/transport/http"
)

// NewHTTPServer new an HTTP server.
func NewHTTPServer(c *conf.Server, greeter *service.GreeterService, review *service.ReviewService, logger log.Logger) *khttp.Server {
    var opts = []khttp.ServerOption{
        khttp.Middleware(
            recovery.Recovery(),
        ),
    }
    if c.Http.Network != "" {
        opts = append(opts, khttp.Network(c.Http.Network))
    }
    if c.Http.Addr != "" {
        opts = append(opts, khttp.Address(c.Http.Addr))
    }
    if c.Http.Timeout != nil {
        opts = append(opts, khttp.Timeout(c.Http.Timeout.AsDuration()))
    }
    srv := khttp.NewServer(opts...)
    hv1.RegisterGreeterHTTPServer(srv, greeter)
    rv1.RegisterReviewHTTPServer(srv, review)

    r := srv.Route("/v1")
    r.POST("/reviews/:id:audit", func(ctx khttp.Context) error {
        req := ctx.Request()
        role := req.Header.Get("X-Role")
        if role != "O" {
            ctx.Response().WriteHeader(http.StatusForbidden)
            _, _ = ctx.Response().Write([]byte(`{"error":"forbidden"}`))
            return nil
        }
        vs := ctx.Vars()["id"]
        var idStr string
        if len(vs) > 0 { idStr = vs[0] }
        id, _ := strconv.ParseUint(idStr, 10, 64)
        var body struct{
            Decision string `json:"decision"`
            Reason   string `json:"reason"`
            OperatorID uint64 `json:"operator_id"`
        }
        if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
            ctx.Response().WriteHeader(http.StatusBadRequest)
            _, _ = ctx.Response().Write([]byte(`{"error":"bad_request"}`))
            return nil
        }
        if err := review.Uc().Audit(ctx, id, body.Decision, body.Reason, body.OperatorID); err != nil {
            ctx.Response().WriteHeader(http.StatusBadRequest)
            b, _ := json.Marshal(map[string]string{"error": err.Error()})
            _, _ = ctx.Response().Write(b)
            return nil
        }
        ctx.Response().WriteHeader(http.StatusOK)
        _, _ = ctx.Response().Write([]byte(`{"ok":true}`))
        return nil
    })

    r.POST("/reviews/:id:reply", func(ctx khttp.Context) error {
        req := ctx.Request()
        role := req.Header.Get("X-Role")
        if role != "B" {
            ctx.Response().WriteHeader(http.StatusForbidden)
            _, _ = ctx.Response().Write([]byte(`{"error":"forbidden"}`))
            return nil
        }
        vs := ctx.Vars()["id"]
        var idStr string
        if len(vs) > 0 { idStr = vs[0] }
        id, _ := strconv.ParseUint(idStr, 10, 64)
        var body struct{
            MerchantID uint64 `json:"merchant_id"`
            Content string `json:"content"`
        }
        if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
            ctx.Response().WriteHeader(http.StatusBadRequest)
            _, _ = ctx.Response().Write([]byte(`{"error":"bad_request"}`))
            return nil
        }
        if err := review.Uc().AddReply(ctx, &biz.ReviewReply{ReviewID: id, MerchantID: body.MerchantID, Content: body.Content}); err != nil {
            ctx.Response().WriteHeader(http.StatusBadRequest)
            b, _ := json.Marshal(map[string]string{"error": err.Error()})
            _, _ = ctx.Response().Write(b)
            return nil
        }
        ctx.Response().WriteHeader(http.StatusOK)
        _, _ = ctx.Response().Write([]byte(`{"ok":true}`))
        return nil
    })

    r.GET("/reviews/:id/replies", func(ctx khttp.Context) error {
        vs := ctx.Vars()["id"]
        var idStr string
        if len(vs) > 0 { idStr = vs[0] }
        id, _ := strconv.ParseUint(idStr, 10, 64)
        list, err := review.Uc().ListReplies(ctx, id)
        if err != nil {
            ctx.Response().WriteHeader(http.StatusBadRequest)
            b, _ := json.Marshal(map[string]string{"error": err.Error()})
            _, _ = ctx.Response().Write(b)
            return nil
        }
        b, _ := json.Marshal(list)
        ctx.Response().Header().Set("Content-Type", "application/json")
        _, _ = ctx.Response().Write(b)
        return nil
    })

    r.GET("/reviews:pending", func(ctx khttp.Context) error {
        q := ctx.Request().URL.Query()
        page, _ := strconv.ParseInt(q.Get("page"), 10, 32)
        size, _ := strconv.ParseInt(q.Get("page_size"), 10, 32)
        items, total, err := review.Uc().ListPending(ctx, int32(page), int32(size))
        if err != nil {
            ctx.Response().WriteHeader(http.StatusBadRequest)
            b, _ := json.Marshal(map[string]string{"error": err.Error()})
            _, _ = ctx.Response().Write(b)
            return nil
        }
        b, _ := json.Marshal(map[string]any{"total": total, "reviews": items})
        ctx.Response().Header().Set("Content-Type", "application/json")
        _, _ = ctx.Response().Write(b)
        return nil
    })

    return srv
}
