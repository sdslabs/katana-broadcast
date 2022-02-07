package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"os"

	pb "github.com/sdslabs/katanad/src/proto"

	"google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedFileUploadServiceServer
}

func (s *server) UploadFile(stream pb.FileUploadService_UploadFileServer) error {
	req, err := stream.Recv()
	if err != nil {
		log.Println("Error: ", err)
		return err
	}
	fileName := req.GetFileInfo().GetFileName()
	chalName := req.GetFileInfo().GetChalName()
	log.Printf("Recieving file %s of challenge %s", fileName, chalName)
	fileData := bytes.Buffer{}
	fileSize := 0
	log.Print("Waiting to recive data")
	for {
		req, err := stream.Recv()

		if err == io.EOF {
			log.Print("Finished Recieving")
			break
		}
		if err != nil {
			log.Println("Error: ", err)
			return err
		}
		chunk := req.GetChunkData()
		fileSize += len(chunk)
		_, err = fileData.Write(chunk)
		if err != nil {
			log.Println("Error: ", err)
			return err
		}
	}
	res := &pb.UploadFileResponse{
		Size: uint64(fileSize),
	}

	err = stream.SendAndClose(res)
	if err != nil {
		log.Println("Error: ", err)
		return err
	}
	f, err := os.Create(fmt.Sprintf("%s.zip", fileName))
	if err != nil {
		return err
	}
	_, err = fileData.WriteTo(f)
	if err != nil {
		return err
	}
	return nil
}

func setupServer() error {
	lis, err := net.Listen("tcp", ":5050")
	if err != nil {
		return err
	}
	grpcServer := grpc.NewServer()
	pb.RegisterFileUploadServiceServer(grpcServer, &server{})
	grpcServer.Serve(lis)
	return nil
}

func main() {
	if err := setupServer(); err != nil {
		log.Fatalln(err)
	}
}
