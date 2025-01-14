package controllers

import (
	"encoding/json"
	"net/http"

	"github.com/labstack/echo"
)

type Data struct {
	Name    string `json:"Name"`
	Age     int    `json:"Age"`
	Address string `json:"Address"`
	Email   string `json:"Email"`
}

func PrivateCacheUseCase(c echo.Context) error {
	// ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	// defer cancel()
	// fmt.Println(ctx)

	data := Data{
		Name:    "buranasak",
		Age:     22,
		Address: "153/3 sakon nakhon",
		Email:   "Buranasak.s@kkumail.com",
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	etag := generateETag(string(jsonData))

	if c.Request().Header.Get("If-None-Match") != "" && c.Request().Header.Get("If-None-Match") == etag {
		c.Response().Header().Set("Cache-Control", "private, max-age=300")
		c.Response().Header().Set("Etag", etag)

		return c.NoContent(http.StatusNotModified)
	}

	c.Response().Header().Set("Cache-Control", "private, max-age=300")
	c.Response().Header().Set("Etag", etag)

	return c.JSON(http.StatusOK, data)
}





