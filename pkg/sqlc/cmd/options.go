package cmd

import (
	"io"

	"github.com/tkcrm/pgxgen/pkg/sqlc/config"
)

type Options struct {
	Env          Env
	Stderr       io.Writer
	MutateConfig func(*config.Config)
}

func (o *Options) ReadConfig(dir, filename string) (string, *config.Config, error) {
	path, conf, err := readConfig(o.Stderr, dir, filename)
	if err != nil {
		return path, conf, err
	}
	if o.MutateConfig != nil {
		o.MutateConfig(conf)
	}
	return path, conf, nil
}
