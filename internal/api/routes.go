package api

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"local-finance/internal/db"
	"local-finance/internal/mcp"
	"local-finance/internal/service"
)

func SetupRouter(database *db.DB, svc *service.TransactionService, staticFS fs.FS, managers ...*mcp.Manager) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	// Enable CORS for local dev server
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "http://localhost:3000", "http://127.0.0.1:5173", "http://127.0.0.1:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	manager := mcp.NewManager(database)
	if len(managers) > 0 {
		manager = managers[0]
	}
	h := newHandler(database, svc, manager)

	// API Group
	apiGroup := r.Group("/api")
	apiGroup.Use(h.AuthMiddleware())
	apiGroup.Use(mcpManagementProtection())
	{
		apiGroup.GET("/health", h.HealthCheck)
		apiGroup.GET("/auth/status", h.GetAuthStatus)
		apiGroup.POST("/auth/setup", h.SetupAuth)
		apiGroup.POST("/auth/login", h.Login)
		apiGroup.POST("/auth/logout", h.Logout)
		apiGroup.POST("/auth/change-password", h.ChangePassword)
		apiGroup.POST("/auth/disable", h.DisableAuth)
		apiGroup.PUT("/auth/settings", h.UpdateSecuritySettings)

		apiGroup.GET("/mcp/settings", h.GetMCPSettings)
		apiGroup.PUT("/mcp/settings", h.UpdateMCPSettings)
		apiGroup.POST("/mcp/token/rotate", h.RotateMCPToken)

		apiGroup.GET("/accounts", h.ListAccounts)
		apiGroup.PUT("/accounts/:id", h.UpdateAccount)
		apiGroup.GET("/transactions", h.ListTransactions)
		apiGroup.GET("/transactions/:id", h.GetTransaction)
		apiGroup.PATCH("/transactions/:id", h.UpdateTransaction)
		apiGroup.PUT("/transactions/:id", h.UpdateTransaction)
		apiGroup.POST("/statements/upload", h.UploadStatement)
		apiGroup.POST("/statements/upload-batch", h.UploadBatchStatements)
		apiGroup.POST("/statements/preview", h.PreviewStatement)
		apiGroup.POST("/statements/preview-batch", h.PreviewBatchStatements)
		apiGroup.GET("/statements", h.ListStatementImports)
		apiGroup.GET("/categories", h.ListCategories)
		apiGroup.POST("/categories", h.CreateCategory)
		apiGroup.DELETE("/categories/:id", h.DeleteCategory)
		apiGroup.GET("/rules", h.ListRules)
		apiGroup.POST("/rules", h.CreateRule)
		apiGroup.PUT("/rules/:id", h.UpdateRule)
		apiGroup.DELETE("/rules/:id", h.DeleteRule)
		apiGroup.POST("/rules/reapply", h.ReapplyRules)
		apiGroup.GET("/analytics/overview", h.GetAnalyticsOverview)
		apiGroup.GET("/analytics/monthly-review", h.GetMonthlyReview)
		apiGroup.GET("/analytics/monthly-review/transactions", h.GetMonthlyReviewEvidence)
		apiGroup.GET("/analytics/cashflow", h.GetCashFlowIntelligence)
		apiGroup.GET("/analytics/salary", h.GetSalaryInsights)
		apiGroup.GET("/analytics/wrapped", h.GetWrappedStory)
		apiGroup.GET("/parsers", h.ListParsers)
		apiGroup.GET("/credit-cards/bills", h.ListCreditCardBills)
		apiGroup.GET("/cards/portfolio", h.GetCardPortfolioOverview)
		apiGroup.POST("/cards/recommend", h.RecommendBestCard)
		apiGroup.PUT("/cards/:id", h.UpdateCardMetadata)
		apiGroup.GET("/cards/rules", h.ListCardRewardRules)
		apiGroup.POST("/cards/rules", h.CreateCardRewardRule)
		apiGroup.PUT("/cards/rules/:id", h.UpdateCardRewardRule)
		apiGroup.DELETE("/cards/rules/:id", h.DeleteCardRewardRule)
		apiGroup.GET("/subscriptions", h.GetSubscriptionsSummary)
		apiGroup.POST("/subscriptions/scan", h.ScanSubscriptions)
		apiGroup.POST("/subscriptions", h.CreateSubscription)
		apiGroup.PUT("/subscriptions/:id", h.UpdateSubscription)
		apiGroup.DELETE("/subscriptions/:id", h.DeleteSubscription)
		apiGroup.GET("/budgets", h.GetBudgetSummary)
		apiGroup.POST("/budgets", h.UpsertCategoryBudget)
		apiGroup.DELETE("/budgets/:categoryId", h.DeleteCategoryBudget)
		apiGroup.GET("/reconciliation/summary", h.GetReconciliationSummary)
		apiGroup.POST("/reconciliation/scan", h.ScanReconciliation)
		apiGroup.POST("/reconciliation/link", h.LinkTransferPair)
		apiGroup.POST("/reconciliation/unlink", h.UnlinkTransferPair)
		apiGroup.POST("/reconciliation/toggle-exclude/:id", h.ToggleExcludeTransaction)
		apiGroup.GET("/merchants", h.ListMerchants)
		apiGroup.GET("/merchants/:name", h.GetMerchantProfile)
		apiGroup.GET("/samples", h.ListSampleStatements)
		apiGroup.GET("/samples/download", h.ServeSampleStatement)
		apiGroup.GET("/database/info", h.GetDatabaseInfo)
		apiGroup.GET("/database/backup", h.BackupDatabase)
		apiGroup.POST("/database/restore", h.RestoreDatabase)
		apiGroup.GET("/database/export/json", h.ExportDatabaseJSON)
		apiGroup.GET("/database/export/csv", h.ExportTransactionsCSV)
		apiGroup.POST("/database/reset", h.ResetDatabase)
		apiGroup.GET("/system/version", h.GetSystemVersion)
		apiGroup.POST("/system/update", h.ApplySystemUpdate)
	}

	// Serve Static Frontend if embedded FS provided
	if staticFS != nil {
		httpFS := http.FS(staticFS)
		fileServer := http.FileServer(httpFS)

		r.NoRoute(func(c *gin.Context) {
			path := c.Request.URL.Path
			if strings.HasPrefix(path, "/api") {
				c.JSON(http.StatusNotFound, gin.H{"error": "API endpoint not found"})
				return
			}

			// Check if file exists in static FS
			f, err := staticFS.Open(strings.TrimPrefix(path, "/"))
			if err == nil {
				_ = f.Close()
				fileServer.ServeHTTP(c.Writer, c.Request)
				return
			}

			// Fallback to index.html for client-side routing (TanStack Router)
			c.Request.URL.Path = "/"
			fileServer.ServeHTTP(c.Writer, c.Request)
		})
	}

	return r
}
