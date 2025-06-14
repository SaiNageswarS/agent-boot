package services

import (
	"context"

	pb "agent-boot/proto/generated"

	"github.com/SaiNageswarS/agent-boot/search-core/db"
	"github.com/SaiNageswarS/go-api-boot/auth"
	"github.com/SaiNageswarS/go-api-boot/odm"
	"github.com/SaiNageswarS/go-collection-boot/async"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type LoginService struct {
	pb.UnimplementedLoginServer
	mongo *mongo.Client
}

func ProvideLoginService(mongo *mongo.Client) *LoginService {
	return &LoginService{
		mongo: mongo,
	}
}

// removing auth interceptor
func (u *LoginService) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, nil
}

func (s *LoginService) Login(ctx context.Context, req *pb.LoginRequest) (*pb.AuthResponse, error) {
	loginInfo, err := async.Await(odm.CollectionOf[db.LoginModel](s.mongo, req.Tenant).FindOneByID(ctx, db.LoginModel{
		EmailId: req.Email}.Id()))
	if err != nil || loginInfo == nil {
		return nil, status.Error(codes.NotFound, "User not found")
	}

	// verify login logic

	jwtToken, err := auth.GetToken(req.Tenant, loginInfo.Id(), "client")
	if err != nil {
		return nil, status.Error(codes.PermissionDenied, "Wrong claim")
	}

	return &pb.AuthResponse{
		Jwt: jwtToken,
	}, nil
}
