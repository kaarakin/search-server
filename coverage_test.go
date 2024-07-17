package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type TestServer struct {
	server *httptest.Server
	client SearchClient
}

var (
	accessToken = "qwerty12345"
)

func newTestServer(accessToken string) TestServer {
	server := httptest.NewServer(http.HandlerFunc(SearchServer))
	client := SearchClient{accessToken, server.URL}

	return TestServer{server, client}
}

func (ts *TestServer) Close() {
	ts.server.Close()
}

func TestBadAccessToken(t *testing.T) {
	ts := newTestServer(accessToken + "invalid")
	defer ts.Close()

	_, err := ts.client.FindUsers(SearchRequest{})

	if err.Error() != "bad AccessToken" {
		t.Errorf("Invalid error: %v", err.Error())
	}
}

func TestOpenFile(t *testing.T) {
	originalFilePath := "dataset.xml"
	fileName = "invalid.xml" // Путь к тестовому файлу
	defer func() { fileName = originalFilePath }()

	ts := newTestServer(accessToken)
	defer ts.Close()

	_, err := ts.client.FindUsers(SearchRequest{})

	if err.Error() != "SearchServer fatal error" {
		t.Errorf("Invalid error: %v", err.Error())
	}
}

func TestTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(1500 * time.Millisecond)
	}))
	defer server.Close()
	client := SearchClient{accessToken, server.URL}

	_, err := client.FindUsers(SearchRequest{})

	if !strings.Contains(err.Error(), "timeout for") {
		t.Errorf("Invalid error: %v", err.Error())
	}
}

func TestUnknownError(t *testing.T) {
	client := SearchClient{accessToken, "http://invalid/"}

	_, err := client.FindUsers(SearchRequest{})

	if !strings.Contains(err.Error(), "unknown error") {
		t.Errorf("Invalid error: %v", err.Error())
	}
}

func TestUnknownBadRequestError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		jsonStr := `{ "Error": "something" }`
		_, err := w.Write([]byte(jsonStr))
		if err != nil {
			t.Errorf("Invalid error: %v", err.Error())
		}
	}))
	defer server.Close()
	client := SearchClient{accessToken, server.URL}

	_, err := client.FindUsers(SearchRequest{})

	if !strings.Contains(err.Error(), "unknown bad request error") {
		t.Errorf("Invalid error: %v", err.Error())
	}
}

func TestFatalError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "SearchServer fatal error", http.StatusInternalServerError)
	}))
	defer server.Close()
	client := SearchClient{accessToken, server.URL}

	_, err := client.FindUsers(SearchRequest{})

	if err.Error() != "SearchServer fatal error" {
		t.Errorf("Invalid error: %v", err.Error())
	}
}

func TestUnpackErrorJson(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()
	client := SearchClient{accessToken, server.URL}

	_, err := client.FindUsers(SearchRequest{})

	if !strings.Contains(err.Error(), "cant unpack error json") {
		t.Errorf("Invalid error: %v", err.Error())
	}
}

func TestUnpackResultJson(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	}))
	defer server.Close()
	client := SearchClient{accessToken, server.URL}

	_, err := client.FindUsers(SearchRequest{})

	if !strings.Contains(err.Error(), "cant unpack result json") {
		t.Errorf("Invalid error: %v", err.Error())
	}
}

func TestBadOrderField(t *testing.T) {
	ts := newTestServer(accessToken)
	defer ts.Close()

	_, err := ts.client.FindUsers(SearchRequest{Query: "do", OrderField: "random", OrderBy: OrderByAsc})

	if err.Error() != "OrderFeld random invalid" {
		t.Errorf("Invalid error: %v", err.Error())
	}
}

func TestOrderFieldId(t *testing.T) {
	ts := newTestServer(accessToken)
	defer ts.Close()

	srchResp, err := ts.client.FindUsers(SearchRequest{Query: "", OrderField: "id", OrderBy: OrderByAsc})

	if err != nil {
		t.Errorf("Invalid error: %v", err.Error())
	}

	for idx := 0; idx < 35; idx++ {
		expectedUserID := srchResp.Users[idx].ID

		if expectedUserID != idx {
			t.Errorf("Expected: %v, got: %v", idx, expectedUserID)
		}
	}
}

