package accrual

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"net/http"
)

type Service struct {
	apiUrl     string
	httpClient *http.Client
	logger     zerolog.Logger
}

func (s *Service) LoggerComponent() string {
	return "Accrual.Service"
}

func NewService(apiURL string, opts ...ServiceOption) (*Service, error) {
	c := &Service{
		apiUrl:     apiURL,
		httpClient: http.DefaultClient,
		logger:     log.Logger,
	}

	for _, o := range opts {
		o(c)
	}

	c.logger = c.logger.With().Str("component", c.LoggerComponent()).Logger()

	return c, nil
}

type ServiceOption func(*Service)

func WithLogger(l zerolog.Logger) ServiceOption {
	return func(s *Service) {
		s.logger = l
	}
}

func (s *Service) GetOrder(ctx context.Context, in *GetOrderRequest, out *GetOrderResponse) error {
	l := s.logger.With().
		Str("method", "GetOrder").
		Logger()
	ctx = l.WithContext(ctx)

	err := s.genericCall(ctx, http.MethodGet, fmt.Sprintf("/api/orders/%s", in.ExternalOrderID), nil, out)
	if err != nil {
		return err
	}

	return nil
}

type RemoteError struct {
	ResponseBody string
	StatusCode   int
}

func NewRemoteError(responseBody string, statusCode int) *RemoteError {
	return &RemoteError{ResponseBody: responseBody, StatusCode: statusCode}
}

func (e *RemoteError) Error() string {
	return e.ResponseBody
}

func (s *Service) genericCall(ctx context.Context, method, endpoint string, in interface{}, out interface{}) error {
	l := zerolog.Ctx(ctx).With().Str("http_method", method).Str("endpoint", endpoint).Logger()
	ctx = l.WithContext(ctx)

	res, err := s.request(ctx, method, endpoint, in)
	if err != nil {
		l.Error().Err(err).
			Msg("Service request failed")
		return fmt.Errorf("request: %w", err)
	}

	if res.StatusCode >= 400 {
		resBody := readString(res.Body)
		l.Error().
			Str("http_body", resBody).
			Msg("Service responded with error")
		return NewRemoteError(resBody, res.StatusCode)
	}

	if err := readJson(res.Body, out); err != nil {
		return fmt.Errorf("body read: %w", err)
	}

	return nil
}

func (s *Service) request(
	ctx context.Context,
	method string,
	endpoint string,
	bodyParams interface{},
) (*http.Response, error) {
	fullUrl := s.apiUrl + endpoint
	l := zerolog.Ctx(ctx).With().
		Str("http_method", method).
		Str("endpoint", endpoint).
		Str("url", fullUrl).
		Str("method", method).
		Str("endpoint", endpoint).
		Logger()
	l.Debug().Msg("HTTP request")

	rawJson, err := json.Marshal(bodyParams)
	if err != nil {
		return nil, fmt.Errorf("json encode: %w", err)
	}

	req, err := http.NewRequest(method, fullUrl, bytes.NewReader(rawJson))
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	req = req.WithContext(ctx)

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")

	l.Debug().Str("request_body", string(rawJson)).Msg("Doing request")

	res, err := s.httpClient.Do(req)
	if err != nil {
		l.Error().Err(err).
			Msg("Call failed")
		return nil, fmt.Errorf("do request: %w", err)
	}

	return res, nil
}
