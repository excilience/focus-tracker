package main

import "time"

type Activity struct {
	ID         string
	Title      string
	IsArchived bool
	CreatedAt  time.Time
}

type ActivityResponse struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	IsArchived bool   `json:"is_archived"`
	CreatedAt  string `json:"created_at"`
}

type createActivityRequest struct {
	Title string `json:"title"`
}

type updateActivityRequest struct {
	Title      *string `json:"title,omitempty"`
	IsArchived *bool   `json:"is_archived,omitempty"`
}
