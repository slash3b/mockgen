## mockgen

1. Install this package with `go install github.com/slash3b/mockgen@latest`
2. run `go generate ./...`


example file bar.go:
```golang
package bar

//go:generate mockgen

type Jest interface {
        Sum(i, j int) (int, error)
}

type Stringer interface {
        String() string
}
```

result file bar_test.go:
```golang
// Auto-generated. Do Not Edit!
package bar

import (
        "github.com/stretchr/testify/mock"
)

var _ Stringer = (*StringerMock)(nil)

type StringerMock struct {
        mock.Mock
}

func (sm *StringerMock) String() string {
        args := sm.Called()

        return args.String(0)
}

var _ Jest = (*JestMock)(nil)

type JestMock struct {
        mock.Mock
}

func (jm *JestMock) Sum(i int, j int) (int, error) {
        args := jm.Called(i, j)

        return args.Int(0), args.Error(1)
}

```
