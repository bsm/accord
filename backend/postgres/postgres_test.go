package postgres_test

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/bsm/accord/backend/internal/testdata"
	"github.com/bsm/accord/backend/postgres"
	. "github.com/bsm/ginkgo/v2"
	. "github.com/bsm/gomega"
	"github.com/joho/godotenv"
)

var _ = Describe("Backend", func() {
	var data testdata.BehavesLikeBackendData
	var db *sql.DB

	BeforeEach(func() {
		dsn := os.Getenv("DATABASE_DSN")
		if dsn == "" {
			Fail("missing DATABASE_DSN environment variable")
		}

		var err error
		db, err = sql.Open("postgres", dsn)
		Expect(err).NotTo(HaveOccurred())

		rows, err := db.Query("SELECT table_name FROM information_schema.tables WHERE table_type = 'BASE TABLE' AND table_schema = 'public'")
		Expect(err).NotTo(HaveOccurred())
		defer rows.Close()

		for rows.Next() {
			var name string
			Expect(rows.Scan(&name)).To(Succeed())
			Expect(db.Exec("DROP TABLE " + name)).NotTo(BeNil())
		}

		data.Subject, err = postgres.OpenDB(context.Background(), db)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		if db != nil {
			Expect(db.Close()).To(Succeed())
		}
	})

	Context("defaults", testdata.BehavesLikeBackend(&data))
})

// ------------------------------------------------------------------------

var _ = BeforeSuite(func() {
	_ = godotenv.Load("../../.env", "../.env", ".env")
})

func TestSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "accord/backend/postgres")
}
