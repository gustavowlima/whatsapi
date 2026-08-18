package whatsmiau

import (
	"context"
	"fmt"
	"sort"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

type ListContactsRequest struct{ InstanceID string }

type ContactResponse struct {
	JID           string `json:"jid"`
	Found         bool   `json:"found"`
	FirstName     string `json:"firstName,omitempty"`
	FullName      string `json:"fullName,omitempty"`
	PushName      string `json:"pushName,omitempty"`
	BusinessName  string `json:"businessName,omitempty"`
	RedactedPhone string `json:"redactedPhone,omitempty"`
}

type FindContactRequest struct {
	InstanceID string     `json:"instance_id"`
	Number     *types.JID `json:"number"`
}

type FindContactResponse struct {
	ID                string `json:"id"`
	Wuid              string `json:"wuid"`
	Name              string `json:"name,omitempty"`
	FirstName         string `json:"firstName,omitempty"`
	FullName          string `json:"fullName,omitempty"`
	PushName          string `json:"pushName,omitempty"`
	BusinessName      string `json:"businessName,omitempty"`
	RedactedPhone     string `json:"redactedPhone,omitempty"`
	IsWhatsApp        bool   `json:"isWhatsApp"`
	Found             bool   `json:"found"`
	ProfilePictureUrl string `json:"profilePictureUrl"`
}

// ListContacts returns the current Whatsmeow contact cache. It intentionally
// excludes profile pictures and any media so the manager can bootstrap an MCP
// dataset without moving binary or expiring URL data between services.
func (s *Whatsmiau) ListContacts(ctx context.Context, req *ListContactsRequest) ([]ContactResponse, error) {
	client, ok := s.clients.Load(req.InstanceID)
	if !ok || client.Store == nil || client.Store.Contacts == nil {
		return nil, whatsmeow.ErrClientIsNil
	}
	contacts, err := client.Store.Contacts.GetAllContacts(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]ContactResponse, 0, len(contacts))
	for jid, info := range contacts {
		result = append(result, ContactResponse{JID: jid.ToNonAD().String(), Found: info.Found, FirstName: info.FirstName, FullName: info.FullName, PushName: info.PushName, BusinessName: info.BusinessName, RedactedPhone: info.RedactedPhone})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].JID < result[j].JID })
	return result, nil
}

// FindContact fetches contact details from local store and combines with profile picture URL.
func (s *Whatsmiau) FindContact(ctx context.Context, req *FindContactRequest) (*FindContactResponse, error) {
	client, ok := s.clients.Load(req.InstanceID)
	if !ok || client == nil {
		return nil, whatsmeow.ErrClientIsNil
	}

	if !client.IsConnected() {
		return nil, fmt.Errorf("client not connected for instance %s", req.InstanceID)
	}

	target := s.resolveJID(ctx, client, *req.Number)

	resp := &FindContactResponse{
		ID:         target.String(),
		Wuid:       target.String(),
		IsWhatsApp: true,
	}

	if client.Store != nil && client.Store.Contacts != nil {
		info, err := client.Store.Contacts.GetContact(ctx, target)
		if err == nil && info.Found {
			resp.Found = true
			resp.FirstName = info.FirstName
			resp.FullName = info.FullName
			resp.PushName = info.PushName
			resp.BusinessName = info.BusinessName
			resp.RedactedPhone = info.RedactedPhone

			if info.FullName != "" {
				resp.Name = info.FullName
			} else if info.FirstName != "" {
				resp.Name = info.FirstName
			} else if info.BusinessName != "" {
				resp.Name = info.BusinessName
			} else if info.PushName != "" {
				resp.Name = info.PushName
			}
		}
	}

	pic, err := client.GetProfilePictureInfo(ctx, target, &whatsmeow.GetProfilePictureParams{
		Preview:     false,
		IsCommunity: false,
	})
	if err == nil && pic != nil {
		resp.ProfilePictureUrl = pic.URL
	}

	return resp, nil
}

