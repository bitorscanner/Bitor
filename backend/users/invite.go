package users

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"net/mail"
	"bitor/middleware"
	"bitor/templates/email"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/forms"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/models/schema"
	"github.com/pocketbase/pocketbase/tools/mailer"
)

// generateSecureToken generates a cryptographically secure random token
func generateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

func EnsureInvitationsCollection(app *pocketbase.PocketBase) error {
	// Check if the collection already exists
	existingCollection, _ := app.Dao().FindCollectionByNameOrId("invitations")
	if existingCollection != nil {
		return nil
	}

	// Create a pointer to int with value 1
	maxSelect := new(int)
	*maxSelect = 1

	// Define the collection schema
	invitationsCollection := &models.Collection{
		Name:   "invitations",
		Type:   models.CollectionTypeBase,
		System: false,
		Schema: schema.NewSchema(
			&schema.SchemaField{
				Name:     "email",
				Type:     schema.FieldTypeEmail,
				Required: true,
			},
			&schema.SchemaField{
				Name:     "token",
				Type:     schema.FieldTypeText,
				Required: true,
				Unique:   true,
			},
			&schema.SchemaField{
				Name:     "expires",
				Type:     schema.FieldTypeDate,
				Required: true,
			},
			&schema.SchemaField{
				Name:     "used",
				Type:     schema.FieldTypeBool,
				Required: true,
			},
			&schema.SchemaField{
				Name: "group",
				Type: schema.FieldTypeRelation,
				Options: &schema.RelationOptions{
					CollectionId: "groups",
					MaxSelect:    maxSelect,
				},
			},
		),
	}

	// Save the collection
	if err := app.Dao().SaveCollection(invitationsCollection); err != nil {
		return err
	}

	return nil
}

