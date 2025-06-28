package browser

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/usadamasa/orm-discovery-mcp-go/browser/generated/api"
)

// DefaultFilterQuery Default question request parameters based on the provided JSON specification
const (
	DefaultFilterQuery = "(type:book OR type:video OR type:article) AND language:(\"en\" OR \"EN\" OR \"en-au\" OR \"en-gb\" OR \"en-GB\" OR \"en-us\" OR \"en-US\") AND ( NOT custom_attributes.required_p_permissions:aia ) AND ( NOT custom_attributes.required_p_permissions:cldsc ) AND ( NOT custom_attributes.required_p_permissions:cprex ) AND ( NOT custom_attributes.required_p_permissions:lvtrg ) AND ( NOT custom_attributes.required_p_permissions:ntbks ) AND ( NOT custom_attributes.required_p_permissions:scnrio )"
)

var (
	DefaultSourceFields = []string{
		"custom_attributes.ourn",
		"custom_attributes.publishers",
		"custom_attributes.marketing_type*",
		"custom_attributes.required_p_permissions",
		"url",
		"cover_image",
		"authors",
		"html",
	}

	DefaultRelatedResourceFields = []string{
		"custom_attributes.ourn",
		"custom_attributes.publishers",
		"custom_attributes.marketing_type*",
		"custom_attributes.required_p_permissions",
		"url",
		"cover_image",
		"authors",
		"html",
	}
)

// createQuestionRequest creates a question request with default parameters
func createQuestionRequest(question string) QuestionRequest {
	return QuestionRequest{
		Question:              question,
		FilterQuery:           DefaultFilterQuery,
		SourceFields:          DefaultSourceFields,
		RelatedResourceFields: DefaultRelatedResourceFields,
		PipelineConfig: PipelineConfig{
			SnippetLength:   500,
			HighlightLength: 200,
		},
	}
}

// SubmitQuestion submits a question to O'Reilly Answers and returns the question ID
// NOTE: This functionality has not been fully tested in production
func (bc *BrowserClient) SubmitQuestion(question string) (*QuestionResponse, error) {
	slog.Info("質問を送信します", "question", question)

	// Create OpenAPI client with answers-specific referer
	client := &api.ClientWithResponses{
		ClientInterface: &api.Client{
			Server:         APIEndpointBase,
			Client:         bc.httpClient,
			RequestEditors: []api.RequestEditorFn{bc.CreateRequestEditorWithReferer("https://learning.oreilly.com/answers2/")},
		},
	}

	// Create a question request
	questionReq := createQuestionRequest(question)

	slog.Debug("OpenAPI client経由で質問を送信中", "question", question)

	// Convert to API request format
	apiRequest := api.QuestionRequest{
		Question:          questionReq.Question,
		Fq:                questionReq.FilterQuery,
		SourceFl:          questionReq.SourceFields,
		RelatedResourceFl: questionReq.RelatedResourceFields,
		PipelineConfig: api.PipelineConfig{
			SnippetLength:   &questionReq.PipelineConfig.SnippetLength,
			HighlightLength: &questionReq.PipelineConfig.HighlightLength,
		},
	}

	// Submit question
	resp, err := client.SubmitQuestionWithResponse(context.Background(), apiRequest)
	if err != nil {
		return nil, fmt.Errorf("質問送信失敗: %w", err)
	}

	// Check response status
	if resp.HTTPResponse.StatusCode != 201 {
		return nil, fmt.Errorf("質問送信がステータス%dで失敗", resp.HTTPResponse.StatusCode)
	}

	if resp.JSON201 == nil {
		return nil, fmt.Errorf("有効なJSON応答を受信できませんでした")
	}

	// Convert generated response to our type
	result := &QuestionResponse{
		QuestionID: safeStringValue(resp.JSON201.QuestionId),
		Status:     safeStringValue(resp.JSON201.Status),
		Message:    safeStringValue(resp.JSON201.Message),
	}

	slog.Info("質問が正常に送信されました", "question_id", result.QuestionID)
	return result, nil
}

