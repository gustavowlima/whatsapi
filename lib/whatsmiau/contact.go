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

	var contactInfo *types.ContactInfo

	if client.Store != nil && client.Store.Contacts != nil {
		// 1. Try target JID
		if info, err := client.Store.Contacts.GetContact(ctx, target); err == nil && info.Found {
			contactInfo = &info
		}
		// 2. Try raw requested number JID if different
		if contactInfo == nil && req.Number != nil && *req.Number != target {
			if info, err := client.Store.Contacts.GetContact(ctx, *req.Number); err == nil && info.Found {
				contactInfo = &info
			}
		}
		// 3. Try Brazilian alternate JID
		if contactInfo == nil {
			if alt := buildBrazilianAlternate(target.User); alt != "" {
				altJID := types.NewJID(alt, types.DefaultUserServer)
				if info, err := client.Store.Contacts.GetContact(ctx, altJID); err == nil && info.Found {
					contactInfo = &info
				}
			}
		}
		// 4. Try searching all contacts in store for matching user or phone digits
		if contactInfo == nil {
			if all, err := client.Store.Contacts.GetAllContacts(ctx); err == nil {
				targetDigits := onlyDigits(target.User)
				for j, info := range all {
					jDigits := onlyDigits(j.User)
					if j == target || (targetDigits != "" && (jDigits == targetDigits || j.User == target.User)) {
						if info.FullName != "" || info.FirstName != "" || info.PushName != "" || info.BusinessName != "" {
							contactInfo = &info
							break
						}
					}
				}
			}
		}
	}

	if contactInfo != nil {
		resp.Found = true
		resp.FirstName = contactInfo.FirstName
		resp.FullName = contactInfo.FullName
		resp.PushName = contactInfo.PushName
		resp.BusinessName = contactInfo.BusinessName
		resp.RedactedPhone = contactInfo.RedactedPhone

		if contactInfo.FullName != "" {
			resp.Name = contactInfo.FullName
		} else if contactInfo.FirstName != "" {
			resp.Name = contactInfo.FirstName
		} else if contactInfo.BusinessName != "" {
			resp.Name = contactInfo.BusinessName
		} else if contactInfo.PushName != "" {
			resp.Name = contactInfo.PushName
		}
	}

	// 5. If name is still empty, try client.GetUserInfo to fetch verified name
	if resp.Name == "" {
		if users, err := client.GetUserInfo(ctx, []types.JID{target}); err == nil {
			if uInfo, ok := users[target]; ok && uInfo.VerifiedName != nil && uInfo.VerifiedName.Details != nil {
				vName := uInfo.VerifiedName.Details.GetVerifiedName()
				if vName != "" {
					resp.Name = vName
					resp.BusinessName = vName
				}
			}
		}
	}

	picURL, err := s.FetchProfilePictureURL(ctx, req.InstanceID, target)
	if err == nil && picURL != "" {
		resp.ProfilePictureUrl = picURL
	}

	return resp, nil
}
