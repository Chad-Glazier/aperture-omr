package sys

import (
	"context"
	"database/sql"
	_ "embed"
	"testing"
	"time"

	"ubco-team15/omr/internal/pdf"
	"ubco-team15/omr/internal/sys/sqlc"

	_ "modernc.org/sqlite" // sqlite3 driver
)

//
// We use SQLite3 to store certain persistent data about the system.
//

//go:embed sqlc/schema.sql
var databaseInit string

// The interface we used to interact with the SQLite3 database.
var db sqlc.Querier

// The path to the SQLite data file.
const dbPath = "data/sys.sqlite3"

func init() {

	//
	// Start up the database.
	//

	path := dbPath
	if testing.Testing() {
		path = ":memory:"
	}

	cnx, err := sql.Open("sqlite", path)
	if err != nil {
		panic("failed to open sys database")
	}

	cnx.SetMaxOpenConns(1)

	db = sqlc.New(cnx)

	ctx := context.Background()
	if _, err := cnx.ExecContext(ctx, databaseInit); err != nil {
		panic("failed to initialize sys database")
	}

	//
	// Once the database is set up, we check it for cached performance sampling
	// results. If the caches miss, we run the sampling right now.
	//

	if testing.Testing() { // Skip sampling if we're using "go test"
		return
	}

	cachedPdfMemCost, err := db.GetPdfRenderCosts(context.Background())
	if err != nil {
		Log("sampling PDF rendering costs...")
		pdfMemCost := pdf.MustRunSampling()
		pdf.SetMemoryCostVars(pdfMemCost)
		db.CreatePdfRenderCosts(
			context.Background(),
			sqlc.CreatePdfRenderCostsParams{
				PdfRenderBaseline:  int64(pdfMemCost.Baseline),
				PdfRenderIncrement: int64(pdfMemCost.Increment),
			},
		)
		Log("sampling complete", "values", pdfMemCost)
	} else {
		pdf.SetMemoryCostVars(pdf.MemCostVars{
			Baseline:  uint64(cachedPdfMemCost.PdfRenderBaseline),
			Increment: uint64(cachedPdfMemCost.PdfRenderIncrement),
		})
		Log(
			"sampled PDF rendering costs loaded from cache",
			"sampling date",
			time.
				UnixMilli(cachedPdfMemCost.SampledAt).
				Local().
				Format("2006-01-02 15:04:05"),
		)
	}
}
