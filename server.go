package main

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
)

// Структура для разбора XML-данных
type xmlData struct {
	XMLName xml.Name `xml:"root"`
	Rows    []row    `xml:"row"`
}

// Структура строки данных из XML
type row struct {
	ID        int    `xml:"id"`
	FirstName string `xml:"first_name"`
	LastName  string `xml:"last_name"`
	Age       int    `xml:"age"`
	About     string `xml:"about"`
	Gender    string `xml:"gender"`
}

// queryDTO содержит параметры запроса
type queryDTO struct {
	query      string
	orderField string
	orderBy    int
	offset     int
	limit      int
}

func (q *queryDTO) parseParams(r *http.Request) error {
	var (
		queryValues = r.URL.Query()
		err         error
	)

	q.query = queryValues.Get("query")
	q.orderField = queryValues.Get("order_field")

	q.orderBy, err = strconv.Atoi(queryValues.Get("order_by"))
	if err != nil {
		q.orderBy = 0
	}

	q.offset, err = strconv.Atoi(queryValues.Get("offset"))
	if err != nil {
		q.offset = 0
	}

	q.limit, err = strconv.Atoi(queryValues.Get("limit"))
	if err != nil {
		q.limit = 0
	}

	return err
}

func isRowMatching(row row, query string) bool {
	return strings.Contains(row.FirstName, query) ||
		strings.Contains(row.LastName, query) ||
		strings.Contains(row.About, query)
}

// Имя xml-файла с данными
var fileName = "dataset.xml"

// Фильтрация данных по заданному query
func filterData(data xmlData, query string) []User {
	result := make([]User, 0)

	for _, row := range data.Rows {
		if query != "" {
			// Проверка соответствия запросу в полях FirstName, LastName и About
			if !isRowMatching(row, query) {
				continue
			}
		}

		// Добавление соответствующих данных в результат
		result = append(result, User{
			ID:     row.ID,
			Name:   row.FirstName + " " + row.LastName,
			Age:    row.Age,
			About:  row.About,
			Gender: row.Gender,
		})
	}
	return result
}

// Сортировка данных в соответствии с orderField и orderBy
func sortData(data []User, orderField string, orderBy int) ([]User, error) {
	var isLess func(i, j int) bool

	switch orderField {
	case "":
		fallthrough
	case "name":
		isLess = func(i, j int) bool {
			return (data[i].Name < data[j].Name) && (orderBy == OrderByAsc)
		}
	case "id":
		isLess = func(i, j int) bool {
			return (data[i].ID < data[j].ID) && (orderBy == OrderByAsc)
		}
	case "age":
		isLess = func(i, j int) bool {
			return (data[i].Age < data[j].Age) && (orderBy == OrderByAsc)
		}
	default:
		return nil, errors.New("OrderField invalid")
	}

	sort.Slice(data, isLess)
	return data, nil
}

// Пагинация данных
func paginateData(data []User, offset, limit int) []User {
	if offset > 0 {
		if offset < len(data) {
			data = data[offset:]
		} else {
			data = []User{}
		}
	}

	if (limit - 1) > 0 {
		data = data[:limit]
	}

	return data
}

// Отправка ответа в формате JSON
func sendResponse(w http.ResponseWriter, data interface{}) {
	jsonFile, err := json.Marshal(data)
	if err != nil {
		http.Error(w, "cant marshal json", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(jsonFile)
	if err != nil {
		http.Error(w, "cant write json", http.StatusInternalServerError)
		return
	}
}

// Обработчик запроса поиска
func SearchServer(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("AccessToken") != accessToken {
		http.Error(w, "Invalid AccessToken", http.StatusUnauthorized)
		return
	}

	xmlFile, err := os.Open(fileName)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer xmlFile.Close()

	var (
		data   xmlData
		result []User
	)

	b, err := io.ReadAll(xmlFile)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = xml.Unmarshal(b, &data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Парсинг параметров запроса
	params := &queryDTO{}
	err = params.parseParams(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	// Фильтрация данных
	result = filterData(data, params.query)

	if params.orderBy != OrderByAsIs {
		// Сортировка данных
		sortedData, err := sortData(result, params.orderField, params.orderBy)
		if err != nil {
			// В случае отпраляется ответ с ошибкой
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			jsonStr := `{ "Error": "` + err.Error() + `" }`
			_, err = w.Write([]byte(jsonStr))
			if err != nil {
				http.Error(w, "cant write json", http.StatusInternalServerError)
			}
			return
		}
		result = sortedData
	}

	// Пагинация данных
	result = paginateData(result, params.offset, params.limit)
	// Отправка результата
	sendResponse(w, result)
}

