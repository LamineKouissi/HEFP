package util

import (
	"errors"
	"testing"
)

func TestIsStructEmpty(t *testing.T) {
	type SimpleStruct struct {
		A int
		B string
	}

	type ComplexStruct struct {
		A int
		B string
		C SimpleStruct
		D []int
		E map[string]int
		F func()
	}

	tests := []struct {
		name         string
		input        interface{}
		expectedRslt bool
		expectedErr  error
	}{
		{
			name:         "Empty simple struct",
			input:        SimpleStruct{},
			expectedRslt: true,
			expectedErr:  nil,
		},
		{
			name: "Non-empty simple struct",
			input: SimpleStruct{
				A: 1,
				B: "hello",
			},
			expectedRslt: false,
			expectedErr:  nil,
		},
		{
			name:         "Empty complex struct",
			input:        ComplexStruct{},
			expectedRslt: true,
			expectedErr:  nil,
		},
		{
			name: "Complex struct with non-empty basic field",
			input: ComplexStruct{
				A: 1,
			},
			expectedRslt: false,
			expectedErr:  nil,
		},
		{
			name: "Complex struct with non-empty nested struct",
			input: ComplexStruct{
				C: SimpleStruct{A: 1},
			},
			expectedRslt: false,
			expectedErr:  nil,
		},
		{
			name: "Complex struct with non-empty slice",
			input: ComplexStruct{
				D: []int{1, 2, 3},
			},
			expectedRslt: false,
			expectedErr:  nil,
		},
		{
			name: "Complex struct with non-empty map",
			input: ComplexStruct{
				E: map[string]int{"a": 1},
			},
			expectedRslt: false,
			expectedErr:  nil,
		},
		{
			name: "Complex struct with non-nil function",
			input: ComplexStruct{
				F: func() {},
			},
			expectedRslt: false,
			expectedErr:  nil,
		},
		{
			name:         "Nil pointer to struct",
			input:        (*ComplexStruct)(nil),
			expectedRslt: false,
			expectedErr:  errors.New("input type is not a struct"),
		},
		{
			name:         "Non-struct type",
			input:        42,
			expectedRslt: false,
			expectedErr:  errors.New("input type is not a struct"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := IsStructEmpty(tt.input)
			if result != tt.expectedRslt && err != tt.expectedErr {
				t.Errorf("isStructEmpty(%v) = [%v, %v], want [%v, %v]", tt.input, result, err, tt.expectedRslt, tt.expectedErr)
			}
		})
	}
}
