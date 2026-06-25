package main

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"daddy/apps/breezy-api/internal/shared"
)

func newUserRouter(service *userService) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	router.HandleMethodNotAllowed = true
	router.MaxMultipartMemory = maxBannerSize

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, shared.NewHealthResponse("user-service"))
	})

	// Toutes les routes /users arrivent du gateway, qui a déjà validé le JWT et
	// posé X-User-Id. RequireIdentity rejette en 401 si le header est absent.

	// Route de lookup déclarée avant le groupe pour éviter le conflict avec /:id.
	router.GET("/users/lookup", shared.RequireIdentity(), service.getUserByUsernameHandler)

	users := router.Group("/users", shared.RequireIdentity())
	{
		users.GET("", service.listUsersHandler)
		users.GET("/suggestions", service.suggestionsHandler)
		users.PATCH("/me", service.updateProfileHandler)
		users.GET("/me/follow-requests", service.listFollowRequestsHandler)
		users.POST("/me/follow-requests/:id/accept", service.acceptFollowRequestHandler)
		users.DELETE("/me/follow-requests/:id", service.rejectFollowRequestHandler)
		users.PUT("/me/avatar", service.uploadAvatarHandler)
		users.DELETE("/me/avatar", service.deleteAvatarHandler)
		users.PUT("/me/banner", service.uploadBannerHandler)
		users.DELETE("/me/banner", service.deleteBannerHandler)
		users.GET("/me/blocked", service.listBlockedHandler)
		users.GET("/me/blocked-by", service.listBlockedByHandler)
		users.GET("/:id", service.getProfileHandler)
		users.POST("/:id/follow", service.followHandler)
		users.DELETE("/:id/follow", service.unfollowHandler)
		users.GET("/:id/following", service.listFollowingHandler)
		users.GET("/:id/followers", service.listFollowersHandler)
		users.POST("/:id/block", service.blockHandler)
		users.DELETE("/:id/block", service.unblockHandler)
	}

	// Modération : sanctions (ban/suspension). Réservé au staff (modérateur /
	// administrateur). Le rôle est lu dans le header X-User-Role posé par le gateway.
	staff := router.Group("/users", shared.RequireRole(shared.RoleModerator, shared.RoleAdministrator))
	{
		staff.POST("/:id/sanctions", service.sanctionUserHandler)
		staff.DELETE("/:id/sanctions", service.liftSanctionHandler)
		staff.GET("/:id/sanctions", service.listSanctionsHandler)
	}

	// Routes internes service-à-service (jamais exposées par le gateway).
	internal := router.Group("/internal", requireInternalKey(service.cfg.InternalKey))
	{
		internal.POST("/users", service.createInternalUserHandler)
		internal.GET("/users/hidden-authors", service.hiddenAuthorsHandler)
		internal.GET("/users/:id/can-view", service.canViewHandler)
		internal.GET("/users/:id/following-ids", service.followingIDsHandler)
		internal.POST("/users/resolve-mentions", service.resolveUsernamesHandler)
		internal.POST("/users/batch-avatars", service.batchAvatarsHandler)
	}

	router.NoMethod(func(c *gin.Context) {
		c.String(http.StatusMethodNotAllowed, "method not allowed")
	})

	return router
}
