package main

import (
	"context"
	"fmt"
	"github.com/xumc/dockerdaydayup/example/videoreport/proto"
	"google.golang.org/grpc"
	"log"
	"net"
)
type VideoReportServer struct {}

func (VideoReportServer) GetVideosViewCount(ctx context.Context, req *proto.GetVideoReportRequest) (*proto.GetVideoReportResponse, error) {
	videoItems := make([]*proto.VideoItem, len(req.GetId()))
	for i, id := range req.GetId() {
		videoItems[i] = &proto.VideoItem {
			Id: id,
			ViewCount: id * 10000,
		}
	}

	return &proto.GetVideoReportResponse{Reply: videoItems,}, nil
}

func main() {
	lis, err := net.Listen("tcp", ":80")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()

	proto.RegisterReportServiceServer(s, &VideoReportServer{})

	err = s.Serve(lis)
	if err != nil{
		panic(fmt.Sprintf("can not serve %s", err.Error()))
	}
}
