package models

import (
	"context"
	"time"

	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type RecipeModel struct {
	pool *pgxpool.Pool
}

func NewRecipeModel(pool *pgxpool.Pool) *RecipeModel {
	return &RecipeModel{
		pool: pool,
	}
}

type Recipe struct {
	Id       int
	Title    string
	Book     string
	PageNum  int
	Notes    string
	LastMade time.Time
}

// endpoints for handlers
func (rm *RecipeModel) Create(ctx context.Context, title, book, notes string, page int, lastMade time.Time) error {
	_, err := rm.pool.Exec(ctx,
		"insert into recipes (title, book, page_num, notes, last_made) values ($1, $2, $3, $4, $5)",
		title,
		book,
		page,
		notes,
		lastMade,
	)

	return err
}

func (rm *RecipeModel) Update(ctx context.Context, recipe Recipe) error {
	_, err := rm.pool.Exec(ctx,
		"update recipes set title = $1, book = $2, page_num = $3, notes = $4, last_made = $5 where id = $6",
		recipe.Title,
		recipe.Book,
		recipe.PageNum,
		recipe.Notes,
		recipe.LastMade,
		recipe.Id,
	)

	return err
}

func (rm *RecipeModel) All(ctx context.Context) ([]Recipe, error) {
	rs, err := rm.pool.Query(ctx, "select id, title, book, page_num, notes, last_made from recipes")
	if err != nil {
		return []Recipe{}, err
	}

	recipes := make([]Recipe, 0)
	for rs.Next() {
		recipe, err := scanToRecipe(rs)
		if err != nil {
			return []Recipe{}, err
		}

		recipes = append(recipes, recipe)
	}
	return recipes, nil
}

func (rm *RecipeModel) GetById(ctx context.Context, id int) (Recipe, error) {
	r := rm.pool.QueryRow(ctx, "select id, title, book, page_num, notes, last_made from recipes where id = $1",
		id,
	)

	return scanToRecipe(r)
}

func (rm *RecipeModel) GetRandom(ctx context.Context) (Recipe, error) {
	r := rm.pool.QueryRow(ctx, "select id, title, book, page_num, notes, last_made from recipes order by RANDOM() limit 1")

	return scanToRecipe(r)
}

func (rm *RecipeModel) Delete(ctx context.Context, id int) error {
	_, err := rm.pool.Exec(ctx, "delete from recipes where id = $1", id)
	return err
}

func (rm *RecipeModel) Search(ctx context.Context, text string) ([]Recipe, error) {
	// this query is not optimal performance-wise, I don't think
	// suggestion from an SO link I closed: create a column called search_fields that combines all desired
	// searchable text and create an index on that column

	queryStr := `
	select id, title, book, page_num, notes, last_made
	from recipes
	where
	LOWER(title) || LOWER(book) || LOWER(notes) like '%' || LOWER($1) || '%'
	`
	rs, err := rm.pool.Query(ctx, queryStr, text)
	if err != nil {
		return []Recipe{}, err
	}

	recipes := make([]Recipe, 0)
	for rs.Next() {
		recipe, err := scanToRecipe(rs)
		if err != nil {
			return []Recipe{}, err
		}

		recipes = append(recipes, recipe)
	}
	return recipes, nil
}

// ----------
// helpers

// scans a pgx.Row into a Recipe struct
// works for pgx.Rows as well since pgx.Row is defined as type Row Rows
func scanToRecipe(r pgx.Row) (Recipe, error) {
	var id int
	var title, book string
	var pageNum int
	var lastMadePg pgtype.Date
	var notesPg pgtype.Text

	if err := r.Scan(&id, &title, &book, &pageNum, &notesPg, &lastMadePg); err != nil {
		return Recipe{}, err
	}

	// for nullable fields, check for presence or leave as zero value
	lastMade := time.Time{} // time.Time{}.IsZero() == true
	if lastMadePg.Status == pgtype.Present {
		lastMade = lastMadePg.Time
	}

	notes := ""
	if notesPg.Status == pgtype.Present {
		notes = notesPg.String
	}

	return Recipe{
		Id:       id,
		Title:    title,
		Book:     book,
		PageNum:  pageNum,
		Notes:    notes,
		LastMade: lastMade,
	}, nil
}
