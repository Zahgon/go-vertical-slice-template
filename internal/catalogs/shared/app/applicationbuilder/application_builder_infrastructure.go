package applicationbuilder

import (
	"github.com/mehdihadeli/go-vertical-slice-template/config"
	"github.com/mehdihadeli/go-vertical-slice-template/internal/pkg/database"
	"github.com/mehdihadeli/go-vertical-slice-template/internal/pkg/http/ginweb"
)

func (b *ApplicationBuilder) AddInfrastructure() {
	err := config.AddAppConfig(b.Container)
	if err != nil {
		b.Logger.Fatal(err)
	}

	err = database.AddGorm(b.Container)
	if err != nil {
		b.Logger.Fatal(err)
	}

	err = ginweb.AddGin(b.Container)
	if err != nil {
		b.Logger.Fatal(err)
	}
}
