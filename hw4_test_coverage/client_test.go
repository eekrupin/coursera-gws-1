package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

// код писать тут

func SearchServer(w http.ResponseWriter, r *http.Request) {

	defer func() {
		if r := recover(); r != nil {
			log.Println(r)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprint("error: ", r)))
		}
	}()

	//Limit      int
	//Offset     int    // Можно учесть после сортировки
	//Query      string // подстрока в 1 из полей по полям Name или About
	//OrderField string
	//// -1 по убыванию, 0 как встретилось, 1 по возрастанию
	//OrderBy int

	accessToken := r.Header.Get("AccessToken")

	limit, _ := strconv.Atoi(r.FormValue("limit"))
	offset, _ := strconv.Atoi(r.FormValue("offset"))
	query := r.FormValue("query")
	orderField := r.FormValue("order_field")
	orderBy, _ := strconv.Atoi(r.FormValue("order_by"))

	//Данные для работы лежаит в файле dataset.xml
	//Параметр query ищет по полям Name и About
	//Параметр order_field работает по полям Id, Age, Name, если пустой - то возвращаем по Name, если что-то другое - SearchServer ругается ошибкой.
	// 	Name - это first_name + last_name из xml.
	//	Если query пустой, то делаем только сортировку, т.е. возвращаем все записи
	//Как работать с XML смотрите в xml/*
	//Запускать как go test -cover
	//Построение покрытия: go test -coverprofile=cover.out && go tool cover -html=cover.out -o cover.html. Для построения покрытия ваш код должен находиться внутри GOPATH

	if query == "timeOut" {
		query = ""
		time.Sleep(5 * time.Second)
	}

	if query == "wrongJson" {
		query = ""
		w.Write([]byte("wrongJson"))
		return
	}

	if query == "wrongJsonBadRequest" {
		query = ""
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("wrongJsonBadRequest"))
		return
	}

	if query == "unknownBadRequest" {
		query = ""
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "Unknown Bad Request"}`))
		return
	}

	if query == "panic" {
		query = ""
		panic("panic")
		return
	}

	if accessToken == "" {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("accessToken is empty"))
		return
	}

	if orderField == "" {
		orderField = "Name"
	}

	if !strings.Contains("Id, Age, Name", orderField) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "ErrorBadOrderField"}`))
		return
	}

	xmlfile := "dataset.xml"
	xmlStream, err := os.Open(xmlfile)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("can't open db"))
		return
	}
	defer xmlStream.Close()

	decoder := xml.NewDecoder(xmlStream)

	root := &RootXML{}
	err = decoder.Decode(&root)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("can't decode db"))
		return
	}

	var rows []*UserXML
	for _, user := range root.Users {
		user.Name = user.First_name + " " + user.Last_name
		if query != "" {
			if !(strings.Contains(user.Name, query) || strings.Contains(user.About, query)) {
				continue
			}
		}
		rows = append(rows, user)
	}

	//Параметр order_field работает по полям Id, Age, Name, если пустой - то возвращаем по Name
	if orderBy != 0 {
		sort.Slice(rows, func(i, j int) bool {
			ri := reflect.ValueOf(rows[i])
			rj := reflect.ValueOf(rows[j])
			vi := reflect.Indirect(ri).FieldByName(orderField)
			vj := reflect.Indirect(rj).FieldByName(orderField)
			switch vi.Kind() {
			case reflect.Int:
				if orderBy < 0 {
					return vi.Int() < vj.Int()
				} else {
					return vi.Int() > vj.Int()
				}
			case reflect.String:
				if orderBy < 0 {
					return vi.String() < vj.String()
				} else {
					return vi.String() > vj.String()
				}
			default:
				panic("found unprocess sort field: " + orderField)
			}
			return false
		})
	}

	if offset > 0 {
		rows = rows[offset:]
	}

	if limit > 0 {
		rows = rows[:Min(limit, len(rows))]
	}

	data, err := json.Marshal(rows)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("can't prepare responce: " + err.Error()))
		return
	}

	w.Write(data)

}

func Min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

type RootXML struct {
	XMLName xml.Name   `xml:"root"`
	Users   []*UserXML `xml:"row"`
}

