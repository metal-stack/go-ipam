package service

import (
	"context"
	"fmt"

	"github.com/gogo/status"
	"github.com/metal-stack/go-ipam"
	v1 "github.com/metal-stack/go-ipam/server/api/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
)

type IpamService struct {
	ipamer ipam.Ipamer
	log    *zap.Logger
}

func NewIpamService(storage ipam.Storage, l *zap.Logger) *IpamService {
	return &IpamService{
		ipamer: ipam.NewWithStorage(storage),
		log:    l,
	}
}

func (i *IpamService) CreatePrefix(ctx context.Context, req *v1.CreatePrefixRequest) (*v1.PrefixResponse, error) {
	i.ipamer.SetNamespace(req.Namespace)
	p, err := i.ipamer.NewPrefix(req.Cidr)
	if err != nil {
		return nil, err
	}
	return &v1.PrefixResponse{Prefix: &v1.Prefix{Cidr: p.Cidr, Namespace: p.Namespace}}, nil
}
func (i *IpamService) DeletePrefix(ctx context.Context, req *v1.DeletePrefixRequest) (*v1.PrefixResponse, error) {
	i.ipamer.SetNamespace(req.Namespace)
	p, err := i.ipamer.DeletePrefix(req.Cidr)
	if err != nil {
		return nil, err
	}
	return &v1.PrefixResponse{Prefix: &v1.Prefix{Cidr: p.Cidr, ParentCidr: p.ParentCidr, Namespace: p.Namespace}}, nil
}
func (i *IpamService) GetPrefix(ctx context.Context, req *v1.GetPrefixRequest) (*v1.PrefixResponse, error) {
	i.ipamer.SetNamespace(req.Namespace)
	p := i.ipamer.PrefixFrom(req.Cidr)
	if p == nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("no prefix with cidr:%s found", req.Cidr))
	}
	return &v1.PrefixResponse{Prefix: &v1.Prefix{Cidr: p.Cidr, ParentCidr: p.ParentCidr, Namespace: p.Namespace}}, nil
}
func (i *IpamService) AcquireChildPrefix(ctx context.Context, req *v1.AcquireChildPrefixRequest) (*v1.PrefixResponse, error) {
	i.ipamer.SetNamespace(req.Namespace)
	if req.Length > 128 {
		return nil, status.Error(codes.Internal, fmt.Sprintf("child prefix length:%d must be between 0-128", req.Length))
	}
	p, err := i.ipamer.AcquireChildPrefix(req.Cidr, uint8(req.Length))
	if err != nil {
		return nil, err
	}
	return &v1.PrefixResponse{Prefix: &v1.Prefix{Cidr: p.Cidr, ParentCidr: p.ParentCidr, Namespace: p.Namespace}}, nil
}
func (i *IpamService) ReleaseChildPrefix(ctx context.Context, req *v1.ReleaseChildPrefixRequest) (*v1.PrefixResponse, error) {
	i.ipamer.SetNamespace(req.Namespace)
	p := i.ipamer.PrefixFrom(req.Cidr)
	if p == nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("no prefix with cidr:%s found", req.Cidr))
	}
	err := i.ipamer.ReleaseChildPrefix(p)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &v1.PrefixResponse{Prefix: &v1.Prefix{Cidr: p.Cidr, ParentCidr: p.ParentCidr, Namespace: p.Namespace}}, nil
}
func (i *IpamService) AcquireIP(ctx context.Context, req *v1.AcquireIPRequest) (*v1.AcquireIPResponse, error) {
	i.ipamer.SetNamespace(req.Namespace)
	if req.Ip == nil {
		ip, err := i.ipamer.AcquireIP(req.PrefixCidr)
		if err != nil {
			return nil, err
		}
		return &v1.AcquireIPResponse{Ip: &v1.IP{Ip: ip.IP.String(), ParentPrefix: ip.ParentPrefix, Namespace: ip.Namespace}}, nil
	}
	ip, err := i.ipamer.AcquireSpecificIP(req.PrefixCidr, req.Ip.Value)
	if err != nil {
		return nil, err
	}
	return &v1.AcquireIPResponse{Ip: &v1.IP{Ip: ip.IP.String(), ParentPrefix: ip.ParentPrefix, Namespace: ip.Namespace}}, nil
}
func (i *IpamService) ReleaseIP(ctx context.Context, req *v1.ReleaseIPRequest) (*v1.ReleaseIPResponse, error) {
	i.ipamer.SetNamespace(req.Namespace)
	err := i.ipamer.ReleaseIPFromPrefix(req.PrefixCidr, req.Ip)
	if err != nil {
		return nil, err
	}
	return &v1.ReleaseIPResponse{Ip: &v1.IP{Ip: req.Ip, ParentPrefix: req.PrefixCidr, Namespace: req.Namespace}}, nil
}
