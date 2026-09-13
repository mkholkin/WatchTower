package mapper_test

import (
	"WatchTower/internal/testutil"
	"WatchTower/pkg/mapper"
	"errors"
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"github.com/stretchr/testify/require"
	"strconv"
	"testing"
)

func TestMapper_Convert(t *testing.T) {
	for _, tc := range []struct {
		name, key, input string
		bad              bool
	}{{"valid", "integer", "42", false}, {"unknown", "other", "42", true}, {"invalid_value", "integer", "x", true}} {
		t.Run(tc.name, func(t *testing.T) {
			testutil.Case(t, "Mapper", "Convert", "equivalence", "classic", func(t *testing.T, a *allure.Context) {
				var m mapper.Mapper[string, int, string]
				var got int
				var err error
				a.Step("Arrange", func(*allure.Context) { m = mapper.New[string, int, string](); m.Register("integer", strconv.Atoi) })
				a.Step("Act", func(*allure.Context) { got, err = m.Convert(tc.key, tc.input) })
				a.Step("Assert", func(*allure.Context) {
					if tc.bad {
						require.Error(t, err)
						if tc.name == "unknown" {
							require.ErrorIs(t, err, mapper.ErrUnknown)
						} else {
							var num *strconv.NumError
							require.True(t, errors.As(err, &num))
						}
					} else {
						require.NoError(t, err)
						require.Equal(t, 42, got)
					}
				})
			})
		})
	}
}
func TestMapper_Register(t *testing.T) {
	for _, replace := range []bool{false, true} {
		t.Run(map[bool]string{false: "new_key", true: "replace_key"}[replace], func(t *testing.T) {
			testutil.Case(t, "Mapper", "Register", "state-transition", "classic", func(t *testing.T, a *allure.Context) {
				var m mapper.Mapper[string, int, string]
				a.Step("Arrange", func(*allure.Context) {
					m = mapper.New[string, int, string]()
					if replace {
						m.Register("integer", func(string) (int, error) { return -1, nil })
					}
				})
				a.Step("Act", func(*allure.Context) { m.Register("integer", strconv.Atoi) })
				a.Step("Assert", func(*allure.Context) {
					got, err := m.Convert("integer", "42")
					require.NoError(t, err)
					require.Equal(t, 42, got)
				})
			})
		})
	}
}
