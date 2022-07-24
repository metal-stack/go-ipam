package service

import (
	"context"

	connect_go "github.com/bufbuild/connect-go"
	goipam "github.com/metal-stack/go-ipam"
	v1 "github.com/metal-stack/go-ipam/api/v1"
	"go.uber.org/zap"
)

type IPAMService struct {
	log    *zap.SugaredLogger
	ipamer goipam.Ipamer
}

func New(log *zap.SugaredLogger, ipamer goipam.Ipamer) *IPAMService {
	return &IPAMService{
		log:    log,
		ipamer: ipamer,
	}
}

func (i *IPAMService) CreatePrefix(_ context.Context, req *connect_go.Request[v1.CreatePrefixRequest]) (*connect_go.Response[v1.CreatePrefixResponse], error) {
	i.log.Infow("createprefix", "req", req)
	resp, err := i.ipamer.NewPrefix(req.Msg.Cidr)
	if err != nil {
		return nil, err
	}
	return &connect_go.Response[v1.CreatePrefixResponse]{
		Msg: &v1.CreatePrefixResponse{
			Prefix: &v1.Prefix{
				Cidr:       resp.Cidr,
				ParentCidr: resp.ParentCidr,
			},
		},
	}, nil
}
func (i *IPAMService) DeletePrefix(context.Context, *connect_go.Request[v1.DeletePrefixRequest]) (*connect_go.Response[v1.DeletePrefixResponse], error) {
	return nil, nil
}
func (i *IPAMService) GetPrefix(context.Context, *connect_go.Request[v1.GetPrefixRequest]) (*connect_go.Response[v1.GetPrefixResponse], error) {
	return nil, nil
}
func (i *IPAMService) AcquireChildPrefix(context.Context, *connect_go.Request[v1.AcquireChildPrefixRequest]) (*connect_go.Response[v1.AcquireChildPrefixResponse], error) {
	return nil, nil
}
func (i *IPAMService) ReleaseChildPrefix(context.Context, *connect_go.Request[v1.ReleaseChildPrefixRequest]) (*connect_go.Response[v1.ReleaseChildPrefixResponse], error) {
	return nil, nil
}
func (i *IPAMService) AcquireIP(context.Context, *connect_go.Request[v1.AcquireIPRequest]) (*connect_go.Response[v1.AcquireIPResponse], error) {
	return nil, nil
}
func (i *IPAMService) ReleaseIP(context.Context, *connect_go.Request[v1.ReleaseIPRequest]) (*connect_go.Response[v1.ReleaseIPResponse], error) {
	return nil, nil
}
