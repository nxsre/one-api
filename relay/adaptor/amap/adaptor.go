package amap

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common/client"
	"github.com/songquanpeng/one-api/relay/adaptor"
	"github.com/songquanpeng/one-api/relay/adaptor/openai"
	"github.com/songquanpeng/one-api/relay/channeltype"
	"github.com/songquanpeng/one-api/relay/meta"
	"github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/relaymode"
)

const (
	channelName    = "amap-poi"
	ModelAmapPOI   = "amap-poi"
	defaultBaseURL = "https://restapi.amap.com"
)

type Adaptor struct{}

type poiRequest struct {
	Location   string `json:"location,omitempty"`
	Keywords   string `json:"keywords,omitempty"`
	Types      string `json:"types,omitempty"`
	Radius     string `json:"radius,omitempty"`
	SortRule   string `json:"sortrule,omitempty"`
	Region     string `json:"region,omitempty"`
	CityLimit  string `json:"city_limit,omitempty"`
	ShowFields string `json:"show_fields,omitempty"`
	PageSize   string `json:"page_size,omitempty"`
	PageNum    string `json:"page_num,omitempty"`
}

type amapResponse struct {
	Status   string   `json:"status"`
	Info     string   `json:"info"`
	InfoCode string   `json:"infocode"`
	Count    string   `json:"count"`
	POIs     poiItems `json:"pois"`
}

type poiItems []poi

type poi struct {
	Name     string `json:"name,omitempty"`
	ID       string `json:"id,omitempty"`
	Parent   string `json:"parent,omitempty"`
	Location string `json:"location,omitempty"`
	Distance string `json:"distance,omitempty"`
	Type     string `json:"type,omitempty"`
	TypeCode string `json:"typecode,omitempty"`
	Province string `json:"pname,omitempty"`
	City     string `json:"cityname,omitempty"`
	District string `json:"adname,omitempty"`
	Address  string `json:"address,omitempty"`
	PCode    string `json:"pcode,omitempty"`
	ADCode   string `json:"adcode,omitempty"`
	CityCode string `json:"citycode,omitempty"`
	Children any    `json:"children,omitempty"`
	Business any    `json:"business,omitempty"`
	Indoor   any    `json:"indoor,omitempty"`
	Navi     any    `json:"navi,omitempty"`
	Photos   any    `json:"photos,omitempty"`
}

func (items *poiItems) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*items = nil
		return nil
	}

	var list []poi
	if err := json.Unmarshal(data, &list); err == nil {
		*items = list
		return nil
	}

	var wrapped struct {
		POI []poi `json:"poi"`
	}
	if err := json.Unmarshal(data, &wrapped); err != nil {
		return err
	}
	*items = wrapped.POI
	return nil
}

func (a *Adaptor) Init(_ *meta.Meta) {}

func (a *Adaptor) GetRequestURL(meta *meta.Meta) (string, error) {
	if meta == nil {
		return "", errors.New("amap-poi: meta is nil")
	}
	base := strings.TrimSpace(meta.BaseURL)
	if base == "" && meta.ChannelType > 0 && meta.ChannelType < len(channeltype.ChannelBaseURLs) {
		base = channeltype.ChannelBaseURLs[meta.ChannelType]
	}
	if base == "" {
		base = defaultBaseURL
	}
	return strings.TrimRight(base, "/") + "/v5/place/around", nil
}

func (a *Adaptor) SetupRequestHeader(c *gin.Context, req *http.Request, meta *meta.Meta) error {
	adaptor.SetupCommonRequestHeader(c, req, meta)
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "application/json")
	}
	return nil
}

func (a *Adaptor) ConvertRequest(_ *gin.Context, relayMode int, request *model.GeneralOpenAIRequest) (any, error) {
	if request == nil {
		return nil, errors.New("amap-poi: request is nil")
	}
	if relayMode != relaymode.ChatCompletions {
		return nil, errors.New("amap-poi: only chat completions supported")
	}
	if request.Stream {
		return nil, errors.New("amap-poi: streaming is not supported; set stream to false")
	}
	return request, nil
}

func (a *Adaptor) ConvertImageRequest(*model.ImageRequest) (any, error) {
	return nil, errors.New("amap-poi: not supported")
}

