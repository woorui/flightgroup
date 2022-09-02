package bunch

import (
	"context"
	"reflect"
	"testing"
	"time"
)

func Test_Server(t *testing.T) {
	actor := newMockActor(1, 2, 3, 4, 5, 6, 7)

	srv := NewServer[int](actor, Context(context.Background()), HandlerNum(3), Timeout(time.Second))

	srv.Serve()

	defer srv.Shutdown(context.Background())

	var (
		source = []int{1, 2, 3, 4, 5, 6, 7}
		to     = map[int]int{1: 1, 2: 1, 3: 1, 4: 1, 5: 1, 6: 1, 7: 1}
		errstr = ""
	)

	if !reflect.DeepEqual(actor.source, source) {
		t.Fatal("source")
	}
	if !reflect.DeepEqual(actor.to, to) {
		t.Fatal("to")
	}
	if !reflect.DeepEqual(actor.errstr, errstr) {
		t.Fatal("errstr")
	}
}
