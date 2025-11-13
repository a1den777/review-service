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

func (s *ReviewService) Uc() *biz.ReviewUsecase { return s.uc }

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
		Id:          r.ID,
		UserId:      r.UserID,
		Subject:     r.Subject,
		Content:     r.Content,
		Rating:      r.Rating,
		CreatedAt:   0,
		Status:      r.Status,
		AuditReason: "",
		AuditBy:     0,
		AuditAt:     0,
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
			Id:          r.ID,
			UserId:      r.UserID,
			Subject:     r.Subject,
			Content:     r.Content,
			Rating:      r.Rating,
			CreatedAt:   0,
			Status:      r.Status,
			AuditReason: "",
			AuditBy:     0,
			AuditAt:     0,
		})
	}
	return &pb.ListReviewReply{Total: total, Reviews: items}, nil
}

func (s *ReviewService) AuditReview(ctx context.Context, req *pb.AuditReviewRequest) (*pb.AuditReviewReply, error) {
	if err := s.uc.Audit(ctx, req.Id, req.Decision, req.Reason, req.OperatorId); err != nil {
		return nil, err
	}
	return &pb.AuditReviewReply{}, nil
}

func (s *ReviewService) CreateReply(ctx context.Context, req *pb.CreateReplyRequest) (*pb.CreateReplyReply, error) {
	if err := s.uc.AddReply(ctx, &biz.ReviewReply{ReviewID: req.Id, MerchantID: req.MerchantId, Content: req.Content}); err != nil {
		return nil, err
	}
	return &pb.CreateReplyReply{}, nil
}

func (s *ReviewService) ListReplies(ctx context.Context, req *pb.ListRepliesRequest) (*pb.ListRepliesReply, error) {
	list, err := s.uc.ListReplies(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	items := make([]*pb.ReplyRecord, 0, len(list))
	for _, r := range list {
		items = append(items, &pb.ReplyRecord{Id: r.ID, ReviewId: r.ReviewID, MerchantId: r.MerchantID, Content: r.Content, CreatedAt: r.CreatedAt})
	}
	return &pb.ListRepliesReply{Replies: items}, nil
}

func (s *ReviewService) ListPendingReview(ctx context.Context, req *pb.ListPendingReviewRequest) (*pb.ListPendingReviewReply, error) {
	rs, total, err := s.uc.ListPending(ctx, req.Page, req.PageSize)
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
			Status:    r.Status,
		})
	}
	return &pb.ListPendingReviewReply{Total: total, Reviews: items}, nil
}
