package main

import (
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	ginapiutil "github.com/zfair/ginapi/utils"

	"github.com/zfair/ginapi/examples/generated/ginapi"
	_ "github.com/zfair/ginapi/examples/statik"
)

func init() {
	ginapi.RegisterPetsService(
		&DefaultPetsService{},
		recovery(),
		ginapiutil.UseValidation("/petstore.yaml"),
	)
}

//go:generate statik -src=. -dest=. -include=petstore.yaml
func main() {
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
					return
				}

				c.String(
					http.StatusInternalServerError,
					"internal server error: %v",
					err,
				)
			}
		}()
		c.Next()
	}
}

func (p *DefaultPetsService) CreatePets() (*ginapi.Result, error) {
	id := uuid.NewString()
	p.m.Store(id, &ginapi.Pet{
		Id:   atomic.AddInt64(&p.c, 1),
		Name: id,
	})
	return &ginapi.Result{Message: "ok"}, nil
}

func (p *DefaultPetsService) ListPets(q ginapi.ListPetsQueries) (*ginapi.Pets, error) {
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
		})

		return true
	})

	return &ret, nil
}

func (p *DefaultPetsService) ShowPetById(vars ginapi.ShowPetByIdPathVars) (*ginapi.Pet, error) {
	pet, ok := p.m.Load(vars.PetId)
	if ok {
		return pet.(*ginapi.Pet), nil
	}
	return nil, fmt.Errorf("not found: %s", vars.PetId)
}
