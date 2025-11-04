package vectorstore

import (
	"context"
	"fmt"
	"llm-mcp-rag-simple/types"
	"math"
	"sort"
	"sync"
)

// 实现向量存储 ,简单实现
// 基于内存的向量存储
type InMemoryVectorStore struct {
	mu    sync.RWMutex
	items []types.VectorStoreItem
}

func NewInMemoryVectorStore() *InMemoryVectorStore {
	return &InMemoryVectorStore{
		items: make([]types.VectorStoreItem, 0),
	}
}

// 添加向量数据
func (vs *InMemoryVectorStore) AddEmbedding(ctx context.Context, embedding []float64, document string, metadata map[string]interface{}) error {
	if len(embedding) == 0 {
		return fmt.Errorf("embedding 不能为空")
	}
	if document == "" {
		return fmt.Errorf("document 不能为空")
	}
	vs.mu.Lock()
	defer vs.mu.Unlock()

	//创建向量存储项
	item := types.VectorStoreItem{
		Embedding: make([]float64, len(embedding)),
		Document:  document,
		Metadata:  metadata,
	}
	copy(item.Embedding, embedding)
	vs.items = append(vs.items, item)
	return nil
}

// 向量相似度实时，返回最相似的limit个文档
// 计算余弦相似度，按相似度排序
// 返回相似度排序的文档列表
func (vs *InMemoryVectorStore) Search(ctx context.Context, queryEmbedding []float64, limit int) ([]string, error) {
	if len(queryEmbedding) == 0 {
		return nil, fmt.Errorf("query embedding 不存在")
	}
	if limit <= 0 {
		return nil, fmt.Errorf("limit 必须为正数")
	}
	vs.mu.RLock()
	defer vs.mu.RUnlock()
	//储存为空，返回空结果
	if len(vs.items) == 0 {
		return []string{}, nil
	}
	//用于排序
	type scoredItem struct {
		document string
		score    float64
	}
	//计算相似度
	scored := make([]scoredItem, 0, len(vs.items))
	for _, item := range vs.items {
		similarity, err := cosineSimilarity(queryEmbedding, item.Embedding)
		if err != nil {
			//跳过维度不兼容的项，如不同模型生成的向量
			continue
		}
		scored = append(scored, scoredItem{
			document: item.Document,
			score:    similarity,
		})
	}
	//排序
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})
	//如果limit超过总数，返回所有结果
	if limit > len(scored) {
		limit = len(scored)
	}

	result := make([]string, limit)
	for i := 0; i < limit; i++ {
		result[i] = scored[i].document
	}
	return result, nil
}

// 返回存储中向量项总数
func (vs *InMemoryVectorStore) Size() int {
	vs.mu.RLock()
	defer vs.mu.RUnlock()
	return len(vs.items)
}

// 清空内存中所有向量项
func (vs *InMemoryVectorStore) Clear() {
	vs.mu.Lock()
	defer vs.mu.Unlock()
	vs.items = vs.items[:0]
}

// 返回存储中所有文档内容
func (vs *InMemoryVectorStore) GetAllDocuments() []string {
	vs.mu.RLock()
	defer vs.mu.RUnlock()

	documents := make([]string, len(vs.items))

	for i, item := range vs.items {
		documents[i] = item.Document
	}
	return documents
}

// 计算两向量之间余弦相似度
func cosineSimilarity(vecA, VecB []float64) (float64, error) {
	if len(vecA) != len(VecB) {
		return 0, fmt.Errorf("向量必须具有相同维度")
	}
	if len(VecB) == 0 {
		return 0, fmt.Errorf("向量不存在")
	}
	var dotProduct, normA, normB float64

	for i := 0; i < len(vecA); i++ {
		dotProduct += vecA[i] * VecB[i]
		normA += vecA[i] * vecA[i]
		normB += VecB[i] * VecB[i]
	}
	//计算模长
	normA = math.Sqrt(normA)
	normB = math.Sqrt(normB)

	//零向量情况(零向量与任何向量相似度为0)
	if normA == 0 || normB == 0 {
		return 0, nil
	}

	return dotProduct / (normA * normB), nil
}
