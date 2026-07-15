package main

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "baguette",
	})
}

func rootHandler(c *gin.Context, collection *mongo.Collection) {
	now := time.Now()

	res, err := collection.InsertOne(c.Request.Context(), bson.M{"created_at": now.Format(time.RFC3339)})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"_id":        res.InsertedID.(bson.ObjectID).String(),
		"created_at": now.Format(time.RFC3339),
	})
}

func logsHandler(c *gin.Context, collection *mongo.Collection) {
	cur, err := collection.Find(c.Request.Context(), bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	defer cur.Close(c.Request.Context())

	var results []bson.M
	if err = cur.All(c.Request.Context(), &results); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"logs": results,
	})
}
