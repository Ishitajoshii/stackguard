// Package correlation carries explicit in-process request metadata, never tokens.
package correlation

import "context"

type Boundary struct{ Session, RequestID string }
type key struct{}

func With(ctx context.Context, b Boundary) context.Context { return context.WithValue(ctx, key{}, b) }
func From(ctx context.Context) Boundary                    { v, _ := ctx.Value(key{}).(Boundary); return v }
