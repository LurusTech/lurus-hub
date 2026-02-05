package repo

import (
	entity "github.com/QuantumNous/lurus-api/internal/domain/entity"
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
