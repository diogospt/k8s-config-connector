// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mockcompute

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/GoogleCloudPlatform/k8s-config-connector/mockgcp/common/projects"
	pb "github.com/GoogleCloudPlatform/k8s-config-connector/mockgcp/generated/mockgcp/cloud/compute/v1"
)

type NetworksV1 struct {
	*MockService
	pb.UnimplementedNetworksServer
}

func (s *NetworksV1) Get(ctx context.Context, req *pb.GetNetworkRequest) (*pb.Network, error) {
	name, err := s.newNetworkName(req.GetProject(), req.GetNetwork())
	if err != nil {
		return nil, err
	}

	fqn := name.String()

	obj := &pb.Network{}
	if err := s.storage.Get(ctx, fqn, obj); err != nil {
		return nil, err
	}

	return obj, nil
}

func (s *NetworksV1) Insert(ctx context.Context, req *pb.InsertNetworkRequest) (*pb.Operation, error) {
	name, err := s.newNetworkName(req.GetProject(), req.GetNetworkResource().GetName())
	if err != nil {
		return nil, err
	}

	fqn := name.String()

	id := s.generateID()

	obj := proto.Clone(req.GetNetworkResource()).(*pb.Network)
	obj.SelfLink = PtrTo("https://compute.googleapis.com/compute/v1/" + name.String())
	obj.CreationTimestamp = PtrTo(s.nowString())
	obj.Id = &id
	obj.SelfLinkWithId = PtrTo(fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/global/networks/%d", name.Project.ID, id))
	obj.Kind = PtrTo("compute#network")

	if err := s.storage.Create(ctx, fqn, obj); err != nil {
		return nil, status.Errorf(codes.Internal, "error creating network: %v", err)
	}

	return s.newLRO(ctx, name.Project.ID)
}

// Patches the specified network with the data included in the request.
// Only the following fields can be modified: routingConfig.routingMode.
func (s *NetworksV1) Patch(ctx context.Context, req *pb.PatchNetworkRequest) (*pb.Operation, error) {
	name, err := s.newNetworkName(req.GetProject(), req.GetNetwork())
	if err != nil {
		return nil, err
	}

	fqn := name.String()
	obj := &pb.Network{}
	if err := s.storage.Get(ctx, fqn, obj); err != nil {
		return nil, err
	}

	if req.GetNetworkResource().RoutingConfig != nil {
		if req.GetNetworkResource().GetRoutingConfig().RoutingMode != nil {
			if obj.RoutingConfig == nil {
				obj.RoutingConfig = &pb.NetworkRoutingConfig{}
			}
			obj.RoutingConfig.RoutingMode = req.GetNetworkResource().GetRoutingConfig().RoutingMode
		}
	}

	if err := s.storage.Update(ctx, fqn, obj); err != nil {
		return nil, err
	}

	return s.newLRO(ctx, name.Project.ID)
}

func (s *NetworksV1) Delete(ctx context.Context, req *pb.DeleteNetworkRequest) (*pb.Operation, error) {
	name, err := s.newNetworkName(req.GetProject(), req.GetNetwork())
	if err != nil {
		return nil, err
	}

	fqn := name.String()

	deleted := &pb.Network{}
	if err := s.storage.Delete(ctx, fqn, deleted); err != nil {
		return nil, err
	}

	return s.newLRO(ctx, name.Project.ID)
}

type networkName struct {
	Project *projects.ProjectData
	Name    string
}

func (n *networkName) String() string {
	return "projects/" + n.Project.ID + "/global" + "/networks/" + n.Name
}

// parseNetworkName parses a string into a networkName.
// The expected form is `projects/*/global/networks/*`.
func (s *MockService) parseNetworkName(name string) (*networkName, error) {
	tokens := strings.Split(name, "/")

	if len(tokens) == 5 && tokens[0] == "projects" && tokens[2] == "global" && tokens[3] == "networks" {
		return s.newNetworkName(tokens[1], tokens[4])
	} else {
		return nil, status.Errorf(codes.InvalidArgument, "name %q is not valid", name)
	}
}

// newNetworkName builds a normalized networkName from the constituent parts.
func (s *MockService) newNetworkName(project string, name string) (*networkName, error) {
	projectObj, err := s.projects.GetProjectByID(project)
	if err != nil {
		return nil, err
	}

	return &networkName{
		Project: projectObj,
		Name:    name,
	}, nil
}
