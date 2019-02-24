package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/lib/pq"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"
)

var db *sql.DB
var methods = map[string]string{
	"GET":    "get",
	"DELETE": "del",
	"POST":   "ins",
	"PUT":    "upd",
}

func init() {
	var err error
	db, err = sql.Open("postgres", "user=postgres password=1 dbname=postgres sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}

	if err = db.Ping(); err != nil {
		log.Fatal(err)
	}
	db.SetMaxOpenConns(4)
}

type route struct {
	pattern *regexp.Regexp
	handler http.Handler
}

type RegexpHandler struct {
	routes []*route
}

func (h *RegexpHandler) Handler(pattern *regexp.Regexp, handler http.Handler) {
	h.routes = append(h.routes, &route{pattern, handler})
}

func (h *RegexpHandler) HandleFunc(pattern *regexp.Regexp, handler func(http.ResponseWriter, *http.Request)) {
	h.routes = append(h.routes, &route{pattern, http.HandlerFunc(handler)})
}

func (h *RegexpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for _, route := range h.routes {
		if route.pattern.MatchString(r.URL.Path) {
			route.handler.ServeHTTP(w, r)
			return
		}
	}
	// no pattern matched; send 404 response
	http.NotFound(w, r)
}

func mustWrite(w http.ResponseWriter, msg []byte) {
	if _, err := w.Write(msg); err != nil {
		panic(err)
	}
}

func mustCloseRows(rows *sql.Rows) {
	if err := rows.Close(); err != nil {
		panic(err)
	}
}

func sendErrorMsg(w http.ResponseWriter, msg string) {
	errMsg := map[string]string{
		"error": fmt.Sprintf(`%s`, msg),
	}
	js, err := json.Marshal(errMsg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	mustWrite(w, js)
}

func sendOK(w http.ResponseWriter, msg string) {
	js, err := json.Marshal(msg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	mustWrite(w, js)
}

func NewDbError(msg string) error {
	return &dbErrorString{msg}
}

type dbErrorString struct {
	s string
}

func (e *dbErrorString) Error() string {
	return e.s
}

type Response struct {
	Data string
}

func sendQuery(query string) (result string, err error) {
	rows, err := db.Query(query)
	if err != nil {
		return "", NewDbError(err.Error())
	}
	defer mustCloseRows(rows)

	response := new(Response)
	for rows.Next() {
		err := rows.Scan(&response.Data)
		if err != nil {
			return "", NewDbError(err.Error())
		}
	}
	if err = rows.Err(); err != nil {
		return "", NewDbError(err.Error())
	}

	return response.Data, nil
}

func CRUDHandler(w http.ResponseWriter, r *http.Request) {
	sqlMethod := methods[r.Method]

	parts := strings.Split(r.RequestURI, "/")
	firstArg := parts[len(parts)-1]
	entity := parts[len(parts)-2]
	secondArg := parts[len(parts)-3]
	secondEntity := parts[len(parts)-4]

	data := make([]byte, r.ContentLength)
	data, _ = ioutil.ReadAll(r.Body)

	entities := []string{"user", "comment"}
	entityState := "single"
	for _, elem := range entities {
		if elem == secondEntity {
			entityState = "multiple"
		}
	}

	if firstArg == "" {
		firstArg = "0"
	}
	queryMaps := map[string]map[string]string{
		"single": {
			"get": fmt.Sprintf(`select test.%s_get(%s)`, entity, firstArg),
			"ins": fmt.Sprintf(`select test.%s_ins('%s')`, entity, string(data)),
			"upd": fmt.Sprintf(`select test.%s_upd(%s,'%s')`, entity, firstArg, string(data)),
			"del": fmt.Sprintf(`select test.%s_del(%s)`, entity, firstArg),
		},
		"multiple": {
			"get": fmt.Sprintf(`select test.user_comment_get(%s, %s)`, firstArg, secondArg),
			"ins": fmt.Sprintf(`select test.user_comment_ins(%s,'%s')`, firstArg, string(data)),
		},
	}

	res, err := sendQuery(queryMaps[entityState][sqlMethod])
	if err != nil {
		sendErrorMsg(w, err.Error())
	} else {
		sendOK(w, res)
	}
}

func main() {
	regexpHandler := RegexpHandler{}
	regexpHandler.HandleFunc(
		regexp.MustCompile("^/api/v1/((user|comment)|(user/[0-9]+/comment))/[0-9]*$"),
		CRUDHandler)
	log.Fatal(http.ListenAndServe(":8080", &regexpHandler))
}
