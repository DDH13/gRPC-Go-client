package main

import (
	"context"
	"crypto/tls"
	// "flag"
	"fmt"
	"io"
	"log"
	"time"

	"goclientapk/student"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials" // Import the credentials package
	"google.golang.org/grpc/metadata"
)

// authInterceptor creates a UnaryClientInterceptor to insert the authorization token into the request metadata
func authInterceptor(token string) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		// Inject the token into the context
		newCtx := metadata.AppendToOutgoingContext(ctx, "Authorization", "Bearer "+token)
		// Forward the call to the invoker
		return invoker(newCtx, method, req, reply, cc, opts...)
	}
}
func main() {
	log.Println("Starting gRPC client...")

	token := "eyJhbGciOiJSUzI1NiIsICJ0eXAiOiJKV1QiLCAia2lkIjoiZ2F0ZXdheV9jZXJ0aWZpY2F0ZV9hbGlhcyJ9.eyJpc3MiOiJodHRwczovL2lkcC5hbS53c28yLmNvbS90b2tlbiIsICJzdWIiOiI0NWYxYzVjOC1hOTJlLTExZWQtYWZhMS0wMjQyYWMxMjAwMDIiLCAiYXVkIjoiYXVkMSIsICJleHAiOjE3MTgyMzc3ODQsICJuYmYiOjE3MTQ2Mzc3ODQsICJpYXQiOjE3MTQ2Mzc3ODQsICJqdGkiOiIwMWVmMDg1Yy00M2Q3LTFiYzYtOGFlNC1lM2UzMmZjNTcxYjgiLCAiY2xpZW50SWQiOiI0NWYxYzVjOC1hOTJlLTExZWQtYWZhMS0wMjQyYWMxMjAwMDIiLCAic2NvcGUiOiJhcGs6YXBpX2NyZWF0ZSJ9.ZVHADQ7icq7zWyai4KvjtCaPgGmEq7E86TooHPW22sBNYW0zzc5cHWa2mPwEUncf9LBuhD5ZwENfwUXeoa1xidn9NESUt0hNxw9bwGXZiyHyS8dRaZN4NSd1OHdSLSpqtVYiMwnlKJZX7rmX3qc5KEJ-02-U9hfrjGxEdhStUYZfPIp3t7DZ-KxwUor_LkdPCVUAfyQ3lt6F34qI8Msti_4TqVRQG3ScWTwsp-mCaOpbUTRTR-mRY7yLt0nmoqr2j8UvHvQNllcdC9H-7NdH_UUeRUzPmvwkcQTqDHVb8VCjCZU4iZFvbPHZWyKmjyWi7BvkncS51QpSrJahK6OgPg"
	serverAddr := "default.gw.wso2.com:9095"

	conn, err := dialServer(serverAddr, token)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()
	log.Println("Successfully connected to the server.")

	c := student.NewStudentServiceClient(conn)

	// Unary call
	makeUnaryCall(c)

	// Server streaming call
	// makeServerStreamCall(c)

	// Test rate limit
	testRateLimit(c, 1000)
}

func dialServer(serverAddr, token string) (*grpc.ClientConn, error) {
	config := &tls.Config{
		InsecureSkipVerify: true, // CAUTION: This disables SSL certificate verification.
	}
	creds := credentials.NewTLS(config)

	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
		grpc.WithBlock(),
		grpc.WithUnaryInterceptor(authInterceptor(token)),
	}

	dialCtx, dialCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer dialCancel()
	log.Printf("Dialing to server at %s with timeout...", serverAddr)

	return grpc.DialContext(dialCtx, serverAddr, dialOpts...)
}

func makeUnaryCall(c student.StudentServiceClient) {
	log.Println("Sending request to the server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	r := &student.StudentRequest{Id: 1234}
	response, err := c.GetStudent(ctx, r)
	if err != nil {
		log.Fatalf("Could not fetch student: %v", err)
	}
	fmt.Printf("Student Details: %v\n\n", response)
}

func makeServerStreamCall(c student.StudentServiceClient) {
	log.Println("Sending streaming request to the server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	r := &student.StudentRequest{Id: 1234}
	stream, err := c.GetStudentStream(ctx, r)
	if err != nil {
		log.Fatalf("Could not fetch student stream: %v", err)
	}

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			log.Println("End of student stream.")
			break
		}
		if err != nil {
			log.Fatalf("Error while receiving from student stream: %v", err)
		}
		fmt.Printf("Streamed Student Details: %v\n", resp)
	}
}

func testRateLimit(c student.StudentServiceClient, numrequests int) {
	r := &student.StudentRequest{Id: 1234}
	log.Println("Testing rate limit...")
	t1 := time.Now()
	for i := 0; i < numrequests; i++ {
		ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
		_, err := c.GetStudent(ctx, r)
		if err != nil {
			log.Fatalf("Could not fetch student: %v", err)
		} else {
			fmt.Printf("   Attempt %v\t\t", i+1)
			fmt.Printf("\r")
		}
	}
	t2 := time.Now()
	log.Printf("Time taken to make %v requests: %v\n\n", numrequests, t2.Sub(t1))
}
