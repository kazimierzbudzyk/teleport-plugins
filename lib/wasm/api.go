/*
Copyright 2015-2022 Gravitational, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package wasm

import (
	"context"

	"github.com/gravitational/teleport/api/types"
	"github.com/gravitational/trace"
	log "github.com/sirupsen/logrus"
	"github.com/wasmerio/wasmer-go/wasmer"
)

// TeleportClient represents interface to Teleport client API wrapper
type TeleportClient interface {
	UpsertLock(context.Context, types.Lock) error
}

// TeleportAPI represents Teleport API functions
type TeleportAPI struct {
	log    log.FieldLogger
	traits []*TeleportAPITrait
	client TeleportClient
	pb     *ProtobufInterop
}

// TeleportAPITrait represents Teleport API functions bound to the specific instance
type TeleportAPITrait struct {
	ec  *ExecutionContext
	api *TeleportAPI
}

// NewTeleportAPI creates new NewTeleportAPI collection instance
func NewTeleportAPI(log log.FieldLogger, client TeleportClient, protobufInterop *ProtobufInterop) *TeleportAPI {
	return &TeleportAPI{log: log, pb: protobufInterop, client: client, traits: make([]*TeleportAPITrait, 0)}
}

// CreateTrait creates TeleportAPITrait
func (e *TeleportAPI) CreateTrait() Trait {
	t := &TeleportAPITrait{api: e}
	e.traits = append(e.traits, t)
	return t
}

// Bind binds TeleportAPITrait to the execution context
func (e *TeleportAPITrait) Bind(ec *ExecutionContext) error {
	e.ec = ec
	return nil
}

// RegisterExports registers protobuf interop exports (nothing in our case)
func (e *TeleportAPITrait) Export(store *wasmer.Store, importObject *wasmer.ImportObject) error {
	importObject.Register("api", map[string]wasmer.IntoExtern{
		"upsertLock": wasmer.NewFunction(store, wasmer.NewFunctionType(
			wasmer.NewValueTypes(wasmer.I32), // lock: DataView
			wasmer.NewValueTypes(),           // void
		), e.upsertLock),
	})
	return nil
}

// upsertLock upserts new lock
func (e *TeleportAPITrait) upsertLock(args []wasmer.Value) ([]wasmer.Value, error) {
	lock := &types.LockV2{}

	handle := args[0].I32()

	ec := e.api.pb.For(e.ec)
	if ec == nil {
		return nil, trace.Errorf("API trait not found for execution context %v", ec)
	}

	err := ec.ReceiveMessage(handle, lock)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	err = lock.CheckAndSetDefaults()
	if err != nil {
		return nil, trace.Wrap(err)
	}

	err = e.api.client.UpsertLock(e.ec.currentContext, lock)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	return []wasmer.Value{}, nil
}
