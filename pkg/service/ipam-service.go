package service

import (
	"context"
	"fmt"

	"net/netip"

	"github.com/bufbuild/connect-go"
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

func (i *IPAMService) CreatePrefix(ctx context.Context, req *connect.Request[v1.CreatePrefixRequest]) (*connect.Response[v1.CreatePrefixResponse], error) {
	i.log.Debugw("createprefix", "req", req)
	resp, err := i.ipamer.NewPrefix(ctx, req.Msg.Cidr)
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
func (i *IPAMService) DeletePrefix(ctx context.Context, req *connect.Request[v1.DeletePrefixRequest]) (*connect.Response[v1.DeletePrefixResponse], error) {
	i.log.Debugw("deleteprefix", "req", req)
	resp, err := i.ipamer.DeletePrefix(ctx, req.Msg.Cidr)
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
func (i *IPAMService) GetPrefix(ctx context.Context, req *connect.Request[v1.GetPrefixRequest]) (*connect.Response[v1.GetPrefixResponse], error) {
	i.log.Debugw("getprefix", "req", req)

	resp := i.ipamer.PrefixFrom(ctx, req.Msg.Cidr)
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
func (i *IPAMService) ListPrefixes(ctx context.Context, req *connect.Request[v1.ListPrefixesRequest]) (*connect.Response[v1.ListPrefixesResponse], error) {
	i.log.Debugw("listprefixes", "req", req)

	resp, err := i.ipamer.ReadAllPrefixCidrs(ctx)
	if err != nil {
		i.log.Errorw("listprefixes", "error", err)
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	var result []*v1.Prefix
	for _, cidr := range resp {
		p := i.ipamer.PrefixFrom(ctx, cidr)
		if p == nil {
			i.log.Warnw("skipping nil prefix of cidr:%q", cidr)
			continue
		}
		result = append(result, &v1.Prefix{Cidr: cidr, ParentCidr: p.ParentCidr})
	}
	return &connect.Response[v1.ListPrefixesResponse]{
		Msg: &v1.ListPrefixesResponse{
			Prefixes: result,
		},
	}, nil
}

func (i *IPAMService) AcquireChildPrefix(ctx context.Context, req *connect.Request[v1.AcquireChildPrefixRequest]) (*connect.Response[v1.AcquireChildPrefixResponse], error) {
	i.log.Debugw("acquirechildprefix", "req", req)
	resp, err := i.ipamer.AcquireChildPrefix(ctx, req.Msg.Cidr, uint8(req.Msg.Length))
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

func (i *IPAMService) ReleaseChildPrefix(ctx context.Context, req *connect.Request[v1.ReleaseChildPrefixRequest]) (*connect.Response[v1.ReleaseChildPrefixResponse], error) {
	i.log.Debugw("releasechildprefix", "req", req)

	prefix := i.ipamer.PrefixFrom(ctx, req.Msg.Cidr)
	if prefix == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("prefix:%q not found", req.Msg.Cidr))
	}

	err := i.ipamer.ReleaseChildPrefix(ctx, prefix)
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
func (i *IPAMService) AcquireIP(ctx context.Context, req *connect.Request[v1.AcquireIPRequest]) (*connect.Response[v1.AcquireIPResponse], error) {
	i.log.Debugw("acquireip", "req", req)

	resp, err := i.ipamer.AcquireIP(ctx, req.Msg.PrefixCidr)
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
func (i *IPAMService) ReleaseIP(ctx context.Context, req *connect.Request[v1.ReleaseIPRequest]) (*connect.Response[v1.ReleaseIPResponse], error) {
	i.log.Debugw("releaseip", "req", req)
	netip, err := netip.ParseAddr(req.Msg.Ip)
	if err != nil {
		i.log.Errorw("releaseip", "error", err)
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	ip := &goipam.IP{
		IP:           netip,
		ParentPrefix: req.Msg.PrefixCidr,
	}
	resp, err := i.ipamer.ReleaseIP(ctx, ip)
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
func (i *IPAMService) Dump(ctx context.Context, req *connect.Request[v1.DumpRequest]) (*connect.Response[v1.DumpResponse], error) {
	i.log.Debugw("dump", "req", req)
	dump, err := i.ipamer.Dump(ctx)
	if err != nil {
		i.log.Errorw("dump", "error", err)
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return &connect.Response[v1.DumpResponse]{
		Msg: &v1.DumpResponse{
			Dump: dump,
		},
	}, nil
}

func (i *IPAMService) Load(ctx context.Context, req *connect.Request[v1.LoadRequest]) (*connect.Response[v1.LoadResponse], error) {
	i.log.Debugw("load", "req", req)
	err := i.ipamer.Load(ctx, req.Msg.Dump)
	if err != nil {
		i.log.Errorw("load", "error", err)
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return &connect.Response[v1.LoadResponse]{}, nil
}
func (i *IPAMService) PrefixUsage(ctx context.Context, req *connect.Request[v1.PrefixUsageRequest]) (*connect.Response[v1.PrefixUsageResponse], error) {
	i.log.Debugw("prefixusage", "req", req)
	p := i.ipamer.PrefixFrom(ctx, req.Msg.Cidr)
	if p == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("prefix:%q not found", req.Msg.Cidr))
	}
	u := p.Usage()
	return &connect.Response[v1.PrefixUsageResponse]{
		Msg: &v1.PrefixUsageResponse{
			AvailableIps:              u.AvailableIPs,
			AcquiredIps:               u.AcquiredIPs,
			AvailableSmallestPrefixes: u.AvailableSmallestPrefixes,
			AvailablePrefixes:         u.AvailablePrefixes,
			AcquiredPrefixes:          u.AcquiredPrefixes,
		},
	}, nil
}
