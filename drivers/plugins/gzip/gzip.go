package gzip

import (
	"bytes"
	"compress/gzip"
	"github.com/eolinker/eosc"
	http_service "github.com/eolinker/eosc/http-service"
	"strings"
)

type Gzip struct {
	*Driver
	id   string
	name string
	conf *Config
}

func (g *Gzip) DoFilter(ctx http_service.IHttpContext, next http_service.IChain) (err error) {
	head := ctx.Request().Header().GetHeader("Accept-encoding")
	if next != nil {
		err = next.DoChain(ctx)
	}
	if err == nil && strings.Contains(head,"gzip") {
		err = g.doCompress(ctx)
	}
	return
}

func (g *Gzip) doCompress(ctx http_service.IHttpContext) error {
	flag := false
	resp := ctx.Response()
	if resp.BodyLen() < g.conf.MinLength {
		// 小于要求的最低长度，不压缩
		return nil
	}
	contentType := resp.GetHeader("Content-Type")
	if len(g.conf.Types) == 0 {
		flag = true
	}else {
		for _, t := range g.conf.Types {
			if strings.Contains(contentType, t) {
				flag = true
				break
			}
		}
	}
	if flag {
		res, err := g.compress(resp.GetBody())
		if err != nil {
			return err
		}
		resp.SetBody(res)
		resp.SetHeader("Content-Encoding", "gzip")
		if g.conf.Vary {
			resp.SetHeader("Vary","Accept-Encoding")
		}
	}
	return nil
}
func (g *Gzip) compress(content []byte) ([]byte, error) {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	defer zw.Close()
	_, err := zw.Write(content)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (g *Gzip) Destroy() {
	g.conf = nil
}

func (g *Gzip) Id() string {
	return g.id
}

func (g *Gzip) Start() error {
	return nil
}

func (g *Gzip) Reset(conf interface{}, workers map[eosc.RequireId]interface{}) error {
	cfg, err := g.check(conf)
	if err != nil {
		return err
	}
	g.conf = cfg
	return nil
}

func (g *Gzip) Stop() error {
	return nil
}

func (g *Gzip) CheckSkill(skill string) bool {
	return http_service.FilterSkillName == skill
}
