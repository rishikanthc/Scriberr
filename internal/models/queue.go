package models

type QueueStatusByUser struct {
	UserID   uint
	Username string
	Status   JobStatus
	Count    int64
}
