package service

import (
	"bytes"
	"context"
	"errors"
	"github.com/google/uuid"
	pb "github.com/pcbook/api/v1/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
	"log"
)

const maxImageSize = 1 << 20

type LaptopServer struct {
	laptopStore LaptopStore
	imageStore ImageStore
	ratingStore RatingStore
}

func NewLaptopServer(laptopStore LaptopStore, imageStore ImageStore, ratingStore RatingStore) *LaptopServer {
	return &LaptopServer{
		laptopStore,
		imageStore,
		ratingStore,
	}
}

func (server *LaptopServer) CreateLaptop(ctx context.Context, req *pb.CreateLaptopRequest) (*pb.CreateLaptopResponse, error) {
	laptop := req.GetLaptop()
	log.Printf("receive a create-laptop request with id: %s", laptop.Id)

	if len(laptop.Id) > 0 {
		// check for valid uuid
		_, err := uuid.Parse(laptop.Id)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "laptop ID is not a valid UUID: %v", err)
		}
	} else {
		id, err := uuid.NewRandom()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "cannot generate a new laptop ID: %v", err)
		}
		laptop.Id = id.String()
	}

	if ctx.Err() == context.Canceled {
		log.Print("request canceled")
		return nil, status.Error(codes.Canceled, "request canceled")
	}

	if ctx.Err() == context.DeadlineExceeded {
		log.Print("deadline exceeded")
		return nil, status.Error(codes.DeadlineExceeded, "deadline exceeded")
	}

	err := server.laptopStore.Save(laptop)
	if err != nil {
		code := codes.Internal
		if errors.Is(err, ErrAlreadyExists) {
			code = codes.AlreadyExists
		}

		return nil, status.Errorf(code, "cannot save laptop to the store: %v", err)
	}

	log.Printf("saved laptop with id: %s", laptop.Id)

	res := &pb.CreateLaptopResponse{Id: laptop.Id}
	return res, nil
}

func (server *LaptopServer) SearchLaptop(req *pb.SearchLaptopRequest, stream pb.LaptopService_SearchLaptopServer) error {
	filter := req.GetFilter()
	log.Printf("receive a search-laptop request with filter: %v", filter)

	err := server.laptopStore.Search(stream.Context(), filter, func(laptop *pb.Laptop) error {
		res := &pb.SearchLaptopResponse{Laptop: laptop}
		err := stream.Send(res)
		if err != nil {
			return err
		}

		log.Printf("sent laptop with id: %v", laptop.Id)
		return nil
	})

	if err != nil {
		return status.Errorf(codes.Internal, "unexpected error: %v", err)
	}

	return nil
}

func (server *LaptopServer) UploadImage(stream pb.LaptopService_UploadImageServer) error {
	req, err := stream.Recv()
	if err != nil {
		return logError(status.Errorf(codes.Unknown, "cannot receive image info"))
	}

	laptopID := req.GetInfo().GetLaptopId()
	imageType := req.GetInfo().GetImageType()
	log.Printf("received an upload-image request for laptop %s with image type %s", laptopID, imageType)

	laptop, err := server.laptopStore.Find(laptopID)
	if err != nil {
		return logError(status.Errorf(codes.Internal, "cannot find laptop: %v", err))
	}
	if laptop == nil {
		return logError(status.Errorf(codes.InvalidArgument, "laptop id %s does not exist", laptopID))
	}

	imageData := bytes.Buffer{}
	imageSize := 0

	for{
		err := contextError(stream.Context())
		if err != nil {
			return err
		}

		log.Print("waiting to receive more data")

		req, err := stream.Recv()
		if err == io.EOF {
			log.Print("no more data")
			break
		}
		if err != nil {
			return logError(status.Errorf(codes.Unknown, "cannot receive chunk data: %v", err))
		}

		chunk := req.GetChunkData()
		size := len(chunk)
		log.Printf("Received chunk with size: %d", size)

		imageSize += size
		if imageSize > maxImageSize {
			return logError(status.Errorf(codes.InvalidArgument, "image exceeds max image size: %d > %d", imageSize, maxImageSize))
		}

		_, err = imageData.Write(chunk)
		if err != nil {
			return logError(status.Errorf(codes.Internal, "cannot write chunk data: %v", err))
		}
	}

	imageID, err := server.imageStore.Save(laptopID, imageType, imageData)
	if err != nil {
		return logError(status.Errorf(codes.Internal, "cannot save image to the store: %v", err))
	}

	res := &pb.UploadImageResponse{
		Id:   imageID,
		Size: uint32(imageSize),
	}

	err = stream.SendAndClose(res)
	if err != nil {
		return logError(status.Errorf(codes.Unknown, "cannot send response: %v", err))
	}

	log.Printf("saved image with id: %s, size: %d", imageID, imageSize)
	return nil
}

func logError(err error) error {
	if err != nil {
		log.Print(err)
	}
	return err
}

func contextError(ctx context.Context) error {
	switch ctx.Err() {
	case context.Canceled:
		return logError(status.Errorf(codes.Canceled, "request is canceled"))
	case context.DeadlineExceeded:
		return logError(status.Errorf(codes.DeadlineExceeded, "deadline is exceeded"))
	default:
		return nil
	}
}

func (server *LaptopServer) RateLaptop(stream pb.LaptopService_RateLaptopServer) error {
	for {
		err := contextError(stream.Context())
		if err != nil {
			return err
		}

		req, err := stream.Recv()
		if err != io.EOF {
			break
		}
		if err != nil {
			return logError(status.Errorf(codes.Unknown, "cannot receive stream request: %v", err))
		}

		laptopID := req.GetLaptopId()
		score := req.GetScore()

		log.Printf("received a rate-laptop request: id = %s, score = %.2f", laptopID, score)

		found, err := server.laptopStore.Find(laptopID)
		if err != nil {
			return logError(status.Errorf(codes.Internal, "cannot find laptop: %v", err))
		}
		if found == nil {
			return logError(status.Errorf(codes.NotFound, "laptop %s could not be found", laptopID))
		}

		rating, err := server.ratingStore.Add(laptopID, score)
		if err != nil {
			return logError(status.Errorf(codes.Internal, "cannot add rating to the store: %v", err))
		}

		res := &pb.RateLaptopResponse{
			LaptopId:     laptopID,
			RatedCount:   rating.Count,
			AverageScore: rating.Sum / float64(rating.Count),
		}

		err = stream.Send(res)
		if err != nil {
			return logError(status.Errorf(codes.Unknown, "cannot send stream response: %v", err))
		}

	}

	return nil
}