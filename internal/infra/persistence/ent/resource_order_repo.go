package ent

import (
	"context"
	"fmt"
	"time"

	"github.com/anzhiyu-c/anheyu-app/ent"
	"github.com/anzhiyu-c/anheyu-app/ent/resourceorder"
	"github.com/anzhiyu-c/anheyu-app/modules/commerce"
	"github.com/anzhiyu-c/anheyu-app/pkg/idgen"
)

type ResourceOrderRepo struct {
	db *ent.Client
}

func NewResourceOrderRepo(db *ent.Client) *ResourceOrderRepo {
	return &ResourceOrderRepo{db: db}
}

func (r *ResourceOrderRepo) Create(ctx context.Context, input commerce.ResourceOrderCreateDTO) (commerce.ResourceOrderRecordDTO, error) {
	resourceDBID, err := decodeResourceID(input.ResourceID)
	if err != nil {
		return commerce.ResourceOrderRecordDTO{}, err
	}

	create := r.db.ResourceOrder.Create().
		SetUserID(input.UserID).
		SetResourceID(resourceDBID).
		SetBusinessOrderNo(input.BusinessOrderNo).
		SetAmount(input.Amount).
		SetStatus(defaultString(input.Status, "pending")).
		SetSnapshot(input.Snapshot)

	if input.ExternalOrderNo != "" {
		create.SetExternalOrderNo(input.ExternalOrderNo)
	}
	if input.ResourceItemID != "" {
		resourceItemDBID, err := decodeResourceItemID(input.ResourceItemID)
		if err != nil {
			return commerce.ResourceOrderRecordDTO{}, err
		}
		create.SetResourceItemID(resourceItemDBID)
	}
	if input.PaidAt != nil {
		create.SetPaidAt(*input.PaidAt)
	}

	entity, err := create.Save(ctx)
	if err != nil {
		return commerce.ResourceOrderRecordDTO{}, err
	}

	return toResourceOrderRecordDTO(entity)
}

func (r *ResourceOrderRepo) MarkPaid(ctx context.Context, businessOrderNo string, externalOrderNo string, paidAt *time.Time) (bool, error) {
	update := r.db.ResourceOrder.Update().
		Where(resourceorder.BusinessOrderNoEQ(businessOrderNo), resourceorder.StatusNEQ("paid")).
		SetStatus("paid")

	if externalOrderNo != "" {
		update.SetExternalOrderNo(externalOrderNo)
	}
	if paidAt != nil {
		update.SetPaidAt(*paidAt)
	}

	affected, err := update.Save(ctx)
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func (r *ResourceOrderRepo) UpdateExternalOrderNo(ctx context.Context, businessOrderNo string, externalOrderNo string) error {
	_, err := r.db.ResourceOrder.Update().
		Where(resourceorder.BusinessOrderNoEQ(businessOrderNo)).
		SetExternalOrderNo(externalOrderNo).
		Save(ctx)
	return err
}

func (r *ResourceOrderRepo) FindByBusinessOrderNo(ctx context.Context, businessOrderNo string) (commerce.ResourceOrderRecordDTO, error) {
	entity, err := r.db.ResourceOrder.Query().
		Where(resourceorder.BusinessOrderNoEQ(businessOrderNo)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return commerce.ResourceOrderRecordDTO{}, commerce.ErrResourceOrderNotFound
		}
		return commerce.ResourceOrderRecordDTO{}, err
	}

	return toResourceOrderRecordDTO(entity)
}

func (r *ResourceOrderRepo) FindByExternalOrderNo(ctx context.Context, externalOrderNo string) (commerce.ResourceOrderRecordDTO, error) {
	entity, err := r.db.ResourceOrder.Query().
		Where(resourceorder.ExternalOrderNoEQ(externalOrderNo)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return commerce.ResourceOrderRecordDTO{}, commerce.ErrResourceOrderNotFound
		}
		return commerce.ResourceOrderRecordDTO{}, err
	}

	return toResourceOrderRecordDTO(entity)
}

func toResourceOrderRecordDTO(entity *ent.ResourceOrder) (commerce.ResourceOrderRecordDTO, error) {
	resourcePublicID, err := idgen.GeneratePublicID(entity.ResourceID, idgen.EntityTypeResource)
	if err != nil {
		return commerce.ResourceOrderRecordDTO{}, fmt.Errorf("encode resource order resource id: %w", err)
	}

	dto := commerce.ResourceOrderRecordDTO{
		UserID:          entity.UserID,
		ResourceID:      resourcePublicID,
		BusinessOrderNo: entity.BusinessOrderNo,
		ExternalOrderNo: entity.ExternalOrderNo,
		Amount:          entity.Amount,
		Status:          entity.Status,
		Snapshot:        entity.Snapshot,
		PaidAt:          entity.PaidAt,
	}

	if entity.ResourceItemID != nil {
		resourceItemPublicID, err := idgen.GeneratePublicID(*entity.ResourceItemID, idgen.EntityTypeResourceItem)
		if err != nil {
			return commerce.ResourceOrderRecordDTO{}, fmt.Errorf("encode resource order item id: %w", err)
		}
		dto.ResourceItemID = resourceItemPublicID
	}

	return dto, nil
}