func TestOrderFieldName(t *testing.T) {
	expectedNames := [...]string{"Allison Valdez", "Annie Osborn", "Bell Bauer", "Beth Wynn", "Beulah Stark",
		"Boyd Wolf", "Brooks Aguilar", "Christy Knapp", "Clarissa Henry", "Cohen Hines",
		"Cruz Guerrero", "Dickson Silva", "Dillard Mccoy", "Everett Dillard", "Gates Spencer",
		"Gilmore Guerra", "Glenn Jordan", "Gonzalez Anderson", "Henderson Maxwell", "Hilda Mayer",
		"Jennings Mays", "Johns Whitney", "Kane Sharp", "Katheryn Jacobs", "Leann Travis",
		"Lowery York", "Nicholson Newman", "Owen Lynn", "Palmer Scott", "Rebekah Sutton",
		"Rose Carney", "Sims Cotton", "Terrell Hall", "Twila Snow", "Whitley Davidson"}

	ts := newTestServer(accessToken)
	defer ts.Close()

	srchResp, err := ts.client.FindUsers(SearchRequest{Query: "", OrderField: "name", OrderBy: OrderByAsc})

	if err != nil {
		t.Errorf("Invalid error: %v", err.Error())
	}

	for idx, user := range srchResp.Users {
		if user.Name != expectedNames[idx] {
			t.Errorf("Expected: %v, got: %v", expectedNames[idx], user.Name)
		}
	}

	// тестирование случая с пустым OrderField
	srchResp, err = ts.client.FindUsers(SearchRequest{Query: "", OrderField: "", OrderBy: OrderByAsc})

	if err != nil {
		t.Errorf("Invalid error: %v", err.Error())
	}

	for idx, user := range srchResp.Users {
		if user.Name != expectedNames[idx] {
			t.Errorf("Expected: %v, got: %v", expectedNames[idx], user.Name)
		}
	}
}

func TestOrderFieldAge(t *testing.T) {
	expectedAges := [...]int{21, 21, 21, 22, 23, 25, 26, 26, 26, 27, 27, 27, 29, 30, 30, 30, 31, 32,
		32, 32, 32, 33, 34, 34, 34, 35, 36, 36, 36, 36, 37, 39, 39, 40, 40}

	ts := newTestServer(accessToken)
	defer ts.Close()

	srchResp, err := ts.client.FindUsers(SearchRequest{Query: "", OrderField: "age", OrderBy: OrderByAsc})

	if err != nil {
		t.Errorf("Invalid error: %v", err.Error())
	}

	for idx, user := range srchResp.Users {
		if user.Age != expectedAges[idx] {
			t.Errorf("Expected: %v, got: %v", expectedAges[idx], user.Age)
		}
	}
}

func TestOffset(t *testing.T) {
	ts := newTestServer(accessToken)
	defer ts.Close()

	referenceResp, err := ts.client.FindUsers(SearchRequest{Query: " ", OrderBy: OrderByAsIs})

	if err != nil {
		t.Errorf("Invalid error: %v", err.Error())
	}

	for offset := 0; offset < len(referenceResp.Users)-1; offset++ {
		srchResp, err := ts.client.FindUsers(SearchRequest{Query: " ", OrderField: "id", OrderBy: OrderByAsc, Offset: offset})

		if err != nil {
			t.Errorf("Invalid error: %v", err.Error())
		}

		firstUserID := srchResp.Users[0].ID
		if firstUserID != offset {
			t.Errorf("Expected: %v, got: %v", offset, firstUserID)
		}

		usersLen := len(srchResp.Users)
		referenceUsersLen := len(referenceResp.Users) - offset
		if usersLen != referenceUsersLen {
			t.Errorf("Expected: %v, got: %v", referenceUsersLen, usersLen)
		}
	}
}

func TestInvalidOffset(t *testing.T) {
	ts := newTestServer(accessToken)
	defer ts.Close()

	srchResp, err := ts.client.FindUsers(SearchRequest{Query: " ", OrderField: "id", OrderBy: OrderByAsc, Offset: 35})

	if err != nil {
		t.Errorf("Invalid error: %v", err.Error())
	}

	usersLen := len(srchResp.Users)
	if usersLen != 0 {
		t.Errorf("Expected: %v, got: %v", 0, usersLen)
	}

}

func TestInvalidRequestOffsetLow(t *testing.T) {
	ts := newTestServer(accessToken)
	defer ts.Close()

	_, err := ts.client.FindUsers(SearchRequest{Offset: -100})

	if err.Error() != "offset must be > 0" {
		t.Errorf("Invalid error: %v", err.Error())
	}
}

func TestCorrectLimit(t *testing.T) {
	ts := newTestServer(accessToken)
	defer ts.Close()

	for limit := 1; limit < 26; limit++ {
		srchResp, err := ts.client.FindUsers(SearchRequest{Query: " ", OrderField: "id", OrderBy: OrderByAsc, Limit: limit})

		if err != nil {
			t.Errorf("Invalid error: %v", err.Error())
		}

		usersLen := len(srchResp.Users)
		if usersLen != limit {
			t.Errorf("Expected: %v, got: %v", limit, usersLen)
		}
	}
}

