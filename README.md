Certainly, here is the `README.md` file content in Markdown format, updated to reflect the Apache 2.0 license.

```markdown
# dagor-grpc

## Overview

`dagor-grpc` is a gRPC interceptor library designed to extend the functionalities of gRPC servers and clients. The library aims to enhance gRPC communications by introducing features such as logging, authentication, and more.

## Installation

To install the `dagor-grpc` package, run the following command:

```bash
go get -u github.com/Jiali-Xing/dagor-grpc
```

Ensure that you have Go installed and your workspace is correctly configured.

## Usage

### Server Interceptor

To use `dagor` as a server interceptor, refer to the following example:

```go
import (
  "github.com/Jiali-Xing/dagor-grpc/dagor"
  "google.golang.org/grpc"
)

func main() {
  serverOptions := []grpc.ServerOption{
    grpc.UnaryInterceptor(dagor.UnaryServerInterceptor),
  }

  server := grpc.NewServer(serverOptions...)
  // Register services and start the server
}
```

### Client Interceptor

To integrate `dagor` as a client interceptor, consult the following example:

```go
import (
  "github.com/Jiali-Xing/dagor-grpc/dagor"
  "google.golang.org/grpc"
)

func main() {
  clientOptions := []grpc.DialOption{
    grpc.WithUnaryInterceptor(dagor.UnaryClientInterceptor),
  }

  conn, err := grpc.Dial("localhost:50051", clientOptions...)
  // Handle the connection and execute client logic
}
```

## Contributing

Contributions from the community are welcome. For more information, please read the [contribution guidelines](CONTRIBUTING.md).

## License

This project is licensed under the Apache License 2.0. See the [LICENSE](LICENSE) file for details.
```

You can place this content into your `README.md` file in the root directory of your repository. Feel free to adjust it according to your project's specific requirements.