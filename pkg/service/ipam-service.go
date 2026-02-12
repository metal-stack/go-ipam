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

func (i *IPAMService) Version(context.Context, *v1.VersionRequest) (*v1.VersionResponse, error) {
	return &v1.VersionResponse{
		Version:   v.Version,
		Revision:  v.Revision,
		GitSha1:   v.GitSHA1,
		BuildDate: v.BuildDate,
	}, nil
}

func (i *IPAMService) CreatePrefix(ctx context.Context, req *v1.CreatePrefixRequest) (*v1.CreatePrefixResponse, error) {
	if req.GetNamespace() != "" {
		ctx = goipam.NewContextWithNamespace(ctx, req.GetNamespace())
	}
	resp, err := i.ipamer.NewPrefix(ctx, req.GetCidr())
	if err != nil {
		if errors.Is(err, goipam.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return &v1.CreatePrefixResponse{
		Prefix: &v1.Prefix{
			Cidr:       resp.Cidr,
			ParentCidr: resp.ParentCidr,
		},
	}, nil
}
func (i *IPAMService) DeletePrefix(ctx context.Context, req *v1.DeletePrefixRequest) (*v1.DeletePrefixResponse, error) {
	if req.GetNamespace() != "" {
		ctx = goipam.NewContextWithNamespace(ctx, req.GetNamespace())
	}
	resp, err := i.ipamer.DeletePrefix(ctx, req.GetCidr())
	if err != nil {
		if errors.Is(err, goipam.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	return &v1.DeletePrefixResponse{
		Prefix: &v1.Prefix{
			Cidr:       resp.Cidr,
			ParentCidr: resp.ParentCidr,
		},
	}, nil
}
func (i *IPAMService) GetPrefix(ctx context.Context, req *v1.GetPrefixRequest) (*v1.GetPrefixResponse, error) {
	if req.GetNamespace() != "" {
		ctx = goipam.NewContextWithNamespace(ctx, req.GetNamespace())
	}
	resp, err := i.ipamer.PrefixFrom(ctx, req.GetCidr())
	if err != nil {
		if errors.Is(err, goipam.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	return &v1.GetPrefixResponse{
		Prefix: &v1.Prefix{
			Cidr:       resp.Cidr,
			ParentCidr: resp.ParentCidr,
		},
	}, nil
}
func (i *IPAMService) ListPrefixes(ctx context.Context, req *v1.ListPrefixesRequest) (*v1.ListPrefixesResponse, error) {
	if req.GetNamespace() != "" {
		ctx = goipam.NewContextWithNamespace(ctx, req.GetNamespace())
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
	return &v1.ListPrefixesResponse{
		Prefixes: result,
	}, nil
}

func (i *IPAMService) AcquireChildPrefix(ctx context.Context, req *v1.AcquireChildPrefixRequest) (*v1.AcquireChildPrefixResponse, error) {
	if req.GetNamespace() != "" {
		ctx = goipam.NewContextWithNamespace(ctx, req.GetNamespace())
	}
	var (
		resp       *goipam.Prefix
		err        error
		parentCidr = req.GetCidr()
		childCidr  = req.GetChildCidr()
		length     = req.GetLength()
	)
	if length > 128 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("length must not be greater than 128"))
	}
	if req.GetChildCidr() != "" {
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
	return &v1.AcquireChildPrefixResponse{
		Prefix: &v1.Prefix{
			Cidr:       resp.Cidr,
			ParentCidr: resp.ParentCidr,
		},
	}, nil
}

func (i *IPAMService) ReleaseChildPrefix(ctx context.Context, req *v1.ReleaseChildPrefixRequest) (*v1.ReleaseChildPrefixResponse, error) {
	if req.GetNamespace() != "" {
		ctx = goipam.NewContextWithNamespace(ctx, req.GetNamespace())
	}
	prefix, err := i.ipamer.PrefixFrom(ctx, req.GetCidr())
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
	return &v1.ReleaseChildPrefixResponse{
		Prefix: &v1.Prefix{
			Cidr:       prefix.Cidr,
			ParentCidr: prefix.ParentCidr,
		},
	}, nil
}

func (i *IPAMService) AcquireIP(ctx context.Context, req *v1.AcquireIPRequest) (*v1.AcquireIPResponse, error) {
	if req.GetNamespace() != "" {
		ctx = goipam.NewContextWithNamespace(ctx, req.GetNamespace())
	}
	var resp *goipam.IP
	var err error
	if req.GetIp() != "" {
		resp, err = i.ipamer.AcquireSpecificIP(ctx, req.GetPrefixCidr(), req.GetIp())
		if err != nil {
			if errors.Is(err, goipam.ErrAlreadyAllocated) {
				return nil, connect.NewError(connect.CodeAlreadyExists, err)
			}
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
	} else {
		resp, err = i.ipamer.AcquireIP(ctx, req.GetPrefixCidr())
		if err != nil {
			if errors.Is(err, goipam.ErrNoIPAvailable) {
				return nil, connect.NewError(connect.CodeNotFound, err)
			}
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
	}
	return &v1.AcquireIPResponse{
		Ip: &v1.IP{
			Ip:           resp.IP.String(),
			ParentPrefix: resp.ParentPrefix,
		},
	}, nil
}
func (i *IPAMService) ReleaseIP(ctx context.Context, req *v1.ReleaseIPRequest) (*v1.ReleaseIPResponse, error) {
	if req.GetNamespace() != "" {
		ctx = goipam.NewContextWithNamespace(ctx, req.GetNamespace())
	}
	netip, err := netip.ParseAddr(req.GetIp())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	ip := &goipam.IP{
		IP:           netip,
		ParentPrefix: req.GetPrefixCidr(),
	}
	resp, err := i.ipamer.ReleaseIP(ctx, ip)
	if err != nil {
		if errors.Is(err, goipam.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return &v1.ReleaseIPResponse{
		Ip: &v1.IP{
			Ip:           req.GetIp(),
			ParentPrefix: resp.ParentCidr,
		},
	}, nil
}
func (i *IPAMService) Dump(ctx context.Context, req *v1.DumpRequest) (*v1.DumpResponse, error) {
	if req.GetNamespace() != "" {
		ctx = goipam.NewContextWithNamespace(ctx, req.GetNamespace())
	}
	dump, err := i.ipamer.Dump(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return &v1.DumpResponse{
		Dump: dump,
	}, nil
}

func (i *IPAMService) Load(ctx context.Context, req *v1.LoadRequest) (*v1.LoadResponse, error) {
	if req.GetNamespace() != "" {
		ctx = goipam.NewContextWithNamespace(ctx, req.GetNamespace())
	}
	err := i.ipamer.Load(ctx, req.GetDump())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return &v1.LoadResponse{}, nil
}
func (i *IPAMService) PrefixUsage(ctx context.Context, req *v1.PrefixUsageRequest) (*v1.PrefixUsageResponse, error) {
	if req.GetNamespace() != "" {
		ctx = goipam.NewContextWithNamespace(ctx, req.GetNamespace())
	}
	p, err := i.ipamer.PrefixFrom(ctx, req.GetCidr())
	if err != nil {
		if errors.Is(err, goipam.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	u := p.Usage()
	return &v1.PrefixUsageResponse{
		AvailableIps:              u.AvailableIPs,
		AcquiredIps:               u.AcquiredIPs,
		AvailableSmallestPrefixes: u.AvailableSmallestPrefixes,
		AvailablePrefixes:         u.AvailablePrefixes,
		AcquiredPrefixes:          u.AcquiredPrefixes,
	}, nil
}

func (i *IPAMService) CreateNamespace(ctx context.Context, req *v1.CreateNamespaceRequest) (*v1.CreateNamespaceResponse, error) {
	err := i.ipamer.CreateNamespace(ctx, req.GetNamespace())
	if err != nil {
		return nil, err
	}
	return &v1.CreateNamespaceResponse{}, nil
}

func (i *IPAMService) DeleteNamespace(ctx context.Context, req *v1.DeleteNamespaceRequest) (*v1.DeleteNamespaceResponse, error) {
	err := i.ipamer.DeleteNamespace(ctx, req.GetNamespace())
	if err != nil {
		return nil, err
	}
	return &v1.DeleteNamespaceResponse{}, nil
}

func (i *IPAMService) ListNamespaces(ctx context.Context, req *v1.ListNamespacesRequest) (*v1.ListNamespacesResponse, error) {
	res, err := i.ipamer.ListNamespaces(ctx)
	if err != nil {
		return nil, err
	}
	return &v1.ListNamespacesResponse{
		Namespace: res,
	}, nil
}
