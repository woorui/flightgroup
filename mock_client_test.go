package bunch

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func Test_Mock(t *testing.T) {
	actor := newMockActor(1, 2, 3, 4, 5, 6, 7)

	ctx := context.Background()

	msgch, _ := actor.Read(ctx)

	for m := range msgch {
		actor.Handle(ctx, m)
	}

	actor.HandleErr(errors.New("mock_error"))

	var (
		source = []int{1, 2, 3, 4, 5, 6, 7}
		to     = map[int]int{1: 1, 2: 1, 3: 1, 4: 1, 5: 1, 6: 1, 7: 1}
		errstr = "mock_error"
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
