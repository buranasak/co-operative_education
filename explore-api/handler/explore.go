package handler

import (
	"context"
	"encoding/json"
	"explore-api/database"
	"explore-api/model"
	"explore-api/tool"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson"
)

func (h *Handler) ExploreServiceUsages(c echo.Context) error {
	var err error

	body := new(model.ExploreRequest)

	err = c.Bind(body)
	if err != nil {
		return c.JSON(http.StatusBadRequest, &model.Exception{
			Status: tool.IntToPointer(http.StatusBadRequest),
			Detail: err.(*echo.HTTPError).Message.(string),
		})
	}

	pipeline := []bson.M{}
	match := bson.M{}



	//
	// filter stage 
	//
	if body.Filter != nil {

		arguments := body.Filter.Arguments
		args := []interface{}{}

		for _, a := range arguments {

			b, _ := json.Marshal(a)

			var arg *model.ExploreFilter
			err = json.Unmarshal(b, &arg)
			if err != nil {
				return c.JSON(http.StatusBadRequest, &model.Exception{
					Status: tool.IntToPointer(http.StatusBadRequest),
					Detail: "Body 'filter' is invalid",
				})
			}
		}

		body.Filter.Arguments = args

		match, err = database.FilterToBsonM(body.Filter)
		if err != nil {
			return c.JSON(http.StatusBadRequest, &model.Exception{
				Status: tool.IntToPointer(http.StatusBadRequest),
				Detail: "Body 'filter' is invalid, " + err.Error(),
			})
		}
	}

	// 
	// match stage
	//
	if len(match) > 0 {
		pipeline = append(pipeline, bson.M{
			"$match": match,
		})
	}





	//
	// group
	//
	groupId := bson.M{}

	//columns
	for _, col := range body.Columns {
		name := strings.ReplaceAll(col.Name, ".", "_")
		groupId[name] = "$" + database.ChangeKeyId(col.Name)
	}

	group := bson.M{}

	if len(groupId) > 0 {
		group["_id"] = groupId

	} else {
		group["_id"] = nil
	}
	
	//aggregate 
	for _, ag := range body.Aggregate { 
		column := strings.ReplaceAll(ag.Column, ".", "_")

		aggregate := bson.M{}

		if strings.ToLower(ag.Aggregate) == "count" {
			aggregate["$sum"] = 1
		} else {
			aggregate["$"+ag.Aggregate] = "$" + database.ChangeKeyId(ag.Column)
		}
		group[column] = aggregate
	}
	
	pipeline = append(pipeline, bson.M{
		"$group": group,
	})




	//
	// project
	//
	project := bson.M{
		"_id": 0,
	}

	for _, col := range body.Columns {
		alias := col.Alias
		if alias == "" {
			alias = col.Name
		}
		project[alias] = "$_id." + strings.ReplaceAll(col.Name, ".", "_")
	}

	for _, ag := range body.Aggregate {
		column := strings.ReplaceAll(ag.Column, ".", "_")

		project[ag.Alias] = "$" + column
	}

	pipeline = append(pipeline, bson.M{
		"$project": project,
	})




	//
	// sort
	//
	sort := bson.D{}

	for _, col := range body.Columns {
		alias := col.Alias
		if alias == "" {
			alias = col.Name
		}

		sort = append(sort, bson.E{Key: alias, Value: 1})
	}

	if len(body.Sorts) > 0 {
		sort = bson.D{}

		for _, s := range body.Sorts {
			direction := 1
			if strings.ToLower(s.Direction) == "desc" {
				direction = -1
			}
			sort = append(sort, bson.E{Key: s.Column, Value: direction})
		}
	}

	pipeline = append(pipeline, bson.M{
		"$sort": sort,
	})




	//
	// offset stage
	//
	offset := 0
	if body.Limit != nil {
		offset = *body.Offset
	}

	pipeline = append(pipeline, bson.M{
		"$skip": offset,
	})




	

	//
	// limit stage
	//
	limit := 10
	if body.Limit != nil {
		limit = *body.Limit
	}

	pipeline = append(pipeline, bson.M{
		"$limit": limit,
	})


	

	//
	// result ผลลัพธ์
	//
	aggServiceUsages, err := h.DB.AggregateServiceUsage(context.Background(), pipeline)
	if err != nil {
		return c.JSON(http.StatusUnprocessableEntity, &model.Exception{
			Status: tool.IntToPointer(http.StatusUnprocessableEntity),
			Detail: "Could not explore service usages, " + err.Error(),
		})
	}

	results := []interface{}{}
	for _, p := range aggServiceUsages {
		results = append(results, p)
	}

	response := &model.Explores{
		Results: results,
	}

	return c.JSON(http.StatusOK, response)
}
