package service

import (
	"context"
	"errors"
	"log/slog"

	"net/netip"

	"connectrpc.com/connect"
	goipam "github.com/metal-stack/go-ipam"
	v1 "github.com/metal-stack/go-ipam/api/v1"
	"github.com/metal-stack/v"
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

func (i *IPAMService) Version(context.Context, *connect.Request[v1.VersionRequest]) (*connect.Response[v1.VersionResponse], error) {
	return connect.NewResponse(&v1.VersionResponse{
		Version:   v.Version,
		Revision:  v.Revision,
		GitSha1:   v.GitSHA1,
		BuildDate: v.BuildDate,
	}), nil
}

func (i *IPAMService) CreatePrefix(ctx context.Context, req *connect.Request[v1.CreatePrefixRequest]) (*connect.Response[v1.CreatePrefixResponse], error) {
	if req.Msg.GetNamespace() != "" {
		ctx = goipam.NewContextWithNamespace(ctx, req.Msg.GetNamespace())
	}
	resp, err := i.ipamer.NewPrefix(ctx, req.Msg.GetCidr())
	if err != nil {
		if errors.Is(err, goipam.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
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
	if req.Msg.GetNamespace() != "" {
		ctx = goipam.NewContextWithNamespace(ctx, req.Msg.GetNamespace())
	}
	resp, err := i.ipamer.DeletePrefix(ctx, req.Msg.GetCidr())
	if err != nil {
		if errors.Is(err, goipam.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
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
	if req.Msg.GetNamespace() != "" {
		ctx = goipam.NewContextWithNamespace(ctx, req.Msg.GetNamespace())
	}
	resp, err := i.ipamer.PrefixFrom(ctx, req.Msg.GetCidr())
	if err != nil {
		if errors.Is(err, goipam.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
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
	if req.Msg.GetNamespace() != "" {
		ctx = goipam.NewContextWithNamespace(ctx, req.Msg.GetNamespace())
	}
	resp, err := i.ipamer.ReadAllPrefixCidrs(ctx)
	if err != nil {
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
	if req.Msg.GetNamespace() != "" {
		ctx = goipam.NewContextWithNamespace(ctx, req.Msg.GetNamespace())
	}
	var (
		resp       *goipam.Prefix
		err        error
		parentCidr = req.Msg.GetCidr()
		childCidr  = req.Msg.GetChildCidr()
		length     = req.Msg.GetLength()
	)
	if length > 128 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("length must not be greater than 128"))
	}
	if req.Msg.GetChildCidr() != "" {
		resp, err = i.ipamer.AcquireSpecificChildPrefix(ctx, parentCidr, childCidr)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
	} else {
		resp, err = i.ipamer.AcquireChildPrefix(ctx, parentCidr, uint8(length)) // nolint:gosec
		if err != nil {
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
	if req.Msg.GetNamespace() != "" {
		ctx = goipam.NewContextWithNamespace(ctx, req.Msg.GetNamespace())
	}
	prefix, err := i.ipamer.PrefixFrom(ctx, req.Msg.GetCidr())
	if err != nil {
		if errors.Is(err, goipam.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	err = i.ipamer.ReleaseChildPrefix(ctx, prefix)
	if err != nil {
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

// PrefixesOverlapping implements apiv1connect.IpamServiceHandler.
func (*IPAMService) PrefixesOverlapping(_ context.Context, req *connect.Request[v1.PrefixesOverlappingRequest]) (*connect.Response[v1.PrefixesOverlappingResponse], error) {
	err := goipam.PrefixesOverlapping(req.Msg.ExistingPrefixes, req.Msg.NewPrefixes)
	if err != nil {
		return nil, connect.NewError(connect.CodeAlreadyExists, err)
	}

	return connect.NewResponse(
		&v1.PrefixesOverlappingResponse{},
	), nil
}

func (i *IPAMService) AcquireIP(ctx context.Context, req *connect.Request[v1.AcquireIPRequest]) (*connect.Response[v1.AcquireIPResponse], error) {
	if req.Msg.GetNamespace() != "" {
		ctx = goipam.NewContextWithNamespace(ctx, req.Msg.GetNamespace())
	}
	var resp *goipam.IP
	var err error
	if req.Msg.GetIp() != "" {
		resp, err = i.ipamer.AcquireSpecificIP(ctx, req.Msg.GetPrefixCidr(), req.Msg.GetIp())
		if err != nil {
			if errors.Is(err, goipam.ErrAlreadyAllocated) {
				return nil, connect.NewError(connect.CodeAlreadyExists, err)
			}
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
	} else {
		resp, err = i.ipamer.AcquireIP(ctx, req.Msg.GetPrefixCidr())
		if err != nil {
			if errors.Is(err, goipam.ErrNoIPAvailable) {
				return nil, connect.NewError(connect.CodeNotFound, err)
			}
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
	if req.Msg.GetNamespace() != "" {
		ctx = goipam.NewContextWithNamespace(ctx, req.Msg.GetNamespace())
	}
	netip, err := netip.ParseAddr(req.Msg.GetIp())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	ip := &goipam.IP{
		IP:           netip,
		ParentPrefix: req.Msg.GetPrefixCidr(),
	}
	resp, err := i.ipamer.ReleaseIP(ctx, ip)
	if err != nil {
		if errors.Is(err, goipam.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
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
	if req.Msg.GetNamespace() != "" {
		ctx = goipam.NewContextWithNamespace(ctx, req.Msg.GetNamespace())
	}
	dump, err := i.ipamer.Dump(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(
		&v1.DumpResponse{
			Dump: dump,
		},
	), nil
}

func (i *IPAMService) Load(ctx context.Context, req *connect.Request[v1.LoadRequest]) (*connect.Response[v1.LoadResponse], error) {
	if req.Msg.GetNamespace() != "" {
		ctx = goipam.NewContextWithNamespace(ctx, req.Msg.GetNamespace())
	}
	err := i.ipamer.Load(ctx, req.Msg.GetDump())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&v1.LoadResponse{}), nil
}
func (i *IPAMService) PrefixUsage(ctx context.Context, req *connect.Request[v1.PrefixUsageRequest]) (*connect.Response[v1.PrefixUsageResponse], error) {
	if req.Msg.GetNamespace() != "" {
		ctx = goipam.NewContextWithNamespace(ctx, req.Msg.GetNamespace())
	}
	p, err := i.ipamer.PrefixFrom(ctx, req.Msg.GetCidr())
	if err != nil {
		if errors.Is(err, goipam.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
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
	err := i.ipamer.CreateNamespace(ctx, req.Msg.GetNamespace())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.CreateNamespaceResponse{}), nil
}

func (i *IPAMService) DeleteNamespace(ctx context.Context, req *connect.Request[v1.DeleteNamespaceRequest]) (*connect.Response[v1.DeleteNamespaceResponse], error) {
	err := i.ipamer.DeleteNamespace(ctx, req.Msg.GetNamespace())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.DeleteNamespaceResponse{}), nil
}

func (i *IPAMService) ListNamespaces(ctx context.Context, req *connect.Request[v1.ListNamespacesRequest]) (*connect.Response[v1.ListNamespacesResponse], error) {
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
