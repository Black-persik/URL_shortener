package service

import (
	"context"
	"crypto/rand"
	"log"
	"net/url"
	"sync"
	"time"
)

type Repository interface {
	CreateLink(ctx context.Context, code, originalURL string) error
	GetLinkByCode(ctx context.Context, code string) (id int64, originalURL string, err error)
	InsertClicks(ctx context.Context, events []ClickEvent) error
	TotalClicks(ctx context.Context, code string) (int64, error)
}

type ClickMeta struct {
	IP        string
	UserAgent string
}

type ClickEvent struct {
	LinkID    int64
	TS        time.Time
	IP        string
	UserAgent string
}

type Config struct {
	CodeLen int

	ClickQueueSize     int
	ClickWorkers       int
	ClickBatchSize     int
	ClickFlushInterval time.Duration
	ClickWriteTimeout  time.Duration
}

type LinksService interface {
	CreateShortLink(ctx context.Context, originalURL string) (string, error)
	Resolve(ctx context.Context, code string, meta ClickMeta) (string, error)
	TotalClicks(ctx context.Context, code string) (int64, error)
	Shutdown(ctx context.Context) error
}

type linksService struct {
	repo Repository
	cfg  Config
	log  *log.Logger

	clickCh chan ClickEvent
	wg      sync.WaitGroup

	mu     sync.RWMutex
	closed bool
}

func NewLinksService(repo Repository, cfg Config, logger *log.Logger) LinksService {
	if cfg.CodeLen <= 0 {
		cfg.CodeLen = 7
	}
	if cfg.ClickQueueSize <= 0 {
		cfg.ClickQueueSize = 1024
	}
	if cfg.ClickWorkers <= 0 {
		cfg.ClickWorkers = 4
	}
	if cfg.ClickBatchSize <= 0 {
		cfg.ClickBatchSize = 200
	}
	if cfg.ClickFlushInterval <= 0 {
		cfg.ClickFlushInterval = 1 * time.Second
	}
	if cfg.ClickWriteTimeout <= 0 {
		cfg.ClickWriteTimeout = 2 * time.Second
	}
	if logger == nil {
		logger = log.Default()
	}

	s := &linksService{
		repo:    repo,
		cfg:     cfg,
		log:     logger,
		clickCh: make(chan ClickEvent, cfg.ClickQueueSize),
	}

	for i := 0; i < cfg.ClickWorkers; i++ {
		s.wg.Add(1)
		go s.worker()
	}

	return s
}

func (s *linksService) CreateShortLink(ctx context.Context, originalURL string) (string, error) {
	if !isValidURL(originalURL) {
		return "", ErrInvalidURL
	}

	// несколько попыток на случай коллизии по code
	for i := 0; i < 8; i++ {
		code := randomBase62(s.cfg.CodeLen)
		if err := s.repo.CreateLink(ctx, code, originalURL); err != nil {
			if err == ErrConflict {
				continue
			}
			return "", err
		}
		return code, nil
	}

	return "", ErrConflict
}

func (s *linksService) Resolve(ctx context.Context, code string, meta ClickMeta) (string, error) {
	id, originalURL, err := s.repo.GetLinkByCode(ctx, code)
	if err != nil {
		return "", err
	}

	ev := ClickEvent{
		LinkID:    id,
		TS:        time.Now().UTC(),
		IP:        meta.IP,
		UserAgent: meta.UserAgent,
	}

	// best-effort enqueue (не тормозим редирект)
	s.mu.RLock()
	closed := s.closed
	ch := s.clickCh
	s.mu.RUnlock()

	if !closed {
		select {
		case ch <- ev:
		default:
			// очередь заполнена — просто дропаем клик
		}
	}

	return originalURL, nil
}

func (s *linksService) TotalClicks(ctx context.Context, code string) (int64, error) {
	return s.repo.TotalClicks(ctx, code)
}

func (s *linksService) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	if !s.closed {
		s.closed = true
		close(s.clickCh)
	}
	s.mu.Unlock()

	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *linksService) worker() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.cfg.ClickFlushInterval)
	defer ticker.Stop()

	buf := make([]ClickEvent, 0, s.cfg.ClickBatchSize)

	flush := func() {
		if len(buf) == 0 {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), s.cfg.ClickWriteTimeout)
		defer cancel()

		if err := s.repo.InsertClicks(ctx, buf); err != nil {
			s.log.Printf("insert clicks failed: %v", err)
			// можно оставить буфер и попробовать позже, но для минималки — просто очищаем
		}
		buf = buf[:0]
	}

	for {
		select {
		case ev, ok := <-s.clickCh:
			if !ok {
				flush()
				return
			}
			buf = append(buf, ev)
			if len(buf) >= s.cfg.ClickBatchSize {
				flush()
			}

		case <-ticker.C:
			flush()
		}
	}
}

func isValidURL(s string) bool {
	u, err := url.ParseRequestURI(s)
	if err != nil {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	return u.Host != ""
}

const base62Alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

func randomBase62(n int) string {
	// crypto/rand, чтобы не было предсказуемо
	b := make([]byte, n)
	for i := 0; i < n; i++ {
		b[i] = base62Alphabet[cryptoRandInt(len(base62Alphabet))]
	}
	return string(b)
}

func cryptoRandInt(max int) byte {
	// max <= 62, поэтому можно просто так
	rb := make([]byte, 1)
	for {
		_, _ = rand.Read(rb)
		v := int(rb[0])
		if v < 256-(256%max) {
			return byte(v % max)
		}
	}
}