func TestInvalidLimitLow(t *testing.T) {
	ts := newTestServer(accessToken)
	defer ts.Close()

	_, err := ts.client.FindUsers(SearchRequest{Limit: -100})

	expected := "limit must be > 0"
	if err.Error() != expected {
		t.Errorf("Expected: %v, got: %v", expected, err.Error())
	}
}

func TestInvalidLimitHigh(t *testing.T) {
	ts := newTestServer(accessToken)
	defer ts.Close()

	srchResp, err := ts.client.FindUsers(SearchRequest{Query: " ", OrderField: "id", OrderBy: OrderByAsc, Limit: 100})

	if err != nil {
		t.Errorf("Invalid error: %v", err.Error())
	}

	usersLen := len(srchResp.Users)
	if usersLen != 25 {
		t.Errorf("Expected: %v, got: %v", 25, usersLen)
	}
}

func TestNextPage(t *testing.T) {
	ts := newTestServer(accessToken)
	defer ts.Close()

	srchResp, err := ts.client.FindUsers(SearchRequest{Query: " ", OrderField: "id", OrderBy: OrderByAsc, Limit: 10})

	if err != nil {
		t.Errorf("Invalid error: %v", err.Error())
	}

	if srchResp.NextPage != true {
		t.Errorf("Expected: %v, got: %v", true, srchResp.NextPage)
	}
}

func TestParseParams(t *testing.T) {
	req := httptest.NewRequest("GET", "/search?query=test&order_field=name&order_by=1&offset=5&limit=10", nil)
	params := &queryDTO{}
	err := params.parseParams(req)
	if err != nil {
		t.Error(err.Error())
	}

	expectedQuery := "test"
	if params.query != expectedQuery {
		t.Errorf("Expected: %s, got: %s", expectedQuery, params.query)
	}

	expectedOrderField := "name"
	if params.orderField != expectedOrderField {
		t.Errorf("Expected: %s, got: %s", expectedOrderField, params.orderField)
	}

	expectedOrderBy := 1
	if params.orderBy != expectedOrderBy {
		t.Errorf("Expected: %d, got: %d", expectedOrderBy, params.orderBy)
	}

	expectedOffset := 5
	if params.offset != expectedOffset {
		t.Errorf("Expected: %d, got: %d", expectedOffset, params.offset)
	}

	expectedLimit := 10
	if params.limit != expectedLimit {
		t.Errorf("Expected: %d, got: %d", expectedLimit, params.limit)
	}
}

func TestInvalidParseParams(t *testing.T) {
	req := httptest.NewRequest("GET", "/search?query=test&order_field=name&order_by=invalid&offset=invalid&limit=invalid", nil)
	params := &queryDTO{}
	err := params.parseParams(req)

	expected := "strconv.Atoi: parsing \"invalid\": invalid syntax"
	if err.Error() != expected {
		t.Errorf("Expected: %s, got: %s", expected, err.Error())
	}

	expectedOrderBy := 0
	if params.orderBy != expectedOrderBy {
		t.Errorf("Expected: %d, got: %d", expectedOrderBy, params.orderBy)
	}

	expectedOffset := 0
	if params.offset != expectedOffset {
		t.Errorf("Expected: %d, got: %d", expectedOffset, params.offset)
	}

	expectedLimit := 0
	if params.limit != expectedLimit {
		t.Errorf("Expected: %d, got: %d", expectedLimit, params.limit)
	}
}

func TestSendResponseMarshalError(t *testing.T) {
	w := httptest.NewRecorder()
	invalidData := make(chan int)
	sendResponse(w, invalidData)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected: %d, got: %d", http.StatusInternalServerError, w.Code)
	}

	expectedBody := "cant marshal json"
	if !strings.Contains(w.Body.String(), expectedBody) {
		t.Errorf("Expected: %s, got: %s", expectedBody, w.Body.String())
	}
}

func TestSendResponseWriteError(t *testing.T) {
	w := &errorResponseWriter{}
	data := map[string]interface{}{"key": "value"}

	sendResponse(w, data)

	if w.Header().Get("Content-Type") == "application/json" {
		t.Error("Expected nil, got application/json")
	}
}

type errorResponseWriter struct{}

func (e *errorResponseWriter) Write([]byte) (int, error) {
	return 0, someError
}

func (e *errorResponseWriter) Header() http.Header {
	return http.Header{}
}

func (e *errorResponseWriter) WriteHeader(statusCode int) {}

var someError = &json.MarshalerError{}

