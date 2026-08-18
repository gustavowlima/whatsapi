package whatsmiau

import (
	"context"
	"errors"
	"testing"

	"go.mau.fi/whatsmeow/types"
)

type fakeGroupAddModeClient struct {
	info          *types.GroupInfo
	infoErr       error
	subGroups     []*types.GroupLinkTarget
	subGroupsErr  error
	setErr        error
	setJID        types.JID
	setMode       types.GroupMemberAddMode
	setCallCount  int
	infoCallCount int
	subCallCount  int
}

func (f *fakeGroupAddModeClient) GetGroupInfo(context.Context, types.JID) (*types.GroupInfo, error) {
	f.infoCallCount++
	return f.info, f.infoErr
}

func (f *fakeGroupAddModeClient) GetSubGroups(context.Context, types.JID) ([]*types.GroupLinkTarget, error) {
	f.subCallCount++
	return f.subGroups, f.subGroupsErr
}

func (f *fakeGroupAddModeClient) SetGroupMemberAddMode(_ context.Context, jid types.JID, mode types.GroupMemberAddMode) error {
	f.setCallCount++
	f.setJID = jid
	f.setMode = mode
	return f.setErr
}

func TestSetGroupAddModeUsesRequestedRegularGroup(t *testing.T) {
	requested := types.NewJID("regular", types.GroupServer)
	client := &fakeGroupAddModeClient{info: &types.GroupInfo{JID: requested}}

	err := setGroupAddMode(context.Background(), client, requested, "all_member_add")
	if err != nil {
		t.Fatalf("setGroupAddMode returned an unexpected error: %v", err)
	}
	if client.setJID != requested {
		t.Fatalf("set target = %s, want %s", client.setJID, requested)
	}
	if client.setMode != types.GroupMemberAddModeAllMember {
		t.Fatalf("set mode = %q, want %q", client.setMode, types.GroupMemberAddModeAllMember)
	}
	if client.subCallCount != 0 {
		t.Fatalf("GetSubGroups calls = %d, want 0", client.subCallCount)
	}
}

func TestSetGroupAddModeResolvesCommunityParent(t *testing.T) {
	parent := types.NewJID("parent", types.GroupServer)
	announcement := types.NewJID("announcement", types.GroupServer)
	client := &fakeGroupAddModeClient{
		info: &types.GroupInfo{JID: parent, GroupParent: types.GroupParent{IsParent: true}},
		subGroups: []*types.GroupLinkTarget{
			{JID: types.NewJID("child", types.GroupServer)},
			{JID: announcement, GroupIsDefaultSub: types.GroupIsDefaultSub{IsDefaultSubGroup: true}},
		},
	}

	err := setGroupAddMode(context.Background(), client, parent, "admin_add")
	if err != nil {
		t.Fatalf("setGroupAddMode returned an unexpected error: %v", err)
	}
	if client.setJID != announcement {
		t.Fatalf("set target = %s, want announcement subgroup %s", client.setJID, announcement)
	}
	if client.setMode != types.GroupMemberAddModeAdmin {
		t.Fatalf("set mode = %q, want %q", client.setMode, types.GroupMemberAddModeAdmin)
	}
	if client.subCallCount != 1 {
		t.Fatalf("GetSubGroups calls = %d, want 1", client.subCallCount)
	}
}

func TestSetGroupAddModeRejectsParentWithoutDefaultSubGroup(t *testing.T) {
	parent := types.NewJID("parent", types.GroupServer)
	client := &fakeGroupAddModeClient{
		info:      &types.GroupInfo{JID: parent, GroupParent: types.GroupParent{IsParent: true}},
		subGroups: []*types.GroupLinkTarget{{JID: types.NewJID("child", types.GroupServer)}},
	}

	err := setGroupAddMode(context.Background(), client, parent, "admin_add")
	if !errors.Is(err, ErrCommunityDefaultSubGroupNotFound) {
		t.Fatalf("error = %v, want ErrCommunityDefaultSubGroupNotFound", err)
	}
	if client.setCallCount != 0 {
		t.Fatalf("SetGroupMemberAddMode calls = %d, want 0", client.setCallCount)
	}
}

func TestSetGroupAddModeRejectsInvalidModeBeforeNetworkCalls(t *testing.T) {
	client := &fakeGroupAddModeClient{}
	err := setGroupAddMode(context.Background(), client, types.NewJID("group", types.GroupServer), "everyone")
	if err == nil {
		t.Fatal("setGroupAddMode returned nil for an invalid mode")
	}
	if client.infoCallCount != 0 || client.setCallCount != 0 {
		t.Fatalf("network calls were made for invalid mode: info=%d set=%d", client.infoCallCount, client.setCallCount)
	}
}
