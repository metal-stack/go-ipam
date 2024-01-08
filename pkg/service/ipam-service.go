package service

import (
	"context"
	"fmt"
	"log/slog"

	"net/netip"

	"connectrpc.com/connect"
	goipam "github.com/metal-stack/go-ipam"
	v1 "github.com/metal-stack/go-ipam/api/v1"
)

type IPAMService struct {
	log    *slog.Logger
	ipamer goipam.Ipamer
}

func New(log *slog.Logger, ipamer goipam.Ipamer) *IPAMService {
	return &IPAMService{
		log:    log,
		ipamer: ipamer,
	}
}

func (i *IPAMService) CreatePrefix(ctx context.Context, req *connect.Request[v1.CreatePrefixRequest]) (*connect.Response[v1.CreatePrefixResponse], error) {
	i.log.Debug("createprefix", "req", req)
	if req.Msg.GetNamespace() != "" {
		ctx = goipam.NewContextWithNamespace(ctx, req.Msg.GetNamespace())
	}
	resp, err := i.ipamer.NewPrefix(ctx, req.Msg.GetCidr())
	if err != nil {
		i.log.Error("createprefix", "error", err)
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(
		&v1.CreatePrefixResponse{
			Prefix: &v1.Prefix{
				Cidr:       resp.Cidr,
				ParentCidr: resp.ParentCidr,
			},
		},
	), nil
}
func (i *IPAMService) DeletePrefix(ctx context.Context, req *connect.Request[v1.DeletePrefixRequest]) (*connect.Response[v1.DeletePrefixResponse], error) {
	i.log.Debug("deleteprefix", "req", req)
	if req.Msg.GetNamespace() != "" {
		ctx = goipam.NewContextWithNamespace(ctx, req.Msg.GetNamespace())
	}
	resp, err := i.ipamer.DeletePrefix(ctx, req.Msg.GetCidr())
	if err != nil {
		i.log.Error("deleteprefix", "error", err)
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	return connect.NewResponse(
		&v1.DeletePrefixResponse{
			Prefix: &v1.Prefix{
				Cidr:       resp.Cidr,
				ParentCidr: resp.ParentCidr,
			},
		},
	), nil
}
func (i *IPAMService) GetPrefix(ctx context.Context, req *connect.Request[v1.GetPrefixRequest]) (*connect.Response[v1.GetPrefixResponse], error) {
	i.log.Debug("getprefix", "req", req)
	if req.Msg.GetNamespace() != "" {
		ctx = goipam.NewContextWithNamespace(ctx, req.Msg.GetNamespace())
	}
	resp, err := i.ipamer.PrefixFrom(ctx, req.Msg.GetCidr())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if resp == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("prefix:%q not found", req.Msg.GetCidr()))
	}

	return connect.NewResponse(
		&v1.GetPrefixResponse{
			Prefix: &v1.Prefix{
				Cidr:       resp.Cidr,
				ParentCidr: resp.ParentCidr,
			},
		},
	), nil
}
func (i *IPAMService) ListPrefixes(ctx context.Context, req *connect.Request[v1.ListPrefixesRequest]) (*connect.Response[v1.ListPrefixesResponse], error) {
	i.log.Debug("listprefixes", "req", req)
	if req.Msg.GetNamespace() != "" {
		ctx = goipam.NewContextWithNamespace(ctx, req.Msg.GetNamespace())
	}
	resp, err := i.ipamer.ReadAllPrefixCidrs(ctx)
	if err != nil {
		i.log.Error("listprefixes", "error", err)
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	var result []*v1.Prefix
	for _, cidr := range resp {
		p, err := i.ipamer.PrefixFrom(ctx, cidr)
		if err != nil || p == nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		result = append(result, &v1.Prefix{Cidr: cidr, ParentCidr: p.ParentCidr})
	}
	return connect.NewResponse(
		&v1.ListPrefixesResponse{
			Prefixes: result,
		},
	), nil
}

func (i *IPAMService) AcquireChildPrefix(ctx context.Context, req *connect.Request[v1.AcquireChildPrefixRequest]) (*connect.Response[v1.AcquireChildPrefixResponse], error) {
	i.log.Debug("acquirechildprefix", "req", req)
	if req.Msg.GetNamespace() != "" {
		ctx = goipam.NewContextWithNamespace(ctx, req.Msg.GetNamespace())
	}
	var resp *goipam.Prefix
	var err error
	if req.Msg.GetChildCidr() != "" {
		resp, err = i.ipamer.AcquireSpecificChildPrefix(ctx, req.Msg.GetCidr(), req.Msg.GetChildCidr())
		if err != nil {
			i.log.Error("acquirechildprefix", "error", err)
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
	} else {
		resp, err = i.ipamer.AcquireChildPrefix(ctx, req.Msg.GetCidr(), uint8(req.Msg.GetLength()))
		if err != nil {
			i.log.Error("acquirechildprefix", "error", err)
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
	}
	return connect.NewResponse(
		&v1.AcquireChildPrefixResponse{
			Prefix: &v1.Prefix{
				Cidr:       resp.Cidr,
				ParentCidr: resp.ParentCidr,
			},
		},
	), nil
}

func (i *IPAMService) ReleaseChildPrefix(ctx context.Context, req *connect.Request[v1.ReleaseChildPrefixRequest]) (*connect.Response[v1.ReleaseChildPrefixResponse], error) {
	i.log.Debug("releasechildprefix", "req", req)
	if req.Msg.GetNamespace() != "" {
		ctx = goipam.NewContextWithNamespace(ctx, req.Msg.GetNamespace())
	}
	prefix, err := i.ipamer.PrefixFrom(ctx, req.Msg.GetCidr())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("prefix:%q not parsable:%s", req.Msg.GetCidr(), err.Error()))
	}
	if prefix == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("prefix:%q not found", req.Msg.GetCidr()))
	}

	err = i.ipamer.ReleaseChildPrefix(ctx, prefix)
	if err != nil {
		i.log.Error("releasechildprefix", "error", err)
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(
		&v1.ReleaseChildPrefixResponse{
			Prefix: &v1.Prefix{
				Cidr:       prefix.Cidr,
				ParentCidr: prefix.ParentCidr,
			},
		},
	), nil
}
func (i *IPAMService) AcquireIP(ctx context.Context, req *connect.Request[v1.AcquireIPRequest]) (*connect.Response[v1.AcquireIPResponse], error) {
	i.log.Debug("acquireip", "req", req)
	if req.Msg.GetNamespace() != "" {
		ctx = goipam.NewContextWithNamespace(ctx, req.Msg.GetNamespace())
	}
	var resp *goipam.IP
	var err error
	if req.Msg.GetIp() != "" {
		resp, err = i.ipamer.AcquireSpecificIP(ctx, req.Msg.GetPrefixCidr(), req.Msg.GetIp())
		if err != nil {
			i.log.Error("acquireip", "error", err)
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
	} else {
		resp, err = i.ipamer.AcquireIP(ctx, req.Msg.GetPrefixCidr())
		if err != nil {
			i.log.Error("acquireip", "error", err)
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
	}
	return connect.NewResponse(
		&v1.AcquireIPResponse{
			Ip: &v1.IP{
				Ip:           resp.IP.String(),
				ParentPrefix: resp.ParentPrefix,
			},
		},
	), nil
}
func (i *IPAMService) ReleaseIP(ctx context.Context, req *connect.Request[v1.ReleaseIPRequest]) (*connect.Response[v1.ReleaseIPResponse], error) {
	i.log.Debug("releaseip", "req", req)
	if req.Msg.GetNamespace() != "" {
		ctx = goipam.NewContextWithNamespace(ctx, req.Msg.GetNamespace())
	}
	netip, err := netip.ParseAddr(req.Msg.GetIp())
	if err != nil {
		i.log.Error("releaseip", "error", err)
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	ip := &goipam.IP{
		IP:           netip,
		ParentPrefix: req.Msg.GetPrefixCidr(),
	}
	resp, err := i.ipamer.ReleaseIP(ctx, ip)
	if err != nil {
		i.log.Error("releaseip", "error", err)
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(
		&v1.ReleaseIPResponse{
			Ip: &v1.IP{
				Ip:           req.Msg.GetIp(),
				ParentPrefix: resp.ParentCidr,
			},
		},
	), nil
}
func (i *IPAMService) Dump(ctx context.Context, req *connect.Request[v1.DumpRequest]) (*connect.Response[v1.DumpResponse], error) {
	i.log.Debug("dump", "req", req)
	if req.Msg.GetNamespace() != "" {
		ctx = goipam.NewContextWithNamespace(ctx, req.Msg.GetNamespace())
	}
	dump, err := i.ipamer.Dump(ctx)
	if err != nil {
		i.log.Error("dump", "error", err)
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(
		&v1.DumpResponse{
			Dump: dump,
		},
	), nil
}

func (i *IPAMService) Load(ctx context.Context, req *connect.Request[v1.LoadRequest]) (*connect.Response[v1.LoadResponse], error) {
	i.log.Debug("load", "req", req)
	if req.Msg.GetNamespace() != "" {
		ctx = goipam.NewContextWithNamespace(ctx, req.Msg.GetNamespace())
	}
	err := i.ipamer.Load(ctx, req.Msg.GetDump())
	if err != nil {
		i.log.Error("load", "error", err)
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&v1.LoadResponse{}), nil
}
func (i *IPAMService) PrefixUsage(ctx context.Context, req *connect.Request[v1.PrefixUsageRequest]) (*connect.Response[v1.PrefixUsageResponse], error) {
	i.log.Debug("prefixusage", "req", req)
	if req.Msg.GetNamespace() != "" {
		ctx = goipam.NewContextWithNamespace(ctx, req.Msg.GetNamespace())
	}
	p, err := i.ipamer.PrefixFrom(ctx, req.Msg.GetCidr())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("prefix:%q not parsable:%s", req.Msg.GetCidr(), err.Error()))
	}
	if p == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("prefix:%q not found", req.Msg.GetCidr()))
	}
	u := p.Usage()
	return connect.NewResponse(
		&v1.PrefixUsageResponse{
			AvailableIps:              u.AvailableIPs,
			AcquiredIps:               u.AcquiredIPs,
			AvailableSmallestPrefixes: u.AvailableSmallestPrefixes,
			AvailablePrefixes:         u.AvailablePrefixes,
			AcquiredPrefixes:          u.AcquiredPrefixes,
		},
	), nil
}

func (i *IPAMService) CreateNamespace(ctx context.Context, req *connect.Request[v1.CreateNamespaceRequest]) (*connect.Response[v1.CreateNamespaceResponse], error) {
	i.log.Debug("createnamespace", "req", req)
	err := i.ipamer.CreateNamespace(ctx, req.Msg.GetNamespace())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.CreateNamespaceResponse{}), nil
}

func (i *IPAMService) DeleteNamespace(ctx context.Context, req *connect.Request[v1.DeleteNamespaceRequest]) (*connect.Response[v1.DeleteNamespaceResponse], error) {
	i.log.Debug("deletenamespace", "req", req)
	err := i.ipamer.DeleteNamespace(ctx, req.Msg.GetNamespace())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.DeleteNamespaceResponse{}), nil
}

func (i *IPAMService) ListNamespaces(ctx context.Context, req *connect.Request[v1.ListNamespacesRequest]) (*connect.Response[v1.ListNamespacesResponse], error) {
	i.log.Debug("", "req", req)
	res, err := i.ipamer.ListNamespaces(ctx)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(
		&v1.ListNamespacesResponse{
			Namespace: res,
		},
	), nil
}
