package common

import (
	"github.com/xtls/xray-core/common/net"
	"golang.org/x/net/context"
	"golang.org/x/time/rate"
	"io"
	"time"
)

const burstLimit = 100 * 1000 * 1000

type RateLimitConn struct {
	net.Conn
	r *Reader
	w *Writer
}
func NewRateLimitConn(conn net.Conn, r *Reader, w *Writer) * RateLimitConn {
	c := RateLimitConn{
		Conn : conn,
		r : r,
		w : w,
	}
	return &c
}
func (c *RateLimitConn) Read(p []byte) (int, error) {
	if c.r != nil {
		return c.r.Read(p)
	}
	return c.Conn.Read(p)
}
//func (c *RateLimitConn) Write(p []byte) (int, error) {
//	if c.w != nil {
//		return c.w.Write(p)
//	}
//	return c.Conn.Write(p)
//}

var LimiterMapper = make(map[string]*rate.Limiter)

func RegisterRateLimited(protocol string, rateBPS uint64) *rate.Limiter {
	rl, ok := LimiterMapper[protocol]
	if !ok {
		rl = rate.NewLimiter(rate.Limit(rateBPS), burstLimit)
		rl.AllowN(time.Now(), burstLimit)
		LimiterMapper[protocol] = rl
	} else {
		limit := float64(rl.Limit()) + float64(rateBPS)
		rl.SetLimit(rate.Limit(limit))
	}
	return rl
}
func LimitConnReader(conn net.Conn, ctx context.Context, rl *rate.Limiter) net.Conn {
	r := NewReaderWithContext(conn, ctx)
	r.limiter = rl
	return NewRateLimitConn(conn, r, nil)
}
type Reader struct {
	r       io.Reader
	limiter *rate.Limiter
	ctx     context.Context
}

type Writer struct {
	w       io.Writer
	limiter *rate.Limiter
	ctx     context.Context
}

// NewReader returns a reader that implements io.Reader with rate limiting.
func NewReader(r io.Reader) *Reader {
	return &Reader{
		r:   r,
		ctx: context.Background(),
	}
}

// NewReaderWithContext returns a reader that implements io.Reader with rate limiting.
func NewReaderWithContext(r io.Reader, ctx context.Context) *Reader {
	return &Reader{
		r:   r,
		ctx: ctx,
	}
}

// NewWriter returns a writer that implements io.Writer with rate limiting.
func NewWriter(w io.Writer) *Writer {
	return &Writer{
		w:   w,
		ctx: context.Background(),
	}
}

// NewWriterWithContext returns a writer that implements io.Writer with rate limiting.
func NewWriterWithContext(w io.Writer, ctx context.Context) *Writer {
	return &Writer{
		w:   w,
		ctx: ctx,
	}
}

// SetRateLimit sets rate limit (bytes/sec) to the reader.
func (s *Reader) SetRateLimit(bytesPerSec float64) {
	s.limiter = rate.NewLimiter(rate.Limit(bytesPerSec), burstLimit)
	s.limiter.AllowN(time.Now(), burstLimit) // spend initial burst
}

// Read reads bytes into p.
func (s *Reader) Read(p []byte) (int, error) {
	if s.limiter == nil {
		return s.r.Read(p)
	}
	n, err := s.r.Read(p)
	if err != nil {
		return n, err
	}
	if err := s.limiter.WaitN(s.ctx, n); err != nil {
		return n, err
	}
	return n, nil
}

// SetRateLimit sets rate limit (bytes/sec) to the writer.
func (s *Writer) SetRateLimit(bytesPerSec float64) {
	s.limiter = rate.NewLimiter(rate.Limit(bytesPerSec), burstLimit)
	s.limiter.AllowN(time.Now(), burstLimit) // spend initial burst
}

// Write writes bytes from p.
func (s *Writer) Write(p []byte) (int, error) {
	if s.limiter == nil {
		return s.w.Write(p)
	}
	n, err := s.w.Write(p)
	if err != nil {
		return n, err
	}
	if err := s.limiter.WaitN(s.ctx, n); err != nil {
		return n, err
	}
	return n, err
}


//
//type MBReader struct {
//	r buf.Reader
//	limiter *rate.Limiter
//	ctx     context.Context
//}
//func (m *MBReader)ReadMultiBuffer() (buf.MultiBuffer, error) {
//	if m.limiter == nil {
//		return m.r.ReadMultiBuffer()
//	}
//	b, err := m.r.ReadMultiBuffer()
//	if err != nil {
//		return b, err
//	}
//	if err = m.limiter.WaitN(m.ctx, int(b.Len())); err != nil {
//		return b, err
//	}
//	return b, nil
//}
//func NewMBReaderWithContext(r buf.Reader, ctx context.Context) *MBReader {
//	return &MBReader{
//		r: r,
//		ctx: ctx,
//	}
//}
//// SetRateLimit sets rate limit (bytes/sec) to the writer.
//func (s *MBReader) SetRateLimit(bytesPerSec float64) {
//	s.limiter = rate.NewLimiter(rate.Limit(bytesPerSec), burstLimit)
//	s.limiter.AllowN(time.Now(), burstLimit) // spend initial burst
//}
//
//type MBWriter struct {
//	w buf.Writer
//	limiter *rate.Limiter
//	ctx     context.Context
//}
//func (m *MBWriter) WriteMultiBuffer(buffer buf.MultiBuffer) error {
//	if m.limiter == nil {
//		return m.w.WriteMultiBuffer(buffer)
//	}
//	n := buffer.Len()
//	err := m.w.WriteMultiBuffer(buffer)
//	if err != nil {
//		return err
//	}
//	if err = m.limiter.WaitN(m.ctx, int(n)); err != nil {
//		return err
//	}
//	return nil
//}
//
//func NewMBWriterWithContext(w buf.Writer, ctx context.Context) *MBWriter {
//	return &MBWriter{
//		w:   w,
//		ctx: ctx,
//	}
//}
//// SetRateLimit sets rate limit (bytes/sec) to the writer.
//func (s *MBWriter) SetRateLimit(bytesPerSec float64) {
//	s.limiter = rate.NewLimiter(rate.Limit(bytesPerSec), burstLimit)
//	s.limiter.AllowN(time.Now(), burstLimit) // spend initial burst
//}