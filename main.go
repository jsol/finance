package main

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	_ "embed"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	store "github.com/jsol/finance/store"
	_ "modernc.org/sqlite"

	"github.com/google/uuid"
)

type PatternsData struct {
	Patterns   []store.Pattern
	Categories []store.Category
}

type Line struct {
	Id  string
	Sum int64
}

type Option struct {
	Name  string
	Value string
}

type Total struct {
	Deptor  string
	Lender  string
	Owed    int64
	Sums    map[string]int64
	Targets map[string]int64
	Sum     int64
	Income  int64
}

type User struct {
	Lines []Line
	Name  string
	Sum   int64
}

type Category struct {
	Name  string
	ID    []byte
	Users map[string]*User
	Sum   int64
}

type Page struct {
	Period     string
	Previous   string
	Next       string
	PeriodNice string
	Total      Total
	Categories map[string]*Category
	Users      []string
	Incomes    map[string]int64
	Patterns   map[string][]string
}

type ImportModal struct {
	Period     string
	PeriodNice string
	Users      []string
}

type application struct {
	auth struct {
		username [32]byte
		password [32]byte
	}
	port       int
	templates  *template.Template
	debug      bool
	queries    *store.Queries
	Users      []string
	Incomes    map[string]int64
}

//go:embed schema.sql
var ddl string

func toString(id []byte) string {
	uuid, err := uuid.FromBytes(id)

	if err != nil {
		fmt.Println(err)
	}
	return uuid.String()
}

func catToString(id []byte) string {
	return string(id)
}

func (u *User) AddItem(item store.Item) {
	id, _ := uuid.FromBytes(item.ID)
	l := Line{Id: id.String(), Sum: item.Sum.Int64}
	u.Lines = append(u.Lines, l)
	u.Sum += item.Sum.Int64
}

func (cat *Category) AddItem(item store.Item) {
	cat.Users[item.Who].AddItem(item)
	cat.Sum += item.Sum.Int64
}

func (p *Page) AddItem(item store.Item) {
	cat := p.Categories[string(item.Cat)]
	cat.AddItem(item)
	p.Total.Sum += item.Sum.Int64

	p.Total.Sums[item.Who] += item.Sum.Int64

	for who, paid := range p.Total.Sums {
		factor := float64(p.Incomes[who]) / float64(p.Total.Income)
		p.Total.Targets[who] = int64(float64(p.Total.Sum) * factor)

		if paid < p.Total.Targets[who] {
			p.Total.Deptor = who
			p.Total.Owed = p.Total.Targets[who] - paid
		} else if paid == p.Total.Targets[who] {
			p.Total.Owed = 0
		} else {
			p.Total.Lender = who
		}
	}
}

func NewCategory(id []byte, name string, users []string) *Category {
	c := &Category{ID: id, Name: name}

	c.Users = make(map[string]*User)
	for _, n := range users {
		c.Users[n] = &User{Name: n, Sum: 0, Lines: []Line{}}
	}

	return c
}

func NewPage(period string, users []string, incomes map[string]int64) *Page {
	p := &Page{Period: period, Users: users, Incomes: incomes}
	p.Total.Targets = make(map[string]int64)
	parts := strings.Split(period, "-")

	for _, income := range incomes {
		p.Total.Income += income
	}

	var month time.Month
	var year int

	if len(parts) == 2 {
		year, _ = strconv.Atoi(parts[0])
		tmp, _ := strconv.Atoi(parts[1])
		month = time.Month(tmp)
	} else {
		year, month, _ = time.Now().Date()
		p.Period = fmt.Sprintf("%d-%d", year, month)
	}

	if month == 1 {
		p.Previous = fmt.Sprintf("%d-%d", year-1, 12)
		p.Next = fmt.Sprintf("%d-%d", year, 2)
	} else if month == 12 {
		p.Previous = fmt.Sprintf("%d-%d", year, 11)
		p.Next = fmt.Sprintf("%d-%d", year+1, 1)
	} else {
		p.Previous = fmt.Sprintf("%d-%d", year, month-1)
		p.Next = fmt.Sprintf("%d-%d", year, month+1)
	}

	p.PeriodNice = fmt.Sprintf("%s, %d", time.Month(month), year)

	p.Categories = make(map[string]*Category)
	p.Total.Sum = 0
	p.Total.Sums = make(map[string]int64)
	for _, u := range users {
		p.Total.Sums[u] = 0
	}

	return p
}

