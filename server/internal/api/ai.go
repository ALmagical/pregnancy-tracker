package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"pregnancy-tracker/server/pkg/resp"
)

var aiRiskPattern = regexp.MustCompile(`大出血|胎盘早剥|子痫|抽搐|意识不清|剧烈腹痛|阴道大量出血`)

func (s *Server) postAIChat(c *gin.Context) {
	userID, ok := uid(c)
	if !ok {
		resp.Unauthorized(c, "无效用户")
		return
	}
	var body struct {
		Question string                 `json:"question"`
		Context  map[string]interface{} `json:"context"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		resp.BadRequest(c, "参数错误", "E_PARAM_INVALID", nil)
		return
	}
	q := strings.TrimSpace(body.Question)
	if q == "" {
		resp.BadRequest(c, "请输入问题", "E_PARAM_INVALID", nil)
		return
	}
	if utf8.RuneCountInString(q) > 500 {
		resp.BadRequest(c, "问题过长", "E_PARAM_INVALID", nil)
		return
	}
	if matched := aiRiskPattern.FindString(q); matched != "" {
		resp.Err(c, http.StatusOK, 10042, "如症状严重请尽快就医或拨打急救电话", "E_AI_SAFETY_BLOCKED", gin.H{
			"answer":     "您描述的情况可能属于高风险症状，请立即前往医院急诊或拨打当地急救电话，不要仅依赖线上信息。",
			"sources":    []gin.H{{"title": "孕产期就医指引", "url": ""}},
			"disclaimer": "以上内容仅供参考，如有不适请及时就医",
		})
		return
	}

	if s.Cfg.OpenAIAPIKey == "" {
		resp.OK(c, gin.H{
			"answer":     "当前服务端未配置大模型密钥。可先浏览「知识」栏目中的科普文章获取可靠信息。",
			"sources":    []gin.H{{"title": "胎动计数怎么数", "url": ""}},
			"disclaimer": "以上内容仅供参考，如有不适请及时就医",
		})
		return
	}

	ans, err := s.callOpenAI(q, body.Context)
	if err != nil {
		resp.Internal(c, "AI 服务暂不可用")
		return
	}
	_ = userID
	resp.OK(c, gin.H{
		"answer":     ans,
		"sources":    []gin.H{{"title": "孕期健康科普", "url": ""}},
		"disclaimer": "以上内容仅供参考，如有不适请及时就医",
	})
}

func (s *Server) callOpenAI(question string, ctx map[string]interface{}) (string, error) {
	sys := "你是孕期健康科普助手，回答简短、温暖、非诊断性，结尾提醒遵医嘱。"
	payload := map[string]interface{}{
		"model": s.Cfg.OpenAIModel,
		"messages": []map[string]string{
			{"role": "system", "content": sys},
			{"role": "user", "content": question},
		},
		"temperature": 0.4,
		"max_tokens":  600,
	}
	b, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, "https://api.openai.com/v1/chat/completions", bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+s.Cfg.OpenAIAPIKey)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 45 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 400 {
		return "", fmt.Errorf("openai: %s", string(raw))
	}
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if json.Unmarshal(raw, &out) != nil || len(out.Choices) == 0 {
		return "", fmt.Errorf("parse")
	}
	return strings.TrimSpace(out.Choices[0].Message.Content), nil
}
