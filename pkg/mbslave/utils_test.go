package mbslave

import (
	"fmt"
	"testing"
)

func TestConvUint16s(t *testing.T) {
	uint16Values := []interface{}{true, false, 100, 75536, "100", 3.14, "3.14"}
	for _, value := range uint16Values {
		results, err := convUint16s("uint16", value)
		if err != nil {
			t.Error(err)
		}
		fmt.Println(results)
	}

	float32Values := []interface{}{true, false, 100, 75536, "100", 3.14, "3.14"}
	for _, value := range float32Values {
		results, err := convUint16s("float32", value)
		if err != nil {
			t.Error(err)
		}
		fmt.Println(results)
	}
}
