package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"

	pb "github.com/sdslabs/katana-broadcast/protobuf"

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
	log.Println(chalName)
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
	artifactPath := fileName
	f, err := os.Create(artifactPath)
	if err != nil {
		return err
	}
	_, err = fileData.WriteTo(f)
	if err != nil {
		return err
	}
	go handleFile(artifactPath, chalName)
	return nil
}

func handleFile(artifactPath string, chalName string) {
	challengeRoot := filepath.Join(os.Getenv("CHALLENGE_DIR"), chalName)
	zipFile, err := zip.OpenReader(artifactPath)
	defer zipFile.Close()
	var filenames []string
	if err != nil {
		log.Fatalln("Error: ", err)
	}
	for _, f := range zipFile.File {
		fpath := filepath.Join(challengeRoot, f.Name)
		filenames = append(filenames, fpath)
		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}
		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			log.Fatalln("Error: ", err)
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			log.Fatalln("Error: ", err)
		}

		rc, err := f.Open()
		if err != nil {
			log.Fatalln("Error: ", err)
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			log.Fatalln("Error: ", err)
		}
	}
	err = os.Remove(artifactPath)
	if err != nil {
		log.Fatalln("Error: ", err)
	}
	initFile := os.Getenv("INIT_FILE")
	if !checkFile(filenames, initFile, filepath.Join(challengeRoot)) {
		log.Fatalln("Error: ", os.ErrNotExist)
	}
	outlog, err := os.Create(filepath.Join(challengeRoot, "out.log"))
	if err != nil {
		log.Fatalln("Error: ", err)
	}
	errlog, err := os.Create(filepath.Join(challengeRoot, "out.log"))
	if err != nil {
		log.Fatalln("Error: ", err)
	}
	err = os.Chmod(filepath.Join(challengeRoot, initFile), 0o755)
	cmd := exec.Command("bash", filepath.Join(challengeRoot, initFile))
	cmd.Stdout = outlog
	cmd.Stderr = errlog
	err = cmd.Run()
	if err != nil {
		log.Fatalln("Error: ", err)
	}
}

func checkFile(filenames []string, file string, path string) bool {
	for _, file := range filenames {
		log.Println(file)
		if file == filepath.Join(path, os.Getenv("INIT_FILE")) {
			return true
		}
	}
	return false
}
func setupServer() error {
	port := os.Getenv("DAEMON_PORT")
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
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
