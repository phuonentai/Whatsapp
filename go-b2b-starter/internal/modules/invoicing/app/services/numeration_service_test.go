package services

import (
	"context"
	"errors"
	"testing"

	"github.com/moasq/go-b2b-starter/internal/modules/invoicing/domain"
)

type fakeNumerationReader struct {
	info domain.NumerationInfo
	err  error
}

func (f *fakeNumerationReader) GetNumeration(ctx context.Context, orgID int32) (domain.NumerationInfo, error) {
	return f.info, f.err
}

type fakeNumerationRepo struct {
	snapshots map[int32]*domain.NumerationSnapshot
}

func newFakeNumerationRepo() *fakeNumerationRepo {
	return &fakeNumerationRepo{snapshots: map[int32]*domain.NumerationSnapshot{}}
}

func (f *fakeNumerationRepo) Get(ctx context.Context, orgID int32) (*domain.NumerationSnapshot, error) {
	if s, ok := f.snapshots[orgID]; ok {
		return s, nil
	}
	return nil, domain.ErrConnectionNotFound
}

func (f *fakeNumerationRepo) UpsertConfirmed(ctx context.Context, snapshot *domain.NumerationSnapshot) (*domain.NumerationSnapshot, error) {
	f.snapshots[snapshot.OrganizationID] = snapshot
	return snapshot, nil
}

type fakeNumerationConnSvc struct {
	transitions []domain.ConnectionStatus
	err         error
}

func (f *fakeNumerationConnSvc) Status(ctx context.Context, orgID int32) (*domain.OrgConnection, error) {
	return nil, nil
}
func (f *fakeNumerationConnSvc) Connect(ctx context.Context, orgID int32, req ConnectRequest) (*domain.OrgConnection, error) {
	return nil, nil
}
func (f *fakeNumerationConnSvc) RequestAssisted(ctx context.Context, orgID int32) (*domain.OrgConnection, error) {
	return nil, nil
}
func (f *fakeNumerationConnSvc) Provision(ctx context.Context, orgID int32, req ConnectRequest) (*domain.OrgConnection, error) {
	return nil, nil
}
func (f *fakeNumerationConnSvc) Pause(ctx context.Context, orgID int32) (*domain.OrgConnection, error) {
	return nil, nil
}
func (f *fakeNumerationConnSvc) Resume(ctx context.Context, orgID int32) (*domain.OrgConnection, error) {
	return nil, nil
}
func (f *fakeNumerationConnSvc) Activate(ctx context.Context, orgID int32) (*domain.OrgConnection, error) {
	return nil, nil
}
func (f *fakeNumerationConnSvc) Disable(ctx context.Context, orgID int32) (*domain.OrgConnection, error) {
	return nil, nil
}
func (f *fakeNumerationConnSvc) ConfirmNumeration(ctx context.Context, orgID int32) (*domain.OrgConnection, error) {
	f.transitions = append(f.transitions, domain.ConnStatusNumeracionOK)
	return nil, f.err
}
func (f *fakeNumerationConnSvc) ConfirmSandboxOK(ctx context.Context, orgID int32) (*domain.OrgConnection, error) {
	return nil, nil
}
func (f *fakeNumerationConnSvc) IsLive(ctx context.Context, orgID int32) (bool, error) {
	return false, nil
}

func TestNumeration_ConfirmStoresSnapshotAndAdvances(t *testing.T) {
	reader := &fakeNumerationReader{info: domain.NumerationInfo{Mode: domain.NumerationAuto}}
	repo := newFakeNumerationRepo()
	connSvc := &fakeNumerationConnSvc{}

	svc := NewNumerationService(reader, repo, connSvc, nopLogger{})
	snapshot, err := svc.Confirm(context.Background(), 7)
	if err != nil {
		t.Fatalf("confirm failed: %v", err)
	}
	if snapshot.Mode != domain.NumerationAuto {
		t.Fatalf("unexpected snapshot mode: %+v", snapshot)
	}
	if len(connSvc.transitions) != 1 || connSvc.transitions[0] != domain.ConnStatusNumeracionOK {
		t.Fatalf("expected connected→numeracion_ok transition, got %v", connSvc.transitions)
	}
}

func TestNumeration_FailedReadLeavesStateUnchanged(t *testing.T) {
	reader := &fakeNumerationReader{err: errors.New("provider down")}
	repo := newFakeNumerationRepo()
	connSvc := &fakeNumerationConnSvc{}

	svc := NewNumerationService(reader, repo, connSvc, nopLogger{})
	if _, err := svc.Confirm(context.Background(), 7); err == nil {
		t.Fatal("expected error on failed read")
	}
	if len(repo.snapshots) != 0 {
		t.Fatal("no snapshot may be stored on failed read")
	}
	if len(connSvc.transitions) != 0 {
		t.Fatal("no transition may occur on failed read")
	}
}

func TestNumeration_GetLivePassthrough(t *testing.T) {
	reader := &fakeNumerationReader{info: domain.NumerationInfo{Mode: domain.NumerationAuto}}
	svc := NewNumerationService(reader, newFakeNumerationRepo(), &fakeNumerationConnSvc{}, nopLogger{})
	info, err := svc.GetLive(context.Background(), 7)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode != domain.NumerationAuto {
		t.Fatalf("unexpected info: %+v", info)
	}
}
