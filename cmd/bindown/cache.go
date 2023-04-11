package main

type cacheCmd struct {
	Clear cacheClearCmd `kong:"cmd,help='clear the cache'"`
}

type cacheClearCmd struct{}

func (c *cacheClearCmd) Run(ctx *runContext) error {
	config, err := loadConfigFile(ctx, true)
	if err != nil {
		return err
	}
	return config.ClearCache()
}
