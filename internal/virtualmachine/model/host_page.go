package model

import "controlplane/internal/virtualmachine/domain/entity"

func HostPageFromEntity(page *entity.HostPage) *entity.HostPage {
	if page == nil {
		return &entity.HostPage{Items: []*entity.Host{}}
	}
	return page
}
