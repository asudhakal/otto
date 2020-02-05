package otto

import (
	"context"
	"strings"
)

func (self *_runtime) waitOneTick() {
	if self.otto.Limiter == nil {
		return
	}
	ctx := self.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	if waitErr := self.otto.Limiter.Wait(ctx); waitErr != nil {
		if self.ctx == nil {
			panic(waitErr)
		}
		if ctxErr := self.ctx.Err(); ctxErr != nil {
			panic(ctxErr)
		}
		if strings.Contains(waitErr.Error(), "would exceed") {
			panic(context.DeadlineExceeded)
		}
		panic(waitErr)
	}
}
