package routes

import (
	"main/auth"
	"main/handlers"
	"os"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func Setup(r *gin.Engine, db *gorm.DB) {
	vehicleHandler := &handlers.VehicleHandler{DB: db}
	authHandler := &handlers.AuthHandler{DB: db}
	notesHandler := &handlers.NotesHandler{DB: db}
	categoriesHandler := &handlers.CategoryHandler{DB: db}
	adminHandler := &handlers.AdminHandler{DB: db}
	historyHandler := &handlers.HistoryHandler{DB: db}
	dnrHandler    := &handlers.DNRHandler{DB: db}
	partsHandler  := &handlers.PartsHandler{DB: db}
	importHandler := &handlers.ImportHandler{DB: db}
	gmHandler     := &handlers.GMHandler{DB: db}
	forkHandler   := &handlers.ForkHandler{DB: db}

	base := r.Group("/api")

	api := base.Group("/")
	api.Use(auth.RequireAuth(db))
	{
		api.GET("/vin/:vin", vehicleHandler.GetVehicle)
		api.PATCH("/update/:vin", vehicleHandler.UpdateVehicle)
		api.GET("/id/:id", vehicleHandler.GetVehicleById)
		api.GET("/vehicles", vehicleHandler.ListVehicles)

		// GM Parts Giant — live RPO/build-option lookup for a specific VIN.
		// Not persisted; data is VIN-specific and must not be stored under a build key.
		api.GET("/gm/decode/:vin", gmHandler.DecodeGMLive)

		// Build-number fork/range data for a VIN or build key.
		api.GET("/fork/:vin", forkHandler.GetForkData)
		// Write paths (trusted users enforced inside the handlers):
		api.POST("/fork/:vin/point", forkHandler.RecordForkPoint) // a verified VIN sighting
		api.POST("/fork/:vin/range", forkHandler.RecordForkRange) // an authoritative manual span
	}

	a := base.Group("/auth")
	{
		a.POST("/register", authHandler.Register)
		a.POST("/login", authHandler.Login)
		a.POST("/refresh", authHandler.Refresh)
		a.POST("/reset-password", auth.RequireAuth(db), authHandler.ResetPassword)
	}

	notes := base.Group("/notes")
	notes.Use(auth.RequireAuth(db))
	{
		notes.GET("/listing-error", notesHandler.GetListingErrorNotes)
		notes.PATCH("/:note_id/resolve", notesHandler.MarkAsResolved)
		notes.PATCH("/:note_id", notesHandler.UpdateNote)
		notes.DELETE("/:note_id", notesHandler.DeleteNote)
		notes.POST("/:vin", notesHandler.AddNote)
	}

	categories := base.Group("/category")
	categories.Use(auth.RequireAuth(db))
	{
		categories.POST("/", categoriesHandler.AddCategory)
		categories.DELETE("/:category_id", categoriesHandler.DeleteCategory)
		categories.PATCH("/:category_id", categoriesHandler.UpdateCategory)
		categories.GET("/", categoriesHandler.GetCategories)
	}

	admin := base.Group("/admin", auth.RequireAuth(db), auth.RequireAdmin)
	{
		admin.GET("/stats", adminHandler.AdminStatus)
		admin.GET("/users", adminHandler.ListUsers)
		admin.PATCH("/users/:id", adminHandler.UpdateUser)
		admin.DELETE("/users/:id", adminHandler.DeleteUser)

		admin.GET("/notes/chart", adminHandler.GetNotesChart)
		admin.GET("/notes/recent", adminHandler.GetRecentNotes)
		admin.POST("/users", adminHandler.CreateUser)
	}

	history := base.Group("/history")
	history.Use(auth.RequireAuth(db))
	{
		history.GET("/", historyHandler.ListHistory)
		history.PATCH("/:id/verify", historyHandler.VerifyEntry)
		history.DELETE("/:id", historyHandler.DeleteEntry)
	}

	dnr := base.Group("/dnr", auth.RequireAuth(db), auth.RequireDNR)
	{
		dnr.GET("/queue", dnrHandler.GetQueue)
		dnr.GET("/stats", dnrHandler.GetStats)
		dnr.GET("/similar", dnrHandler.GetSimilar)
		dnr.POST("/propagate", dnrHandler.Propagate)
		dnr.POST("/vehicles", dnrHandler.CreateVehicle)
	}

	// Bulk VIN import — admin and DNR team
	importGroup := base.Group("/import", auth.RequireAuth(db), auth.RequireDNR)
	{
		importGroup.POST("/", importHandler.StartImport)
		importGroup.GET("/:job_id", importHandler.GetImportJob)
		importGroup.DELETE("/:job_id", importHandler.CancelImport)
	}

	// Parts catalog — read: any authenticated user; write: handled inside handler
	parts := base.Group("/parts", auth.RequireAuth(db))
	{
		parts.GET("/",                    partsHandler.ListParts)
		parts.GET("/categories",          partsHandler.ListCategories)
		parts.GET("/by-vehicle/:vin",     partsHandler.GetCompatibleParts)
		parts.GET("/:id",                 partsHandler.GetPart)
		parts.POST("/",                   partsHandler.CreatePart)
		parts.PATCH("/:id",               partsHandler.UpdatePart)
		parts.DELETE("/:id",              partsHandler.DeletePart)
		parts.POST("/:id/rules",          partsHandler.AddRule)
		parts.PATCH("/:id/rules/:rule_id",partsHandler.UpdateRule)
		parts.DELETE("/:id/rules/:rule_id",partsHandler.DeleteRule)
		parts.GET("/:id/vehicles",         partsHandler.GetCompatibleVehicles)
		parts.POST("/:id/clone",           partsHandler.ClonePart)
	}

	// Serve frontend
	const frontendDir = "/home/developer/frontend"
	r.Static("/assets", frontendDir+"/assets")
	r.StaticFile("/", frontendDir+"/index.html")
	r.NoRoute(func(c *gin.Context) {
		// If the requested path is a real file in the frontend directory
		// (favicon.svg, robots.txt, manifest.json, etc.), serve it directly.
		// Otherwise fall back to index.html so the SPA handles routing.
		candidate := frontendDir + c.Request.URL.Path
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			c.File(candidate)
			return
		}
		c.File(frontendDir + "/index.html")
	})
}
