package gramgo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"reflect"
	"strings"

	"github.com/OhMyDitzzy/gramgo/types"
)

func (b *GramGoBot) rawRequest(ctx context.Context, method string, params any, result any) error {
	url := b.apiURL + "/" + method

	req, err := b.buildRequest(ctx, url, params)
	if err != nil {
		return fmt.Errorf("failed to build request for %s: %w", method, err)
	}

	resp, err := b.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request for %s: %w", method, err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			fmt.Printf("warning: failed to close response body: %v\n", closeErr)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response for %s: %w", method, err)
	}

	var apiResp types.APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return fmt.Errorf("failed to parse response for %s: %w (body: %s)", method, err, body)
	}

	if !apiResp.Ok {
		return b.handleAPIError(method, &apiResp)
	}

	if result != nil && len(apiResp.Result) > 0 {
		if err := json.Unmarshal(apiResp.Result, result); err != nil {
			return fmt.Errorf("failed to parse result for %s: %w", method, err)
		}
	}

	return nil
}

func (b *GramGoBot) buildRequest(ctx context.Context, url string, params any) (*http.Request, error) {
	// Check if we need multipart (for file uploads)
	if shouldUseMultipart(params) {
		return b.buildMultipartRequest(ctx, url, params)
	}
	return b.buildJSONRequest(ctx, url, params)
}

func (b *GramGoBot) buildJSONRequest(ctx context.Context, url string, params any) (*http.Request, error) {
	var body io.Reader

	if params != nil && !isNilOrEmpty(params) {
		data, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal params: %w", err)
		}
		body = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func (b *GramGoBot) buildMultipartRequest(ctx context.Context, url string, params any) (*http.Request, error) {
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	go func() {
		defer pw.Close()
		defer writer.Close()

		if err := b.writeFormFields(writer, params); err != nil {
			pw.CloseWithError(fmt.Errorf("failed to write form fields: %w", err))
			return
		}
	}()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, pr)
	if err != nil {
		pr.Close()
		return nil, err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req, nil
}

func (b *GramGoBot) writeFormFields(writer *multipart.Writer, params any) error {
	if params == nil {
		return nil
	}

	v := reflect.ValueOf(params)
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return fmt.Errorf("params must be a struct, got %s", v.Kind())
	}

	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		if !field.CanInterface() {
			continue
		}

		jsonTag := fieldType.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		fieldName, omitEmpty := parseJSONTag(jsonTag)

		if omitEmpty && isZeroValue(field) {
			continue
		}

		if err := b.writeFormField(writer, fieldName, field); err != nil {
			return fmt.Errorf("failed to write field %s: %w", fieldName, err)
		}
	}

	return nil
}

func (b *GramGoBot) writeFormField(writer *multipart.Writer, fieldName string, field reflect.Value) error {
	if field.Kind() == reflect.Interface && field.IsNil() {
		return nil
	}

	if field.Kind() == reflect.Interface {
		field = field.Elem()
	}

	if field.Kind() == reflect.Ptr && !field.IsNil() {
		if fileUpload, ok := field.Interface().(*types.InputFileUpload); ok {
			return b.writeFileUpload(writer, fieldName, fileUpload)
		}
		if fileString, ok := field.Interface().(*types.InputFileString); ok {
			return writer.WriteField(fieldName, fileString.Data)
		}
	}

	switch field.Kind() {
	case reflect.String:
		return writer.WriteField(fieldName, field.String())

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return writer.WriteField(fieldName, fmt.Sprintf("%d", field.Int()))

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return writer.WriteField(fieldName, fmt.Sprintf("%d", field.Uint()))

	case reflect.Float32, reflect.Float64:
		return writer.WriteField(fieldName, fmt.Sprintf("%f", field.Float()))

	case reflect.Bool:
		return writer.WriteField(fieldName, fmt.Sprintf("%t", field.Bool()))

	case reflect.Slice, reflect.Array, reflect.Map, reflect.Struct, reflect.Ptr:
		data, err := json.Marshal(field.Interface())
		if err != nil {
			return fmt.Errorf("failed to marshal field: %w", err)
		}
		
		data = bytes.Trim(data, "\"")
		return writer.WriteField(fieldName, string(data))

	default:
		return fmt.Errorf("unsupported field type: %s", field.Kind())
	}
}

func (b *GramGoBot) writeFileUpload(writer *multipart.Writer, fieldName string, file *types.InputFileUpload) error {
	if file.Data == nil {
		return fmt.Errorf("file data is nil for field %s", fieldName)
	}

	part, err := writer.CreateFormFile(fieldName, file.Filename)
	if err != nil {
		return fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := io.Copy(part, file.Data); err != nil {
		return fmt.Errorf("failed to copy file data: %w", err)
	}

	return nil
}

func shouldUseMultipart(params any) bool {
	if params == nil {
		return false
	}

	v := reflect.ValueOf(params)
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return false
		}
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return false
	}

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		
		if !field.CanInterface() {
			continue
		}

		if field.Kind() == reflect.Ptr && !field.IsNil() {
			if _, ok := field.Interface().(*types.InputFileUpload); ok {
				return true
			}
		}

		if field.Kind() == reflect.Interface && !field.IsNil() {
			elem := field.Elem()
			if elem.Kind() == reflect.Ptr && !elem.IsNil() {
				if _, ok := elem.Interface().(*types.InputFileUpload); ok {
					return true
				}
			}
		}
	}

	return false
}

func (b *GramGoBot) handleAPIError(method string, resp *types.APIResponse) error {
	baseErr := &APIError{
		Code:        resp.ErrorCode,
		Description: resp.Description,
		Parameters:  resp.Parameters,
	}

	switch resp.ErrorCode {
	case http.StatusBadRequest: // 400
		baseErr.Description = fmt.Sprintf("bad request for %s: %s", method, resp.Description)
	case http.StatusUnauthorized: // 401
		baseErr.Description = fmt.Sprintf("unauthorized for %s: %s (check your bot token)", method, resp.Description)
	case http.StatusForbidden: // 403
		baseErr.Description = fmt.Sprintf("forbidden for %s: %s", method, resp.Description)
	case http.StatusNotFound: // 404
		baseErr.Description = fmt.Sprintf("not found for %s: %s", method, resp.Description)
	case http.StatusConflict: // 409
		baseErr.Description = fmt.Sprintf("conflict for %s: %s", method, resp.Description)
	case http.StatusTooManyRequests: // 429
		baseErr.Description = fmt.Sprintf("rate limited for %s: %s", method, resp.Description)
	default:
		baseErr.Description = fmt.Sprintf("API error for %s (%d): %s", method, resp.ErrorCode, resp.Description)
	}

	return baseErr
}

func parseJSONTag(tag string) (name string, omitEmpty bool) {
	parts := strings.Split(tag, ",")
	name = parts[0]
	
	for i := 1; i < len(parts); i++ {
		if parts[i] == "omitempty" {
			omitEmpty = true
			break
		}
	}
	
	return name, omitEmpty
}

func isZeroValue(v reflect.Value) bool {
	if !v.IsValid() {
		return true
	}
	
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	}
	
	return false
}

func isNilOrEmpty(i any) bool {
	if i == nil {
		return true
	}
	
	v := reflect.ValueOf(i)
	switch v.Kind() {
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	case reflect.Slice, reflect.Map:
		return v.Len() == 0
	}
	
	return false
}