func (a *Adaptor) DoRequest(c *gin.Context, meta *meta.Meta, requestBody io.Reader) (*http.Response, error) {
	if meta == nil {
		return nil, fmt.Errorf("amap-poi: meta is nil")
	}
	if meta.Mode != relaymode.ChatCompletions {
		return nil, fmt.Errorf("amap-poi: only /v1/chat/completions is supported")
	}
	raw, err := io.ReadAll(requestBody)
	if err != nil {
		return nil, err
	}
	var openAIReq model.GeneralOpenAIRequest
	if err = json.Unmarshal(raw, &openAIReq); err != nil {
		return nil, fmt.Errorf("amap-poi: bad request: %w", err)
	}
	params, err := buildPOIRequest(&openAIReq)
	if err != nil {
		return nil, err
	}

	reqURL, err := a.GetRequestURL(meta)
	if err != nil {
		return nil, err
	}
	upstreamReq, err := newAmapRequest(reqURL, apiKey(meta), params)
	if err != nil {
		return nil, err
	}
	if err = a.SetupRequestHeader(c, upstreamReq, meta); err != nil {
		return nil, err
	}

	resp, err := httpClient().Do(upstreamReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return errorResponse(resp.StatusCode, fmt.Sprintf("amap-poi upstream http %d: %s", resp.StatusCode, string(body))), nil
	}

	var amapResp amapResponse
	if err = json.Unmarshal(body, &amapResp); err != nil {
		return nil, fmt.Errorf("amap-poi: parse upstream response: %w", err)
	}
	if amapResp.Status != "1" {
		msg := strings.TrimSpace(amapResp.Info)
		if msg == "" {
			msg = string(body)
		}
		return errorResponse(http.StatusBadGateway, fmt.Sprintf("amap-poi upstream error infocode=%s info=%s", amapResp.InfoCode, msg)), nil
	}

	return okResponse(meta, &openAIReq, amapResp)
}

func (a *Adaptor) DoResponse(c *gin.Context, resp *http.Response, meta *meta.Meta) (*model.Usage, *model.ErrorWithStatusCode) {
	if resp == nil {
		return nil, openai.ErrorWrapper(errors.New("amap-poi: empty response"), "bad_response", http.StatusInternalServerError)
	}
	apiErr, usage := openai.Handler(c, resp, meta.PromptTokens, meta.ActualModelName)
	return usage, apiErr
}

func (a *Adaptor) GetModelList() []string {
	return []string{ModelAmapPOI}
}

func (a *Adaptor) GetChannelName() string {
	return channelName
}

func TestAround(baseURL, key string) error {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	req, err := newAmapRequest(strings.TrimRight(baseURL, "/")+"/v5/place/around", key, poiRequest{
		Location: "116.473168,39.993015",
		Types:    "050000",
		Radius:   "1000",
		PageSize: "1",
		PageNum:  "1",
	})
	if err != nil {
		return err
	}
	resp, err := httpClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("amap-poi test http %d: %s", resp.StatusCode, string(body))
	}
	var parsed amapResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return err
	}
	if parsed.Status != "1" {
		return fmt.Errorf("amap-poi test failed infocode=%s info=%s", parsed.InfoCode, parsed.Info)
	}
	return nil
}

func newAmapRequest(endpoint, key string, params poiRequest) (*http.Request, error) {
	if strings.TrimSpace(key) == "" {
		return nil, fmt.Errorf("amap-poi: channel key is required")
	}
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("key", key)
	q.Set("output", "json")
	setQuery(q, "location", params.Location)
	setQuery(q, "keywords", params.Keywords)
	setQuery(q, "types", params.Types)
	setQuery(q, "radius", defaultIfEmpty(params.Radius, "5000"))
	setQuery(q, "sortrule", defaultIfEmpty(params.SortRule, "distance"))
	setQuery(q, "region", params.Region)
	setQuery(q, "city_limit", params.CityLimit)
	setQuery(q, "show_fields", params.ShowFields)
	setQuery(q, "page_size", defaultIfEmpty(params.PageSize, "10"))
	setQuery(q, "page_num", defaultIfEmpty(params.PageNum, "1"))
	u.RawQuery = q.Encode()
	return http.NewRequest(http.MethodGet, u.String(), nil)
}

func buildPOIRequest(request *model.GeneralOpenAIRequest) (poiRequest, error) {
	params := poiRequest{
		Radius:   "5000",
		SortRule: "distance",
		PageSize: "10",
		PageNum:  "1",
	}
	mergeParams(&params, paramsFromAny(request.Metadata))
	if text := lastUserText(request); text != "" {
		mergeParams(&params, paramsFromText(text))
	}
	if strings.TrimSpace(params.Location) == "" {
		return params, fmt.Errorf("amap-poi: location is required; pass it in metadata or the last user message, e.g. {\"location\":\"116.473168,39.993015\",\"keywords\":\"咖啡\"}")
	}
	return params, nil
}

func paramsFromText(text string) poiRequest {
	text = strings.TrimSpace(text)
	if text == "" {
		return poiRequest{}
	}
	var params poiRequest
	if strings.HasPrefix(text, "{") {
		var raw map[string]any
		if err := json.Unmarshal([]byte(text), &raw); err == nil {
			return paramsFromAny(raw)
		}
	}
	if values, err := url.ParseQuery(strings.TrimPrefix(text, "?")); err == nil && len(values) > 0 && hasKnownPOIKey(values) {
		return paramsFromValues(values)
	}
	params.Keywords = text
	return params
}