func (p *Page) AddCategory(c *Category) {
	p.Categories[string(c.ID)] = c
}

func (app *application) executeTemplate(wr io.Writer, name string, content any) {

	err := app.templates.ExecuteTemplate(wr, name, content)
	if err != nil {
		fmt.Println(err)
	}
}

func newId() []byte {
	rowid, _ := uuid.NewV7()
	return rowid[:]
}

func (app *application) handler(w http.ResponseWriter, r *http.Request) {
	app.printStart(w, r)
}

func (app *application) printStart(w http.ResponseWriter, r *http.Request) {
	page := NewPage(r.URL.Query().Get("period"), app.Users, app.Incomes)
	ctx := r.Context()

	categories, _ := app.queries.ListCategories(ctx)
	for _, c := range categories {
		cat := NewCategory(c.ID, c.Name, app.Users)
		page.AddCategory(cat)
	}

	records, _ := app.queries.GetPeriod(ctx, page.Period)
	for _, r := range records {
		page.AddItem(r)
	}
	patterns, _ := app.queries.GetPatterns(ctx)
	page.Patterns = make(map[string][]string)

	for _, p := range patterns {
		cat := catToString(p.Category)
		page.Patterns[cat] = append(page.Patterns[cat], p.Pattern)
	}

	app.executeTemplate(w, "start", page)
}

func parseSum(parts []string) int64 {

	if len(parts) != 2 {
		return -1
	}

	kr, err := strconv.Atoi(strings.Trim(parts[0], " "))

	if err != nil {
		return -1
	}

	tmp := strings.Trim(parts[1], " ")

	tmp = tmp + "00"

	fraction, err := strconv.Atoi(tmp[:2])
	if err != nil {
		return -1
	}

	return int64(kr*100 + fraction)
}

func (app *application) delete(w http.ResponseWriter, r *http.Request) {

	id, err := uuid.Parse(strings.TrimPrefix(r.URL.Path, "/delete/"))

	if err != nil {
		log.Println(err)
		w.WriteHeader(400)
		return
	}

	item, err := app.queries.GetItem(r.Context(), []byte(id[:]))

	if err != nil {
		log.Println(err)
		w.WriteHeader(400)
		return
	}
	app.queries.RemoveItem(r.Context(), []byte(id[:]))
	if app.debug {
		fmt.Println("Removing id", id)
	}

	app.executePageTemplate(r.Context(), w, item.Period)
}

func (app *application) add(w http.ResponseWriter, r *http.Request) {

	period := r.PostFormValue("period")
	who := r.PostFormValue("who")
	cat := []byte(r.PostFormValue("cat"))

	sum := r.PostFormValue("sum")

	sums := strings.Split(sum, "+")

	if app.debug {
		fmt.Println("Sums", sums)
	}

	for _, s := range sums {
		parts := strings.Split(s, ".")
		parsedSum := parseSum(parts)

		if parsedSum < 0 {
			parts = strings.Split(s, ",")
			parsedSum = parseSum(parts)
		}

		if parsedSum < 0 {
			tmp, err := strconv.Atoi(strings.Trim(parts[0], " "))
			parsedSum = int64(tmp * 100)
			if err != nil {
				parsedSum = -1
			}
		}

		if parsedSum > 0 {
			if app.debug {
				fmt.Println("Adding sum", parsedSum)
			}
			args := store.AddItemParams{ID: newId(), Who: who, Cat: cat, Sum: sql.NullInt64{Valid: true, Int64: parsedSum}, Period: period}
			app.queries.AddItem(r.Context(), args)
		} else {
			if app.debug {
				fmt.Println("Skipping sum", parsedSum)
			}
		}
	}
	app.executePageTemplate(r.Context(), w, period)
}

func (app *application) importModal(w http.ResponseWriter, r *http.Request) {
	data := ImportModal{}

	data.Period = r.URL.Query().Get("period")

	data.Users = app.Users

	parts := strings.Split(data.Period, "-")

	var month time.Month
	var year int

	if len(parts) != 2 {
		return
	}

	year, _ = strconv.Atoi(parts[0])
	tmp, _ := strconv.Atoi(parts[1])
	month = time.Month(tmp)

	data.PeriodNice = fmt.Sprintf("%s, %d", time.Month(month), year)

	app.executeTemplate(w, "import_modal", data)
}

