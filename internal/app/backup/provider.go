package backup

import (
	"context"

	"xui-sells-v2/internal/app/instance"
	"xui-sells-v2/internal/domain"
)

// InstanceStoreDataProvider adapts instance.Store to DataProvider for backup operations.
type InstanceStoreDataProvider struct {
	store instance.Store
}

// NewInstanceStoreDataProvider creates a provider using an instance store.
func NewInstanceStoreDataProvider(store instance.Store) *InstanceStoreDataProvider {
	return &InstanceStoreDataProvider{store: store}
}

func (p *InstanceStoreDataProvider) GetGlobalData(ctx context.Context) (*BackupData, error) {
	list, err := p.store.List(ctx)
	if err != nil {
		return nil, err
	}
	return &BackupData{
		Instances: list,
	}, nil
}

func (p *InstanceStoreDataProvider) GetInstanceData(ctx context.Context, instanceID int64) (*BackupData, error) {
	inst, err := p.store.Get(ctx, instanceID)
	if err != nil {
		return nil, err
	}

	children, err := p.store.ListChildBots(ctx, instanceID)
	if err != nil {
		return nil, err
	}

	all := append([]domain.Instance{*inst}, children...)
	return &BackupData{
		Instances: all,
	}, nil
}

func (p *InstanceStoreDataProvider) RestoreGlobalData(ctx context.Context, data *BackupData) error {
	for _, inst := range data.Instances {
		_ = p.store.Create(ctx, &inst)
	}
	return nil
}

func (p *InstanceStoreDataProvider) RestoreInstanceData(ctx context.Context, instanceID int64, data *BackupData) error {
	for _, inst := range data.Instances {
		_ = p.store.Create(ctx, &inst)
	}
	return nil
}
