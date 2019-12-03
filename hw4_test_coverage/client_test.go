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
		w.Write([]byte("ErrorBadOrderField"))
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

	root := &Root{}
	err = decoder.Decode(&root)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("can't decode db"))
		return
	}

	var rows []*Row
	for _, row := range root.Rows {
		row.Name = row.First_name + " " + row.Last_name
		if query != "" {
			if !(strings.Contains(row.Name, query) || strings.Contains(row.About, query)) {
				continue
			}
		}
		rows = append(rows, &row)
	}

	//Параметр order_field работает по полям Id, Age, Name, если пустой - то возвращаем по Name
	if orderBy != 0 {
		sort.Slice(rows, func(i, j int) bool {
			ri := reflect.ValueOf(rows[i])
			rj := reflect.ValueOf(rows[j])
			vi := reflect.Indirect(ri).FieldByName(orderField).FieldByName(orderField)
			vj := reflect.Indirect(rj).FieldByName(orderField).FieldByName(orderField)
			switch vi.Kind() {
			case reflect.Int:
				return vi.Int() < vj.Int()
			case reflect.String:
				return vi.String() < vj.String()
			default:
				panic("found unprocess sort field: " + orderField)
			}
			return false
		})
	}

	if offset > 0 {
		rows = rows[offset-1:]
	}

	if limit > 0 {
		rows = rows[:limit]
	}

	data, err := json.Marshal(rows)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("can't prepare responce: " + err.Error()))
		return
	}

	w.Write(data)
	w.WriteHeader(http.StatusOK)

}

/*
    <id>0</id>
    <guid>1a6fa827-62f1-45f6-b579-aaead2b47169</guid>
    <isActive>false</isActive>
    <balance>$2,144.93</balance>
    <picture>http://placehold.it/32x32</picture>
    <age>22</age>
    <eyeColor>green</eyeColor>
    <first_name>Boyd</first_name>
    <last_name>Wolf</last_name>
    <gender>male</gender>
    <company>HOPELI</company>
    <email>boydwolf@hopeli.com</email>
    <phone>+1 (956) 593-2402</phone>
    <address>586 Winthrop Street, Edneyville, Mississippi, 9555</address>
    <about>Nulla cillum enim voluptate consequat laborum esse excepteur occaecat commodo nostrud excepteur ut cupidatat. Occaecat minim incididunt ut proident ad sint nostrud ad laborum sint pariatur. Ut nulla commodo dolore officia. Consequat anim eiusmod amet commodo eiusmod deserunt culpa. Ea sit dolore nostrud cillum proident nisi mollit est Lorem pariatur. Lorem aute officia deserunt dolor nisi aliqua consequat nulla nostrud ipsum irure id deserunt dolore. Minim reprehenderit nulla exercitation labore ipsum.
</about>
    <registered>2017-02-05T06:23:27 -03:00</registered>
    <favoriteFruit>apple</favoriteFruit>
*/

type Root struct {
	XMLName xml.Name `xml:"root"`
	Rows    []Row    `xml:"row"`
}

type Row struct {
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

func TestGetData(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))

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
