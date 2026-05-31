package rerank

import (
	"context"
	"slices"

	"github.com/cloudwego/eino/components/document"
	"github.com/cloudwego/eino/schema"
)

// Config 重排序器配置
type Config struct {
	// ScoreFieldKey 指定 metadata 中存储分数的字段名。为 nil 时使用 Document.Score() 方法获取分数。
	ScoreFieldKey *string
}

// NewReranker 创建基于分数的文档重排序器。
func NewReranker(ctx context.Context, config *Config) (document.Transformer, error) {
	var getter func(doc *schema.Document) float64
	if config == nil || config.ScoreFieldKey == nil {
		getter = func(doc *schema.Document) float64 {
			return doc.Score()
		}
	} else {
		key := *config.ScoreFieldKey
		getter = func(doc *schema.Document) float64 {
			if doc.MetaData == nil {
				return 0
			}
			v, ok := doc.MetaData[key]
			if !ok {
				return 0
			}
			vv, okk := v.(float64)
			if !okk {
				return 0
			}
			return vv
		}
	}
	return &scoreReranker{scoreGetter: getter}, nil
}

type scoreReranker struct {
	scoreGetter func(doc *schema.Document) float64
}

func (r *scoreReranker) Transform(_ context.Context, src []*schema.Document, _ ...document.TransformerOption) ([]*schema.Document, error) {
	if len(src) <= 1 {
		return src, nil
	}

	// 复制一份避免修改原切片
	copied := make([]*schema.Document, len(src))
	copy(copied, src)

	// 按分数降序排列
	slices.SortFunc(copied, func(a, b *schema.Document) int {
		if r.scoreGetter(a) < r.scoreGetter(b) {
			return 1
		} else if r.scoreGetter(a) > r.scoreGetter(b) {
			return -1
		}
		return 0
	})

	// 首因/近因交织排列: 高分放两端，低分放中间
	ret := make([]*schema.Document, len(src))
	for i, d := range copied {
		if i%2 == 0 {
			ret[i/2] = d
		} else {
			ret[len(ret)-1-i/2] = d
		}
	}

	return ret, nil
}

func (r *scoreReranker) GetType() string {
	return "ScoreReranker"
}