func paramsFromAny(v any) poiRequest {
	if v == nil {
		return poiRequest{}
	}
	raw, ok := v.(map[string]any)
	if !ok {
		return poiRequest{}
	}
	if nested, ok := raw["amap_poi"].(map[string]any); ok {
		raw = nested
	}
	return poiRequest{
		Location:   anyString(raw["location"]),
		Keywords:   anyString(raw["keywords"]),
		Types:      anyString(raw["types"]),
		Radius:     anyString(raw["radius"]),
		SortRule:   anyString(raw["sortrule"]),
		Region:     anyString(raw["region"]),
		CityLimit:  anyString(raw["city_limit"]),
		ShowFields: anyString(raw["show_fields"]),
		PageSize:   anyString(raw["page_size"]),
		PageNum:    anyString(raw["page_num"]),
	}
}

func paramsFromValues(values url.Values) poiRequest {
	return poiRequest{
		Location:   values.Get("location"),
		Keywords:   values.Get("keywords"),
		Types:      values.Get("types"),
		Radius:     values.Get("radius"),
		SortRule:   values.Get("sortrule"),
		Region:     values.Get("region"),
		CityLimit:  values.Get("city_limit"),
		ShowFields: values.Get("show_fields"),
		PageSize:   values.Get("page_size"),
		PageNum:    values.Get("page_num"),
	}
}

func mergeParams(dst *poiRequest, src poiRequest) {
	if strings.TrimSpace(src.Location) != "" {
		dst.Location = src.Location
	}
	if strings.TrimSpace(src.Keywords) != "" {
		dst.Keywords = src.Keywords
	}
	if strings.TrimSpace(src.Types) != "" {
		dst.Types = src.Types
	}
	if strings.TrimSpace(src.Radius) != "" {
		dst.Radius = src.Radius
	}
	if strings.TrimSpace(src.SortRule) != "" {
		dst.SortRule = src.SortRule
	}
	if strings.TrimSpace(src.Region) != "" {
		dst.Region = src.Region
	}
	if strings.TrimSpace(src.CityLimit) != "" {
		dst.CityLimit = src.CityLimit
	}
	if strings.TrimSpace(src.ShowFields) != "" {
		dst.ShowFields = src.ShowFields
	}
	if strings.TrimSpace(src.PageSize) != "" {
		dst.PageSize = src.PageSize
	}
	if strings.TrimSpace(src.PageNum) != "" {
		dst.PageNum = src.PageNum
	}
}

func lastUserText(req *model.GeneralOpenAIRequest) string {
	if req == nil {
		return ""
	}
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			return strings.TrimSpace(req.Messages[i].StringContent())
		}
	}
	return ""
}

func okResponse(meta *meta.Meta, request *model.GeneralOpenAIRequest, amapResp amapResponse) (*http.Response, error) {
	content, err := json.MarshalIndent(amapResp, "", "  ")
	if err != nil {
		return nil, err
	}
	modelName := meta.OriginModelName
	if modelName == "" && request != nil {
		modelName = request.Model
	}
	if modelName == "" {
		modelName = ModelAmapPOI
	}
	out := openai.TextResponse{
		Id:      "chatcmpl-amap-poi-" + strconv.FormatInt(time.Now().UnixNano(), 10),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   modelName,
		Choices: []openai.TextResponseChoice{{
			Index: 0,
			Message: model.Message{
				Role:    "assistant",
				Content: string(content),
			},
			FinishReason: "stop",
		}},
	}
	out.Usage = model.Usage{}
	b, err := json.Marshal(out)
	if err != nil {
		return nil, err
	}
	return jsonResponse(http.StatusOK, b), nil
}

func errorResponse(status int, message string) *http.Response {
	b, _ := json.Marshal(map[string]any{
		"error": model.Error{
			Message: message,
			Type:    "upstream_error",
			Code:    "amap_poi_error",
		},
	})
	return jsonResponse(status, b)
}

func jsonResponse(status int, body []byte) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(string(body))),
	}
}

func httpClient() *http.Client {
	if client.HTTPClient != nil {
		return client.HTTPClient
	}
	return http.DefaultClient
}

func apiKey(meta *meta.Meta) string {
	if meta == nil {
		return ""
	}
	if strings.TrimSpace(meta.ChannelKey) != "" {
		return strings.TrimSpace(meta.ChannelKey)
	}
	return strings.TrimSpace(meta.APIKey)
}

func setQuery(values url.Values, key, value string) {
	if strings.TrimSpace(value) != "" {
		values.Set(key, value)
	}
}

func defaultIfEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func hasKnownPOIKey(values url.Values) bool {
	for _, key := range []string{"location", "keywords", "types", "radius", "sortrule", "region", "city_limit", "show_fields", "page_size", "page_num"} {
		if values.Has(key) {
			return true
		}
	}
	return false
}

func anyString(v any) string {
	switch val := v.(type) {
	case string:
		return strings.TrimSpace(val)
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case int:
		return strconv.Itoa(val)
	case int64:
		return strconv.FormatInt(val, 10)
	case bool:
		return strconv.FormatBool(val)
	default:
		return ""
	}
}
