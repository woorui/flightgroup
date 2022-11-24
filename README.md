# flightgroup

`Handler` call `handle` to handle data from `Reader.Read()`, `Close` makes graceful shotdown.

call `group := FlightGroup()`, `group.Close()`, `for range group.ErrChan` in 3 different goroutinue.