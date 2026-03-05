package snapshot

import (
	"time"

	"gitlab.com/distributed_lab/figure/v3"
	"gitlab.com/distributed_lab/kit/kv"
	"gitlab.com/distributed_lab/logan/v3/errors"
)

type Config struct {
	RPC          string        `fig:"rpc,required"`
	Denom        string        `fig:"denom,required"`
	Height       int64         `fig:"height"`
	Output       string        `fig:"output,required"`
	RequestDelay time.Duration `fig:"request_delay,required"`
}

func NewConfig(getter kv.Getter) Config {
	var cfg Config
	err := figure.Out(&cfg).
		From(kv.MustGetStringMap(getter, "snapshot")).
		With(figure.BaseHooks).
		Please()
	if err != nil {
		panic(errors.Wrap(err, "failed to figure out snapshot config"))
	}
	return cfg
}
