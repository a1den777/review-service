package biz

import (
    "context"

    "github.com/go-kratos/kratos/v2/errors"
    "github.com/go-kratos/kratos/v2/log"
)

var (
    ErrReviewNotFound = errors.NotFound("REVIEW_NOT_FOUND", "review not found")
)

type Review struct {
    ID      uint64
    UserID  uint64
    Subject string
    Content string
    Rating  int32
}

type ReviewRepo interface {
    Create(context.Context, *Review) (uint64, error)
    Update(context.Context, *Review) error
    Delete(context.Context, uint64) error
    Get(context.Context, uint64) (*Review, error)
    List(context.Context, *ReviewQuery) ([]*Review, int64, error)
}

type ReviewUsecase struct {
    repo ReviewRepo
    log  *log.Helper
}

func NewReviewUsecase(repo ReviewRepo, logger log.Logger) *ReviewUsecase {
    return &ReviewUsecase{repo: repo, log: log.NewHelper(logger)}
}

func (uc *ReviewUsecase) CreateDemo(ctx context.Context) (uint64, error) {
    uc.log.WithContext(ctx).Info("Create demo review")
    return uc.repo.Create(ctx, &Review{UserID: 1, Subject: "Demo", Content: "First review", Rating: 5})
}

func (uc *ReviewUsecase) Get(ctx context.Context, id uint64) (*Review, error) {
    return uc.repo.Get(ctx, id)
}

func (uc *ReviewUsecase) Create(ctx context.Context, in *Review) (uint64, error) {
    uc.log.WithContext(ctx).Infof("Create review user=%d", in.UserID)
    return uc.repo.Create(ctx, in)
}

func (uc *ReviewUsecase) Update(ctx context.Context, in *Review) error {
    uc.log.WithContext(ctx).Infof("Update review id=%d", in.ID)
    return uc.repo.Update(ctx, in)
}

func (uc *ReviewUsecase) Delete(ctx context.Context, id uint64) error {
    uc.log.WithContext(ctx).Infof("Delete review id=%d", id)
    return uc.repo.Delete(ctx, id)
}

// ReviewQuery defines search filters for listing reviews
type ReviewQuery struct {
    Page     int32
    PageSize int32
    Q        string
    UserID   uint64
    RatingMin int32
    RatingMax int32
    Sort     string // relevance|ts|rating
    Order    string // asc|desc
}

func (uc *ReviewUsecase) List(ctx context.Context, in *ReviewQuery) ([]*Review, int64, error) {
    // defaults
    if in == nil { in = &ReviewQuery{} }
    if in.Page < 1 { in.Page = 1 }
    if in.PageSize <= 0 || in.PageSize > 100 { in.PageSize = 20 }
    if in.Order == "" { in.Order = "desc" }
    if in.Sort == "" { in.Sort = "relevance" }
    return uc.repo.List(ctx, in)
}