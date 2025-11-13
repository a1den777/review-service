package service

import (
    "context"

    pb "review-service/api/review/v1"
    "review-service/internal/biz"
)

type ReviewService struct {
    pb.UnimplementedReviewServer
    uc *biz.ReviewUsecase
}

func NewReviewService(uc *biz.ReviewUsecase) *ReviewService {
    return &ReviewService{uc: uc}
}

func (s *ReviewService) CreateReview(ctx context.Context, req *pb.CreateReviewRequest) (*pb.CreateReviewReply, error) {
    id, err := s.uc.Create(ctx, &biz.Review{
        UserID:  uint64(req.UserId),
        Subject: req.Subject,
        Content: req.Content,
        Rating:  req.Rating,
    })
    if err != nil {
        return nil, err
    }
    return &pb.CreateReviewReply{Id: id}, nil
}

func (s *ReviewService) UpdateReview(ctx context.Context, req *pb.UpdateReviewRequest) (*pb.UpdateReviewReply, error) {
    err := s.uc.Update(ctx, &biz.Review{
        ID:      req.Id,
        Subject: req.Subject,
        Content: req.Content,
        Rating:  req.Rating,
    })
    if err != nil {
        return nil, err
    }
    return &pb.UpdateReviewReply{}, nil
}

func (s *ReviewService) DeleteReview(ctx context.Context, req *pb.DeleteReviewRequest) (*pb.DeleteReviewReply, error) {
    if err := s.uc.Delete(ctx, req.Id); err != nil {
        return nil, err
    }
    return &pb.DeleteReviewReply{}, nil
}

func (s *ReviewService) GetReview(ctx context.Context, req *pb.GetReviewRequest) (*pb.GetReviewReply, error) {
    r, err := s.uc.Get(ctx, req.Id)
    if err != nil {
        return nil, err
    }
    return &pb.GetReviewReply{Review: &pb.ReviewRecord{
        Id:        r.ID,
        UserId:    r.UserID,
        Subject:   r.Subject,
        Content:   r.Content,
        Rating:    r.Rating,
        CreatedAt: 0,
    }}, nil
}

func (s *ReviewService) ListReview(ctx context.Context, req *pb.ListReviewRequest) (*pb.ListReviewReply, error) {
    rs, total, err := s.uc.List(ctx, &biz.ReviewQuery{
        Page:      req.Page,
        PageSize:  req.PageSize,
        Q:         req.Q,
        UserID:    req.UserId,
        RatingMin: req.RatingMin,
        RatingMax: req.RatingMax,
        Sort:      req.Sort,
        Order:     req.Order,
    })
    if err != nil {
        return nil, err
    }
    items := make([]*pb.ReviewRecord, 0, len(rs))
    for _, r := range rs {
        items = append(items, &pb.ReviewRecord{
            Id:        r.ID,
            UserId:    r.UserID,
            Subject:   r.Subject,
            Content:   r.Content,
            Rating:    r.Rating,
            CreatedAt: 0,
        })
    }
    return &pb.ListReviewReply{Total: total, Reviews: items}, nil
}
