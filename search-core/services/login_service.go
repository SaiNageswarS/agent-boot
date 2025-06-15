package services

import (
	"context"

	pb "agent-boot/proto/generated"

	"github.com/SaiNageswarS/agent-boot/search-core/appconfig"
	"github.com/SaiNageswarS/agent-boot/search-core/db"
	"github.com/SaiNageswarS/go-api-boot/auth"
	"github.com/SaiNageswarS/go-api-boot/logger"
	"github.com/SaiNageswarS/go-api-boot/odm"
	"github.com/SaiNageswarS/go-collection-boot/async"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type LoginService struct {
	pb.UnimplementedLoginServer
	mongo *mongo.Client
	ccfgg *appconfig.AppConfig
}

func ProvideLoginService(mongo *mongo.Client, ccfgg *appconfig.AppConfig) *LoginService {
	return &LoginService{
		mongo: mongo,
		ccfgg: ccfgg,
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
	err = bcrypt.CompareHashAndPassword([]byte(loginInfo.HashedPassword), []byte(req.Password))
	if err != nil {
		return nil, status.Error(codes.PermissionDenied, "Wrong password")
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
	if !s.ccfgg.SignUpAllowed || req.Email == "" || req.Password == "" {
		return nil, status.Error(codes.PermissionDenied, "Sign up is not allowed")
	}

	hashedPassword, err := hashPassword(req.Password)
	if err != nil {
		logger.Error("Failed to hash password", zap.Error(err))
		return nil, status.Error(codes.Internal, "Failed to hash password: "+err.Error())
	}

	loginInfo := db.LoginModel{
		EmailId:        req.Email,
		HashedPassword: hashedPassword,
	}

	// Save the login info to the database
	_, err = async.Await(odm.CollectionOf[db.LoginModel](s.mongo, req.Tenant).Save(ctx, loginInfo))
	if err != nil {
		logger.Error("Failed to save login info", zap.Error(err))
		return nil, status.Error(codes.Internal, "Failed to save login info: "+err.Error())
	}

	jwtToken, err := auth.GetToken(req.Tenant, loginInfo.Id(), "client")
	if err != nil {
		logger.Error("Failed to generate JWT token", zap.Error(err))
		return nil, status.Error(codes.Internal, "Failed to generate JWT token: "+err.Error())
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
