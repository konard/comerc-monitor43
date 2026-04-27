// Package auth предоставляет JWT interceptor для gRPC.
//
// Использование:
//
//	interceptor := auth.NewJWTInterceptor("secret-key")
//	grpcServer := grpc.NewServer(
//	    grpc.UnaryInterceptor(interceptor.Unary()),
//	)
//
// Извлечение user ID в handler:
//
//	userID, err := auth.RequireAuth(ctx)
//	if err != nil {
//	    return nil, err
//	}
package auth
