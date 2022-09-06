package bunch

import (
	"context"
	"reflect"
	"testing"
	"time"
)

func Test_Server(t *testing.T) {
	driver := newMockDriver(1, 2, 3, 4, 5, 6, 7)

	srv := NewServer[int](driver, Context(context.Background()), HandlerNum(3), Timeout(time.Second))

	srv.Serve()

	defer srv.Shutdown(context.Background())

	var (
		source = []int{1, 2, 3, 4, 5, 6, 7}
		to     = map[int]int{1: 1, 2: 1, 3: 1, 4: 1, 5: 1, 6: 1, 7: 1}
		errstr = ""
	)

	if !reflect.DeepEqual(driver.source, source) {
		t.Fatal("source")
	}
	if !reflect.DeepEqual(driver.to, to) {
		t.Fatal("to")
	}
	if !reflect.DeepEqual(driver.errstr, errstr) {
		t.Fatal("errstr")
	}
}
