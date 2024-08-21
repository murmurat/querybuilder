package graphql

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
)

type Client struct {
	url        string
	httpClient *http.Client
}

func NewClient(url string) *Client {
	c := &Client{
		url: url,
	}
	if c.httpClient == nil {
		c.httpClient = http.DefaultClient
	}
	return c
}

type Request struct {
	q      string
	vars   map[string]interface{}
	Header http.Header
}

func NewRequest(q string) *Request {
	req := &Request{
		q:      q,
		Header: make(map[string][]string),
	}
	return req
}

func (req *Request) Var(key string, value interface{}) {
	if req.vars == nil {
		req.vars = make(map[string]interface{})
	}
	req.vars[key] = value
}

func (c *Client) Run(ctx context.Context, req *Request, resp interface{}) error {
	var requestBody bytes.Buffer
	requestBodyObj := struct {
		Query     string                 `json:"query"`
		Variables map[string]interface{} `json:"variables"`
	}{
		Query:     req.q,
		Variables: req.vars,
	}

	if err := json.NewEncoder(&requestBody).Encode(requestBodyObj); err != nil {
		return err
	}

	r, err := http.NewRequest(http.MethodPost, c.url, &requestBody)
	if err != nil {
		return err
	}
	r.Header.Set("Content-Type", "application/json; charset=utf-8")
	r.Header.Set("Accept", "application/json; charset=utf-8")
	for key, values := range req.Header {
		for _, value := range values {
			r.Header.Add(key, value)
		}
	}

	r = r.WithContext(ctx)
	res, err := c.httpClient.Do(r)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, res.Body); err != nil {
		return err
	}

	if err := json.NewDecoder(&buf).Decode(&resp); err != nil {
		return err
	}
	return nil
}

// BuildQuery builds GraphQl query for a struct using "gql" tag, ex:
/*
query ($limit: Int, $contract_filter: ContractFiltersInput!) {
	Contract(limit: $limit, filter: $filter) {
		id
		name
		description
		ContractUnits {
			id
			name
			description
		}
	}
}
*/
func BuildQuery(objectType reflect.Type, filterType string) string {
	filter, fconst := "", ""
	if filterType != "" {
		filter = fmt.Sprintf(", $filter: %s!", filterType)
		fconst = fmt.Sprintf(", filter: $filter")
	}

	fields := getStructFields(objectType, 2)
	query := fmt.Sprintf(`
query ($after: Int, $limit: Int%s) {
	%s(after: $after, limit: $limit%s) {
%s
	}
}`, filter, objectType.Name(), fconst, fields)

	return query
}

func getStructFields(t reflect.Type, indentLevel int) string {
	var fields []string
	indent := strings.Repeat("\t", indentLevel)
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if tag := field.Tag.Get("gql"); tag != "" {
			switch field.Type.Kind() {

			case reflect.Slice:
				if field.Type.Elem().Kind() == reflect.Struct {
					// null.String, null.Int, etc.
					if strings.HasPrefix(field.Type.Elem().String(), "null.") {
						fields = append(fields, fmt.Sprintf("%s%s", indent, tag))
						continue
					}
					nestedFields := getStructFields(field.Type.Elem(), indentLevel+1)
					fields = append(fields, fmt.Sprintf("%s%s {\n%s%s\n%s}", indent, tag, nestedFields, indent, indent))
				} else {
					fields = append(fields, fmt.Sprintf("%s%s", indent, tag))
				}

			case reflect.Struct:
				// null.String, null.Int, etc.
				if strings.HasPrefix(field.Type.String(), "null.") {
					fields = append(fields, fmt.Sprintf("%s%s", indent, tag))
					continue
				}

				nestedFields := getStructFields(field.Type, indentLevel+1)
				fields = append(fields, fmt.Sprintf("%s%s {\n%s%s\n%s}", indent, tag, nestedFields, indent, indent))

			default:
				fields = append(fields, fmt.Sprintf("%s%s", indent, tag))
			}
		}
	}
	return strings.Join(fields, "\n")
}
