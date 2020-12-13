package sample

import (
	"github.com/golang/protobuf/ptypes"
	"github.com/google/uuid"
	"github.com/pcbook/api/v1/proto"
	"math"
	"math/rand"
)

// NewKeyboard returns a new sample keyboard
func NewKeyboard() *pb.Keyboard {
	keyboard := &pb.Keyboard{
		Layout: randomKeyboardLayout(),
		Backlit: randomBool(),
	}

	return keyboard
}

func NewCPU() *pb.CPU {
	brand := randomCPUBrand()
	cores := uint32(randomInt(2, 8))
	baseFreq := randomFloat(2.5, 4.0)
	boostFreq := randomFloat(baseFreq, 5.0)

	cpu := &pb.CPU{
		Brand:             brand,
		Name:              randomCPUName(brand),
		CoreCount:         cores,
		ThreadCount:       cores * 2,
		CoreFrequencyGhz:  baseFreq,
		BoostFrequencyGhz: boostFreq,
	}

	return cpu
}

func NewGPU() *pb.GPU {
	brand := randomGPUBrand()
	name := randomGPUName(brand)

	minGhz := randomFloat(1.0, 1.5)
	maxGhz := randomFloat(minGhz, 2.0)
	memGB := randomInt(2, 6)

	gpu := &pb.GPU{
		Brand:             brand,
		Name:              name,
		BaseFrequencyGhz:  minGhz,
		BoostFrequencyGhz: maxGhz,
		Memory:            &pb.Memory{
			Size: uint64(memGB),
			Unit: pb.Memory_GIGABYTE,
		},
	}

	return gpu
}

func NewRam() *pb.Memory {
	ram := &pb.Memory{
		Size: uint64(math.Pow(2, float64(rand.Intn(7)))),
		Unit: pb.Memory_GIGABYTE,
	}

	return ram
}

// NewSSD returns a new sample SSD
func NewSSD() *pb.Storage {
	memGB := randomInt(128, 4)

	ssd := &pb.Storage{
		Driver: pb.Storage_SSD,
		Memory: &pb.Memory{
			Size: uint64(memGB),
			Unit:  pb.Memory_GIGABYTE,
		},
	}

	return ssd
}

// NewHDD returns a new sample HDD
func NewHDD() *pb.Storage {
	memTB := randomInt(1, 3)

	hdd := &pb.Storage{
		Driver: pb.Storage_HDD,
		Memory: &pb.Memory{
			Size: uint64(memTB),
			Unit:  pb.Memory_TERABYTE,
		},
	}

	return hdd
}

func randomScreenResolution() *pb.Screen_Resolution {
	height := randomInt(1080, 4) + 1
	width := height * 16 / 9

	resolution := &pb.Screen_Resolution{
		Width:  uint32(width),
		Height: uint32(height),
	}
	return resolution
}

func randomScreenPanel() pb.Screen_Panel {
	switch rand.Intn(3) {
	case 1:
		return pb.Screen_IPS
	case 2:
		return pb.Screen_TN
	default:
		return pb.Screen_OLED
	}
}

// NewScreen returns a new sample Screen
func NewScreen() *pb.Screen {
	screen := &pb.Screen{
		SizeInch:   randomFloat32(13, 17),
		Resolution: randomScreenResolution(),
		Panel:      randomScreenPanel(),
		Multitouch: randomBool(),
	}

	return screen
}

func randomID() string {
	return uuid.New().String()
}

func randomLaptopBrand() string {
	return randomStringFromSet("Apple", "Dell", "Lenovo")
}

func randomLaptopName(brand string) string {
	switch brand {
	case "Apple":
		return randomStringFromSet("Macbook Air", "Macbook Pro")
	case "Dell":
		return randomStringFromSet("Latitude", "Vostro", "XPS", "Alienware")
	default:
		return randomStringFromSet("Thinkpad X1", "Thinkpad P1", "Thinkpad P53")
	}
}

func NewLaptop() *pb.Laptop {
	brand := randomLaptopBrand()
	name := randomLaptopName(brand)
	years := []uint32{2015, 2016, 2017, 2018, 2019, 2020}

	laptop := &pb.Laptop{
		Id:          randomID(),
		Brand:       brand,
		Name:        name,
		Cpu:         NewCPU(),
		Gpu:         []*pb.GPU{NewGPU()},
		Ram:         NewRam(),
		Storage:     []*pb.Storage{NewSSD(), NewHDD()},
		Screen:      NewScreen(),
		Keyboard:    NewKeyboard(),
		Weight:      &pb.Laptop_WeightKg{WeightKg: float64(randomFloat32(1.0, 3.0))},
		PriceUsd:    float64(randomFloat32(1500, 3500)),
		ReleaseYear: years[rand.Intn(len(years))],
		UpdatedAt:   ptypes.TimestampNow(),
	}

	return laptop
}

