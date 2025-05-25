package services

import (
	"context"

	"github.com/SaiNageswarS/agent-boot/db"
	pb "github.com/SaiNageswarS/agent-boot/generated/pb"
	"github.com/SaiNageswarS/go-api-boot/auth"
	"github.com/SaiNageswarS/go-api-boot/logger"
	"github.com/SaiNageswarS/go-api-boot/odm"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type LoginService struct {
	pb.UnimplementedLoginServer
	mongo odm.MongoClient
}

func ProvideLoginService(mongo odm.MongoClient) *LoginService {
	return &LoginService{
		mongo: mongo,
	}
}

// removing auth interceptor
func (u *LoginService) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, nil
}

func (s *LoginService) Login(ctx context.Context, req *pb.LoginRequest) (*pb.AuthResponse, error) {
	loginInfo, err := odm.Await(odm.CollectionOf[db.LoginModel](s.mongo, req.Tenant).FindOneByID(ctx, db.LoginModel{
		EmailId: req.Email}.Id()))

	if err != nil || loginInfo == nil {
		return nil, status.Error(codes.NotFound, "User not found")
	}

	jwtToken, err := auth.GetToken(req.Tenant, loginInfo.Id(), "client")
	if err != nil {
		return nil, status.Error(codes.PermissionDenied, "Wrong claim")
	}

	return &pb.AuthResponse{
		Jwt: jwtToken,
	}, nil
}

func (s *LoginService) SignUp(ctx context.Context, req *pb.SignUpRequest) (*pb.AuthResponse, error) {
	loginModel := db.LoginModel{EmailId: req.Email}
	isExists, err := odm.Await(odm.CollectionOf[db.LoginModel](s.mongo, req.Tenant).Exists(ctx, loginModel.Id()))
	if err != nil {
		logger.Error("Error checking user existence", zap.Error(err))
		return nil, status.Error(codes.Internal, "Internal server error")
	}

	if isExists {
		return nil, status.Error(codes.AlreadyExists, "User already exists")
	}

	loginModel.HashedPassword, err = hashPassword(req.Password)
	if err != nil {
		logger.Error("Error hashing password", zap.Error(err))
		return nil, status.Error(codes.Internal, "Internal server error")
	}

	_, err = odm.Await(odm.CollectionOf[db.LoginModel](s.mongo, req.Tenant).Save(ctx, loginModel))
	if err != nil {
		logger.Error("Error saving user", zap.Error(err))
		return nil, status.Error(codes.Internal, "Internal server error")
	}

	jwtToken, err := auth.GetToken(req.Tenant, loginModel.Id(), "client")
	if err != nil {
		logger.Error("Error generating JWT token", zap.Error(err))
		return nil, status.Error(codes.PermissionDenied, "Wrong claim")
	}

	return &pb.AuthResponse{
		Jwt: jwtToken,
	}, nil
}

func hashPassword(password string) (string, error) {
	// Generate a hashed password using bcrypt
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	// Return the hashed password as a string
	return string(hashedPassword), nil
}
