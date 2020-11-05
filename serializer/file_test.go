package serializer

import (
	"github.com/pcbook/api/v1/sample"
	"testing"
)

func TestFileSerializer(t *testing.T) {
	t.Parallel()

	binaryFile := "../tmp/laptop.bin"

	laptop := sample.NewLaptop()
	println("%v, %v", binaryFile, laptop)
}