// RegisterRoutes registers the invitation routes
func RegisterRoutes(app *pocketbase.PocketBase, e *core.ServeEvent) {
	log.Printf("Registering invitation routes...")
	// Create a group for the invitation routes
	invitationsGroup := e.Router.Group("/api/invitations", middleware.RequirePermission(app, "manage_users", "users"))
	log.Printf("Created invitations group with path: /api/invitations")

	// Add the invite endpoint that handles both creation and email sending
	invitationsGroup.POST("/invite", func(c echo.Context) error {
		log.Printf("Handling POST /api/invitations/invite request")
		// Parse request body
		var data struct {
			Email string `json:"email"`
			Group string `json:"group"`
		}

		if err := c.Bind(&data); err != nil {
			return apis.NewBadRequestError("Invalid request body", err)
		}

		log.Printf("Received invitation request - Email: %s, Group: %s", data.Email, data.Group)

		// Generate a secure random token
		token, err := generateSecureToken(32) // 32 bytes = 256 bits
		if err != nil {
			return apis.NewApiError(http.StatusInternalServerError, "Failed to generate secure token", err)
		}

		expires := time.Now().Add(24 * time.Hour)

		// Create the invitation record
		collection, err := app.Dao().FindCollectionByNameOrId("invitations")
		if err != nil {
			return apis.NewApiError(http.StatusInternalServerError, "Failed to find invitations collection", err)
		}

		record := models.NewRecord(collection)
		record.Set("email", data.Email)
		record.Set("token", token)
		record.Set("expires", expires)
		record.Set("used", false)

		// Set the group relation
		if data.Group != "" {
			record.Set("group", data.Group)
			log.Printf("Setting group relation: %s", data.Group)
		} else {
			log.Printf("No group provided in request")
		}

		if err := app.Dao().SaveRecord(record); err != nil {
			log.Printf("Error saving invitation record: %v", err)
			return apis.NewApiError(http.StatusInternalServerError, "Failed to create invitation", err)
		}

		// Verify the saved record
		savedRecord, err := app.Dao().FindRecordById(collection.Id, record.Id)
		if err != nil {
			log.Printf("Error fetching saved record: %v", err)
		} else {
			log.Printf("Saved record - Group: %v", savedRecord.Get("group"))
		}

		// Create the invite URL using PocketBase's built-in settings
		baseURL := app.Settings().Meta.AppUrl
		if baseURL == "" {
			// Fallback to constructing URL from the request if app URL is not set
			baseURL = fmt.Sprintf("%s://%s", c.Scheme(), c.Request().Host)
		}
		inviteUrl := fmt.Sprintf("%s/accept-invite?token=%s", baseURL, token)

		// Send invitation email
		from := mail.Address{
			Name:    app.Settings().Meta.AppName,
			Address: app.Settings().Meta.SenderAddress,
		}

		to := mail.Address{
			Address: data.Email,
		}

		message := &mailer.Message{
			From:    from,
			To:      []mail.Address{to},
			Subject: fmt.Sprintf("Invitation to join %s", app.Settings().Meta.AppName),
			HTML:    email.GetInvitationEmailTemplate(inviteUrl, app.Settings().Meta.AppName),
		}

		if err := app.NewMailClient().Send(message); err != nil {
			return apis.NewApiError(http.StatusInternalServerError, "Failed to send invitation email", err)
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"message": "Invitation sent successfully",
			"token":   token,
		})
	})

	// Add the accept invite endpoint
	e.Router.POST("/api/invitations/accept", func(c echo.Context) error {
		// Parse request body
		var data struct {
			Token     string `json:"token"`
			Username  string `json:"username"`
			Password  string `json:"password"`
			FirstName string `json:"first_name"`
			LastName  string `json:"last_name"`
		}

		if err := c.Bind(&data); err != nil {
			return apis.NewBadRequestError("Invalid request body", err)
		}

		// Find the invitation
		invitationsCollection, err := app.Dao().FindCollectionByNameOrId("invitations")
		if err != nil {
			return apis.NewApiError(http.StatusInternalServerError, "Failed to find invitations collection", err)
		}

		invitation, err := app.Dao().FindFirstRecordByData(invitationsCollection.Id, "token", data.Token)
		if err != nil {
			return apis.NewNotFoundError("Invalid or expired invitation token", err)
		}

		// Check if invitation is expired
		expiresTime := invitation.GetDateTime("expires").Time()
		if expiresTime.Before(time.Now()) {
			return apis.NewBadRequestError("Invitation has expired", nil)
		}

		// Check if invitation is already used
		if invitation.GetBool("used") {
			return apis.NewBadRequestError("Invitation has already been used", nil)
		}

		// Create the user
		collection, err := app.Dao().FindCollectionByNameOrId("users")
		if err != nil {
			return apis.NewApiError(http.StatusInternalServerError, "Failed to find users collection", err)
		}

		record := models.NewRecord(collection)
		record.Set("username", data.Username)
		record.Set("password", data.Password)
		record.Set("passwordConfirm", data.Password)
		record.Set("first_name", data.FirstName)
		record.Set("last_name", data.LastName)
		record.Set("email", invitation.GetString("email"))
		record.Set("emailVisibility", true)
		record.Set("verified", true)

		// Set the group if one was specified in the invitation
		if group := invitation.GetString("group"); group != "" {
			record.Set("group", group)
		}

		// Create the user with the password
		form := forms.NewRecordUpsert(app, record)
		form.Password = data.Password
		form.PasswordConfirm = data.Password

		if err := form.Submit(); err != nil {
			log.Printf("Error creating user: %v", err)
			return apis.NewApiError(http.StatusInternalServerError, "Failed to create user", err)
		}

		// Mark invitation as used
		invitation.Set("used", true)
		if err := app.Dao().SaveRecord(invitation); err != nil {
			log.Printf("Failed to mark invitation as used: %v", err)
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"message": "Account created successfully",
		})
	})

	// Add the validate endpoint
	e.Router.GET("/api/invitations/validate", func(c echo.Context) error {
		token := c.QueryParam("token")
		if token == "" {
			return apis.NewBadRequestError("Token is required", nil)
		}

		// Find the invitation
		invitationsCollection, err := app.Dao().FindCollectionByNameOrId("invitations")
		if err != nil {
			return apis.NewApiError(http.StatusInternalServerError, "Failed to find invitations collection", err)
		}

		invitation, err := app.Dao().FindFirstRecordByData(invitationsCollection.Id, "token", token)
		if err != nil {
			return apis.NewNotFoundError("Invalid invitation token", err)
		}

		// Check if invitation is expired
		expiresTime := invitation.GetDateTime("expires").Time()
		if expiresTime.Before(time.Now()) {
			return apis.NewBadRequestError("Invitation has expired", nil)
		}

		// Check if invitation is already used
		if invitation.GetBool("used") {
			return apis.NewBadRequestError("Invitation has already been used", nil)
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"valid": true,
			"email": invitation.GetString("email"),
		})
	})
}
