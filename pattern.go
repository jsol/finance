package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/google/uuid"
	store "github.com/jsol/finance/store"
)

func (app *application) printPatterns(w http.ResponseWriter, r *http.Request, template string) {
	ctx := r.Context()
	data := PatternsData{}

	patterns, _ := app.queries.GetPatterns(ctx)
	categories, _ := app.queries.ListCategories(ctx)

	data.Patterns = patterns
	data.Categories = categories

	app.executeTemplate(w, template, data)
}

func (app *application) patternHandler(w http.ResponseWriter, r *http.Request) {

	app.printPatterns(w, r, "patterns")
}

func (app *application) addPatternHandler(w http.ResponseWriter, r *http.Request) {

	pattern := r.PostFormValue("pattern")
	category := r.PostFormValue("category")

	args := store.AddPatternParams{Pattern: pattern, Category: []byte(category), ID: newId()}

	err := app.queries.AddPattern(r.Context(), args)

	if err != nil {
		fmt.Printf("Error adding pattern %s: ", pattern)
		fmt.Println(err)
	}

	app.printPatterns(w, r, "pattern_list")
}

func (app *application) deletePatternHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	uuid, err := uuid.Parse(id)

	if err != nil {
		log.Println(err)
		w.WriteHeader(400)
		return
	}

	err = app.queries.RemovePattern(r.Context(), []byte(uuid[:]))

	if err != nil {
		fmt.Println(err)
		w.WriteHeader(400)
		return
	}

	app.printPatterns(w, r, "pattern_list")
}
