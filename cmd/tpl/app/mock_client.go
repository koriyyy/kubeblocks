/*
Copyright ApeCloud, Inc.

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

package app

import (
	"context"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cfgcore "github.com/apecloud/kubeblocks/internal/configuration"
	testutil "github.com/apecloud/kubeblocks/internal/testutil/k8s"
)

type mockClient struct {
	objects        map[client.ObjectKey]client.Object
	kindObjectList map[string][]runtime.Object
}

func newMockClient(objs []client.Object) client.Client {
	return &mockClient{
		objects:        fromObjects(objs),
		kindObjectList: splitRuntimeObject(objs),
	}
}

func fromObjects(objs []client.Object) map[client.ObjectKey]client.Object {
	r := make(map[client.ObjectKey]client.Object)
	for _, obj := range objs {
		if obj != nil {
			r[client.ObjectKeyFromObject(obj)] = obj
		}
	}
	return r
}

func splitRuntimeObject(objects []client.Object) map[string][]runtime.Object {
	r := make(map[string][]runtime.Object)
	for _, object := range objects {
		kind := object.GetObjectKind().GroupVersionKind().Kind
		if _, ok := r[kind]; !ok {
			r[kind] = make([]runtime.Object, 0)
		}
		r[kind] = append(r[kind], object)
	}
	return r
}

func (m *mockClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	objKey := key
	if object, ok := m.objects[objKey]; ok {
		testutil.SetGetReturnedObject(obj, object)
		return nil
	}
	objKey.Namespace = ""
	if object, ok := m.objects[objKey]; ok {
		testutil.SetGetReturnedObject(obj, object)
	}
	return nil
}

func (m *mockClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	r := m.kindObjectList[list.GetObjectKind().GroupVersionKind().Kind]
	if r != nil {
		return testutil.SetListReturnedObjects(list, r)
	}
	return nil
}

func (m mockClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	return cfgcore.MakeError("not support")
}

func (m mockClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	return cfgcore.MakeError("not support")
}

func (m mockClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	return cfgcore.MakeError("not support")
}

func (m mockClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	return cfgcore.MakeError("not support")
}

func (m mockClient) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	return cfgcore.MakeError("not support")
}

func (m mockClient) Status() client.SubResourceWriter {
	panic("implement me")
}

func (m mockClient) SubResource(subResource string) client.SubResourceClient {
	panic("implement me")
}

func (m mockClient) Scheme() *runtime.Scheme {
	panic("implement me")
}

func (m mockClient) RESTMapper() meta.RESTMapper {
	panic("implement me")
}