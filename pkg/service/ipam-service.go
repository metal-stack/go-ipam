package service

import (
	"context"

	v1 "github.com/metal-stack/go-ipam/api/v1"
)

type IPAMService struct {
}

func New() *IPAMService {
	return &IPAMService{}
}

func (i *IPAMService) CreatePrefix(ctx context.Context, in *v1.CreatePrefixRequest) (*v1.CreatePrefixResponse, error) {
	return nil, nil
}
func (i *IPAMService) DeletePrefix(ctx context.Context, in *v1.DeletePrefixRequest) (*v1.DeletePrefixResponse, error) {
	return nil, nil
}
func (i *IPAMService) GetPrefix(ctx context.Context, in *v1.GetPrefixRequest) (*v1.GetPrefixResponse, error) {
	return nil, nil
}
func (i *IPAMService) AcquireChildPrefix(ctx context.Context, in *v1.AcquireChildPrefixRequest) (*v1.AcquireChildPrefixResponse, error) {
	return nil, nil
}
func (i *IPAMService) ReleaseChildPrefix(ctx context.Context, in *v1.ReleaseChildPrefixRequest) (*v1.ReleaseChildPrefixResponse, error) {
	return nil, nil
}
func (i *IPAMService) AcquireIP(ctx context.Context, in *v1.AcquireIPRequest) (*v1.AcquireIPResponse, error) {
	return nil, nil
}
func (i *IPAMService) ReleaseIP(ctx context.Context, in *v1.ReleaseIPRequest) (*v1.ReleaseIPResponse, error) {
	return nil, nil
}
