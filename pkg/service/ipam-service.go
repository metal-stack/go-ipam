package service

import (
	"context"
	"fmt"

	"github.com/bufbuild/connect-go"
	goipam "github.com/metal-stack/go-ipam"
	v1 "github.com/metal-stack/go-ipam/api/v1"
	"go.uber.org/zap"
	"inet.af/netaddr"
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

func (i *IPAMService) CreatePrefix(_ context.Context, req *connect.Request[v1.CreatePrefixRequest]) (*connect.Response[v1.CreatePrefixResponse], error) {
	i.log.Debugw("createprefix", "req", req)
	// FIXME context must be passed here
	resp, err := i.ipamer.NewPrefix(req.Msg.Cidr)
	if err != nil {
		i.log.Errorw("createprefix", "error", err)
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return &connect.Response[v1.CreatePrefixResponse]{
		Msg: &v1.CreatePrefixResponse{
			Prefix: &v1.Prefix{
				Cidr:       resp.Cidr,
				ParentCidr: resp.ParentCidr,
			},
		},
	}, nil
}
func (i *IPAMService) DeletePrefix(_ context.Context, req *connect.Request[v1.DeletePrefixRequest]) (*connect.Response[v1.DeletePrefixResponse], error) {
	i.log.Debugw("deleteprefix", "req", req)
	resp, err := i.ipamer.DeletePrefix(req.Msg.Cidr)
	if err != nil {
		i.log.Errorw("deleteprefix", "error", err)
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	return &connect.Response[v1.DeletePrefixResponse]{
		Msg: &v1.DeletePrefixResponse{
			Prefix: &v1.Prefix{
				Cidr:       resp.Cidr,
				ParentCidr: resp.ParentCidr,
			},
		},
	}, nil
}
func (i *IPAMService) GetPrefix(_ context.Context, req *connect.Request[v1.GetPrefixRequest]) (*connect.Response[v1.GetPrefixResponse], error) {
	i.log.Debugw("getprefix", "req", req)

	resp := i.ipamer.PrefixFrom(req.Msg.Cidr)
	if resp == nil {
		return &connect.Response[v1.GetPrefixResponse]{}, connect.NewError(connect.CodeNotFound, fmt.Errorf("prefix:%q not found", req.Msg.Cidr))
	}

	return &connect.Response[v1.GetPrefixResponse]{
		Msg: &v1.GetPrefixResponse{
			Prefix: &v1.Prefix{
				Cidr:       resp.Cidr,
				ParentCidr: resp.ParentCidr,
			},
		},
	}, nil
}
func (i *IPAMService) ListPrefixes(_ context.Context, req *connect.Request[v1.ListPrefixesRequest]) (*connect.Response[v1.ListPrefixesResponse], error) {
	i.log.Debugw("listprefixes", "req", req)

	resp, err := i.ipamer.ReadAllPrefixCidrs()
	if err != nil {
		i.log.Errorw("listprefixes", "error", err)
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	var result []*v1.Prefix
	for _, cidr := range resp {
		p := i.ipamer.PrefixFrom(cidr)
		if p == nil {
			continue
		}
		result = append(result, &v1.Prefix{Cidr: cidr, Namespace: req.Msg.Namespace, ParentCidr: p.ParentCidr})
	}
	return &connect.Response[v1.ListPrefixesResponse]{
		Msg: &v1.ListPrefixesResponse{
			Prefixes: result,
		},
	}, nil
}

func (i *IPAMService) AcquireChildPrefix(_ context.Context, req *connect.Request[v1.AcquireChildPrefixRequest]) (*connect.Response[v1.AcquireChildPrefixResponse], error) {
	i.log.Debugw("acquirechildprefix", "req", req)
	resp, err := i.ipamer.AcquireChildPrefix(req.Msg.Cidr, uint8(req.Msg.Length))
	if err != nil {
		i.log.Errorw("acquirechildprefix", "error", err)
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return &connect.Response[v1.AcquireChildPrefixResponse]{
		Msg: &v1.AcquireChildPrefixResponse{
			Prefix: &v1.Prefix{
				Cidr:       resp.Cidr,
				ParentCidr: resp.ParentCidr,
			},
		},
	}, nil
}

func (i *IPAMService) ReleaseChildPrefix(_ context.Context, req *connect.Request[v1.ReleaseChildPrefixRequest]) (*connect.Response[v1.ReleaseChildPrefixResponse], error) {
	i.log.Debugw("releasechildprefix", "req", req)

	prefix := i.ipamer.PrefixFrom(req.Msg.Cidr)
	if prefix == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("prefix:%q not found", req.Msg.Cidr))
	}

	err := i.ipamer.ReleaseChildPrefix(prefix)
	if err != nil {
		i.log.Errorw("releasechildprefix", "error", err)
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return &connect.Response[v1.ReleaseChildPrefixResponse]{
		Msg: &v1.ReleaseChildPrefixResponse{
			Prefix: &v1.Prefix{
				Cidr:       prefix.Cidr,
				ParentCidr: prefix.ParentCidr,
			},
		},
	}, nil
}
func (i *IPAMService) AcquireIP(_ context.Context, req *connect.Request[v1.AcquireIPRequest]) (*connect.Response[v1.AcquireIPResponse], error) {
	i.log.Debugw("acquireip", "req", req)

	resp, err := i.ipamer.AcquireIP(req.Msg.PrefixCidr)
	if err != nil {
		i.log.Errorw("acquireip", "error", err)
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return &connect.Response[v1.AcquireIPResponse]{
		Msg: &v1.AcquireIPResponse{
			Ip: &v1.IP{
				Ip:           resp.IP.String(),
				ParentPrefix: resp.ParentPrefix,
			},
		},
	}, nil
}
func (i *IPAMService) ReleaseIP(_ context.Context, req *connect.Request[v1.ReleaseIPRequest]) (*connect.Response[v1.ReleaseIPResponse], error) {
	i.log.Debugw("releaseip", "req", req)
	netip, err := netaddr.ParseIP(req.Msg.Ip)
	if err != nil {
		i.log.Errorw("releaseip", "error", err)
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	ip := &goipam.IP{
		IP:           netip,
		ParentPrefix: req.Msg.PrefixCidr,
	}
	resp, err := i.ipamer.ReleaseIP(ip)
	if err != nil {
		i.log.Errorw("releaseip", "error", err)
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return &connect.Response[v1.ReleaseIPResponse]{
		Msg: &v1.ReleaseIPResponse{
			Ip: &v1.IP{
				Ip:           req.Msg.Ip,
				ParentPrefix: resp.ParentCidr,
			},
		},
	}, nil
}
