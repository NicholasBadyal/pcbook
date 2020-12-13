package service_test

import (
	"bufio"
	"context"
	"fmt"
	pb "github.com/pcbook/api/v1/proto"
	"github.com/pcbook/api/v1/sample"
	"github.com/pcbook/api/v1/serializer"
	"github.com/pcbook/api/v1/service"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"io"
	"net"
	"os"
	"path/filepath"
	"testing"
)

func startTestLaptopServer(t *testing.T, laptopStore service.LaptopStore, imageStore *service.DiskImageStore, ratingStore service.RatingStore) (*service.LaptopServer, string) {
	laptopServer := service.NewLaptopServer(laptopStore, imageStore, ratingStore)

	grpcServer := grpc.NewServer()
	pb.RegisterLaptopServiceServer(grpcServer, laptopServer)

	lis, err := net.Listen("tcp", ":0") // random available port
	require.NoError(t, err)

	go grpcServer.Serve(lis)

	return laptopServer, lis.Addr().String()
}

func newTestLaptopClient(t *testing.T, serverAddr string) pb.LaptopServiceClient {
	conn, err := grpc.Dial(serverAddr, grpc.WithInsecure())
	require.NoError(t, err)
	return pb.NewLaptopServiceClient(conn)
}

func requireSameLaptop(t *testing.T, lhs *pb.Laptop, rhs *pb.Laptop){
	jsonLHS, err := serializer.ProtobufToJSON(lhs)
	require.NoError(t, err)

	jsonRHS, err := serializer.ProtobufToJSON(rhs)
	require.NoError(t, err)

	require.Equal(t, jsonLHS, jsonRHS)
}

func TestClientSearchLaptop(t *testing.T) {
	t.Parallel()

	filter := &pb.Filter{
		MaxPriceUsd: 2000,
		MinCpuCores: 4,
		MinCpuGhz:   2.2,
		MinRam:      &pb.Memory{
			Size: 8,
			Unit: pb.Memory_GIGABYTE,
		},
	}

	store := service.NewInMemoryLaptopStore()
	imageStore := service.NewDiskImageStore("img")
	ratingStore := service.NewInMemoryRatingStore()

	expectedIDs := make(map[string]bool)
	for i := 0; i < 6; i++{
		laptop := sample.NewLaptop()

		switch i {
		case 0:
			laptop.PriceUsd = 2500
		case 1:
			laptop.Cpu.CoreCount = 2
		case 2:
			laptop.Cpu.CoreFrequencyGhz = 2.0
		case 3:
			laptop.Ram = &pb.Memory{Size: 4096, Unit: pb.Memory_MEGABYTE}
		case 4:
			laptop.PriceUsd = 1999
			laptop.Cpu.CoreCount = 4
			laptop.Cpu.CoreFrequencyGhz = 2.5
			laptop.Cpu.BoostFrequencyGhz = laptop.Cpu.CoreFrequencyGhz + 2.0
			laptop.Ram = &pb.Memory{Size: 16, Unit: pb.Memory_GIGABYTE}
			expectedIDs[laptop.Id] = true
		case 5:
			laptop.PriceUsd = 2000
			laptop.Cpu.CoreCount = 6
			laptop.Cpu.CoreFrequencyGhz = 2.8
			laptop.Cpu.BoostFrequencyGhz = laptop.Cpu.CoreFrequencyGhz + 2.0
			laptop.Ram = &pb.Memory{Size: 64, Unit: pb.Memory_GIGABYTE}
			expectedIDs[laptop.Id] = true
		}

		err := store.Save(laptop)
		require.NoError(t, err)
	}

	_, serverAddress := startTestLaptopServer(t, store, imageStore, ratingStore)
	laptopClient := newTestLaptopClient(t, serverAddress)

	req := &pb.SearchLaptopRequest{Filter: filter}
	stream, err := laptopClient.SearchLaptop(context.Background(), req)
	require.NoError(t, err)

	found := 0
	for{
		res, err := stream.Recv()
		if err == io.EOF {
			break
		}

		require.NoError(t, err)
		require.Contains(t, expectedIDs, res.GetLaptop().GetId())

		found += 1
	}

	require.Equal(t, len(expectedIDs), found)


}

func TestClientCreateLaptop(t *testing.T) {
	t.Parallel()

	laptopStore := service.NewInMemoryLaptopStore()
	imageStore := service.NewDiskImageStore("img")
	ratingStore := service.NewInMemoryRatingStore()
	_, serverAddr := startTestLaptopServer(t, laptopStore, imageStore, ratingStore)
	laptopClient := newTestLaptopClient(t, serverAddr)

	laptop := sample.NewLaptop()
	expectedID := laptop.Id
	req := &pb.CreateLaptopRequest{Laptop: laptop}

	res, err := laptopClient.CreateLaptop(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, expectedID, res.Id)

	// check that the laptop is saved to the store
	other, err := laptopStore.Find(res.Id)
	require.NoError(t, err)
	require.NotNil(t, other)

	// check that the saved laptop is the same as the one we sent
	requireSameLaptop(t, laptop, other)
}

func TestClientUploadImage(t *testing.T) {
	t.Parallel()

	laptopStore := service.NewInMemoryLaptopStore()
	imageStore := service.NewDiskImageStore("../img")
	ratingStore := service.NewInMemoryRatingStore()

	laptop := sample.NewLaptop()
	err := laptopStore.Save(laptop)
	require.NoError(t, err)

	_, serverAddress := startTestLaptopServer(t, laptopStore, imageStore, ratingStore)
	laptopClient := newTestLaptopClient(t, serverAddress)

	imagePath := fmt.Sprint("../tmp/laptop.jpg")
	file, err := os.Open(imagePath)
	require.NoError(t, err)
	defer file.Close()

	stream, err := laptopClient.UploadImage(context.Background())
	require.NoError(t, err)

	imageType := filepath.Ext(imagePath)
	req := &pb.UploadImageRequest{
		Data: &pb.UploadImageRequest_Info{
			Info: &pb.ImageInfo{
				LaptopId:  laptop.GetId(),
				ImageType: imageType,
			},
		},
	}

	err = stream.Send(req)
	require.NoError(t, err)

	reader := bufio.NewReader(file)
	buffer := make([]byte, 1024)
	size := 0

	for {
		n, err := reader.Read(buffer)
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		size += n

		req := &pb.UploadImageRequest{
			Data: &pb.UploadImageRequest_ChunkData{
				ChunkData: buffer[:n],
			},
		}

		err = stream.Send(req)
		require.NoError(t, err)
	}

	res, err := stream.CloseAndRecv()
	require.NoError(t, err)
	require.NotZero(t, res.GetId())
	require.EqualValues(t, size, res.GetSize())

	savedImagePath := fmt.Sprintf("../img/%s%s", res.GetId(), imageType)
	require.FileExists(t, savedImagePath)
	require.NoError(t, os.Remove(savedImagePath))
}