// GetAnswer retrieves the answer for a submitted question
// NOTE: This functionality has not been fully tested in production
func (bc *BrowserClient) GetAnswer(questionID string, includeUnfinished bool) (*AnswerResponse, error) {
	slog.Debug("回答を取得中", "question_id", questionID)

	// Create OpenAPI client with answers-specific referer
	client := &api.ClientWithResponses{
		ClientInterface: &api.Client{
			Server:         APIEndpointBase,
			Client:         bc.httpClient,
			RequestEditors: []api.RequestEditorFn{bc.CreateRequestEditorWithReferer("https://learning.oreilly.com/answers2/")},
		},
	}

	// Create parameters
	params := &api.GetAnswerParams{
		IncludeUnfinished: &includeUnfinished,
	}

	// Get answer
	resp, err := client.GetAnswerWithResponse(context.Background(), questionID, params)
	if err != nil {
		return nil, fmt.Errorf("回答取得失敗: %w", err)
	}

	// Check response status
	if resp.HTTPResponse.StatusCode != 200 {
		return nil, fmt.Errorf("回答取得がステータス%dで失敗", resp.HTTPResponse.StatusCode)
	}

	if resp.JSON200 == nil {
		return nil, fmt.Errorf("有効なJSON応答を受信できませんでした")
	}

	// Convert generated response to our type
	result := &AnswerResponse{
		QuestionID: safeStringValue(resp.JSON200.QuestionId),
		IsFinished: safeBoolValue(resp.JSON200.IsFinished),
		MisoResponse: MisoResponse{
			Data: convertAnswerData(resp.JSON200.MisoResponse.Data),
		},
	}

	if result.IsFinished {
		slog.Info("回答が完了しました", "question_id", questionID)
	} else {
		slog.Debug("回答はまだ生成中です", "question_id", questionID)
	}

	return result, nil
}

// convertAnswerData converts the generated AnswerData to our type
func convertAnswerData(data *api.AnswerData) AnswerData {
	if data == nil {
		return AnswerData{}
	}

	result := AnswerData{
		Answer:            safeStringValue(data.Answer),
		FollowupQuestions: safeStringSliceValue(data.FollowupQuestions),
	}

	// Convert sources
	if data.Sources != nil {
		for _, source := range *data.Sources {
			result.Sources = append(result.Sources, AnswerSource{
				Title:      safeStringValue(source.Title),
				URL:        safeStringValue(source.Url),
				Authors:    safeStringSliceValue(source.Authors),
				CoverImage: safeStringValue(source.CoverImage),
				Excerpt:    safeStringValue(source.Excerpt),
			})
		}
	}

	// Convert related resources
	if data.RelatedResources != nil {
		for _, resource := range *data.RelatedResources {
			result.RelatedResources = append(result.RelatedResources, RelatedResource{
				Title:       safeStringValue(resource.Title),
				URL:         safeStringValue(resource.Url),
				Authors:     safeStringSliceValue(resource.Authors),
				ContentType: safeStringValue(resource.ContentType),
			})
		}
	}

	// Convert affiliation products
	if data.AffiliationProducts != nil {
		for _, product := range *data.AffiliationProducts {
			result.AffiliationProducts = append(result.AffiliationProducts, AffiliationProduct{
				ProductID:   safeStringValue(product.ProductId),
				Title:       safeStringValue(product.Title),
				URL:         safeStringValue(product.Url),
				Authors:     safeStringSliceValue(product.Authors),
				ContentType: safeStringValue(product.ContentType),
			})
		}
	}

	return result
}

// AskQuestion asks a question and polls for the answer until completion
// NOTE: This functionality has not been fully tested in production
func (bc *BrowserClient) AskQuestion(question string, maxWaitTime time.Duration) (*AnswerResponse, error) {
	slog.Info("質問を開始します", "question", question)

	// Submit question
	questionResp, err := bc.SubmitQuestion(question)
	if err != nil {
		return nil, fmt.Errorf("質問送信失敗: %w", err)
	}

	// Poll for answer
	start := time.Now()
	pollInterval := 2 * time.Second
	maxInterval := 10 * time.Second

	for {
		// Check timeout
		if time.Since(start) > maxWaitTime {
			return nil, fmt.Errorf("タイムアウト: %v経過後も回答が完了しませんでした", maxWaitTime)
		}

		// Get answer
		answer, err := bc.GetAnswer(questionResp.QuestionID, true)
		if err != nil {
			slog.Warn("回答取得エラー（リトライ中）", "error", err)
			time.Sleep(pollInterval)
			continue
		}

		// Check if finished
		if answer.IsFinished {
			slog.Info("質問への回答が完了しました", "question", question)
			return answer, nil
		}

		// Wait before next poll
		slog.Debug("回答生成中...", "elapsed_time", time.Since(start).Round(time.Second))
		time.Sleep(pollInterval)

		// Gradually increase poll interval to reduce load
		if pollInterval < maxInterval {
			pollInterval = time.Duration(float64(pollInterval) * 1.2)
			if pollInterval > maxInterval {
				pollInterval = maxInterval
			}
		}
	}
}

// GetQuestionByID retrieves a previously asked question and its answer
// NOTE: This functionality has not been fully tested in production
func (bc *BrowserClient) GetQuestionByID(questionID string) (*AnswerResponse, error) {
	slog.Debug("質問IDで回答を取得", "question_id", questionID)
	return bc.GetAnswer(questionID, true)
}

// Helper functions for safe type conversion from API pointer types
func safeStringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func safeBoolValue(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

func safeStringSliceValue(s *[]string) []string {
	if s == nil {
		return []string{}
	}
	return *s
}