type UserXML struct {
	Id            int    `xml:"id" json:"id"`
	GUID          string `xml:"guid"`
	IsActive      bool   `xml:"isActive"`
	Balance       string `xml:"balance"`
	Picture       string `xml:"picture"`
	Age           int    `xml:"age" json:"age"`
	EyeColor      string `xml:"eyeColor"`
	First_name    string `xml:"first_name"`
	Last_name     string `xml:"last_name"`
	Name          string `xml:"-" json:"name"`
	Company       string `xml:"company"`
	Email         string `xml:"email"`
	Phone         string `xml:"phone"`
	Address       string `xml:"address"`
	About         string `xml:"about" json:"about"`
	Registered    string `xml:"registered"`
	FavoriteFruit string `xml:"favoriteFruit"`
	Gender        string `xml:"gender" json:"gender"`
}

func TestFailAccessToken(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	srv := &SearchClient{
		AccessToken: "",
		URL:         ts.URL,
	}
	got, err := srv.FindUsers(SearchRequest{})
	wantErr := "Bad AccessToken"

	if err == nil {
		t.Errorf("SearchClient.FindUsers() = %v, want %v", got, wantErr)
		return
	}

	if err.Error() != wantErr {
		t.Errorf("SearchClient.FindUsers() error = %v, wantErr %v", err, wantErr)
		return
	}
	//if !reflect.DeepEqual(got, want) {
	//	t.Errorf("SearchClient.FindUsers() = %v, want %v", got, tt.want)
	//}

}

func TestTop10(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	srv := &SearchClient{
		AccessToken: "AccessToken",
		URL:         ts.URL,
	}
	got, err := srv.FindUsers(SearchRequest{Limit: 10})

	if err != nil {
		t.Errorf("SearchClient.FindUsers() = %v, want nil error", err)
		return
	}

	if len(got.Users) != 10 {
		t.Errorf("SearchClient.FindUsers() not has count 10")
	}

}

func TestTop30(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	srv := &SearchClient{
		AccessToken: "AccessToken",
		URL:         ts.URL,
	}
	got, err := srv.FindUsers(SearchRequest{Limit: 30})

	if err != nil {
		t.Errorf("SearchClient.FindUsers() = %v, want nil error", err)
		return
	}

	if len(got.Users) != 25 {
		t.Errorf("SearchClient.FindUsers() not has count 25")
	}

	if !got.NextPage {
		t.Errorf("NextPage must be")
	}

}

func TestOffset(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	srv := &SearchClient{
		AccessToken: "AccessToken",
		URL:         ts.URL,
	}
	got5, err := srv.FindUsers(SearchRequest{Limit: 5})
	if err != nil {
		t.Errorf("SearchClient.FindUsers() = %v, want nil : ", err)
		return
	}

	gotOffset4, err := srv.FindUsers(SearchRequest{Limit: 5, Offset: 4})
	if err != nil {
		t.Errorf("SearchClient.FindUsers() = %v, want nil error: ", err)
		return
	}

	if got5.Users[4].Id != gotOffset4.Users[0].Id {
		t.Errorf("Offset not work")
	}

}

func TestWrongLimit(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	srv := &SearchClient{
		AccessToken: "AccessToken",
		URL:         ts.URL,
	}
	_, err := srv.FindUsers(SearchRequest{Limit: -5})

	if err == nil {
		t.Errorf("err must be not nil")
		return
	}

	want := "limit must be > 0"
	if err.Error() != want {
		t.Errorf("err must be: %s, get: %s", want, err)
		return
	}

}

func TestWrongOffset(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	srv := &SearchClient{
		AccessToken: "AccessToken",
		URL:         ts.URL,
	}
	_, err := srv.FindUsers(SearchRequest{Offset: -5})

	if err == nil {
		t.Errorf("err must be not nil")
		return
	}

	want := "offset must be > 0"
	if err.Error() != want {
		t.Errorf("err must be: %s, get: %s", want, err)
		return
	}

}

func TestWrongSortName(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	srv := &SearchClient{
		AccessToken: "AccessToken",
		URL:         ts.URL,
	}
	_, err := srv.FindUsers(SearchRequest{Limit: 10, OrderField: "Company"})

	if err == nil {
		t.Errorf("err must be not nil")
		return
	}

	want := "OrderFeld Company invalid"
	if err.Error() != want {
		t.Errorf("err must be: %s, get: %s", want, err)
	}

}

