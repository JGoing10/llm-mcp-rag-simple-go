package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"llm-mcp-rag-simple/types"
	"llm-mcp-rag-simple/utils"
	"net/http"
	"time"
)

//文档嵌入，语义检索

type Retriever struct {
	embeddingModel string
	baseURL        string
	apiKey         string
	vectorStore    types.VectorStore //向量存储接口
	httpClient     *http.Client
}

func NewRetriever(embeddingModel, baseURL, apiKey string, vectorStore types.VectorStore) *Retriever {
	return &Retriever{
		embeddingModel: embeddingModel,
		baseURL:        baseURL,
		apiKey:         apiKey,
		vectorStore:    vectorStore,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

//将文档嵌入为向量，并存储

func (r *Retriever) EmbedDocument(ctx context.Context, document string) ([]float64, error) {
	utils.LogTitle("EMBEDDING DOCUMENT")
	embedding, err := r.embed(ctx, document)
	if err != nil {
		return nil, fmt.Errorf("文本向量化失败%w", err)
	}
	err = r.vectorStore.AddEmbedding(ctx, embedding, document, nil)
	if err != nil {
		return nil, fmt.Errorf("存储向量失败%w", err)
	}
	fmt.Printf("文本嵌入成功（维度：%d）\n", len(embedding))

	return embedding, nil
}

// 查询请求向量化
func (r *Retriever) EmbedQuery(ctx context.Context, query string) ([]float64, error) {
	utils.LogTitle("EMBEDDING QUERY")
	embedding, err := r.embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("查询请求向量化失败%w", err)
	}
	fmt.Printf("查询请求向量化成功（维度：%d）\n", len(embedding))
	return embedding, nil
}

// 执行语义实时，返回最相似的limit个文档
func (r *Retriever) Retrieve(ctx context.Context, query string, limit int) ([]string, error) {
	queryEmbedding, err := r.EmbedQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("查询请求向量化失败：%w", err)
	}
	//检索
	result, err := r.vectorStore.Search(ctx, queryEmbedding, limit)
	if err != nil {
		return nil, fmt.Errorf("查询向量失败:%w", err)
	}
	utils.LogTitle("RETRIEVAL RESULTS")
	fmt.Printf("查询到%d份文档\n", len(result))
	fmt.Println(result)
	return result, nil
}

// 使用嵌入模型，将文本向量化
func (r *Retriever) embed(ctx context.Context, text string) ([]float64, error) {
	if text == "" {
		return nil, fmt.Errorf("文本不存在")
	}
	reqBody := types.EmbeddingRequest{
		Model:          r.embeddingModel,
		Input:          text,
		EncodingFormat: "float",
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("请求序列号失败：%w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", r.baseURL+"/embeddings", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建请求错误：%W", err)
	}
	//设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer"+r.apiKey) // BearerToken 认证

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求错误：%w", err)
	}
	defer resp.Body.Close()
	//检查响应状态
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("api 请求错误，错误码：%d,响应信息：%s", resp.StatusCode, body)
	}

	//解析响应
	var embeddingResp types.EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embeddingResp); err != nil {
		return nil, fmt.Errorf("解析响应失败%w", err)
	}

	if len(embeddingResp.Data) == 0 {
		return nil, fmt.Errorf("没有 embedding 数据")
	}

	embedding := embeddingResp.Data[0].Embedding
	if len(embedding) == 0 {
		return nil, fmt.Errorf("embedding为空")
	}

	return embedding, nil
}

// 返回向量数据库中文档的数量
func (r *Retriever) GetVectorStoreSize() int {
	return r.vectorStore.Size()
}
