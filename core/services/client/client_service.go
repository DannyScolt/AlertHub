package client

import (
	"context"

	clientDto "alerthub/core/dto/client"
	clientRepo "alerthub/core/repository/client"

	"github.com/google/uuid"
)

type ClientService interface {
	GetMe(ctx context.Context, clientID uuid.UUID) (clientDto.ClientResponse, error)
}

type clientService struct{ repo clientRepo.ClientRepository }

func NewClientService(repo clientRepo.ClientRepository) ClientService {
	return &clientService{repo: repo}
}

func (s *clientService) GetMe(ctx context.Context, clientID uuid.UUID) (clientDto.ClientResponse, error) {
	c, err := s.repo.FindByID(ctx, clientID)
	if err != nil {
		return clientDto.ClientResponse{}, err
	}
	return clientDto.ClientResponse{ID: c.ID.String(), Email: c.Email, Name: c.Name, CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt}, nil
}
