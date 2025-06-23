package models

import (
	"encoding/json"
	"math"
	"strconv"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

type PaginatorQueryUri struct {
	Page  json.Number `form:"page" binding:"number,gte_number=1,lte_number=1000"`
	Limit json.Number `form:"limit" binding:"number,gte_number=1,lte_number=1000"`
}

type PaginatorQuery struct {
	Page  int `json:"page"`
	Limit int `json:"limit"`
}

type PaginatorData struct {
	CurrentPage int  `json:"current_page"`
	NextPage    *int `json:"next_page"`
	PrevPage    *int `json:"prev_page"`
	TotalCount  int  `json:"total_count"`
	TotalPages  int  `json:"total_pages"`
}

func init() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("gte_number", gteNumber)
		v.RegisterValidation("lte_number", lteNumber)
	}
}

func TransformPaginator(pagUri PaginatorQueryUri) PaginatorQuery {
	var paginatorQuery PaginatorQuery

	paginatorQuery.Page, _ = strconv.Atoi(pagUri.Page.String())
	paginatorQuery.Limit, _ = strconv.Atoi(pagUri.Limit.String())

	return paginatorQuery
}

func GetPaginatorMeta(currentPage, limit, totalCount int) PaginatorData {
	var prevPage, nextPage *int
	totalPages := int(math.Ceil(float64(totalCount) / float64(limit)))

	if currentPage > 1 {
		p := currentPage - 1
		prevPage = &p
	}

	if currentPage < totalPages {
		n := currentPage + 1
		nextPage = &n
	}

	return PaginatorData{
		CurrentPage: currentPage,
		PrevPage:    prevPage,
		NextPage:    nextPage,
		TotalPages:  totalPages,
		TotalCount:  totalCount,
	}
}
