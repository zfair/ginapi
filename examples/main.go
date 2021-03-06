package main

import (
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"

	ginapiutil "github.com/anqur/ginapi/utils"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/anqur/ginapi/examples/generated/ginapi"
	_ "github.com/anqur/ginapi/examples/generated/statik"
)

//go:generate docker run --rm -v $PWD:/local openapitools/openapi-generator-cli generate -i /local/petstore.yaml -g go-gin-server -o /local/generated
//go:generate ginapi -i generated -vars {"server":"http://localhost:8088"} -ctx -ignored-tags ignored
//go:generate statik -src=. -dest=./generated -include=petstore.yaml
func main() {
	ginapi.RegisterPetsService(
		&DefaultPetsService{},
		recovery(),
		ginapiutil.UseValidation("/petstore.yaml"),
	)
	r := ginapi.Initialize(gin.Default())
	if err := r.Run("localhost:8088"); err != nil {
		panic(err)
	}
}

type DefaultPetsService struct {
	m sync.Map
	c int64
}

func recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				err, ok := r.(error)
				if !ok {
					panic(r)
				}

				if _, ok := err.(*openapi3filter.RequestError); ok {
					c.String(
						http.StatusBadRequest,
						"invalid parameter: %v",
						err,
					)
					c.Abort()
					return
				}

				c.String(
					http.StatusInternalServerError,
					"internal server error: %v",
					err,
				)
				c.Abort()
			}
		}()
		c.Next()
	}
}

func (p *DefaultPetsService) CreatePets(_ *gin.Context, h ginapi.CreatePetsHeaders) (*ginapi.Result, error) {
	id := uuid.NewString()
	pet := &ginapi.Pet{
		Id:   atomic.AddInt64(&p.c, 1),
		Name: id,
		Tag:  id,
	}
	if h.XTag != nil {
		pet.Tag = *h.XTag
	}
	p.m.Store(id, pet)
	return &ginapi.Result{Message: "ok"}, nil
}

func (p *DefaultPetsService) ListPets(_ *gin.Context, q ginapi.ListPetsQueries) (*ginapi.Pets, error) {
	var n int32
	var ret ginapi.Pets

	p.m.Range(func(key, value interface{}) bool {
		if n++; q.Limit != nil && n > *q.Limit {
			return false
		}

		pet := value.(*ginapi.Pet)
		ret = append(ret, ginapi.Pet{
			Id:   pet.Id,
			Name: pet.Name,
			Tag:  pet.Tag,
		})

		return true
	})

	return &ret, nil
}

func (p *DefaultPetsService) ShowPetById(_ *gin.Context, vars ginapi.ShowPetByIdPathVars) (*ginapi.Pet, error) {
	pet, ok := p.m.Load(vars.PetId)
	if ok {
		return pet.(*ginapi.Pet), nil
	}
	return nil, fmt.Errorf("not found: %s", vars.PetId)
}

func (p *DefaultPetsService) DeletePet(_ *gin.Context, vars ginapi.DeletePetPathVars) error {
	p.m.Delete(vars.PetId)
	return nil
}

func (p *DefaultPetsService) UploadFile(
	_ *gin.Context,
	vars ginapi.UploadFilePathVars,
	q ginapi.UploadFileQueries,
	req []byte,
) (*ginapi.Result, error) {
	petID := vars.PetId
	fmt.Printf("petID: %s, fileLen: %d\n", petID, len(req))

	if metadata := q.AdditionalMetadata; metadata != nil {
		fmt.Printf("petID: %s, metadata: %s\n", petID, *metadata)
	}

	return &ginapi.Result{Message: "ok"}, nil
}
