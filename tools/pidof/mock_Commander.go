// Code generated by mockery v0.0.0-dev. DO NOT EDIT.

package pidof

import mock "github.com/stretchr/testify/mock"

// MockCommander is an autogenerated mock type for the Commander type
type MockCommander struct {
	mock.Mock
}

// Output provides a mock function with given fields: cmd
func (_m *MockCommander) Output(cmd string) ([]byte, error) {
	ret := _m.Called(cmd)

	var r0 []byte
	if rf, ok := ret.Get(0).(func(string) []byte); ok {
		r0 = rf(cmd)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(cmd)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
