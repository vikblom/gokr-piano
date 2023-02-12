package piano

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/url"
	"sync"

	"github.com/amacneil/dbmate/pkg/dbmate"
)

var migrate sync.Once

// MigrateUp to and including the latest sql under ./sql/migrations.
func MigrateUp(url *url.URL) error {
	var err error
	migrate.Do(func() {
		dbm := dbmate.New(url)

		buf := bytes.NewBuffer(nil)
		dbm.Log = buf
		dbm.AutoDumpSchema = false

		err = dbm.Migrate()
		out, _ := io.ReadAll(buf)
		log.Printf("%s\n", out)
		if err != nil {
			err = fmt.Errorf("migrate: %w", err)
		}
	})
	return err
}
