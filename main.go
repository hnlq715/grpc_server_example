/*
 *
 * Copyright 2015 gRPC authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

//go:generate protoc -I ../helloworld --go_out=plugins=grpc:../helloworld ../helloworld/helloworld.proto

// Package main implements a server for Greeter service.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"

	pb "github.com/api7/grpc_server_example/proto"
)

var (
	grpcAddr  = ":50051"
	grpcsAddr = ":50052"

	crtFilePath = "../t/cert/apisix.crt"
	keyFilePath = "../t/cert/apisix.key"
)

func init() {
	flag.StringVar(&grpcAddr, "grpc-address", grpcAddr, "address for grpc")
	flag.StringVar(&grpcsAddr, "grpcs-address", grpcsAddr, "address for grpcs")
	flag.StringVar(&crtFilePath, "crt", crtFilePath, "path to certificate")
	flag.StringVar(&keyFilePath, "key", keyFilePath, "path to key")
}

// server is used to implement helloworld.GreeterServer.
type server struct{}

// SayHello implements helloworld.GreeterServer
func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	log.Printf("Received: %v", in.Name)
	return &pb.HelloReply{Message: "Hello " + in.Name, Items: in.GetItems()}, nil
}

func (s *server) SayHelloAfterDelay(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {

	select {
	case <-time.After(1 * time.Second):
		fmt.Println("overslept")
	case <-ctx.Done():
		errStr := ctx.Err().Error()
		if ctx.Err() == context.DeadlineExceeded {
			return nil, status.Error(codes.DeadlineExceeded, errStr)
		}
	}

	time.Sleep(1 * time.Second)

	log.Printf("Received: %v", in.Name)

	return &pb.HelloReply{Message: "Hello delay " + in.Name}, nil
}

func (s *server) Plus(ctx context.Context, in *pb.PlusRequest) (*pb.PlusReply, error) {
	log.Printf("Received: %v %v", in.A, in.B)
	return &pb.PlusReply{Result: in.A + in.B}, nil
}

func main() {
	flag.Parse()

	go func() {
		lis, err := net.Listen("tcp", grpcAddr)
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}
		s := grpc.NewServer()
		pb.RegisterGreeterServer(s, &server{})
		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	go func() {
		lis, err := net.Listen("tcp", grpcsAddr)
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}

		c, err := credentials.NewServerTLSFromFile(crtFilePath, keyFilePath)
		if err != nil {
			log.Fatalf("credentials.NewServerTLSFromFile err: %v", err)
		}
		s := grpc.NewServer(grpc.Creds(c))
		pb.RegisterGreeterServer(s, &server{})
		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	sig := <-signals
	log.Printf("get signal %s, exit\n", sig.String())
}
