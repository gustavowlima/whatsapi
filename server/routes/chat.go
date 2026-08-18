package routes

import (
	"github.com/labstack/echo/v4"
	"github.com/verbeux-ai/whatsmiau/lib/whatsmiau"
	"github.com/verbeux-ai/whatsmiau/repositories/instances"
	"github.com/verbeux-ai/whatsmiau/server/controllers"
	"github.com/verbeux-ai/whatsmiau/services"
)

func Chat(group *echo.Group) {
	redisInstance := instances.NewRedis(services.Redis())
	controller := controllers.NewChats(redisInstance, whatsmiau.Get())
	contactController := controllers.NewContacts(whatsmiau.Get())

	group.POST("/presence", controller.SendChatPresence)
	group.POST("/read-messages", controller.ReadMessages)
	group.POST("/fetchProfilePictureUrl", controller.FetchProfilePicture)
	group.POST("/findContact", contactController.FindContact)
	group.DELETE("/deleteMessageForEveryone", controller.DeleteMessageForEveryone)
}

func ChatEVO(group *echo.Group) {
	redisInstance := instances.NewRedis(services.Redis())
	controller := controllers.NewChats(redisInstance, whatsmiau.Get())
	contactController := controllers.NewContacts(whatsmiau.Get())

	// Evolution API Compatibility (partially REST)
	group.POST("/markMessageAsRead/:instance", controller.ReadMessages)
	group.POST("/sendPresence/:instance", controller.SendChatPresence)
	group.POST("/whatsappNumbers/:instance", controller.NumberExists)
	group.POST("/fetchProfilePictureUrl/:instance", controller.FetchProfilePicture)
	group.POST("/findContact/:instance", contactController.FindContact)
	group.DELETE("/deleteMessageForEveryone/:instance", controller.DeleteMessageForEveryone)
}
