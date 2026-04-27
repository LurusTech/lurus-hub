package repo

import (
	entity "github.com/LurusTech/lurus-hub/internal/domain/entity"
)

type Setup = entity.Setup

func GetSetup() *Setup {
	var setup Setup
	err := DB.First(&setup).Error
	if err != nil {
		return nil
	}
	return &setup
}
