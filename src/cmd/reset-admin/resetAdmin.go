package resetadmin

import (
	"context"
	"fmt"

	"github.com/DwifteJB/aplsonic/src/config"
	"github.com/DwifteJB/aplsonic/src/db"
	"github.com/DwifteJB/aplsonic/src/db/schema"
	"gorm.io/gorm"
)

func CMD() {
	ctx := context.Background()

	dsn := config.GenerateDSN()
	if err := db.Connect(dsn); err != nil {
		fmt.Printf("Failed to connect to database: %v\n", err)
		panic(err)
	}

	// reset admin
	rows, err := gorm.G[schema.AdminConfig](db.DB).Delete(ctx)

	if err != nil {
		fmt.Printf("Failed to reset admin: %v\n", err)
		panic(err)
	}

	fmt.Printf("Admin reset successfully. Rows affected: %d\n", rows)
}