func (app *application) executePageTemplate(ctx context.Context, w io.Writer, period string) {
	page := NewPage(period, app.Users, app.Incomes)

	categories, _ := app.queries.ListCategories(ctx)
	for _, c := range categories {
		cat := NewCategory(c.ID, c.Name, app.Users)
		page.AddCategory(cat)
	}

	if app.debug {
		fmt.Printf("Building period data %s\n", period)
	}
	records, _ := app.queries.GetPeriod(ctx, page.Period)
	for _, r := range records {
		page.AddItem(r)
	}
	app.executeTemplate(w, "period", page)

}

func (app *application) basicAuth(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if ok {
			usernameHash := sha256.Sum256([]byte(username))
			passwordHash := sha256.Sum256([]byte(password))

			usernameMatch := (subtle.ConstantTimeCompare(usernameHash[:], app.auth.username[:]) == 1)
			passwordMatch := (subtle.ConstantTimeCompare(passwordHash[:], app.auth.password[:]) == 1)

			if usernameMatch && passwordMatch {
				next.ServeHTTP(w, r)
				return
			}
		}

		w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
}

func toNormal(sum int64) string {
	return fmt.Sprintf("%d.%d", sum/100, sum%100)
}

func getUUID() string {
	id, _ := uuid.NewV7()
	return id.String()
}

func (app *application) fetchUsers(ctx context.Context) {
	users, _ := app.queries.GetUsers(ctx)

	app.Incomes = make(map[string]int64)
	app.Users = []string{}
	for _, u := range users {
		app.Users = append(app.Users, u.Name)
		app.Incomes[u.Name] = u.Factor.Int64
	}

}

func main() {
	app := new(application)
	app.port = 9091

	ctx := context.Background()
	user := os.Getenv("AUTH_USERNAME")
	password := os.Getenv("AUTH_PASSWORD")
	database := os.Getenv("DATABASE")
	app.debug = os.Getenv("DEBUG") == "true"

	if user == "" {
		log.Fatal("basic auth username must be provided")
	}

	if password == "" {
		log.Fatal("basic auth password must be provided")
	}

	if database == "" {
		log.Fatal("DATABASE must be path to the sqlite db or :memory:")
	}

	app.auth.username = sha256.Sum256([]byte(user))
	app.auth.password = sha256.Sum256([]byte(password))

	var err error

	db, err := sql.Open("sqlite", database)
	if err != nil {
		log.Panic(err)
	}

	// Create tables
	if _, err := db.ExecContext(ctx, ddl); err != nil {
		log.Panic(err)
	}

	app.queries = store.New(db)

	app.templates, err = template.New("finance").Funcs(template.FuncMap{"toNormal": toNormal, "getUUID": getUUID, "toString": toString, "catToString": catToString}).ParseGlob("templates/*.tmpl")

	if err != nil {
		panic(err)
	}

	app.fetchUsers(ctx)

	//	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	http.Handle("/category/", app.basicAuth(app.categoryHandler))
	http.Handle("/category/add", app.basicAuth(app.addCategoryHandler))
	http.Handle("/category/delete", app.basicAuth(app.deleteCategoryHandler))
	http.Handle("/category/update", app.basicAuth(app.updateCategoryHandler))
	http.Handle("/user/", app.basicAuth(app.userHandler))
	http.Handle("/user/add", app.basicAuth(app.addUserHandler))
	http.Handle("/user/update", app.basicAuth(app.updateUserHandler))
	http.Handle("/user/delete", app.basicAuth(app.deleteUserHandler))
	http.Handle("/pattern/", app.basicAuth(app.patternHandler))
	http.Handle("/pattern/delete", app.basicAuth(app.deletePatternHandler))
	http.Handle("/pattern/add", app.basicAuth(app.addPatternHandler))
	http.Handle("/delete/", app.basicAuth(app.delete))
	http.HandleFunc("/add", app.basicAuth(app.add))
	http.HandleFunc("/import-modal", app.basicAuth(app.importModal))
	http.HandleFunc("/", app.basicAuth(app.handler))

	fmt.Println("Listening on port", app.port)

	err = http.ListenAndServe(fmt.Sprintf(":%d", app.port), nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}

}
