package session

import (
	"context"
	"time"

	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/status"

	"github.com/go-park-mail-ru/2018_2_DeadMolesStudio/logger"
)

// nolint: golint
type SessionManager struct {
	smc      SessionManagerClient
	grpcConn *grpc.ClientConn
}

func ConnectSessionManager(address string) *SessionManager {
	grpcConn, err := grpc.Dial(
		address,
		grpc.WithInsecure(),
		grpc.WithBlock(),
		grpc.WithTimeout(30*time.Second), // nolint: megacheck
	)
	if err != nil {
		logger.Panicf("failed to connect to sessionManager at address %v: %v", address, err)
	}

	smc := NewSessionManagerClient(grpcConn)

	logger.Infof("Successfully connected to sessionManager: %v", address)

	return &SessionManager{smc: smc, grpcConn: grpcConn}
}

func (sm *SessionManager) Create(uID uint) (string, error) {
	if sm.grpcConn == nil {
		return "", ErrConnRefused
	}

	sID, err := sm.smc.Create(
		context.Background(),
		&Session{UID: uint64(uID)},
	)
	if err != nil {
		return "", err
	}
	return sID.UUID, nil
}

func (sm *SessionManager) Get(sID string) (uint, error) {
	if sm.grpcConn == nil {
		return 0, ErrConnRefused
	}

	s, err := sm.smc.Get(
		context.Background(),
		&SessionID{UUID: sID},
	)
	if err != nil {
		s, _ := status.FromError(err)
		if s.Message() == ErrKeyNotFound.Error() {
			return 0, ErrKeyNotFound
		}
		return 0, err
	}
	return uint(s.UID), nil
}

func (sm *SessionManager) Delete(sID string) error {
	if sm.grpcConn == nil {
		return ErrConnRefused
	}

	_, err := sm.smc.Delete(
		context.Background(),
		&SessionID{UUID: sID},
	)
	return err
}

func (sm *SessionManager) Close() error {
	if sm.grpcConn == nil {
		return ErrConnRefused
	}

	err := sm.grpcConn.Close()
	sm.grpcConn = nil
	return err
}
