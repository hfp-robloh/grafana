package rest

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/zeebo/assert"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/apis/example"
)

const kind = "dummy"

var exampleObj = &example.Pod{TypeMeta: metav1.TypeMeta{}, ObjectMeta: metav1.ObjectMeta{Name: "foo"}, Spec: example.PodSpec{}, Status: example.PodStatus{}}

func TestMode1_Create(t *testing.T) {
	type testCase struct {
		name          string
		setupLegacyFn func(m *mock.Mock)
		setupUSFn     func(m *mock.Mock)
		wantErr       bool
	}
	tests :=
		[]testCase{
			{
				name: "creating an object only in the legacy store",
				setupLegacyFn: func(m *mock.Mock) {
					m.On("Create", context.Background(), exampleObj, mock.Anything, mock.Anything).Return(exampleObj, nil)
				},
				setupUSFn: func(m *mock.Mock) {
					m.On("Create", context.Background(), exampleObj, mock.Anything, mock.Anything).Return(exampleObj, nil)
				},
			},
			{
				name: "error when creating object in the legacy store fails",
				setupLegacyFn: func(m *mock.Mock) {
					m.On("Create", context.Background(), exampleObj, mock.Anything, mock.Anything).Return(exampleObj, errors.New("error"))
				},
			},
		}

	for _, tt := range tests {
		l := (LegacyStorage)(nil)
		s := (Storage)(nil)
		m := &mock.Mock{}

		ls := legacyStoreMock{m, l}
		us := unifiedStoreMock{m, s}

		if tt.setupLegacyFn != nil {
			tt.setupLegacyFn(m)
		}
		if tt.setupUSFn != nil {
			tt.setupUSFn(m)
		}

		dw := SelectDualWriter(Mode1, ls, us)

		obj, err := dw.Create(context.Background(), exampleObj, func(context.Context, runtime.Object) error { return nil }, &metav1.CreateOptions{})

		if tt.wantErr {
			assert.Error(t, err)
			continue
		}

		us.AssertNotCalled(t, "Create", context.Background(), exampleObj, func(context.Context, runtime.Object) error { return nil }, &metav1.CreateOptions{})

		assert.Equal(t, obj, exampleObj)
	}
}

func TestMode1_Get(t *testing.T) {
	type testCase struct {
		name          string
		input         string
		setupLegacyFn func(m *mock.Mock, name string)
		setupUSFn     func(m *mock.Mock, name string)
		wantErr       bool
	}
	tests :=
		[]testCase{
			{
				name:  "get an object only in the legacy store",
				input: "foo",
				setupLegacyFn: func(m *mock.Mock, name string) {
					m.On("Get", context.Background(), name, mock.Anything).Return(exampleObj, nil)
				},
				setupUSFn: func(m *mock.Mock, name string) {
					m.On("Get", context.Background(), name, mock.Anything).Return(exampleObj, nil)
				},
			},
			{
				name: "error when getting an object in the legacy store fails",
				setupLegacyFn: func(m *mock.Mock, name string) {
					m.On("Get", context.Background(), name, mock.Anything).Return(exampleObj, errors.New("error"))
				},
			},
		}

	for _, tt := range tests {
		l := (LegacyStorage)(nil)
		s := (Storage)(nil)
		m := &mock.Mock{}

		ls := legacyStoreMock{m, l}
		us := unifiedStoreMock{m, s}

		if tt.setupLegacyFn != nil {
			tt.setupLegacyFn(m, tt.input)
		}
		if tt.setupUSFn != nil {
			tt.setupUSFn(m, tt.input)
		}

		dw := SelectDualWriter(Mode1, ls, us)

		obj, err := dw.Get(context.Background(), tt.input, &metav1.GetOptions{})

		if tt.wantErr {
			assert.Error(t, err)
			continue
		}

		us.AssertNotCalled(t, "Get", context.Background(), tt.name, &metav1.GetOptions{})

		assert.Equal(t, obj, exampleObj)
	}
}

func TestMode1_List(t *testing.T) {
	type testCase struct {
		name          string
		setupLegacyFn func(m *mock.Mock)
		setupUSFn     func(m *mock.Mock)
		wantErr       bool
	}
	tests :=
		[]testCase{
			{
				name: "error when listing an object in the legacy store is not implemented",
				setupLegacyFn: func(m *mock.Mock) {
					m.On("List", context.Background(), mock.Anything).Return(nil, errors.New("error"))
				},
			},
			// TODO: legacy list is missing
		}

	for _, tt := range tests {
		l := (LegacyStorage)(nil)
		s := (Storage)(nil)
		m := &mock.Mock{}

		ls := legacyStoreMock{m, l}
		us := unifiedStoreMock{m, s}

		if tt.setupLegacyFn != nil {
			tt.setupLegacyFn(m)
		}
		if tt.setupUSFn != nil {
			tt.setupUSFn(m)
		}

		dw := SelectDualWriter(Mode1, ls, us)

		_, err := dw.List(context.Background(), &metainternalversion.ListOptions{})

		if tt.wantErr {
			assert.Error(t, err)
			continue
		}

		us.AssertNotCalled(t, "List", context.Background(), &metainternalversion.ListOptions{})
	}
}

func TestMode1_Delete(t *testing.T) {
	type testCase struct {
		name          string
		input         string
		setupLegacyFn func(m *mock.Mock, name string)
		setupUSFn     func(m *mock.Mock, name string)
		wantErr       bool
	}
	tests :=
		[]testCase{
			{
				name: "error when deleting an object in the legacy store",
				setupLegacyFn: func(m *mock.Mock, name string) {
					m.On("Delete", context.Background(), name, mock.Anything, mock.Anything).Return(exampleObj, false, nil)
				},
				setupUSFn: func(m *mock.Mock, name string) {
					m.On("Delete", context.Background(), name, mock.Anything, mock.Anything).Return(exampleObj, false, nil)
				},
			},
		}

	for _, tt := range tests {
		l := (LegacyStorage)(nil)
		s := (Storage)(nil)
		m := &mock.Mock{}

		ls := legacyStoreMock{m, l}
		us := unifiedStoreMock{m, s}

		if tt.setupLegacyFn != nil {
			tt.setupLegacyFn(m, tt.input)
		}
		if tt.setupUSFn != nil {
			tt.setupUSFn(m, tt.input)
		}

		dw := SelectDualWriter(Mode1, ls, us)

		_, _, err := dw.Delete(context.Background(), tt.input, func(ctx context.Context, obj runtime.Object) error { return nil }, &metav1.DeleteOptions{})

		if tt.wantErr {
			assert.Error(t, err)
			continue
		}

		us.AssertNotCalled(t, "Delete", context.Background(), tt.input, func(ctx context.Context, obj runtime.Object) error { return nil }, &metav1.DeleteOptions{})
	}
}
