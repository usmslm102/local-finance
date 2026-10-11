package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"local-finance/internal/db"
	"local-finance/internal/models"
)

func (h *Handler) ListSavingsGoals(c *gin.Context) {
	goals, err := h.db.ListSavingsGoals()
	if err != nil {
		c.JSON(500, gin.H{"error": "Unable to load savings goals"})
		return
	}
	c.JSON(200, goals)
}
func (h *Handler) SaveSavingsGoal(c *gin.Context) {
	var goal models.SavingsGoal
	if err := c.ShouldBindJSON(&goal); err != nil {
		c.JSON(400, gin.H{"error": "Enter a valid savings goal"})
		return
	}
	found, err := h.db.SaveSavingsGoal(&goal, c.Param("id"))
	if err != nil {
		if errors.Is(err, db.ErrInvalidGoal) {
			c.JSON(400, gin.H{"error": err.Error()})
		} else {
			c.JSON(500, gin.H{"error": "Unable to save savings goal"})
		}
		return
	}
	if !found {
		c.JSON(404, gin.H{"error": "Savings goal no longer exists. Refresh and try again."})
		return
	}
	c.Status(http.StatusNoContent)
}
func (h *Handler) DeleteSavingsGoal(c *gin.Context) {
	found, err := h.db.DeleteSavingsGoal(c.Param("id"))
	if err != nil {
		c.JSON(500, gin.H{"error": "Unable to delete savings goal"})
		return
	}
	if !found {
		c.JSON(404, gin.H{"error": "Savings goal not found"})
		return
	}
	c.Status(http.StatusNoContent)
}
func (h *Handler) GetDataHealth(c *gin.Context) {
	result, err := h.db.GetDataHealthAccounts()
	if err != nil {
		c.JSON(500, gin.H{"error": "Unable to load data health"})
		return
	}
	c.JSON(200, result)
}
