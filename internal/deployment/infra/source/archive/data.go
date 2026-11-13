package archive

import (
	"github.com/checkship-deploy/checkship/internal/deployment/app/get_deployment"
	"github.com/checkship-deploy/checkship/internal/deployment/domain"
)

type Data string

func (p Data) Kind() string             { return "archive" }
func (p Data) NeedVersionControl() bool { return false }

func init() {
	domain.SourceDataTypes.Register(Data(""), func(value string) (domain.SourceData, error) {
		return Data(value), nil
	})

	get_deployment.SourceDataTypes.Register(Data(""), func(value string) (get_deployment.SourceData, error) {
		return Data(value), nil
	})
}