func TestSort(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	ts.Config.IdleTimeout = time.Nanosecond
	defer ts.Close()

	srv := &SearchClient{
		AccessToken: "AccessToken",
		URL:         ts.URL,
	}
	gotOrderByAsc, err := srv.FindUsers(SearchRequest{Limit: 5, Offset: 30, OrderBy: OrderByAsc, OrderField: "Id"})
	if err != nil {
		t.Errorf("SearchClient.FindUsers() = %v, want nil : ", err)
		return
	}

	gotOrderByDesc, err := srv.FindUsers(SearchRequest{Limit: 5, OrderBy: OrderByDesc, OrderField: "Id"})
	if err != nil {
		t.Errorf("SearchClient.FindUsers() = %v, want nil error: ", err)
		return
	}

	if gotOrderByAsc.Users[4].Id != gotOrderByDesc.Users[0].Id {
		t.Errorf("Sort not work")
	}

}

func TestTimeOut(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	srv := &SearchClient{
		AccessToken: "AccessToken",
		URL:         ts.URL,
	}
	_, err := srv.FindUsers(SearchRequest{Limit: 5, Offset: 30, Query: "timeOut"})
	if err == nil {
		t.Errorf("must be error timeOut")
		return
	}

	if !strings.Contains(err.Error(), "timeout") {
		t.Errorf("must be error with text timeOut")
		return
	}
}

func TestWrongJsonServer(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	srv := &SearchClient{
		AccessToken: "AccessToken",
		URL:         ts.URL,
	}
	_, err := srv.FindUsers(SearchRequest{Limit: 5, Offset: 30, Query: "wrongJson"})
	if err == nil {
		t.Errorf("must be error")
		return
	}

	want := "cant unpack result json"
	if !strings.Contains(err.Error(), want) {
		t.Errorf("must be error with text '%s'. Get: %s", want, err)
		return
	}
}

func TestWrongJsonBadRequestServer(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	srv := &SearchClient{
		AccessToken: "AccessToken",
		URL:         ts.URL,
	}
	_, err := srv.FindUsers(SearchRequest{Limit: 5, Offset: 30, Query: "wrongJsonBadRequest"})
	if err == nil {
		t.Errorf("must be error")
		return
	}

	want := "cant unpack error json"
	if !strings.Contains(err.Error(), want) {
		t.Errorf("must be error with text '%s'. Get: %s", want, err)
		return
	}
}

func TestPanicServer(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	srv := &SearchClient{
		AccessToken: "AccessToken",
		URL:         ts.URL,
	}
	_, err := srv.FindUsers(SearchRequest{Limit: 5, Offset: 30, Query: "panic"})
	if err == nil {
		t.Errorf("must be error")
		return
	}

	want := "SearchServer fatal error"
	if !strings.Contains(err.Error(), want) {
		t.Errorf("must be error with text '%s'. Get: %s", want, err)
		return
	}
}

func TestUnknownBadRequestServer(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	srv := &SearchClient{
		AccessToken: "AccessToken",
		URL:         ts.URL,
	}
	_, err := srv.FindUsers(SearchRequest{Limit: 5, Offset: 30, Query: "unknownBadRequest"})
	if err == nil {
		t.Errorf("must be error")
		return
	}

	want := "unknown bad request"
	if !strings.Contains(err.Error(), want) {
		t.Errorf("must be error with text '%s'. Get: %s", want, err)
		return
	}
}

func TestTop10Query(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	srv := &SearchClient{
		AccessToken: "AccessToken",
		URL:         ts.URL,
	}

	want := "Nicholson"
	got, err := srv.FindUsers(SearchRequest{Limit: 10, Query: want})

	if err != nil {
		t.Errorf("SearchClient.FindUsers() = %v, want nil error", err)
		return
	}

	for _, user := range got.Users {
		if !strings.Contains(user.Name, want) {
			t.Errorf("User %s dont have '%s'", user.Name, want)
		}
	}

}

func TestUnknownErrorServer(t *testing.T) {

	srv := &SearchClient{
		AccessToken: "AccessToken",
		URL:         "UnknownError",
	}
	_, err := srv.FindUsers(SearchRequest{})
	if err == nil {
		t.Errorf("must be error")
		return
	}

	want := "unknown error"
	if !strings.Contains(err.Error(), want) {
		t.Errorf("must be error with text '%s'. Get: %s", want, err)
		return
	}
}
