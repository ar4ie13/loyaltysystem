package requestor

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/ar4ie13/loyaltysystem/internal/requestor/config"
	"github.com/go-resty/resty/v2"
	"github.com/rs/zerolog"
)

const pollSleepTime = 1 * time.Second

type Requestor struct {
	orders     []string
	conf       config.ReqConf
	zlog       zerolog.Logger
	repo       Repository
	retryAfter int
}

type Repository interface {
	GetUnprocessedOrders(ctx context.Context, limit int) ([]string, error)
	UpdateOrderWithoutAccrual(ctx context.Context, orderNum string, status string) error
	UpdateOrderWithAccrual(ctx context.Context, orderNum string, status string, accrual float64) error
}

func NewRequestor(conf config.ReqConf, zlog zerolog.Logger, repo Repository) *Requestor {
	r := &Requestor{
		orders:     make([]string, 0),
		conf:       conf,
		zlog:       zlog,
		repo:       repo,
		retryAfter: 0,
	}
	go r.StartWorkers()
	return r
}

func (r *Requestor) StartWorkers() {
	for {
		var err error
		wg := &sync.WaitGroup{}
		ctx, cancel := context.WithCancel(context.Background())
		r.orders, err = r.repo.GetUnprocessedOrders(context.Background(), r.conf.WorkerNum)
		if err != nil {
			r.zlog.Error().Err(err).Msg("unable to get unprocessed orders")
		}

		switch {
		case len(r.orders) == 0:
			time.Sleep(pollSleepTime)
			r.zlog.Debug().Msgf("no unprocessed orders, sleeping %v seconds...", pollSleepTime.Seconds())
		default:
			for workerID := 0; workerID < len(r.orders); workerID++ {
				wg.Add(1)
				go r.executeRequestWorker(ctx, wg, workerID, cancel)
			}

			wg.Wait()
			cancel()
			retryAfter := time.Duration(r.retryAfter) * time.Second
			if retryAfter > 0 {
				r.zlog.Debug().Msgf("too many requests, sleeping for %v seconds...", retryAfter.Seconds())
				time.Sleep(retryAfter)
			} else {
				r.zlog.Debug().Msgf("workers finished, sleeping %v seconds...", pollSleepTime.Seconds())
				time.Sleep(pollSleepTime)
			}
		}
	}
}

func (r *Requestor) executeRequestWorker(ctx context.Context, wg *sync.WaitGroup, id int, cancel context.CancelFunc) {
	defer wg.Done()
	select {
	case <-ctx.Done():
		r.zlog.Debug().Msgf("worker %d cancelled while processing order %s", id, r.orders[id])
		return
	default:
	}
	r.zlog.Debug().Msgf("worker %d processing order %s", id, r.orders[id])

	client := resty.New()
	resp, err := client.R().Get(r.conf.AccrualAddr + "/api/orders/" + r.orders[id])
	if err != nil {
		r.zlog.Err(err).Msgf("worker %d unable to process order %s", id, r.orders[id])
		return
	}

	switch resp.StatusCode() {
	case http.StatusOK:
		if resp.IsSuccess() {
			var accrualResponse AccrualResponse
			if err = json.Unmarshal(resp.Body(), &accrualResponse); err != nil {
				r.zlog.Err(err).Msg("unable to unmarshal accrual response")
			}
			var accrual float64
			if accrualResponse.Accrual == nil {
				err = r.repo.UpdateOrderWithoutAccrual(ctx, accrualResponse.OrderNumber, accrualResponse.Status)
				if err != nil {
					r.zlog.Err(err).Msg("unable to update order")
					return
				}
			} else {
				accrual = *accrualResponse.Accrual
			}
			err = r.repo.UpdateOrderWithAccrual(ctx, accrualResponse.OrderNumber, accrualResponse.Status, accrual)
			if err != nil {
				r.zlog.Err(err).Msg("unable to update order")
				return
			}
		}

		r.zlog.Debug().Msgf("worker %d processes order %s", id, r.orders[id])
	case http.StatusNoContent:
		r.zlog.Debug().Msgf("worker %d: order %s wasn't found in accrual", id, r.orders[id])
		return
	case http.StatusTooManyRequests:
		if sleepTimeStr := resp.Header().Get("Retry-After"); sleepTimeStr != "" {
			r.retryAfter, err = strconv.Atoi(sleepTimeStr)
			if err != nil {
				r.zlog.Err(err).Msgf("unable to parse retry after %s", sleepTimeStr)
				return
			}
		}
		r.zlog.Debug().Msgf("worker %d found %d Status Code. Cancelling all workers", resp.StatusCode(), id)
		cancel()
		return
	default:
		r.zlog.Err(err).Msgf("accrual service returned status %d", resp.StatusCode())
		return
	}
}
