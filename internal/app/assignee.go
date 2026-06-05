package app

import (
	"context"
	"fmt"

	"github.com/strider2038/multica-mcp/internal/domain"
)

func (u *UseCase) resolveAssigneeType(ctx context.Context, assigneeID string, explicit *string) (*string, error) {
	if explicit != nil && *explicit != "" {
		if !domain.AssigneeType(*explicit).IsValid() {
			return nil, fmt.Errorf("invalid assignee_type %q; valid values: member, agent, squad", *explicit)
		}
		return explicit, nil
	}
	if assigneeID == "" {
		return nil, nil
	}

	agents, err := u.client.ListAgents(ctx)
	if err != nil {
		member := string(domain.AssigneeTypeMember)
		return &member, nil
	}
	for _, a := range agents {
		if a.ID == assigneeID {
			agent := string(domain.AssigneeTypeAgent)
			return &agent, nil
		}
	}

	member := string(domain.AssigneeTypeMember)
	return &member, nil
}
