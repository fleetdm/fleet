This package contains API Android tests with the real Android service and Android MySQL database.

We use testify Suite to run these tests. Since [testify Suite does not support parallel execution](https://github.com/stretchr/testify/issues/187),
we put each test in their own package/directory. This allows these tests to run in parallel because each package is a separate compile unit. If you
create a large test, please put it in a separate file within the same Suite/package.
